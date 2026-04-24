package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MacroDataSource{}

type MacroDataSource struct {
	client *ZendeskClient
}

type MacroDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Active      types.Bool   `tfsdk:"active"`
	Description types.String `tfsdk:"description"`
	Position    types.Int64  `tfsdk:"position"`
	Default     types.Bool   `tfsdk:"default"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type macrosListResponse struct {
	Macros   []macroReadAPI `json:"macros"`
	NextPage *string        `json:"next_page"`
}

func NewMacroDataSource() datasource.DataSource {
	return &MacroDataSource{}
}

func (d *MacroDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_macro"
}

func (d *MacroDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk macro by ID or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the macro. Provide either id or title.",
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the macro. Provide either id or title.",
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

func (d *MacroDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MacroDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config MacroDataSourceModel
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

	var found *macroReadAPI
	if hasID {
		var result macroReadWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/macros/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading macro", err.Error())
			return
		}
		found = &result.Macro
	} else {
		targetTitle := config.Title.ValueString()
		page := "/api/v2/macros.json?access=account&page[size]=100"
		for page != "" {
			var result macrosListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing macros", err.Error())
				return
			}
			for i := range result.Macros {
				if result.Macros[i].Title == targetTitle {
					found = &result.Macros[i]
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
			resp.Diagnostics.AddError("Macro not found", fmt.Sprintf("No macro found with title %q", targetTitle))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Title = types.StringValue(found.Title)
	config.Active = types.BoolValue(found.Active)
	if found.Description != nil {
		config.Description = types.StringValue(*found.Description)
	} else {
		config.Description = types.StringNull()
	}
	config.Position = types.Int64Value(found.Position)
	config.Default = types.BoolValue(found.Default)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
