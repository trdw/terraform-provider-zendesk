package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &GroupMembershipResource{}
	_ resource.ResourceWithImportState = &GroupMembershipResource{}
)

type GroupMembershipResource struct {
	client *ZendeskClient
}

type GroupMembershipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	GroupID   types.Int64  `tfsdk:"group_id"`
	Default   types.Bool   `tfsdk:"default"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type groupMembershipAPIObject struct {
	ID        int64  `json:"id,omitempty"`
	UserID    int64  `json:"user_id"`
	GroupID   int64  `json:"group_id"`
	Default   *bool  `json:"default,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type groupMembershipWrapper struct {
	GroupMembership groupMembershipAPIObject `json:"group_membership"`
}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

func (r *GroupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk group membership. Group memberships are immutable — changing user_id or group_id forces recreation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the user (agent).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.Int64Attribute{
				Required:    true,
				Description: "The ID of the group.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, tickets assigned directly to the agent will assume this membership's group.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ZendeskClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *ZendeskClient")
		return
	}
	r.client = client
}

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := groupMembershipWrapper{
		GroupMembership: groupMembershipAPIObject{
			UserID:  plan.UserID.ValueInt64(),
			GroupID: plan.GroupID.ValueInt64(),
		},
	}
	if !plan.Default.IsNull() && !plan.Default.IsUnknown() {
		v := plan.Default.ValueBool()
		apiReq.GroupMembership.Default = &v
	}

	var result groupMembershipWrapper
	if err := r.client.Post("/api/v2/group_memberships", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating group membership", err.Error())
		return
	}

	mapGroupMembershipToState(&result.GroupMembership, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result groupMembershipWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/group_memberships/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group membership", err.Error())
		return
	}

	mapGroupMembershipToState(&result.GroupMembership, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Group memberships cannot be updated via API; changes to user_id/group_id
	// trigger replacement via RequiresReplace plan modifiers.
	resp.Diagnostics.AddError(
		"Group memberships cannot be updated in place",
		"Delete and recreate the group membership instead.",
	)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/group_memberships/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting group membership", err.Error())
		return
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapGroupMembershipToState(gm *groupMembershipAPIObject, m *GroupMembershipResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(gm.ID, 10))
	m.UserID = types.Int64Value(gm.UserID)
	m.GroupID = types.Int64Value(gm.GroupID)
	if gm.Default != nil {
		m.Default = types.BoolValue(*gm.Default)
	} else {
		m.Default = types.BoolValue(false)
	}
	m.CreatedAt = types.StringValue(gm.CreatedAt)
	m.UpdatedAt = types.StringValue(gm.UpdatedAt)
}
