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
	_ resource.Resource                = &TriggerResource{}
	_ resource.ResourceWithImportState = &TriggerResource{}
)

type TriggerResource struct {
	client *ZendeskClient
}

type TriggerConditionModel struct {
	Field    types.String `tfsdk:"field"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

type TriggerActionModel struct {
	Field types.String `tfsdk:"field"`
	Value types.String `tfsdk:"value"`
}

type TriggerResourceModel struct {
	ID          types.String            `tfsdk:"id"`
	Title       types.String            `tfsdk:"title"`
	Active      types.Bool              `tfsdk:"active"`
	Description types.String            `tfsdk:"description"`
	CategoryID  types.String            `tfsdk:"category_id"`
	Position    types.Int64             `tfsdk:"position"`
	Default     types.Bool              `tfsdk:"default"`
	CondAll     []TriggerConditionModel `tfsdk:"condition_all"`
	CondAny     []TriggerConditionModel `tfsdk:"condition_any"`
	Actions     []TriggerActionModel    `tfsdk:"actions"`
	CreatedAt   types.String            `tfsdk:"created_at"`
	UpdatedAt   types.String            `tfsdk:"updated_at"`
}

type triggerConditionAPI struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type triggerActionAPI struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type triggerConditionsAPI struct {
	All []triggerConditionAPI `json:"all,omitempty"`
	Any []triggerConditionAPI `json:"any,omitempty"`
}

type triggerAPIObject struct {
	ID          int64                `json:"id,omitempty"`
	Title       string               `json:"title"`
	Active      *bool                `json:"active,omitempty"`
	Description string               `json:"description,omitempty"`
	CategoryID  *string              `json:"category_id,omitempty"`
	Position    int64                `json:"position,omitempty"`
	Default     bool                 `json:"default,omitempty"`
	Conditions  *triggerConditionsAPI `json:"conditions,omitempty"`
	Actions     []triggerActionAPI   `json:"actions"`
	CreatedAt   string               `json:"created_at,omitempty"`
	UpdatedAt   string               `json:"updated_at,omitempty"`
}

type triggerWrapper struct {
	Trigger triggerAPIObject `json:"trigger"`
}

func NewTriggerResource() resource.Resource {
	return &TriggerResource{}
}

func (r *TriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (r *TriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	conditionSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"field": schema.StringAttribute{
				Required:    true,
				Description: "The ticket field to evaluate.",
			},
			"operator": schema.StringAttribute{
				Required:    true,
				Description: "The comparison operator.",
			},
			"value": schema.StringAttribute{
				Optional:    true,
				Description: "The value to compare against.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk ticket trigger.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the trigger.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the trigger is active.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The description of the trigger.",
			},
			"category_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the trigger category.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Position of the trigger, determines the order they fire.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a default system trigger.",
			},
			"condition_all": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions that must ALL be met (AND logic) for the trigger to fire.",
				NestedObject: conditionSchema,
			},
			"condition_any": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions where ANY can be met (OR logic) for the trigger to fire.",
				NestedObject: conditionSchema,
			},
			"actions": schema.ListNestedAttribute{
				Required:    true,
				Description: "Actions to perform when the trigger fires.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"field": schema.StringAttribute{
							Required:    true,
							Description: "The ticket field to modify.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The new value for the field.",
						},
					},
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

func (r *TriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildTriggerAPI(&plan)
	apiReq := triggerWrapper{Trigger: apiObj}

	var result triggerWrapper
	if err := r.client.Post("/api/v2/triggers", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating trigger", err.Error())
		return
	}

	mapTriggerToState(&result.Trigger, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result triggerWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/triggers/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading trigger", err.Error())
		return
	}

	mapTriggerToState(&result.Trigger, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildTriggerAPI(&plan)
	apiReq := triggerWrapper{Trigger: apiObj}

	var result triggerWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/triggers/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating trigger", err.Error())
		return
	}

	mapTriggerToState(&result.Trigger, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/triggers/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting trigger", err.Error())
		return
	}
}

func (r *TriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildTriggerAPI(plan *TriggerResourceModel) triggerAPIObject {
	obj := triggerAPIObject{
		Title:       plan.Title.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	if !plan.CategoryID.IsNull() && !plan.CategoryID.IsUnknown() {
		v := plan.CategoryID.ValueString()
		obj.CategoryID = &v
	}
	if !plan.Position.IsNull() && !plan.Position.IsUnknown() {
		obj.Position = plan.Position.ValueInt64()
	}

	conditions := &triggerConditionsAPI{}
	if plan.CondAll != nil {
		conditions.All = make([]triggerConditionAPI, len(plan.CondAll))
		for i, c := range plan.CondAll {
			conditions.All[i] = triggerConditionAPI{
				Field:    c.Field.ValueString(),
				Operator: c.Operator.ValueString(),
				Value:    stringToValue(c.Value.ValueString()),
			}
		}
	}
	if plan.CondAny != nil {
		conditions.Any = make([]triggerConditionAPI, len(plan.CondAny))
		for i, c := range plan.CondAny {
			conditions.Any[i] = triggerConditionAPI{
				Field:    c.Field.ValueString(),
				Operator: c.Operator.ValueString(),
				Value:    stringToValue(c.Value.ValueString()),
			}
		}
	}
	obj.Conditions = conditions

	if plan.Actions != nil {
		obj.Actions = make([]triggerActionAPI, len(plan.Actions))
		for i, a := range plan.Actions {
			obj.Actions[i] = triggerActionAPI{
				Field: a.Field.ValueString(),
				Value: stringToValue(a.Value.ValueString()),
			}
		}
	}

	return obj
}

func mapTriggerToState(t *triggerAPIObject, m *TriggerResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(t.ID, 10))
	m.Title = types.StringValue(t.Title)
	if t.Active != nil {
		m.Active = types.BoolValue(*t.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	m.Description = types.StringValue(t.Description)
	if t.CategoryID != nil {
		m.CategoryID = types.StringValue(*t.CategoryID)
	} else {
		m.CategoryID = types.StringNull()
	}
	m.Position = types.Int64Value(t.Position)
	m.Default = types.BoolValue(t.Default)
	m.CreatedAt = types.StringValue(t.CreatedAt)
	m.UpdatedAt = types.StringValue(t.UpdatedAt)

	if t.Conditions != nil {
		if len(t.Conditions.All) > 0 {
			m.CondAll = make([]TriggerConditionModel, len(t.Conditions.All))
			for i, c := range t.Conditions.All {
				m.CondAll[i] = TriggerConditionModel{
					Field:    types.StringValue(c.Field),
					Operator: types.StringValue(c.Operator),
					Value:    types.StringValue(valueToString(c.Value)),
				}
			}
		} else {
			m.CondAll = nil
		}
		if len(t.Conditions.Any) > 0 {
			m.CondAny = make([]TriggerConditionModel, len(t.Conditions.Any))
			for i, c := range t.Conditions.Any {
				m.CondAny[i] = TriggerConditionModel{
					Field:    types.StringValue(c.Field),
					Operator: types.StringValue(c.Operator),
					Value:    types.StringValue(valueToString(c.Value)),
				}
			}
		} else {
			m.CondAny = nil
		}
	}

	if t.Actions != nil {
		m.Actions = make([]TriggerActionModel, len(t.Actions))
		for i, a := range t.Actions {
			m.Actions[i] = TriggerActionModel{
				Field: types.StringValue(a.Field),
				Value: types.StringValue(valueToString(a.Value)),
			}
		}
	}
}
