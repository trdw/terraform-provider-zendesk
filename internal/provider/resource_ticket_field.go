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
	_ resource.Resource                = &TicketFieldResource{}
	_ resource.ResourceWithImportState = &TicketFieldResource{}
)

type TicketFieldResource struct {
	client *ZendeskClient
}

type CustomFieldOptionModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Value        types.String `tfsdk:"value"`
	Position     types.Int64  `tfsdk:"position"`
	AllowSolving types.Bool   `tfsdk:"allow_solving"`
}

type TicketFieldResourceModel struct {
	ID                     types.String             `tfsdk:"id"`
	Type                   types.String             `tfsdk:"type"`
	Title                  types.String             `tfsdk:"title"`
	TitleInPortal          types.String             `tfsdk:"title_in_portal"`
	Description            types.String             `tfsdk:"description"`
	AgentDescription       types.String             `tfsdk:"agent_description"`
	Active                 types.Bool               `tfsdk:"active"`
	Required               types.Bool               `tfsdk:"required"`
	RequiredInPortal       types.Bool               `tfsdk:"required_in_portal"`
	VisibleInPortal        types.Bool               `tfsdk:"visible_in_portal"`
	EditableInPortal       types.Bool               `tfsdk:"editable_in_portal"`
	CollapsedForAgents     types.Bool               `tfsdk:"collapsed_for_agents"`
	Position               types.Int64              `tfsdk:"position"`
	RegexpForValidation    types.String             `tfsdk:"regexp_for_validation"`
	Tag                    types.String             `tfsdk:"tag"`
	SubTypeID              types.Int64              `tfsdk:"sub_type_id"`
	RelationshipTargetType types.String             `tfsdk:"relationship_target_type"`
	CustomFieldOptions     []CustomFieldOptionModel `tfsdk:"custom_field_options"`
	System                 types.Bool               `tfsdk:"system"`
	Removable              types.Bool               `tfsdk:"removable"`
	URL                    types.String             `tfsdk:"url"`
	CreatedAt              types.String             `tfsdk:"created_at"`
	UpdatedAt              types.String             `tfsdk:"updated_at"`
}

