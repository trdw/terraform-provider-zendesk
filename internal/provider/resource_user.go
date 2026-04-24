package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

type UserResource struct {
	client *ZendeskClient
}

type UserResourceModel struct {
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
	LastLoginAt         types.String `tfsdk:"last_login_at"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

type userCreateUpdateAPI struct {
	Name                string  `json:"name"`
	Email               string  `json:"email,omitempty"`
	Role                string  `json:"role,omitempty"`
	CustomRoleID        *int64  `json:"custom_role_id,omitempty"`
	DefaultGroupID      *int64  `json:"default_group_id,omitempty"`
	Alias               string  `json:"alias,omitempty"`
	Details             string  `json:"details,omitempty"`
	Notes               string  `json:"notes,omitempty"`
	Signature           string  `json:"signature,omitempty"`
	Phone               string  `json:"phone,omitempty"`
	ExternalID          string  `json:"external_id,omitempty"`
	OrganizationID      *int64  `json:"organization_id,omitempty"`
	Locale              string  `json:"locale,omitempty"`
	TimeZone            string  `json:"time_zone,omitempty"`
	Suspended           *bool   `json:"suspended,omitempty"`
	RestrictedAgent     *bool   `json:"restricted_agent,omitempty"`
	OnlyPrivateComments *bool   `json:"only_private_comments,omitempty"`
	TicketRestriction   string  `json:"ticket_restriction,omitempty"`
	Verified            *bool   `json:"verified,omitempty"`
	SkipVerifyEmail     *bool   `json:"skip_verify_email,omitempty"`
}

type userReadAPI struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	Email               string  `json:"email"`
	Role                string  `json:"role"`
	CustomRoleID        *int64  `json:"custom_role_id"`
	DefaultGroupID      *int64  `json:"default_group_id"`
	Alias               string  `json:"alias"`
	Details             string  `json:"details"`
	Notes               string  `json:"notes"`
	Signature           string  `json:"signature"`
	Phone               string  `json:"phone"`
	ExternalID          *string `json:"external_id"`
	OrganizationID      *int64  `json:"organization_id"`
	Locale              string  `json:"locale"`
	TimeZone            string  `json:"time_zone"`
	Suspended           bool    `json:"suspended"`
	RestrictedAgent     bool    `json:"restricted_agent"`
	OnlyPrivateComments bool    `json:"only_private_comments"`
	TicketRestriction   string  `json:"ticket_restriction"`
	Active              bool    `json:"active"`
	Verified            bool    `json:"verified"`
	LastLoginAt         *string `json:"last_login_at"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type userCreateWrapper struct {
	User userCreateUpdateAPI `json:"user"`
}

type userReadWrapper struct {
	User userReadAPI `json:"user"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk user (agent). Can be used to create and manage agent accounts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The user's name.",
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "The user's primary email address.",
			},
			"role": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's role: end-user, agent, or admin.",
			},
			"custom_role_id": schema.Int64Attribute{
				Optional:    true,
				Description: "A custom role ID if the user is an agent on the Enterprise plan.",
			},
			"default_group_id": schema.Int64Attribute{
				Optional:    true,
				Description: "The default group ID for the user.",
			},
			"alias": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "An alias displayed to end users instead of the agent's real name.",
			},
			"details": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Details about the user, such as an address.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Notes about the user.",
			},
			"signature": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's signature for email responses.",
			},
			"phone": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's phone number.",
			},
			"external_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A unique identifier from another system.",
			},
			"organization_id": schema.Int64Attribute{
				Optional:    true,
				Description: "The ID of the user's organization.",
			},
			"locale": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's locale (e.g. en-US).",
			},
			"time_zone": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's time zone.",
			},
			"suspended": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the user's access is suspended.",
			},
			"restricted_agent": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the agent has restrictions.",
			},
			"only_private_comments": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the user can only create private comments.",
			},
			"ticket_restriction": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Which tickets the user can access: organization, groups, assigned, requested.",
			},
			"active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user is active (not deleted).",
			},
			"verified": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the identity of the user has been verified.",
			},
			"last_login_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the user's last login.",
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*ZendeskClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *ZendeskClient")
		return
	}
	r.client = client
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildUserAPI(&plan)
	apiObj.SkipVerifyEmail = boolPtr(true)
	apiReq := userCreateWrapper{User: apiObj}

	var result userReadWrapper
	if err := r.client.Post("/api/v2/users", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	mapUserToState(&result.User, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result userReadWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/users/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	mapUserToState(&result.User, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildUserAPI(&plan)
	apiReq := userCreateWrapper{User: apiObj}

	var result userReadWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/users/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	mapUserToState(&result.User, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/users/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildUserAPI(plan *UserResourceModel) userCreateUpdateAPI {
	obj := userCreateUpdateAPI{
		Name:  plan.Name.ValueString(),
		Email: plan.Email.ValueString(),
	}

	if !plan.Role.IsNull() && !plan.Role.IsUnknown() {
		obj.Role = plan.Role.ValueString()
	}
	if !plan.CustomRoleID.IsNull() && !plan.CustomRoleID.IsUnknown() {
		v := plan.CustomRoleID.ValueInt64()
		obj.CustomRoleID = &v
	}
	if !plan.DefaultGroupID.IsNull() && !plan.DefaultGroupID.IsUnknown() {
		v := plan.DefaultGroupID.ValueInt64()
		obj.DefaultGroupID = &v
	}
	if !plan.Alias.IsNull() && !plan.Alias.IsUnknown() {
		obj.Alias = plan.Alias.ValueString()
	}
	if !plan.Details.IsNull() && !plan.Details.IsUnknown() {
		obj.Details = plan.Details.ValueString()
	}
	if !plan.Notes.IsNull() && !plan.Notes.IsUnknown() {
		obj.Notes = plan.Notes.ValueString()
	}
	if !plan.Signature.IsNull() && !plan.Signature.IsUnknown() {
		obj.Signature = plan.Signature.ValueString()
	}
	if !plan.Phone.IsNull() && !plan.Phone.IsUnknown() {
		obj.Phone = plan.Phone.ValueString()
	}
	if !plan.ExternalID.IsNull() && !plan.ExternalID.IsUnknown() {
		obj.ExternalID = plan.ExternalID.ValueString()
	}
	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() {
		v := plan.OrganizationID.ValueInt64()
		obj.OrganizationID = &v
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() {
		obj.Locale = plan.Locale.ValueString()
	}
	if !plan.TimeZone.IsNull() && !plan.TimeZone.IsUnknown() {
		obj.TimeZone = plan.TimeZone.ValueString()
	}
	if !plan.Suspended.IsNull() && !plan.Suspended.IsUnknown() {
		v := plan.Suspended.ValueBool()
		obj.Suspended = &v
	}
	if !plan.RestrictedAgent.IsNull() && !plan.RestrictedAgent.IsUnknown() {
		v := plan.RestrictedAgent.ValueBool()
		obj.RestrictedAgent = &v
	}
	if !plan.OnlyPrivateComments.IsNull() && !plan.OnlyPrivateComments.IsUnknown() {
		v := plan.OnlyPrivateComments.ValueBool()
		obj.OnlyPrivateComments = &v
	}
	if !plan.TicketRestriction.IsNull() && !plan.TicketRestriction.IsUnknown() {
		obj.TicketRestriction = plan.TicketRestriction.ValueString()
	}
	if !plan.Verified.IsNull() && !plan.Verified.IsUnknown() {
		v := plan.Verified.ValueBool()
		obj.Verified = &v
	}

	return obj
}

func mapUserToState(u *userReadAPI, m *UserResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(u.ID, 10))
	m.Name = types.StringValue(u.Name)
	m.Email = types.StringValue(u.Email)
	m.Role = types.StringValue(u.Role)
	m.Active = types.BoolValue(u.Active)
	m.Verified = types.BoolValue(u.Verified)
	m.Suspended = types.BoolValue(u.Suspended)
	m.RestrictedAgent = types.BoolValue(u.RestrictedAgent)
	m.OnlyPrivateComments = types.BoolValue(u.OnlyPrivateComments)
	m.Alias = types.StringValue(u.Alias)
	m.Details = types.StringValue(u.Details)
	m.Notes = types.StringValue(u.Notes)
	m.Signature = types.StringValue(u.Signature)
	m.Phone = types.StringValue(u.Phone)
	m.TicketRestriction = types.StringValue(u.TicketRestriction)
	m.Locale = types.StringValue(u.Locale)
	m.TimeZone = types.StringValue(u.TimeZone)
	m.CreatedAt = types.StringValue(u.CreatedAt)
	m.UpdatedAt = types.StringValue(u.UpdatedAt)

	if u.CustomRoleID != nil {
		m.CustomRoleID = types.Int64Value(*u.CustomRoleID)
	} else {
		m.CustomRoleID = types.Int64Null()
	}
	if u.DefaultGroupID != nil {
		m.DefaultGroupID = types.Int64Value(*u.DefaultGroupID)
	} else {
		m.DefaultGroupID = types.Int64Null()
	}
	if u.ExternalID != nil {
		m.ExternalID = types.StringValue(*u.ExternalID)
	} else {
		m.ExternalID = types.StringValue("")
	}
	if u.OrganizationID != nil {
		m.OrganizationID = types.Int64Value(*u.OrganizationID)
	} else {
		m.OrganizationID = types.Int64Null()
	}
	if u.LastLoginAt != nil {
		m.LastLoginAt = types.StringValue(*u.LastLoginAt)
	} else {
		m.LastLoginAt = types.StringNull()
	}
}
