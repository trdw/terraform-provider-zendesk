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
	_ resource.Resource                = &BrandResource{}
	_ resource.ResourceWithImportState = &BrandResource{}
)

type BrandResource struct {
	client *ZendeskClient
}

type BrandResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Subdomain         types.String `tfsdk:"subdomain"`
	Active            types.Bool   `tfsdk:"active"`
	BrandURL          types.String `tfsdk:"brand_url"`
	HostMapping       types.String `tfsdk:"host_mapping"`
	HasHelpCenter     types.Bool   `tfsdk:"has_help_center"`
	HelpCenterState   types.String `tfsdk:"help_center_state"`
	SignatureTemplate types.String `tfsdk:"signature_template"`
	Default           types.Bool   `tfsdk:"default"`
	IsDeleted         types.Bool   `tfsdk:"is_deleted"`
	URL               types.String `tfsdk:"url"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

type brandAPIObject struct {
	ID                int64  `json:"id,omitempty"`
	Name              string `json:"name"`
	Subdomain         string `json:"subdomain"`
	Active            *bool  `json:"active,omitempty"`
	BrandURL          string `json:"brand_url,omitempty"`
	HostMapping       string `json:"host_mapping,omitempty"`
	HasHelpCenter     bool   `json:"has_help_center,omitempty"`
	HelpCenterState   string `json:"help_center_state,omitempty"`
	SignatureTemplate string `json:"signature_template,omitempty"`
	Default           bool   `json:"default,omitempty"`
	IsDeleted         bool   `json:"is_deleted,omitempty"`
	URL               string `json:"url,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type brandWrapper struct {
	Brand brandAPIObject `json:"brand"`
}

func NewBrandResource() resource.Resource {
	return &BrandResource{}
}

func (r *BrandResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_brand"
}

func (r *BrandResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk brand.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the brand.",
			},
			"subdomain": schema.StringAttribute{
				Required:    true,
				Description: "The subdomain of the brand.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the brand is active.",
			},
			"brand_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The URL of the brand.",
			},
			"host_mapping": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The host mapping to this brand.",
			},
			"has_help_center": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the brand has a Help Center.",
			},
			"help_center_state": schema.StringAttribute{
				Computed:    true,
				Description: "The state of the Help Center (enabled, disabled, restricted).",
			},
			"signature_template": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The signature template for the brand.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is the default brand for the account.",
			},
			"is_deleted": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the brand is deleted.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The API URL for this brand.",
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

func (r *BrandResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrandResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BrandResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := brandWrapper{Brand: buildBrandAPI(&plan)}
	var result brandWrapper
	if err := r.client.Post("/api/v2/brands", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating brand", err.Error())
		return
	}

	mapBrandToState(&result.Brand, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrandResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BrandResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result brandWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/brands/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading brand", err.Error())
		return
	}

	mapBrandToState(&result.Brand, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BrandResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BrandResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state BrandResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := brandWrapper{Brand: buildBrandAPI(&plan)}
	var result brandWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/brands/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating brand", err.Error())
		return
	}

	mapBrandToState(&result.Brand, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrandResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BrandResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/brands/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting brand", err.Error())
		return
	}
}

func (r *BrandResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildBrandAPI(plan *BrandResourceModel) brandAPIObject {
	obj := brandAPIObject{
		Name:      plan.Name.ValueString(),
		Subdomain: plan.Subdomain.ValueString(),
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	if !plan.BrandURL.IsNull() && !plan.BrandURL.IsUnknown() {
		obj.BrandURL = plan.BrandURL.ValueString()
	}
	if !plan.HostMapping.IsNull() && !plan.HostMapping.IsUnknown() {
		obj.HostMapping = plan.HostMapping.ValueString()
	}
	if !plan.HasHelpCenter.IsNull() && !plan.HasHelpCenter.IsUnknown() {
		obj.HasHelpCenter = plan.HasHelpCenter.ValueBool()
	}
	if !plan.SignatureTemplate.IsNull() && !plan.SignatureTemplate.IsUnknown() {
		obj.SignatureTemplate = plan.SignatureTemplate.ValueString()
	}
	return obj
}

func mapBrandToState(b *brandAPIObject, m *BrandResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(b.ID, 10))
	m.Name = types.StringValue(b.Name)
	m.Subdomain = types.StringValue(b.Subdomain)
	if b.Active != nil {
		m.Active = types.BoolValue(*b.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	m.BrandURL = types.StringValue(b.BrandURL)
	m.HostMapping = types.StringValue(b.HostMapping)
	m.HasHelpCenter = types.BoolValue(b.HasHelpCenter)
	m.HelpCenterState = types.StringValue(b.HelpCenterState)
	m.SignatureTemplate = types.StringValue(b.SignatureTemplate)
	m.Default = types.BoolValue(b.Default)
	m.IsDeleted = types.BoolValue(b.IsDeleted)
	m.URL = types.StringValue(b.URL)
	m.CreatedAt = types.StringValue(b.CreatedAt)
	m.UpdatedAt = types.StringValue(b.UpdatedAt)
}
