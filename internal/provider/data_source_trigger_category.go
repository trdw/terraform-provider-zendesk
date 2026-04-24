package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TriggerCategoryDataSource{}

type TriggerCategoryDataSource struct {
	client *ZendeskClient
}

type TriggerCategoryDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type triggerCategoryAPI struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Position  int64  `json:"position,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type triggerCategoryWrapper struct {
	TriggerCategory triggerCategoryAPI `json:"trigger_category"`
}

type triggerCategoriesListResponse struct {
	TriggerCategories []triggerCategoryAPI `json:"trigger_categories"`
	NextPage          *string              `json:"next_page"`
}

func NewTriggerCategoryDataSource() datasource.DataSource {
	return &TriggerCategoryDataSource{}
}

func (d *TriggerCategoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger_category"
}

func (d *TriggerCategoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a Zendesk trigger category by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the trigger category.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the trigger category.",
			},
		},
	}
}

func (d *TriggerCategoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TriggerCategoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TriggerCategoryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetName := config.Name.ValueString()
	page := fmt.Sprintf("/api/v2/trigger_categories?sort=position&page[size]=100&filter[name]=%s", url.QueryEscape(targetName))

	for page != "" {
		var result triggerCategoriesListResponse
		if err := d.client.Get(page, &result); err != nil {
			resp.Diagnostics.AddError("Error listing trigger categories", err.Error())
			return
		}

		for _, cat := range result.TriggerCategories {
			if cat.Name == targetName {
				config.ID = types.StringValue(cat.ID)
				resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
				return
			}
		}

		if result.NextPage != nil && *result.NextPage != "" {
			page = *result.NextPage
		} else {
			page = ""
		}
	}

	resp.Diagnostics.AddError(
		"Trigger category not found",
		fmt.Sprintf("No trigger category found with name %q", targetName),
	)
}
