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
	_ resource.Resource                = &CustomRoleResource{}
	_ resource.ResourceWithImportState = &CustomRoleResource{}
)

type CustomRoleResource struct {
	client *ZendeskClient
}

type CustomRoleConfigModel struct {
	AssignTicketsToAnyGroup     types.Bool   `tfsdk:"assign_tickets_to_any_group"`
	ChatAccess                  types.Bool   `tfsdk:"chat_access"`
	EndUserListAccess           types.String `tfsdk:"end_user_list_access"`
	EndUserProfileAccess        types.String `tfsdk:"end_user_profile_access"`
	ExploreAccess               types.String `tfsdk:"explore_access"`
	ForumAccess                 types.String `tfsdk:"forum_access"`
	GroupAccess                 types.Bool   `tfsdk:"group_access"`
	LightAgent                  types.Bool   `tfsdk:"light_agent"`
	MacroAccess                 types.String `tfsdk:"macro_access"`
	ManageBusinessRules         types.Bool   `tfsdk:"manage_business_rules"`
	ManageContextualWorkspaces  types.Bool   `tfsdk:"manage_contextual_workspaces"`
	ManageDynamicContent        types.Bool   `tfsdk:"manage_dynamic_content"`
	ManageExtensionsAndChannels types.Bool   `tfsdk:"manage_extensions_and_channels"`
	ManageFacebook              types.Bool   `tfsdk:"manage_facebook"`
	ManageOrganizationFields    types.Bool   `tfsdk:"manage_organization_fields"`
	ManageTicketFields          types.Bool   `tfsdk:"manage_ticket_fields"`
	ManageTicketForms           types.Bool   `tfsdk:"manage_ticket_forms"`
	ManageUserFields            types.Bool   `tfsdk:"manage_user_fields"`
	ModerateForums              types.Bool   `tfsdk:"moderate_forums"`
	OrganizationEditing         types.Bool   `tfsdk:"organization_editing"`
	OrganizationNotesEditing    types.Bool   `tfsdk:"organization_notes_editing"`
	ReportAccess                types.String `tfsdk:"report_access"`
	SideConversationCreate      types.Bool   `tfsdk:"side_conversation_create"`
	TicketAccess                types.String `tfsdk:"ticket_access"`
	TicketCommentAccess         types.String `tfsdk:"ticket_comment_access"`
	TicketDeletion              types.Bool   `tfsdk:"ticket_deletion"`
	TicketEditing               types.Bool   `tfsdk:"ticket_editing"`
	TicketMerge                 types.Bool   `tfsdk:"ticket_merge"`
	TicketTagEditing            types.Bool   `tfsdk:"ticket_tag_editing"`
	ViewAccess                  types.String `tfsdk:"view_access"`
	ViewDeletedTickets          types.Bool   `tfsdk:"view_deleted_tickets"`
	VoiceAccess                 types.Bool   `tfsdk:"voice_access"`
	VoiceDashboardAccess        types.Bool   `tfsdk:"voice_dashboard_access"`
}

type CustomRoleResourceModel struct {
	ID              types.String           `tfsdk:"id"`
	Name            types.String           `tfsdk:"name"`
	Description     types.String           `tfsdk:"description"`
	RoleType        types.Int64            `tfsdk:"role_type"`
	TeamMemberCount types.Int64            `tfsdk:"team_member_count"`
	Configuration   *CustomRoleConfigModel `tfsdk:"configuration"`
	CreatedAt       types.String           `tfsdk:"created_at"`
	UpdatedAt       types.String           `tfsdk:"updated_at"`
}

