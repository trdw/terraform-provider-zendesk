package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &SupportAddressResource{}
	_ resource.ResourceWithImportState = &SupportAddressResource{}
)

type SupportAddressResource struct {
	client *ZendeskClient
}

type SupportAddressResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Email                    types.String `tfsdk:"email"`
	Name                     types.String `tfsdk:"name"`
	BrandID                  types.Int64  `tfsdk:"brand_id"`
	Default                  types.Bool   `tfsdk:"default"`
	CnameStatus              types.String `tfsdk:"cname_status"`
	DNSResults               types.String `tfsdk:"dns_results"`
	DomainVerificationCode   types.String `tfsdk:"domain_verification_code"`
	DomainVerificationStatus types.String `tfsdk:"domain_verification_status"`
	ForwardingStatus         types.String `tfsdk:"forwarding_status"`
	SPFStatus                types.String `tfsdk:"spf_status"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

type supportAddressAPIObject struct {
	ID                       int64  `json:"id,omitempty"`
	Email                    string `json:"email,omitempty"`
	Name                     string `json:"name,omitempty"`
	BrandID                  *int64 `json:"brand_id,omitempty"`
	Default                  *bool  `json:"default,omitempty"`
	CnameStatus              string `json:"cname_status,omitempty"`
	DNSResults               string `json:"dns_results,omitempty"`
	DomainVerificationCode   string `json:"domain_verification_code,omitempty"`
	DomainVerificationStatus string `json:"domain_verification_status,omitempty"`
	ForwardingStatus         string `json:"forwarding_status,omitempty"`
	SPFStatus                string `json:"spf_status,omitempty"`
	CreatedAt                string `json:"created_at,omitempty"`
	UpdatedAt                string `json:"updated_at,omitempty"`
}

type supportAddressWrapper struct {
	RecipientAddress supportAddressAPIObject `json:"recipient_address"`
}

func NewSupportAddressResource() resource.Resource {
	return &SupportAddressResource{}
}

func (r *SupportAddressResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_support_address"
}

func (r *SupportAddressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk support address (recipient address) — an email address " +
			"tickets can be received at. The Zendesk API calls these recipient_addresses.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Required: true,
				Description: "The support email address. Write-once in Zendesk: changing it " +
					"forces replacement of the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The display name of the address. If omitted, the Zendesk-assigned value is recorded in state.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"brand_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The id of the brand the address belongs to. If omitted, Zendesk assigns one (the default brand) and it is recorded in state.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is the account's default support address.",
			},
			"cname_status": schema.StringAttribute{
				Computed:    true,
				Description: "CNAME record status: unknown, verified, or failed.",
			},
			"dns_results": schema.StringAttribute{
				Computed:    true,
				Description: "DNS verification results.",
			},
			"domain_verification_code": schema.StringAttribute{
				Computed:    true,
				Description: "Verification code for domain verification.",
			},
			"domain_verification_status": schema.StringAttribute{
				Computed:    true,
				Description: "Domain verification status: unknown, verified, or failed.",
			},
			"forwarding_status": schema.StringAttribute{
				Computed:    true,
				Description: "Forwarding status (external addresses only): unknown, waiting, verified, or failed.",
			},
			"spf_status": schema.StringAttribute{
				Computed:    true,
				Description: "SPF record status: unknown, verified, or failed.",
			},
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

func (r *SupportAddressResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SupportAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SupportAddressResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := supportAddressWrapper{RecipientAddress: buildSupportAddressAPI(&plan)}
	var result supportAddressWrapper
	if err := r.client.Post("/api/v2/recipient_addresses", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating support address", err.Error())
		return
	}

	mapSupportAddressToState(&result.RecipientAddress, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SupportAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SupportAddressResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result supportAddressWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/recipient_addresses/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading support address", err.Error())
		return
	}

	mapSupportAddressToState(&result.RecipientAddress, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SupportAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SupportAddressResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SupportAddressResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// email is write-once (RequiresReplace), so updates only carry the
	// mutable fields.
	apiObj := buildSupportAddressAPI(&plan)
	apiObj.Email = ""
	apiReq := supportAddressWrapper{RecipientAddress: apiObj}
	var result supportAddressWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/recipient_addresses/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating support address", err.Error())
		return
	}

	mapSupportAddressToState(&result.RecipientAddress, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SupportAddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SupportAddressResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/recipient_addresses/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting support address", err.Error())
		return
	}
}

func (r *SupportAddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildSupportAddressAPI(plan *SupportAddressResourceModel) supportAddressAPIObject {
	obj := supportAddressAPIObject{
		Email: plan.Email.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		obj.Name = plan.Name.ValueString()
	}
	if !plan.BrandID.IsNull() && !plan.BrandID.IsUnknown() {
		v := plan.BrandID.ValueInt64()
		obj.BrandID = &v
	}
	return obj
}

func mapSupportAddressToState(a *supportAddressAPIObject, m *SupportAddressResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(a.ID, 10))
	m.Email = types.StringValue(a.Email)
	m.Name = types.StringValue(a.Name)
	if a.BrandID != nil {
		m.BrandID = types.Int64Value(*a.BrandID)
	} else {
		m.BrandID = types.Int64Null()
	}
	if a.Default != nil {
		m.Default = types.BoolValue(*a.Default)
	} else {
		m.Default = types.BoolValue(false)
	}
	m.CnameStatus = types.StringValue(a.CnameStatus)
	m.DNSResults = types.StringValue(a.DNSResults)
	m.DomainVerificationCode = types.StringValue(a.DomainVerificationCode)
	m.DomainVerificationStatus = types.StringValue(a.DomainVerificationStatus)
	m.ForwardingStatus = types.StringValue(a.ForwardingStatus)
	m.SPFStatus = types.StringValue(a.SPFStatus)
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)
}
