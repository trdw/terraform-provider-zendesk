package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BrandDataSource{}

type BrandDataSource struct {
	client *ZendeskClient
}

type BrandDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Subdomain         types.String `tfsdk:"subdomain"`
	Active            types.Bool   `tfsdk:"active"`
	BrandURL          types.String `tfsdk:"brand_url"`
	HostMapping       types.String `tfsdk:"host_mapping"`
	HasHelpCenter     types.Bool   `tfsdk:"has_help_center"`
	HelpCenterState   types.String `tfsdk:"help_center_state"`
	SignatureTemplate types.String `tfsdk:"signature_template"`
	Default           types.Bool   `tfsdk:"default"`
	IsDeleted         types.Bool   `tfsdk:"is_deleted"`
	URL               types.String `tfsdk:"url"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

type brandsListResponse struct {
	Brands   []brandAPIObject `json:"brands"`
	NextPage *string          `json:"next_page"`
}

func NewBrandDataSource() datasource.DataSource {
	return &BrandDataSource{}
}

func (d *BrandDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_brand"
}

func (d *BrandDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk brand by ID, name, or subdomain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the brand. Provide one of id, name, or subdomain.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the brand. Provide one of id, name, or subdomain.",
			},
			"subdomain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The subdomain of the brand. Provide one of id, name, or subdomain.",
			},
			"active":             schema.BoolAttribute{Computed: true},
			"brand_url":          schema.StringAttribute{Computed: true},
			"host_mapping":       schema.StringAttribute{Computed: true},
			"has_help_center":    schema.BoolAttribute{Computed: true},
			"help_center_state":  schema.StringAttribute{Computed: true},
			"signature_template": schema.StringAttribute{Computed: true},
			"default":            schema.BoolAttribute{Computed: true},
			"is_deleted":         schema.BoolAttribute{Computed: true},
			"url":                schema.StringAttribute{Computed: true},
			"created_at":         schema.StringAttribute{Computed: true},
			"updated_at":         schema.StringAttribute{Computed: true},
		},
	}
}

func (d *BrandDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrandDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BrandDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	hasSubdomain := !config.Subdomain.IsNull() && !config.Subdomain.IsUnknown() && config.Subdomain.ValueString() != ""

	if !hasID && !hasName && !hasSubdomain {
		resp.Diagnostics.AddError("Missing attribute", "One of 'id', 'name', or 'subdomain' must be provided.")
		return
	}

	var found *brandAPIObject
	if hasID {
		var result brandWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/brands/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading brand", err.Error())
			return
		}
		found = &result.Brand
	} else {
		targetName := config.Name.ValueString()
		targetSubdomain := config.Subdomain.ValueString()
		page := "/api/v2/brands.json?page[size]=100"
		for page != "" {
			var result brandsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing brands", err.Error())
				return
			}
			for i := range result.Brands {
				b := &result.Brands[i]
				if (hasName && b.Name == targetName) || (hasSubdomain && b.Subdomain == targetSubdomain) {
					found = b
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
			resp.Diagnostics.AddError("Brand not found", "No brand matched the supplied name or subdomain.")
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Name = types.StringValue(found.Name)
	config.Subdomain = types.StringValue(found.Subdomain)
	if found.Active != nil {
		config.Active = types.BoolValue(*found.Active)
	} else {
		config.Active = types.BoolValue(true)
	}
	config.BrandURL = types.StringValue(found.BrandURL)
	config.HostMapping = types.StringValue(found.HostMapping)
	config.HasHelpCenter = types.BoolValue(found.HasHelpCenter)
	config.HelpCenterState = types.StringValue(found.HelpCenterState)
	config.SignatureTemplate = types.StringValue(found.SignatureTemplate)
	config.Default = types.BoolValue(found.Default)
	config.IsDeleted = types.BoolValue(found.IsDeleted)
	config.URL = types.StringValue(found.URL)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
