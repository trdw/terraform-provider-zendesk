package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ZendeskProvider{}

type ZendeskProvider struct {
	version string
}

type ZendeskProviderModel struct {
	Subdomain types.String `tfsdk:"subdomain"`
	Email     types.String `tfsdk:"email"`
	APIToken  types.String `tfsdk:"api_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ZendeskProvider{version: version}
	}
}

func (p *ZendeskProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zendesk"
	resp.Version = p.version
}

func (p *ZendeskProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Zendesk Support resources.",
		Attributes: map[string]schema.Attribute{
			"subdomain": schema.StringAttribute{
				Optional:    true,
				Description: "The Zendesk subdomain. Can also be set via ZENDESK_SUBDOMAIN env var.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "The email for API authentication. Can also be set via ZENDESK_EMAIL env var.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The Zendesk API token. Can also be set via ZENDESK_API_TOKEN env var.",
			},
		},
	}
}

func (p *ZendeskProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ZendeskProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subdomain := os.Getenv("ZENDESK_SUBDOMAIN")
	email := os.Getenv("ZENDESK_EMAIL")
	apiToken := os.Getenv("ZENDESK_API_TOKEN")

	if !config.Subdomain.IsNull() {
		subdomain = config.Subdomain.ValueString()
	}
	if !config.Email.IsNull() {
		email = config.Email.ValueString()
	}
	if !config.APIToken.IsNull() {
		apiToken = config.APIToken.ValueString()
	}

	if subdomain == "" {
		resp.Diagnostics.AddError("Missing subdomain", "The Zendesk subdomain must be set in the provider config or ZENDESK_SUBDOMAIN env var.")
		return
	}
	if email == "" {
		resp.Diagnostics.AddError("Missing email", "The email must be set in the provider config or ZENDESK_EMAIL env var.")
		return
	}
	if apiToken == "" {
		resp.Diagnostics.AddError("Missing API token", "The API token must be set in the provider config or ZENDESK_API_TOKEN env var.")
		return
	}

	client := NewZendeskClient(subdomain, email, apiToken)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ZendeskProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGroupResource,
		NewGroupMembershipResource,
		NewViewResource,
		NewTriggerResource,
		NewTriggerCategoryResource,
		NewMacroResource,
		NewCustomRoleResource,
		NewUserResource,
		NewBrandResource,
		NewTicketFieldResource,
		NewUserFieldResource,
		NewTicketFormResource,
		NewAutomationResource,
	}
}

func (p *ZendeskProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGroupDataSource,
		NewGroupMembershipDataSource,
		NewViewDataSource,
		NewTriggerDataSource,
		NewTriggerCategoryDataSource,
		NewMacroDataSource,
		NewCustomRoleDataSource,
		NewUserDataSource,
		NewBrandDataSource,
		NewTicketFieldDataSource,
		NewUserFieldDataSource,
		NewTicketFormDataSource,
		NewAutomationDataSource,
	}
}
