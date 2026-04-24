package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CustomRoleDataSource{}

type CustomRoleDataSource struct {
	client *ZendeskClient
}

type CustomRoleDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	RoleType        types.Int64  `tfsdk:"role_type"`
	TeamMemberCount types.Int64  `tfsdk:"team_member_count"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

type customRolesListResponse struct {
	CustomRoles []customRoleAPIObject `json:"custom_roles"`
	NextPage    *string               `json:"next_page"`
}

func NewCustomRoleDataSource() datasource.DataSource {
	return &CustomRoleDataSource{}
}

func (d *CustomRoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_role"
}

func (d *CustomRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk custom role by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the custom role. Provide either id or name.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the custom role. Provide either id or name.",
			},
			"description":       schema.StringAttribute{Computed: true},
			"role_type":         schema.Int64Attribute{Computed: true},
			"team_member_count": schema.Int64Attribute{Computed: true},
			"created_at":        schema.StringAttribute{Computed: true},
			"updated_at":        schema.StringAttribute{Computed: true},
		},
	}
}

func (d *CustomRoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CustomRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CustomRoleDataSourceModel
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

	var found *customRoleAPIObject
	if hasID {
		var result customRoleWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/custom_roles/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading custom role", err.Error())
			return
		}
		found = &result.CustomRole
	} else {
		targetName := config.Name.ValueString()
		page := "/api/v2/custom_roles.json"
		for page != "" {
			var result customRolesListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing custom roles", err.Error())
				return
			}
			for i := range result.CustomRoles {
				if result.CustomRoles[i].Name == targetName {
					found = &result.CustomRoles[i]
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
			resp.Diagnostics.AddError("Custom role not found", fmt.Sprintf("No custom role found with name %q", targetName))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Name = types.StringValue(found.Name)
	config.Description = types.StringValue(found.Description)
	config.RoleType = types.Int64Value(found.RoleType)
	config.TeamMemberCount = types.Int64Value(found.TeamMemberCount)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
