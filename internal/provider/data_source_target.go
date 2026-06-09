package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TargetDataSource{}

type TargetDataSource struct {
	client *ZendeskClient
}

type TargetDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Type      types.String `tfsdk:"type"`
	Email     types.String `tfsdk:"email"`
	Subject   types.String `tfsdk:"subject"`
	Active    types.Bool   `tfsdk:"active"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type targetsListResponse struct {
	Targets  []targetAPIObject `json:"targets"`
	NextPage *string           `json:"next_page"`
}

func NewTargetDataSource() datasource.DataSource {
	return &TargetDataSource{}
}

func (d *TargetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (d *TargetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk target by ID or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the target. Provide either id or title.",
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the target. Provide either id or title.",
			},
			"type":       schema.StringAttribute{Computed: true},
			"email":      schema.StringAttribute{Computed: true},
			"subject":    schema.StringAttribute{Computed: true},
			"active":     schema.BoolAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TargetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TargetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TargetDataSourceModel
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

	var found *targetAPIObject
	if hasID {
		var result targetWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/targets/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading target", err.Error())
			return
		}
		found = &result.Target
	} else {
		targetTitle := config.Title.ValueString()
		page := "/api/v2/targets.json?page[size]=100"
		for page != "" {
			var result targetsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing targets", err.Error())
				return
			}
			for i := range result.Targets {
				if result.Targets[i].Title == targetTitle {
					found = &result.Targets[i]
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
			resp.Diagnostics.AddError("Target not found", fmt.Sprintf("No target found with title %q", targetTitle))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Title = types.StringValue(found.Title)
	config.Type = types.StringValue(found.Type)
	config.Email = types.StringValue(found.Email)
	config.Subject = types.StringValue(found.Subject)
	config.Active = types.BoolValue(found.Active != nil && *found.Active)
	config.CreatedAt = types.StringValue(found.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
