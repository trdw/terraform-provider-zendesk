package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct {
	client *ZendeskClient
}

type UserDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Email               types.String `tfsdk:"email"`
	Role                types.String `tfsdk:"role"`
	CustomRoleID        types.Int64  `tfsdk:"custom_role_id"`
	DefaultGroupID      types.Int64  `tfsdk:"default_group_id"`
	Alias               types.String `tfsdk:"alias"`
	Details             types.String `tfsdk:"details"`
	Notes               types.String `tfsdk:"notes"`
	Signature           types.String `tfsdk:"signature"`
	Phone               types.String `tfsdk:"phone"`
	ExternalID          types.String `tfsdk:"external_id"`
	OrganizationID      types.Int64  `tfsdk:"organization_id"`
	Locale              types.String `tfsdk:"locale"`
	TimeZone            types.String `tfsdk:"time_zone"`
	Suspended           types.Bool   `tfsdk:"suspended"`
	RestrictedAgent     types.Bool   `tfsdk:"restricted_agent"`
	OnlyPrivateComments types.Bool   `tfsdk:"only_private_comments"`
	TicketRestriction   types.String `tfsdk:"ticket_restriction"`
	Active              types.Bool   `tfsdk:"active"`
	Verified            types.Bool   `tfsdk:"verified"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk user by ID or email address.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the user. Provide either id or email.",
			},
			"name": schema.StringAttribute{Computed: true},
			"email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The email address of the user. Provide either id or email.",
			},
			"role":                  schema.StringAttribute{Computed: true},
			"custom_role_id":        schema.Int64Attribute{Computed: true},
			"default_group_id":      schema.Int64Attribute{Computed: true},
			"alias":                 schema.StringAttribute{Computed: true},
			"details":               schema.StringAttribute{Computed: true},
			"notes":                 schema.StringAttribute{Computed: true},
			"signature":             schema.StringAttribute{Computed: true},
			"phone":                 schema.StringAttribute{Computed: true},
			"external_id":           schema.StringAttribute{Computed: true},
			"organization_id":       schema.Int64Attribute{Computed: true},
			"locale":                schema.StringAttribute{Computed: true},
			"time_zone":             schema.StringAttribute{Computed: true},
			"suspended":             schema.BoolAttribute{Computed: true},
			"restricted_agent":      schema.BoolAttribute{Computed: true},
			"only_private_comments": schema.BoolAttribute{Computed: true},
			"ticket_restriction":    schema.StringAttribute{Computed: true},
			"active":                schema.BoolAttribute{Computed: true},
			"verified":              schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type userSearchResponse struct {
	Users []userReadAPI `json:"users"`
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
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

	var u *userReadAPI

	if hasID {
		var result userReadWrapper
		err := d.client.Get(fmt.Sprintf("/api/v2/users/%s", config.ID.ValueString()), &result)
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		u = &result.User
	} else {
		var result userSearchResponse
		err := d.client.Get(fmt.Sprintf("/api/v2/users/search?query=%s", url.QueryEscape(config.Email.ValueString())), &result)
		if err != nil {
			resp.Diagnostics.AddError("Error searching for user", err.Error())
			return
		}
		email := config.Email.ValueString()
		for i := range result.Users {
			if result.Users[i].Email == email {
				u = &result.Users[i]
				break
			}
		}
		if u == nil {
			resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user found with email %q", email))
			return
		}
	}
	config.ID = types.StringValue(strconv.FormatInt(u.ID, 10))
	config.Name = types.StringValue(u.Name)
	config.Email = types.StringValue(u.Email)
	config.Role = types.StringValue(u.Role)
	config.Alias = types.StringValue(u.Alias)
	config.Details = types.StringValue(u.Details)
	config.Notes = types.StringValue(u.Notes)
	config.Signature = types.StringValue(u.Signature)
	config.Phone = types.StringValue(u.Phone)
	config.TicketRestriction = types.StringValue(u.TicketRestriction)
	config.Locale = types.StringValue(u.Locale)
	config.TimeZone = types.StringValue(u.TimeZone)
	config.Active = types.BoolValue(u.Active)
	config.Verified = types.BoolValue(u.Verified)
	config.Suspended = types.BoolValue(u.Suspended)
	config.RestrictedAgent = types.BoolValue(u.RestrictedAgent)
	config.OnlyPrivateComments = types.BoolValue(u.OnlyPrivateComments)

	if u.CustomRoleID != nil {
		config.CustomRoleID = types.Int64Value(*u.CustomRoleID)
	} else {
		config.CustomRoleID = types.Int64Null()
	}
	if u.DefaultGroupID != nil {
		config.DefaultGroupID = types.Int64Value(*u.DefaultGroupID)
	} else {
		config.DefaultGroupID = types.Int64Null()
	}
	if u.ExternalID != nil {
		config.ExternalID = types.StringValue(*u.ExternalID)
	} else {
		config.ExternalID = types.StringValue("")
	}
	if u.OrganizationID != nil {
		config.OrganizationID = types.Int64Value(*u.OrganizationID)
	} else {
		config.OrganizationID = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