type customFieldOptionAPI struct {
	ID           int64  `json:"id,omitempty"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	Position     int64  `json:"position,omitempty"`
	AllowSolving bool   `json:"allow_solving,omitempty"`
}

type ticketFieldAPIObject struct {
	ID                     int64                  `json:"id,omitempty"`
	Type                   string                 `json:"type,omitempty"`
	Title                  string                 `json:"title,omitempty"`
	TitleInPortal          string                 `json:"title_in_portal,omitempty"`
	Description            string                 `json:"description,omitempty"`
	AgentDescription       string                 `json:"agent_description,omitempty"`
	Active                 *bool                  `json:"active,omitempty"`
	Required               *bool                  `json:"required,omitempty"`
	RequiredInPortal       *bool                  `json:"required_in_portal,omitempty"`
	VisibleInPortal        *bool                  `json:"visible_in_portal,omitempty"`
	EditableInPortal       *bool                  `json:"editable_in_portal,omitempty"`
	CollapsedForAgents     *bool                  `json:"collapsed_for_agents,omitempty"`
	Position               int64                  `json:"position,omitempty"`
	RegexpForValidation    *string                `json:"regexp_for_validation,omitempty"`
	Tag                    *string                `json:"tag,omitempty"`
	SubTypeID              *int64                 `json:"sub_type_id,omitempty"`
	RelationshipTargetType string                 `json:"relationship_target_type,omitempty"`
	CustomFieldOptions     []customFieldOptionAPI `json:"custom_field_options,omitempty"`
	System                 bool                   `json:"system,omitempty"`
	Removable              bool                   `json:"removable,omitempty"`
	URL                    string                 `json:"url,omitempty"`
	CreatedAt              string                 `json:"created_at,omitempty"`
	UpdatedAt              string                 `json:"updated_at,omitempty"`
}

type ticketFieldWrapper struct {
	TicketField ticketFieldAPIObject `json:"ticket_field"`
}

func NewTicketFieldResource() resource.Resource {
	return &TicketFieldResource{}
}

func (r *TicketFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ticket_field"
}

func (r *TicketFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk custom ticket field.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The custom field type: checkbox, date, decimal, dropdown, integer, multiselect, regexp, text, textarea, tagger, lookup.",
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the ticket field.",
			},
			"title_in_portal": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the ticket field for end users in Help Center.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Describes the purpose of the ticket field to users.",
			},
			"agent_description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A description of the ticket field that only agents can see.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this field is available.",
			},
			"required": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, agents must enter a value to change the ticket status to solved.",
			},
			"required_in_portal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, end users must enter a value to create the request.",
			},
			"visible_in_portal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this field is visible to end users in Help Center.",
			},
			"editable_in_portal": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this field is editable by end users in Help Center.",
			},
			"collapsed_for_agents": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, the field is shown collapsed to agents by default.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The relative position of the ticket field on a ticket.",
			},
			"regexp_for_validation": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "For regexp fields only. The validation pattern.",
			},
			"tag": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "For checkbox fields only. A tag added when selected.",
			},
			"sub_type_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Sub type ID for system fields of type priority or status.",
			},
			"relationship_target_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "For lookup fields. e.g. zen:user, zen:organization, zen:ticket, zen:custom_object:{key}.",
			},
			"custom_field_options": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Options for dropdown (tagger), multiselect fields.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.Int64Attribute{Computed: true},
						"name":          schema.StringAttribute{Required: true},
						"value":         schema.StringAttribute{Required: true},
						"position":      schema.Int64Attribute{Optional: true, Computed: true},
						"allow_solving": schema.BoolAttribute{Optional: true, Computed: true},
					},
				},
			},
			"system":    schema.BoolAttribute{Computed: true, Description: "Whether this is a system field."},
			"removable": schema.BoolAttribute{Computed: true, Description: "Whether this field can be removed."},
			"url":       schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *TicketFieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TicketFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TicketFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := ticketFieldWrapper{TicketField: buildTicketFieldAPI(&plan)}
	var result ticketFieldWrapper
	if err := r.client.Post("/api/v2/ticket_fields", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating ticket field", err.Error())
		return
	}

	mapTicketFieldToState(&result.TicketField, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TicketFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TicketFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ticketFieldWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/ticket_fields/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ticket field", err.Error())
		return
	}

	mapTicketFieldToState(&result.TicketField, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TicketFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TicketFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TicketFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildTicketFieldAPI(&plan)
	apiObj.Type = ""
	apiReq := ticketFieldWrapper{TicketField: apiObj}
	var result ticketFieldWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/ticket_fields/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating ticket field", err.Error())
		return
	}

	mapTicketFieldToState(&result.TicketField, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TicketFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TicketFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/ticket_fields/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting ticket field", err.Error())
		return
	}
}

func (r *TicketFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildTicketFieldAPI(plan *TicketFieldResourceModel) ticketFieldAPIObject {
	obj := ticketFieldAPIObject{
		Type:  plan.Type.ValueString(),
		Title: plan.Title.ValueString(),
	}
	if !plan.TitleInPortal.IsNull() && !plan.TitleInPortal.IsUnknown() {
		obj.TitleInPortal = plan.TitleInPortal.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		obj.Description = plan.Description.ValueString()
	}
	if !plan.AgentDescription.IsNull() && !plan.AgentDescription.IsUnknown() {
		obj.AgentDescription = plan.AgentDescription.ValueString()
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	if !plan.Required.IsNull() && !plan.Required.IsUnknown() {
		v := plan.Required.ValueBool()
		obj.Required = &v
	}
	if !plan.RequiredInPortal.IsNull() && !plan.RequiredInPortal.IsUnknown() {
		v := plan.RequiredInPortal.ValueBool()
		obj.RequiredInPortal = &v
	}
	if !plan.VisibleInPortal.IsNull() && !plan.VisibleInPortal.IsUnknown() {
		v := plan.VisibleInPortal.ValueBool()
		obj.VisibleInPortal = &v
	}
	if !plan.EditableInPortal.IsNull() && !plan.EditableInPortal.IsUnknown() {
		v := plan.EditableInPortal.ValueBool()
		obj.EditableInPortal = &v
	}
	if !plan.CollapsedForAgents.IsNull() && !plan.CollapsedForAgents.IsUnknown() {
		v := plan.CollapsedForAgents.ValueBool()
		obj.CollapsedForAgents = &v
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		obj.Position = plan.Position.ValueInt64()
	}
	if !plan.RegexpForValidation.IsNull() && !plan.RegexpForValidation.IsUnknown() {
		v := plan.RegexpForValidation.ValueString()
		obj.RegexpForValidation = &v
	}
	if !plan.Tag.IsNull() && !plan.Tag.IsUnknown() {
		v := plan.Tag.ValueString()
		obj.Tag = &v
	}
	if !plan.SubTypeID.IsNull() && !plan.SubTypeID.IsUnknown() {
		v := plan.SubTypeID.ValueInt64()
		obj.SubTypeID = &v
	}
	if !plan.RelationshipTargetType.IsNull() && !plan.RelationshipTargetType.IsUnknown() {
		obj.RelationshipTargetType = plan.RelationshipTargetType.ValueString()
	}
	if plan.CustomFieldOptions != nil {
		obj.CustomFieldOptions = make([]customFieldOptionAPI, len(plan.CustomFieldOptions))
		for i, opt := range plan.CustomFieldOptions {
			obj.CustomFieldOptions[i] = customFieldOptionAPI{
				Name:         opt.Name.ValueString(),
				Value:        opt.Value.ValueString(),
				AllowSolving: opt.AllowSolving.ValueBool(),
			}
			if !opt.Position.IsNull() && !opt.Position.IsUnknown() {
				obj.CustomFieldOptions[i].Position = opt.Position.ValueInt64()
			}
		}
	}
	return obj
}

func mapTicketFieldToState(f *ticketFieldAPIObject, m *TicketFieldResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(f.ID, 10))
	m.Type = types.StringValue(f.Type)
	m.Title = types.StringValue(f.Title)
	m.TitleInPortal = types.StringValue(f.TitleInPortal)
	m.Description = types.StringValue(f.Description)
	m.AgentDescription = types.StringValue(f.AgentDescription)
	if f.Active != nil {
		m.Active = types.BoolValue(*f.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	m.Required = optBoolToTF(f.Required)
	m.RequiredInPortal = optBoolToTF(f.RequiredInPortal)
	m.VisibleInPortal = optBoolToTF(f.VisibleInPortal)
	m.EditableInPortal = optBoolToTF(f.EditableInPortal)
	m.CollapsedForAgents = optBoolToTF(f.CollapsedForAgents)
	m.Position = types.Int64Value(f.Position)
	if f.RegexpForValidation != nil {
		m.RegexpForValidation = types.StringValue(*f.RegexpForValidation)
	} else {
		m.RegexpForValidation = types.StringNull()
	}
	if f.Tag != nil {
		m.Tag = types.StringValue(*f.Tag)
	} else {
		m.Tag = types.StringNull()
	}
	if f.SubTypeID != nil {
		m.SubTypeID = types.Int64Value(*f.SubTypeID)
	} else {
		m.SubTypeID = types.Int64Null()
	}
	m.RelationshipTargetType = types.StringValue(f.RelationshipTargetType)
	if f.CustomFieldOptions != nil {
		m.CustomFieldOptions = make([]CustomFieldOptionModel, len(f.CustomFieldOptions))
		for i, opt := range f.CustomFieldOptions {
			m.CustomFieldOptions[i] = CustomFieldOptionModel{
				ID:           types.Int64Value(opt.ID),
				Name:         types.StringValue(opt.Name),
				Value:        types.StringValue(opt.Value),
				Position:     types.Int64Value(opt.Position),
				AllowSolving: types.BoolValue(opt.AllowSolving),
			}
		}
	} else {
		m.CustomFieldOptions = nil
	}
	m.System = types.BoolValue(f.System)
	m.Removable = types.BoolValue(f.Removable)
	m.URL = types.StringValue(f.URL)
	m.CreatedAt = types.StringValue(f.CreatedAt)
	m.UpdatedAt = types.StringValue(f.UpdatedAt)
}
