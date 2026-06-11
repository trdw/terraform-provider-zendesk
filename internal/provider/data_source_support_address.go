package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SupportAddressDataSource{}

type SupportAddressDataSource struct {
	client *ZendeskClient
}

type SupportAddressDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Email                    types.String `tfsdk:"email"`
	Name                     types.String `tfsdk:"name"`
	BrandID                  types.Int64  `tfsdk:"brand_id"`
	Default                  types.Bool   `tfsdk:"default"`
	CnameStatus              types.String `tfsdk:"cname_status"`
	DNSResults               types.String `tfsdk:"dns_results"`
	DomainVerificationCode   types.String `tfsdk:"domain_verification_code"`
	DomainVerificationStatus types.String `tfsdk:"domain_verification_status"`
	ForwardingStatus         types.String `tfsdk:"forwarding_status"`
	SPFStatus                types.String `tfsdk:"spf_status"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

type supportAddressesListResponse struct {
	RecipientAddresses []supportAddressAPIObject `json:"recipient_addresses"`
	NextPage           *string                   `json:"next_page"`
}

func NewSupportAddressDataSource() datasource.DataSource {
	return &SupportAddressDataSource{}
}

func (d *SupportAddressDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_support_address"
}

func (d *SupportAddressDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk support address (recipient address) by ID or email.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the support address. Provide either id or email.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The email of the support address. Provide either id or email.",
			},
			"name":                       schema.StringAttribute{Computed: true},
			"brand_id":                   schema.Int64Attribute{Computed: true},
			"default":                    schema.BoolAttribute{Computed: true},
			"cname_status":               schema.StringAttribute{Computed: true},
			"dns_results":                schema.StringAttribute{Computed: true},
			"domain_verification_code":   schema.StringAttribute{Computed: true},
			"domain_verification_status": schema.StringAttribute{Computed: true},
			"forwarding_status":          schema.StringAttribute{Computed: true},
			"spf_status":                 schema.StringAttribute{Computed: true},
			"created_at":                 schema.StringAttribute{Computed: true},
			"updated_at":                 schema.StringAttribute{Computed: true},
		},
	}
}

func (d *SupportAddressDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SupportAddressDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SupportAddressDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasEmail := !config.Email.IsNull() && !config.Email.IsUnknown() && config.Email.ValueString() != ""

	if !hasID && !hasEmail {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id' or 'email' must be provided.")
		return
	}

	var found *supportAddressAPIObject
	if hasID {
		var result supportAddressWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/recipient_addresses/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading support address", err.Error())
			return
		}
		found = &result.RecipientAddress
	} else {
		target := strings.ToLower(config.Email.ValueString())
		page := "/api/v2/recipient_addresses.json?page[size]=100"
		for page != "" {
			var result supportAddressesListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing support addresses", err.Error())
				return
			}
			for i := range result.RecipientAddresses {
				if strings.ToLower(result.RecipientAddresses[i].Email) == target {
					found = &result.RecipientAddresses[i]
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
			resp.Diagnostics.AddError("Support address not found", fmt.Sprintf("No support address found with email %q", config.Email.ValueString()))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Email = types.StringValue(found.Email)
	config.Name = types.StringValue(found.Name)
	if found.BrandID != nil {
		config.BrandID = types.Int64Value(*found.BrandID)
	} else {
		config.BrandID = types.Int64Null()
	}
	config.Default = types.BoolValue(found.Default != nil && *found.Default)
	config.CnameStatus = types.StringValue(found.CnameStatus)
	config.DNSResults = types.StringValue(found.DNSResults)
	config.DomainVerificationCode = types.StringValue(found.DomainVerificationCode)
	config.DomainVerificationStatus = types.StringValue(found.DomainVerificationStatus)
	config.ForwardingStatus = types.StringValue(found.ForwardingStatus)
	config.SPFStatus = types.StringValue(found.SPFStatus)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
