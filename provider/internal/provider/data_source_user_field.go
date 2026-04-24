package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserFieldDataSource{}

type UserFieldDataSource struct {
	client *ZendeskClient
}

type UserFieldDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Title       types.String `tfsdk:"title"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Active      types.Bool   `tfsdk:"active"`
	Position    types.Int64  `tfsdk:"position"`
	Tag         types.String `tfsdk:"tag"`
	System      types.Bool   `tfsdk:"system"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type userFieldsListResponse struct {
	UserFields []userFieldAPIObject `json:"user_fields"`
	NextPage   *string              `json:"next_page"`
}

func NewUserFieldDataSource() datasource.DataSource {
	return &UserFieldDataSource{}
}

func (d *UserFieldDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_field"
}

func (d *UserFieldDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk user field by ID, key, or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the user field. Provide one of id, key, or title.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique key of the user field. Provide one of id, key, or title.",
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the user field. Provide one of id, key, or title.",
			},
			"type":        schema.StringAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"active":      schema.BoolAttribute{Computed: true},
			"position":    schema.Int64Attribute{Computed: true},
			"tag":         schema.StringAttribute{Computed: true},
			"system":      schema.BoolAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *UserFieldDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserFieldDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserFieldDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasKey := !config.Key.IsNull() && !config.Key.IsUnknown() && config.Key.ValueString() != ""
	hasTitle := !config.Title.IsNull() && !config.Title.IsUnknown() && config.Title.ValueString() != ""

	if !hasID && !hasKey && !hasTitle {
		resp.Diagnostics.AddError("Missing attribute", "One of 'id', 'key', or 'title' must be provided.")
		return
	}

	var found *userFieldAPIObject
	if hasID {
		var result userFieldWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/user_fields/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading user field", err.Error())
			return
		}
		found = &result.UserField
	} else {
		targetKey := config.Key.ValueString()
		targetTitle := config.Title.ValueString()
		page := "/api/v2/user_fields.json?page[size]=100"
		for page != "" {
			var result userFieldsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing user fields", err.Error())
				return
			}
			for i := range result.UserFields {
				uf := &result.UserFields[i]
				if (hasKey && uf.Key == targetKey) || (hasTitle && uf.Title == targetTitle) {
					found = uf
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
			resp.Diagnostics.AddError("User field not found", "No user field matched the supplied key or title.")
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Key = types.StringValue(found.Key)
	config.Title = types.StringValue(found.Title)
	config.Type = types.StringValue(found.Type)
	config.Description = types.StringValue(found.Description)
	if found.Active != nil {
		config.Active = types.BoolValue(*found.Active)
	} else {
		config.Active = types.BoolValue(true)
	}
	config.Position = types.Int64Value(found.Position)
	if found.Tag != nil {
		config.Tag = types.StringValue(*found.Tag)
	} else {
		config.Tag = types.StringNull()
	}
	config.System = types.BoolValue(found.System)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