type customRoleConfigAPI struct {
	AssignTicketsToAnyGroup     *bool   `json:"assign_tickets_to_any_group,omitempty"`
	ChatAccess                  *bool   `json:"chat_access,omitempty"`
	EndUserListAccess           string  `json:"end_user_list_access,omitempty"`
	EndUserProfileAccess        string  `json:"end_user_profile_access,omitempty"`
	ExploreAccess               string  `json:"explore_access,omitempty"`
	ForumAccess                 string  `json:"forum_access,omitempty"`
	GroupAccess                 *bool   `json:"group_access,omitempty"`
	LightAgent                  *bool   `json:"light_agent,omitempty"`
	MacroAccess                 string  `json:"macro_access,omitempty"`
	ManageBusinessRules         *bool   `json:"manage_business_rules,omitempty"`
	ManageContextualWorkspaces  *bool   `json:"manage_contextual_workspaces,omitempty"`
	ManageDynamicContent        *bool   `json:"manage_dynamic_content,omitempty"`
	ManageExtensionsAndChannels *bool   `json:"manage_extensions_and_channels,omitempty"`
	ManageFacebook              *bool   `json:"manage_facebook,omitempty"`
	ManageOrganizationFields    *bool   `json:"manage_organization_fields,omitempty"`
	ManageTicketFields          *bool   `json:"manage_ticket_fields,omitempty"`
	ManageTicketForms           *bool   `json:"manage_ticket_forms,omitempty"`
	ManageUserFields            *bool   `json:"manage_user_fields,omitempty"`
	ModerateForums              *bool   `json:"moderate_forums,omitempty"`
	OrganizationEditing         *bool   `json:"organization_editing,omitempty"`
	OrganizationNotesEditing    *bool   `json:"organization_notes_editing,omitempty"`
	ReportAccess                string  `json:"report_access,omitempty"`
	SideConversationCreate      *bool   `json:"side_conversation_create,omitempty"`
	TicketAccess                string  `json:"ticket_access,omitempty"`
	TicketCommentAccess         string  `json:"ticket_comment_access,omitempty"`
	TicketDeletion              *bool   `json:"ticket_deletion,omitempty"`
	TicketEditing               *bool   `json:"ticket_editing,omitempty"`
	TicketMerge                 *bool   `json:"ticket_merge,omitempty"`
	TicketTagEditing            *bool   `json:"ticket_tag_editing,omitempty"`
	ViewAccess                  string  `json:"view_access,omitempty"`
	ViewDeletedTickets          *bool   `json:"view_deleted_tickets,omitempty"`
	VoiceAccess                 *bool   `json:"voice_access,omitempty"`
	VoiceDashboardAccess        *bool   `json:"voice_dashboard_access,omitempty"`
}

