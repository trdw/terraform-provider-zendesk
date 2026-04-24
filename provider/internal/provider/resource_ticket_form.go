package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TicketFormResource{}
	_ resource.ResourceWithImportState = &TicketFormResource{}
)

type TicketFormResource struct {
	client *ZendeskClient
}

type TicketFormResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	DisplayName        types.String `tfsdk:"display_name"`
	Active             types.Bool   `tfsdk:"active"`
	Default            types.Bool   `tfsdk:"default"`
	EndUserVisible     types.Bool   `tfsdk:"end_user_visible"`
	Position           types.Int64  `tfsdk:"position"`
	InAllBrands        types.Bool   `tfsdk:"in_all_brands"`
	RestrictedBrandIDs types.List   `tfsdk:"restricted_brand_ids"`
	TicketFieldIDs     types.List   `tfsdk:"ticket_field_ids"`
	URL                types.String `tfsdk:"url"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

type ticketFormAPIObject struct {
	ID                 int64   `json:"id,omitempty"`
	Name               string  `json:"name,omitempty"`
	DisplayName        string  `json:"display_name,omitempty"`
	Active             *bool   `json:"active,omitempty"`
	Default            *bool   `json:"default,omitempty"`
	EndUserVisible     *bool   `json:"end_user_visible,omitempty"`
	Position           int64   `json:"position,omitempty"`
	InAllBrands        *bool   `json:"in_all_brands,omitempty"`
	RestrictedBrandIDs []int64 `json:"restricted_brand_ids,omitempty"`
	TicketFieldIDs     []int64 `json:"ticket_field_ids,omitempty"`
	URL                string  `json:"url,omitempty"`
	CreatedAt          string  `json:"created_at,omitempty"`
	UpdatedAt          string  `json:"updated_at,omitempty"`
}

type ticketFormWrapper struct {
	TicketForm ticketFormAPIObject `json:"ticket_form"`
}

func NewTicketFormResource() resource.Resource {
	return &TicketFormResource{}
}

func (r *TicketFormResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ticket_form"
}

func (r *TicketFormResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk ticket form.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the form.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the form displayed to end users.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the form is active.",
			},
			"default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this is the default form for the account.",
			},
			"end_user_visible": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the form is visible to end users.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The position of this form in the dropdown.",
			},
			"in_all_brands": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the form is available for use in all brands.",
			},
			"restricted_brand_ids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "IDs of all brands that this ticket form is restricted to.",
			},
			"ticket_field_ids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "IDs of ticket fields in this form. Order determines display order.",
			},
			"url": schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *TicketFormResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TicketFormResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TicketFormResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := ticketFormWrapper{TicketForm: buildTicketFormAPI(ctx, &plan, &resp.Diagnostics)}
	if resp.Diagnostics.HasError() {
		return
	}
	var result ticketFormWrapper
	if err := r.client.Post("/api/v2/ticket_forms", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating ticket form", err.Error())
		return
	}

	mapTicketFormToState(&result.TicketForm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TicketFormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TicketFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ticketFormWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/ticket_forms/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ticket form", err.Error())
		return
	}

	mapTicketFormToState(&result.TicketForm, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TicketFormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TicketFormResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TicketFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := ticketFormWrapper{TicketForm: buildTicketFormAPI(ctx, &plan, &resp.Diagnostics)}
	if resp.Diagnostics.HasError() {
		return
	}
	var result ticketFormWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/ticket_forms/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating ticket form", err.Error())
		return
	}

	mapTicketFormToState(&result.TicketForm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TicketFormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TicketFormResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/ticket_forms/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting ticket form", err.Error())
		return
	}
}

func (r *TicketFormResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func int64ListToSlice(ctx context.Context, list types.List, diags *diag.Diagnostics) []int64 {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []int64
	d := list.ElementsAs(ctx, &out, false)
	diags.Append(d...)
	return out
}

func sliceToInt64List(s []int64) types.List {
	if s == nil {
		return types.ListNull(types.Int64Type)
	}
	vals := make([]attr.Value, len(s))
	for i, v := range s {
		vals[i] = types.Int64Value(v)
	}
	l, _ := types.ListValue(types.Int64Type, vals)
	return l
}

func buildTicketFormAPI(ctx context.Context, plan *TicketFormResourceModel, diags *diag.Diagnostics) ticketFormAPIObject {
	obj := ticketFormAPIObject{
		Name: plan.Name.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		obj.DisplayName = plan.DisplayName.ValueString()
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	if !plan.Default.IsNull() && !plan.Default.IsUnknown() {
		v := plan.Default.ValueBool()
		obj.Default = &v
	}
	if !plan.EndUserVisible.IsNull() && !plan.EndUserVisible.IsUnknown() {
		v := plan.EndUserVisible.ValueBool()
		obj.EndUserVisible = &v
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		obj.Position = plan.Position.ValueInt64()
	}
	if !plan.InAllBrands.IsNull() && !plan.InAllBrands.IsUnknown() {
		v := plan.InAllBrands.ValueBool()
		obj.InAllBrands = &v
	}
	obj.RestrictedBrandIDs = int64ListToSlice(ctx, plan.RestrictedBrandIDs, diags)
	obj.TicketFieldIDs = int64ListToSlice(ctx, plan.TicketFieldIDs, diags)
	return obj
}

func mapTicketFormToState(f *ticketFormAPIObject, m *TicketFormResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(f.ID, 10))
	m.Name = types.StringValue(f.Name)
	m.DisplayName = types.StringValue(f.DisplayName)
	if f.Active != nil {
		m.Active = types.BoolValue(*f.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	if f.Default != nil {
		m.Default = types.BoolValue(*f.Default)
	} else {
		m.Default = types.BoolValue(false)
	}
	if f.EndUserVisible != nil {
		m.EndUserVisible = types.BoolValue(*f.EndUserVisible)
	} else {
		m.EndUserVisible = types.BoolValue(true)
	}
	m.Position = types.Int64Value(f.Position)
	if f.InAllBrands != nil {
		m.InAllBrands = types.BoolValue(*f.InAllBrands)
	} else {
		m.InAllBrands = types.BoolValue(false)
	}
	m.RestrictedBrandIDs = sliceToInt64List(f.RestrictedBrandIDs)
	m.TicketFieldIDs = sliceToInt64List(f.TicketFieldIDs)
	m.URL = types.StringValue(f.URL)
	m.CreatedAt = types.StringValue(f.CreatedAt)
	m.UpdatedAt = types.StringValue(f.UpdatedAt)
}
