package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TriggerCategoryResource{}
	_ resource.ResourceWithImportState = &TriggerCategoryResource{}
)

type TriggerCategoryResource struct {
	client *ZendeskClient
}

type TriggerCategoryResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Position  types.Int64  `tfsdk:"position"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewTriggerCategoryResource() resource.Resource {
	return &TriggerCategoryResource{}
}

func (r *TriggerCategoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger_category"
}

func (r *TriggerCategoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk ticket trigger category.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the trigger category.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of the trigger category.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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

func (r *TriggerCategoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TriggerCategoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TriggerCategoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := triggerCategoryWrapper{
		TriggerCategory: triggerCategoryAPI{
			Name: plan.Name.ValueString(),
		},
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		apiReq.TriggerCategory.Position = plan.Position.ValueInt64()
	}

	var result triggerCategoryWrapper
	if err := r.client.Post("/api/v2/trigger_categories", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating trigger category", err.Error())
		return
	}

	mapTriggerCategoryToState(&result.TriggerCategory, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerCategoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TriggerCategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result triggerCategoryWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/trigger_categories/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading trigger category", err.Error())
		return
	}

	mapTriggerCategoryToState(&result.TriggerCategory, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TriggerCategoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TriggerCategoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TriggerCategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := triggerCategoryWrapper{
		TriggerCategory: triggerCategoryAPI{
			Name: plan.Name.ValueString(),
		},
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		apiReq.TriggerCategory.Position = plan.Position.ValueInt64()
	}

	var result triggerCategoryWrapper
	if err := r.client.Patch(fmt.Sprintf("/api/v2/trigger_categories/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating trigger category", err.Error())
		return
	}

	mapTriggerCategoryToState(&result.TriggerCategory, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerCategoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TriggerCategoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/trigger_categories/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting trigger category", err.Error())
		return
	}
}

func (r *TriggerCategoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapTriggerCategoryToState(c *triggerCategoryAPI, m *TriggerCategoryResourceModel) {
	m.ID = types.StringValue(c.ID)
	m.Name = types.StringValue(c.Name)
	m.Position = types.Int64Value(c.Position)
	m.CreatedAt = types.StringValue(c.CreatedAt)
	m.UpdatedAt = types.StringValue(c.UpdatedAt)
}
