package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GroupDataSource{}

type GroupDataSource struct {
	client *ZendeskClient
}

type GroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsPublic    types.Bool   `tfsdk:"is_public"`
	Default     types.Bool   `tfsdk:"default"`
	Deleted     types.Bool   `tfsdk:"deleted"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type groupsListResponse struct {
	Groups   []groupAPIObject `json:"groups"`
	NextPage *string          `json:"next_page"`
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk group by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the group. Provide either id or name.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the group. Provide either id or name.",
			},
			"description": schema.StringAttribute{Computed: true},
			"is_public":   schema.BoolAttribute{Computed: true},
			"default":     schema.BoolAttribute{Computed: true},
			"deleted":     schema.BoolAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupDataSourceModel
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

	var found *groupAPIObject
	if hasID {
		var result groupWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/groups/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading group", err.Error())
			return
		}
		found = &result.Group
	} else {
		targetName := config.Name.ValueString()
		page := "/api/v2/groups.json?page[size]=100"
		for page != "" {
			var result groupsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing groups", err.Error())
				return
			}
			for i := range result.Groups {
				if result.Groups[i].Name == targetName {
					found = &result.Groups[i]
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
			resp.Diagnostics.AddError("Group not found", fmt.Sprintf("No group found with name %q", targetName))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Name = types.StringValue(found.Name)
	config.Description = types.StringValue(found.Description)
	if found.IsPublic != nil {
		config.IsPublic = types.BoolValue(*found.IsPublic)
	} else {
		config.IsPublic = types.BoolValue(false)
	}
	config.Default = types.BoolValue(found.Default)
	config.Deleted = types.BoolValue(found.Deleted)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
