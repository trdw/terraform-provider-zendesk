package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TargetResource{}
	_ resource.ResourceWithImportState = &TargetResource{}
)

type TargetResource struct {
	client *ZendeskClient
}

type TargetResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Type      types.String `tfsdk:"type"`
	Email     types.String `tfsdk:"email"`
	Subject   types.String `tfsdk:"subject"`
	Active    types.Bool   `tfsdk:"active"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type targetAPIObject struct {
	ID        int64  `json:"id,omitempty"`
	Title     string `json:"title"`
	Type      string `json:"type,omitempty"`
	Email     string `json:"email,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Active    *bool  `json:"active,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type targetWrapper struct {
	Target targetAPIObject `json:"target"`
}

func NewTargetResource() resource.Resource {
	return &TargetResource{}
}

func (r *TargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *TargetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk email target. Targets are notified by triggers and " +
			"automations via a notification_target action. Note: Zendesk has deprecated URL and " +
			"branded targets in favor of webhooks; email targets remain supported, so this resource " +
			"defaults to and is intended for the email_target type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The name of the target.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The target type. Defaults to \"email_target\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "The email address to notify.",
			},
			"subject": schema.StringAttribute{
				Required:    true,
				Description: "The subject line of the notification email.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the target is active.",
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := targetWrapper{Target: buildTargetAPI(&plan)}
	var result targetWrapper
	if err := r.client.Post("/api/v2/targets", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating target", err.Error())
		return
	}

	mapTargetToState(&result.Target, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result targetWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/targets/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading target", err.Error())
		return
	}

	mapTargetToState(&result.Target, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TargetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := targetWrapper{Target: buildTargetAPI(&plan)}
	var result targetWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/targets/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating target", err.Error())
		return
	}

	mapTargetToState(&result.Target, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TargetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/targets/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting target", err.Error())
		return
	}
}

func (r *TargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildTargetAPI(plan *TargetResourceModel) targetAPIObject {
	obj := targetAPIObject{
		Title:   plan.Title.ValueString(),
		Email:   plan.Email.ValueString(),
		Subject: plan.Subject.ValueString(),
	}
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() && plan.Type.ValueString() != "" {
		obj.Type = plan.Type.ValueString()
	} else {
		obj.Type = "email_target"
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	return obj
}

func mapTargetToState(t *targetAPIObject, m *TargetResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(t.ID, 10))
	m.Title = types.StringValue(t.Title)
	m.Type = types.StringValue(t.Type)
	m.Email = types.StringValue(t.Email)
	m.Subject = types.StringValue(t.Subject)
	if t.Active != nil {
		m.Active = types.BoolValue(*t.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	m.CreatedAt = types.StringValue(t.CreatedAt)
}
