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
	_ resource.Resource                = &AutomationResource{}
	_ resource.ResourceWithImportState = &AutomationResource{}
)

type AutomationResource struct {
	client *ZendeskClient
}

type AutomationResourceModel struct {
	ID        types.String            `tfsdk:"id"`
	Title     types.String            `tfsdk:"title"`
	Active    types.Bool              `tfsdk:"active"`
	Position  types.Int64             `tfsdk:"position"`
	Default   types.Bool              `tfsdk:"default"`
	CondAll   []TriggerConditionModel `tfsdk:"condition_all"`
	CondAny   []TriggerConditionModel `tfsdk:"condition_any"`
	Actions   []TriggerActionModel    `tfsdk:"actions"`
	CreatedAt types.String            `tfsdk:"created_at"`
	UpdatedAt types.String            `tfsdk:"updated_at"`
}

type automationAPIObject struct {
	ID         int64                 `json:"id,omitempty"`
	Title      string                `json:"title"`
	Active     *bool                 `json:"active,omitempty"`
	Position   int64                 `json:"position,omitempty"`
	Default    bool                  `json:"default,omitempty"`
	Conditions *triggerConditionsAPI `json:"conditions,omitempty"`
	Actions    []triggerActionAPI    `json:"actions"`
	CreatedAt  string                `json:"created_at,omitempty"`
	UpdatedAt  string                `json:"updated_at,omitempty"`
}

type automationWrapper struct {
	Automation automationAPIObject `json:"automation"`
}

func NewAutomationResource() resource.Resource {
	return &AutomationResource{}
}

func (r *AutomationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation"
}

func (r *AutomationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		Description: "Manages a Zendesk automation (time-based business rule).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the automation.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the automation is active.",
			},
			"position": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The execution position among other automations.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a default system automation.",
			},
			"condition_all": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions that must ALL be met (AND). At least one time-based condition is typically required.",
				NestedObject: conditionSchema,
			},
			"condition_any": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions where ANY can be met (OR).",
				NestedObject: conditionSchema,
			},
			"actions": schema.ListNestedAttribute{
				Required:    true,
				Description: "Actions to perform when the automation fires.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"field": schema.StringAttribute{Required: true, Description: "The ticket field to modify."},
						"value": schema.StringAttribute{Required: true, Description: "The new value for the field."},
					},
				},
			},
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

func (r *AutomationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AutomationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AutomationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := automationWrapper{Automation: buildAutomationAPI(&plan)}
	var result automationWrapper
	if err := r.client.Post("/api/v2/automations", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating automation", err.Error())
		return
	}

	mapAutomationToState(&result.Automation, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutomationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AutomationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result automationWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/automations/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading automation", err.Error())
		return
	}

	mapAutomationToState(&result.Automation, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AutomationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AutomationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AutomationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := automationWrapper{Automation: buildAutomationAPI(&plan)}
	var result automationWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/automations/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating automation", err.Error())
		return
	}

	mapAutomationToState(&result.Automation, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AutomationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AutomationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/automations/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting automation", err.Error())
		return
	}
}

func (r *AutomationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildAutomationAPI(plan *AutomationResourceModel) automationAPIObject {
	obj := automationAPIObject{
		Title: plan.Title.ValueString(),
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
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

func mapAutomationToState(a *automationAPIObject, m *AutomationResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(a.ID, 10))
	m.Title = types.StringValue(a.Title)
	if a.Active != nil {
		m.Active = types.BoolValue(*a.Active)
	} else {
		m.Active = types.BoolValue(true)
	}
	m.Position = types.Int64Value(a.Position)
	m.Default = types.BoolValue(a.Default)
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)

	if a.Conditions != nil {
		if len(a.Conditions.All) > 0 {
			m.CondAll = make([]TriggerConditionModel, len(a.Conditions.All))
			for i, c := range a.Conditions.All {
				m.CondAll[i] = TriggerConditionModel{
					Field:    types.StringValue(c.Field),
					Operator: types.StringValue(c.Operator),
					Value:    types.StringValue(valueToString(c.Value)),
				}
			}
		} else {
			m.CondAll = nil
		}
		if len(a.Conditions.Any) > 0 {
			m.CondAny = make([]TriggerConditionModel, len(a.Conditions.Any))
			for i, c := range a.Conditions.Any {
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

	if a.Actions != nil {
		m.Actions = make([]TriggerActionModel, len(a.Actions))
		for i, ac := range a.Actions {
			m.Actions[i] = TriggerActionModel{
				Field: types.StringValue(ac.Field),
				Value: types.StringValue(valueToString(ac.Value)),
			}
		}
	}
}
