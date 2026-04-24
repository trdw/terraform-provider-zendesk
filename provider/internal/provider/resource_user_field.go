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
	_ resource.Resource                = &UserFieldResource{}
	_ resource.ResourceWithImportState = &UserFieldResource{}
)

type UserFieldResource struct {
	client *ZendeskClient
}

type UserFieldResourceModel struct {
	ID                     types.String             `tfsdk:"id"`
	Key                    types.String             `tfsdk:"key"`
	Type                   types.String             `tfsdk:"type"`
	Title                  types.String             `tfsdk:"title"`
	Description            types.String             `tfsdk:"description"`
	Active                 types.Bool               `tfsdk:"active"`
	Position               types.Int64              `tfsdk:"position"`
	RegexpForValidation    types.String             `tfsdk:"regexp_for_validation"`
	Tag                    types.String             `tfsdk:"tag"`
	RelationshipTargetType types.String             `tfsdk:"relationship_target_type"`
	CustomFieldOptions     []CustomFieldOptionModel `tfsdk:"custom_field_options"`
	System                 types.Bool               `tfsdk:"system"`
	URL                    types.String             `tfsdk:"url"`
	CreatedAt              types.String             `tfsdk:"created_at"`
	UpdatedAt              types.String             `tfsdk:"updated_at"`
}

type userFieldAPIObject struct {
	ID                     int64                  `json:"id,omitempty"`
	Key                    string                 `json:"key,omitempty"`
	Type                   string                 `json:"type,omitempty"`
	Title                  string                 `json:"title,omitempty"`
	Description            string                 `json:"description,omitempty"`
	Active                 *bool                  `json:"active,omitempty"`
	Position               int64                  `json:"position,omitempty"`
	RegexpForValidation    *string                `json:"regexp_for_validation,omitempty"`
	Tag                    *string                `json:"tag,omitempty"`
	RelationshipTargetType string                 `json:"relationship_target_type,omitempty"`
	CustomFieldOptions     []customFieldOptionAPI `json:"custom_field_options,omitempty"`
	System                 bool                   `json:"system,omitempty"`
	URL                    string                 `json:"url,omitempty"`
	CreatedAt              string                 `json:"created_at,omitempty"`
	UpdatedAt              string                 `json:"updated_at,omitempty"`
}

type userFieldWrapper struct {
	UserField userFieldAPIObject `json:"user_field"`
}

func NewUserFieldResource() resource.Resource {
	return &UserFieldResource{}
}

func (r *UserFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_field"
}

func (r *UserFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk custom user field.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "A unique key that identifies this custom field.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The custom field type: checkbox, date, decimal, dropdown, integer, lookup, regexp, text, textarea.",
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the custom field.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "User-defined description of this field's purpose.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, this field is available for use.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Ordering of the field relative to other fields.",
			},
			"regexp_for_validation": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Regular expression field only. The validation pattern.",
			},
			"tag": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional for checkbox type.",
			},
			"relationship_target_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "For lookup fields. e.g. zen:user, zen:organization, zen:ticket, zen:custom_object:{key}.",
			},
			"custom_field_options": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Options for dropdown fields.",
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
			"system": schema.BoolAttribute{Computed: true, Description: "If true, only active and position can be changed."},
			"url":    schema.StringAttribute{Computed: true},
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

func (r *UserFieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := userFieldWrapper{UserField: buildUserFieldAPI(&plan)}
	var result userFieldWrapper
	if err := r.client.Post("/api/v2/user_fields", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating user field", err.Error())
		return
	}

	mapUserFieldToState(&result.UserField, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result userFieldWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/user_fields/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user field", err.Error())
		return
	}

	mapUserFieldToState(&result.UserField, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildUserFieldAPI(&plan)
	apiObj.Type = ""
	apiObj.Key = ""
	apiReq := userFieldWrapper{UserField: apiObj}
	var result userFieldWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/user_fields/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating user field", err.Error())
		return
	}

	mapUserFieldToState(&result.UserField, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/user_fields/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting user field", err.Error())
		return
	}
}

func (r *UserFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildUserFieldAPI(plan *UserFieldResourceModel) userFieldAPIObject {
	obj := userFieldAPIObject{
		Key:   plan.Key.ValueString(),
		Type:  plan.Type.ValueString(),
		Title: plan.Title.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		obj.Description = plan.Description.ValueString()
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
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
	if !plan.RelationshipTargetType.IsNull() && !plan.RelationshipTargetType.IsUnknown() {
		obj.RelationshipTargetType = plan.RelationshipTargetType.ValueString()
	}
	if plan.CustomFieldOptions != nil {
		obj.CustomFieldOptions = make([]customFieldOptionAPI, len(plan.CustomFieldOptions))
		for i, opt := range plan.CustomFieldOptions {
			obj.CustomFieldOptions[i] = customFieldOptionAPI{
				Name:  opt.Name.ValueString(),
				Value: opt.Value.ValueString(),
			}
			if !opt.Position.IsNull() && !opt.Position.IsUnknown() {
				obj.CustomFieldOptions[i].Position = opt.Position.ValueInt64()
			}
		}
	}
	return obj
}

func mapUserFieldToState(f *userFieldAPIObject, m *UserFieldResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(f.ID, 10))
	m.Key = types.StringValue(f.Key)
	m.Type = types.StringValue(f.Type)
	m.Title = types.StringValue(f.Title)
	m.Description = types.StringValue(f.Description)
	if f.Active != nil {
		m.Active = types.BoolValue(*f.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
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
	m.URL = types.StringValue(f.URL)
	m.CreatedAt = types.StringValue(f.CreatedAt)
	m.UpdatedAt = types.StringValue(f.UpdatedAt)
}
