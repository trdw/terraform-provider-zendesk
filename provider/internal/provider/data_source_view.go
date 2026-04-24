package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ViewDataSource{}

type ViewDataSource struct {
	client *ZendeskClient
}

type ViewDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Active      types.Bool   `tfsdk:"active"`
	Description types.String `tfsdk:"description"`
	Position    types.Int64  `tfsdk:"position"`
	Default     types.Bool   `tfsdk:"default"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type viewsListResponse struct {
	Views    []viewReadAPI `json:"views"`
	NextPage *string       `json:"next_page"`
}

func NewViewDataSource() datasource.DataSource {
	return &ViewDataSource{}
}

func (d *ViewDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (d *ViewDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk view by ID or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the view. Provide either id or title.",
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the view. Provide either id or title.",
			},
			"active":      schema.BoolAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"position":    schema.Int64Attribute{Computed: true},
			"default":     schema.BoolAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *ViewDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ViewDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ViewDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasTitle := !config.Title.IsNull() && !config.Title.IsUnknown() && config.Title.ValueString() != ""

	if !hasID && !hasTitle {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id' or 'title' must be provided.")
		return
	}

	var found *viewReadAPI
	if hasID {
		var result viewReadWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/views/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading view", err.Error())
			return
		}
		found = &result.View
	} else {
		targetTitle := config.Title.ValueString()
		page := "/api/v2/views.json?access=account&page[size]=100"
		for page != "" {
			var result viewsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing views", err.Error())
				return
			}
			for i := range result.Views {
				if result.Views[i].Title == targetTitle {
					found = &result.Views[i]
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
			resp.Diagnostics.AddError("View not found", fmt.Sprintf("No view found with title %q", targetTitle))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Title = types.StringValue(found.Title)
	config.Active = types.BoolValue(found.Active)
	config.Description = types.StringValue(found.Description)
	config.Position = types.Int64Value(found.Position)
	config.Default = types.BoolValue(found.Default)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
