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
	_ resource.Resource                = &OrganizationResource{}
	_ resource.ResourceWithImportState = &OrganizationResource{}
)

type OrganizationResource struct {
	client *ZendeskClient
}

type OrganizationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Details            types.String `tfsdk:"details"`
	Notes              types.String `tfsdk:"notes"`
	DomainNames        types.Set    `tfsdk:"domain_names"`
	GroupID            types.Int64  `tfsdk:"group_id"`
	SharedComments     types.Bool   `tfsdk:"shared_comments"`
	SharedTickets      types.Bool   `tfsdk:"shared_tickets"`
	Tags               types.Set    `tfsdk:"tags"`
	ExternalID         types.String `tfsdk:"external_id"`
	OrganizationFields types.Map    `tfsdk:"organization_fields"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	URL                types.String `tfsdk:"url"`
}

type organizationAPIObject struct {
	ID                 int64                  `json:"id,omitempty"`
	Name               string                 `json:"name"`
	Details            *string                `json:"details,omitempty"`
	Notes              *string                `json:"notes,omitempty"`
	DomainNames        []string               `json:"domain_names,omitempty"`
	GroupID            *int64                 `json:"group_id,omitempty"`
	SharedComments     *bool                  `json:"shared_comments,omitempty"`
	SharedTickets      *bool                  `json:"shared_tickets,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	ExternalID         *string                `json:"external_id,omitempty"`
	OrganizationFields map[string]interface{} `json:"organization_fields,omitempty"`
	CreatedAt          string                 `json:"created_at,omitempty"`
	UpdatedAt          string                 `json:"updated_at,omitempty"`
	URL                string                 `json:"url,omitempty"`
}

type organizationWrapper struct {
	Organization organizationAPIObject `json:"organization"`
}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

func (r *OrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A unique name for the organization.",
			},
			"details": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Any details about the organization, such as the address.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Any notes you have about the organization.",
			},
			"domain_names": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Domain names associated with this organization.",
			},
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Description: "New tickets from users in this organization are automatically put in this group.",
			},
			"shared_comments": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether end users in this organization can comment on each other's tickets.",
			},
			"shared_tickets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether end users in this organization can see each other's tickets.",
			},
			"tags": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "The tags of the organization. Note: Zendesk lowercases tags.",
			},
			"external_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A unique external id to associate the organization with an external record.",
			},
			"organization_fields": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Custom organization field values, keyed by field key. Values are sent as strings.",
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
			"url": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *OrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := organizationWrapper{Organization: buildOrganizationAPI(ctx, &plan, &resp.Diagnostics)}
	if resp.Diagnostics.HasError() {
		return
	}

	var result organizationWrapper
	if err := r.client.Post("/api/v2/organizations", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}

	mapOrganizationToState(&result.Organization, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result organizationWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/organizations/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	mapOrganizationToState(&result.Organization, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := organizationWrapper{Organization: buildOrganizationAPI(ctx, &plan, &resp.Diagnostics)}
	if resp.Diagnostics.HasError() {
		return
	}

	var result organizationWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/organizations/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating organization", err.Error())
		return
	}

	mapOrganizationToState(&result.Organization, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/organizations/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting organization", err.Error())
		return
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildOrganizationAPI(ctx context.Context, plan *OrganizationResourceModel, diags *diag.Diagnostics) organizationAPIObject {
	obj := organizationAPIObject{Name: plan.Name.ValueString()}

	if !plan.Details.IsNull() && !plan.Details.IsUnknown() {
		v := plan.Details.ValueString()
		obj.Details = &v
	}
	if !plan.Notes.IsNull() && !plan.Notes.IsUnknown() {
		v := plan.Notes.ValueString()
		obj.Notes = &v
	}
	if !plan.ExternalID.IsNull() && !plan.ExternalID.IsUnknown() {
		v := plan.ExternalID.ValueString()
		obj.ExternalID = &v
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		v := plan.GroupID.ValueInt64()
		obj.GroupID = &v
	}
	if !plan.SharedComments.IsNull() && !plan.SharedComments.IsUnknown() {
		v := plan.SharedComments.ValueBool()
		obj.SharedComments = &v
	}
	if !plan.SharedTickets.IsNull() && !plan.SharedTickets.IsUnknown() {
		v := plan.SharedTickets.ValueBool()
		obj.SharedTickets = &v
	}

	obj.DomainNames = stringSetToSlice(ctx, plan.DomainNames, diags)
	obj.Tags = stringSetToSlice(ctx, plan.Tags, diags)

	if !plan.OrganizationFields.IsNull() && !plan.OrganizationFields.IsUnknown() {
		obj.OrganizationFields = map[string]interface{}{}
		for k, v := range plan.OrganizationFields.Elements() {
			if s, ok := v.(types.String); ok {
				obj.OrganizationFields[k] = s.ValueString()
			}
		}
	}

	return obj
}

func mapOrganizationToState(o *organizationAPIObject, m *OrganizationResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(o.ID, 10))
	m.Name = types.StringValue(o.Name)
	m.Details = strPtrToState(o.Details)
	m.Notes = strPtrToState(o.Notes)
	m.ExternalID = strPtrToState(o.ExternalID)
	m.DomainNames = sliceToStringSet(o.DomainNames)
	m.Tags = sliceToStringSet(o.Tags)
	if o.GroupID != nil {
		m.GroupID = types.Int64Value(*o.GroupID)
	} else {
		m.GroupID = types.Int64Null()
	}
	if o.SharedComments != nil {
		m.SharedComments = types.BoolValue(*o.SharedComments)
	} else {
		m.SharedComments = types.BoolValue(false)
	}
	if o.SharedTickets != nil {
		m.SharedTickets = types.BoolValue(*o.SharedTickets)
	} else {
		m.SharedTickets = types.BoolValue(false)
	}
	m.CreatedAt = types.StringValue(o.CreatedAt)
	m.UpdatedAt = types.StringValue(o.UpdatedAt)
	m.URL = types.StringValue(o.URL)

	// Only echo back the custom-field keys the configuration set, so the API
	// returning the full set of defined org fields doesn't show as drift. A key
	// the API omits keeps its configured value to avoid post-apply inconsistency.
	if m.OrganizationFields.IsNull() || m.OrganizationFields.IsUnknown() {
		m.OrganizationFields = types.MapNull(types.StringType)
	} else {
		vals := map[string]attr.Value{}
		for k, prior := range m.OrganizationFields.Elements() {
			if av, ok := o.OrganizationFields[k]; ok {
				vals[k] = types.StringValue(valueToString(av))
			} else {
				vals[k] = prior
			}
		}
		m.OrganizationFields = types.MapValueMust(types.StringType, vals)
	}
}

func strPtrToState(p *string) types.String {
	if p == nil {
		return types.StringValue("")
	}
	return types.StringValue(*p)
}

func stringSetToSlice(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(set.ElementsAs(ctx, &out, false)...)
	return out
}

func sliceToStringSet(s []string) types.Set {
	if s == nil {
		return types.SetNull(types.StringType)
	}
	vals := make([]attr.Value, len(s))
	for i, v := range s {
		vals[i] = types.StringValue(v)
	}
	set, _ := types.SetValue(types.StringType, vals)
	return set
}
