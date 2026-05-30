package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OrganizationDataSource{}

type OrganizationDataSource struct {
	client *ZendeskClient
}

type OrganizationDataSourceModel struct {
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

type organizationsListResponse struct {
	Organizations []organizationAPIObject `json:"organizations"`
	NextPage      *string                 `json:"next_page"`
}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

func (d *OrganizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk organization by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the organization. Provide either id or name.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the organization. Provide either id or name.",
			},
			"details":             schema.StringAttribute{Computed: true},
			"notes":               schema.StringAttribute{Computed: true},
			"domain_names":        schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"group_id":            schema.Int64Attribute{Computed: true},
			"shared_comments":     schema.BoolAttribute{Computed: true},
			"shared_tickets":      schema.BoolAttribute{Computed: true},
			"tags":                schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"external_id":         schema.StringAttribute{Computed: true},
			"organization_fields": schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"created_at":          schema.StringAttribute{Computed: true},
			"updated_at":          schema.StringAttribute{Computed: true},
			"url":                 schema.StringAttribute{Computed: true},
		},
	}
}

func (d *OrganizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ZendeskClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", "Expected *ZendeskClient")
		return
	}
	d.client = client
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OrganizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id' or 'name' must be provided.")
		return
	}

	var found *organizationAPIObject
	if hasID {
		var result organizationWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/organizations/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading organization", err.Error())
			return
		}
		found = &result.Organization
	} else {
		targetName := config.Name.ValueString()
		page := "/api/v2/organizations.json?page[size]=100"
		for page != "" {
			var result organizationsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing organizations", err.Error())
				return
			}
			for i := range result.Organizations {
				if result.Organizations[i].Name == targetName {
					found = &result.Organizations[i]
					break
				}
			}
			if found != nil {
				break
			}
			if result.NextPage != nil && *result.NextPage != "" {
				page = *result.NextPage
			} else {
				page = ""
			}
		}
		if found == nil {
			resp.Diagnostics.AddError("Organization not found", fmt.Sprintf("No organization found with name %q", targetName))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Name = types.StringValue(found.Name)
	config.Details = strPtrToState(found.Details)
	config.Notes = strPtrToState(found.Notes)
	config.ExternalID = strPtrToState(found.ExternalID)
	config.DomainNames = sliceToStringSet(found.DomainNames)
	config.Tags = sliceToStringSet(found.Tags)
	if found.GroupID != nil {
		config.GroupID = types.Int64Value(*found.GroupID)
	} else {
		config.GroupID = types.Int64Null()
	}
	config.SharedComments = types.BoolValue(found.SharedComments != nil && *found.SharedComments)
	config.SharedTickets = types.BoolValue(found.SharedTickets != nil && *found.SharedTickets)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)
	config.URL = types.StringValue(found.URL)

	if found.OrganizationFields == nil {
		config.OrganizationFields = types.MapNull(types.StringType)
	} else {
		vals := map[string]attr.Value{}
		for k, v := range found.OrganizationFields {
			vals[k] = types.StringValue(valueToString(v))
		}
		config.OrganizationFields = types.MapValueMust(types.StringType, vals)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