type customRoleAPIObject struct {
	ID              int64                `json:"id,omitempty"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	RoleType        int64                `json:"role_type"`
	TeamMemberCount int64                `json:"team_member_count,omitempty"`
	Configuration   *customRoleConfigAPI `json:"configuration,omitempty"`
	CreatedAt       string               `json:"created_at,omitempty"`
	UpdatedAt       string               `json:"updated_at,omitempty"`
}

type customRoleWrapper struct {
	CustomRole customRoleAPIObject `json:"custom_role"`
}

func NewCustomRoleResource() resource.Resource {
	return &CustomRoleResource{}
}

func (r *CustomRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_role"
}

func (r *CustomRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk custom agent role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the custom role.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A description of the custom role.",
			},
			"role_type": schema.Int64Attribute{
				Required:    true,
				Description: "The role type: 0=custom agent, 1=light agent, 2=chat agent, 3=contributor, 4=admin, 5=billing admin.",
			},
			"team_member_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of team members with this role.",
			},
			"configuration": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Permissions configuration for the role.",
				Attributes: map[string]schema.Attribute{
					"assign_tickets_to_any_group": schema.BoolAttribute{Optional: true, Computed: true},
					"chat_access":                 schema.BoolAttribute{Optional: true, Computed: true},
					"end_user_list_access":         schema.StringAttribute{Optional: true, Computed: true, Description: "full, none"},
					"end_user_profile_access":      schema.StringAttribute{Optional: true, Computed: true, Description: "edit, edit-within-org, full, readonly"},
					"explore_access":               schema.StringAttribute{Optional: true, Computed: true, Description: "edit, full, none, readonly"},
					"forum_access":                 schema.StringAttribute{Optional: true, Computed: true, Description: "edit-topics, full, readonly"},
					"group_access":                schema.BoolAttribute{Optional: true, Computed: true},
					"light_agent":                 schema.BoolAttribute{Optional: true, Computed: true},
					"macro_access":                schema.StringAttribute{Optional: true, Computed: true, Description: "full, manage-group, manage-personal, readonly"},
					"manage_business_rules":        schema.BoolAttribute{Optional: true, Computed: true},
					"manage_contextual_workspaces": schema.BoolAttribute{Optional: true, Computed: true},
					"manage_dynamic_content":       schema.BoolAttribute{Optional: true, Computed: true},
					"manage_extensions_and_channels": schema.BoolAttribute{Optional: true, Computed: true},
					"manage_facebook":              schema.BoolAttribute{Optional: true, Computed: true},
					"manage_organization_fields":   schema.BoolAttribute{Optional: true, Computed: true},
					"manage_ticket_fields":         schema.BoolAttribute{Optional: true, Computed: true},
					"manage_ticket_forms":          schema.BoolAttribute{Optional: true, Computed: true},
					"manage_user_fields":           schema.BoolAttribute{Optional: true, Computed: true},
					"moderate_forums":              schema.BoolAttribute{Optional: true, Computed: true},
					"organization_editing":         schema.BoolAttribute{Optional: true, Computed: true},
					"organization_notes_editing":   schema.BoolAttribute{Optional: true, Computed: true},
					"report_access":               schema.StringAttribute{Optional: true, Computed: true, Description: "full, none, readonly"},
					"side_conversation_create":     schema.BoolAttribute{Optional: true, Computed: true},
					"ticket_access":               schema.StringAttribute{Optional: true, Computed: true, Description: "all, assigned-only, within-groups, within-groups-and-public"},
					"ticket_comment_access":        schema.StringAttribute{Optional: true, Computed: true, Description: "public, none"},
					"ticket_deletion":             schema.BoolAttribute{Optional: true, Computed: true},
					"ticket_editing":              schema.BoolAttribute{Optional: true, Computed: true},
					"ticket_merge":                schema.BoolAttribute{Optional: true, Computed: true},
					"ticket_tag_editing":           schema.BoolAttribute{Optional: true, Computed: true},
					"view_access":                 schema.StringAttribute{Optional: true, Computed: true, Description: "full, manage-group, manage-personal, readonly, playonly"},
					"view_deleted_tickets":         schema.BoolAttribute{Optional: true, Computed: true},
					"voice_access":                schema.BoolAttribute{Optional: true, Computed: true},
					"voice_dashboard_access":       schema.BoolAttribute{Optional: true, Computed: true},
				},
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

func (r *CustomRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildCustomRoleAPI(&plan)
	apiReq := customRoleWrapper{CustomRole: apiObj}

	var result customRoleWrapper
	if err := r.client.Post("/api/v2/custom_roles", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating custom role", err.Error())
		return
	}

	mapCustomRoleToState(&result.CustomRole, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result customRoleWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/custom_roles/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom role", err.Error())
		return
	}

	mapCustomRoleToState(&result.CustomRole, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CustomRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CustomRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildCustomRoleAPI(&plan)
	apiReq := customRoleWrapper{CustomRole: apiObj}

	var result customRoleWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/custom_roles/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating custom role", err.Error())
		return
	}

	mapCustomRoleToState(&result.CustomRole, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/custom_roles/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting custom role", err.Error())
		return
	}
}

func (r *CustomRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func boolPtr(v bool) *bool { return &v }

func buildCustomRoleAPI(plan *CustomRoleResourceModel) customRoleAPIObject {
	obj := customRoleAPIObject{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		RoleType:    plan.RoleType.ValueInt64(),
	}

	if plan.Configuration != nil {
		cfg := &customRoleConfigAPI{}
		c := plan.Configuration

		if !c.AssignTicketsToAnyGroup.IsNull() {
			cfg.AssignTicketsToAnyGroup = boolPtr(c.AssignTicketsToAnyGroup.ValueBool())
		}
		if !c.ChatAccess.IsNull() {
			cfg.ChatAccess = boolPtr(c.ChatAccess.ValueBool())
		}
		if !c.EndUserListAccess.IsNull() {
			cfg.EndUserListAccess = c.EndUserListAccess.ValueString()
		}
		if !c.EndUserProfileAccess.IsNull() {
			cfg.EndUserProfileAccess = c.EndUserProfileAccess.ValueString()
		}
		if !c.ExploreAccess.IsNull() {
			cfg.ExploreAccess = c.ExploreAccess.ValueString()
		}
		if !c.ForumAccess.IsNull() {
			cfg.ForumAccess = c.ForumAccess.ValueString()
		}
		if !c.GroupAccess.IsNull() {
			cfg.GroupAccess = boolPtr(c.GroupAccess.ValueBool())
		}
		if !c.LightAgent.IsNull() {
			cfg.LightAgent = boolPtr(c.LightAgent.ValueBool())
		}
		if !c.MacroAccess.IsNull() {
			cfg.MacroAccess = c.MacroAccess.ValueString()
		}
		if !c.ManageBusinessRules.IsNull() {
			cfg.ManageBusinessRules = boolPtr(c.ManageBusinessRules.ValueBool())
		}
		if !c.ManageContextualWorkspaces.IsNull() {
			cfg.ManageContextualWorkspaces = boolPtr(c.ManageContextualWorkspaces.ValueBool())
		}
		if !c.ManageDynamicContent.IsNull() {
			cfg.ManageDynamicContent = boolPtr(c.ManageDynamicContent.ValueBool())
		}
		if !c.ManageExtensionsAndChannels.IsNull() {
			cfg.ManageExtensionsAndChannels = boolPtr(c.ManageExtensionsAndChannels.ValueBool())
		}
		if !c.ManageFacebook.IsNull() {
			cfg.ManageFacebook = boolPtr(c.ManageFacebook.ValueBool())
		}
		if !c.ManageOrganizationFields.IsNull() {
			cfg.ManageOrganizationFields = boolPtr(c.ManageOrganizationFields.ValueBool())
		}
		if !c.ManageTicketFields.IsNull() {
			cfg.ManageTicketFields = boolPtr(c.ManageTicketFields.ValueBool())
		}
		if !c.ManageTicketForms.IsNull() {
			cfg.ManageTicketForms = boolPtr(c.ManageTicketForms.ValueBool())
		}
		if !c.ManageUserFields.IsNull() {
			cfg.ManageUserFields = boolPtr(c.ManageUserFields.ValueBool())
		}
		if !c.ModerateForums.IsNull() {
			cfg.ModerateForums = boolPtr(c.ModerateForums.ValueBool())
		}
		if !c.OrganizationEditing.IsNull() {
			cfg.OrganizationEditing = boolPtr(c.OrganizationEditing.ValueBool())
		}
		if !c.OrganizationNotesEditing.IsNull() {
			cfg.OrganizationNotesEditing = boolPtr(c.OrganizationNotesEditing.ValueBool())
		}
		if !c.ReportAccess.IsNull() {
			cfg.ReportAccess = c.ReportAccess.ValueString()
		}
		if !c.SideConversationCreate.IsNull() {
			cfg.SideConversationCreate = boolPtr(c.SideConversationCreate.ValueBool())
		}
		if !c.TicketAccess.IsNull() {
			cfg.TicketAccess = c.TicketAccess.ValueString()
		}
		if !c.TicketCommentAccess.IsNull() {
			cfg.TicketCommentAccess = c.TicketCommentAccess.ValueString()
		}
		if !c.TicketDeletion.IsNull() {
			cfg.TicketDeletion = boolPtr(c.TicketDeletion.ValueBool())
		}
		if !c.TicketEditing.IsNull() {
			cfg.TicketEditing = boolPtr(c.TicketEditing.ValueBool())
		}
		if !c.TicketMerge.IsNull() {
			cfg.TicketMerge = boolPtr(c.TicketMerge.ValueBool())
		}
		if !c.TicketTagEditing.IsNull() {
			cfg.TicketTagEditing = boolPtr(c.TicketTagEditing.ValueBool())
		}
		if !c.ViewAccess.IsNull() {
			cfg.ViewAccess = c.ViewAccess.ValueString()
		}
		if !c.ViewDeletedTickets.IsNull() {
			cfg.ViewDeletedTickets = boolPtr(c.ViewDeletedTickets.ValueBool())
		}
		if !c.VoiceAccess.IsNull() {
			cfg.VoiceAccess = boolPtr(c.VoiceAccess.ValueBool())
		}
		if !c.VoiceDashboardAccess.IsNull() {
			cfg.VoiceDashboardAccess = boolPtr(c.VoiceDashboardAccess.ValueBool())
		}
		obj.Configuration = cfg
	}

	return obj
}

func mapCustomRoleToState(r *customRoleAPIObject, m *CustomRoleResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(r.ID, 10))
	m.Name = types.StringValue(r.Name)
	m.Description = types.StringValue(r.Description)
	m.RoleType = types.Int64Value(r.RoleType)
	m.TeamMemberCount = types.Int64Value(r.TeamMemberCount)
	m.CreatedAt = types.StringValue(r.CreatedAt)
	m.UpdatedAt = types.StringValue(r.UpdatedAt)

	if r.Configuration != nil {
		cfg := &CustomRoleConfigModel{}
		c := r.Configuration

		cfg.AssignTicketsToAnyGroup = optBoolToTF(c.AssignTicketsToAnyGroup)
		cfg.ChatAccess = optBoolToTF(c.ChatAccess)
		cfg.EndUserListAccess = types.StringValue(c.EndUserListAccess)
		cfg.EndUserProfileAccess = types.StringValue(c.EndUserProfileAccess)
		cfg.ExploreAccess = types.StringValue(c.ExploreAccess)
		cfg.ForumAccess = types.StringValue(c.ForumAccess)
		cfg.GroupAccess = optBoolToTF(c.GroupAccess)
		cfg.LightAgent = optBoolToTF(c.LightAgent)
		cfg.MacroAccess = types.StringValue(c.MacroAccess)
		cfg.ManageBusinessRules = optBoolToTF(c.ManageBusinessRules)
		cfg.ManageContextualWorkspaces = optBoolToTF(c.ManageContextualWorkspaces)
		cfg.ManageDynamicContent = optBoolToTF(c.ManageDynamicContent)
		cfg.ManageExtensionsAndChannels = optBoolToTF(c.ManageExtensionsAndChannels)
		cfg.ManageFacebook = optBoolToTF(c.ManageFacebook)
		cfg.ManageOrganizationFields = optBoolToTF(c.ManageOrganizationFields)
		cfg.ManageTicketFields = optBoolToTF(c.ManageTicketFields)
		cfg.ManageTicketForms = optBoolToTF(c.ManageTicketForms)
		cfg.ManageUserFields = optBoolToTF(c.ManageUserFields)
		cfg.ModerateForums = optBoolToTF(c.ModerateForums)
		cfg.OrganizationEditing = optBoolToTF(c.OrganizationEditing)
		cfg.OrganizationNotesEditing = optBoolToTF(c.OrganizationNotesEditing)
		cfg.ReportAccess = types.StringValue(c.ReportAccess)
		cfg.SideConversationCreate = optBoolToTF(c.SideConversationCreate)
		cfg.TicketAccess = types.StringValue(c.TicketAccess)
		cfg.TicketCommentAccess = types.StringValue(c.TicketCommentAccess)
		cfg.TicketDeletion = optBoolToTF(c.TicketDeletion)
		cfg.TicketEditing = optBoolToTF(c.TicketEditing)
		cfg.TicketMerge = optBoolToTF(c.TicketMerge)
		cfg.TicketTagEditing = optBoolToTF(c.TicketTagEditing)
		cfg.ViewAccess = types.StringValue(c.ViewAccess)
		cfg.ViewDeletedTickets = optBoolToTF(c.ViewDeletedTickets)
		cfg.VoiceAccess = optBoolToTF(c.VoiceAccess)
		cfg.VoiceDashboardAccess = optBoolToTF(c.VoiceDashboardAccess)

		m.Configuration = cfg
	}
}

func optBoolToTF(v *bool) types.Bool {
	if v == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*v)
}
