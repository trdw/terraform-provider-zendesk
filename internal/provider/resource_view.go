package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ViewResource{}
	_ resource.ResourceWithImportState = &ViewResource{}
)

type ViewResource struct {
	client *ZendeskClient
}

type ViewConditionModel struct {
	Field    types.String `tfsdk:"field"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

type ViewRestrictionModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.Int64  `tfsdk:"id"`
}

type ViewResourceModel struct {
	ID          types.String         `tfsdk:"id"`
	Title       types.String         `tfsdk:"title"`
	Active      types.Bool           `tfsdk:"active"`
	Description types.String         `tfsdk:"description"`
	Position    types.Int64          `tfsdk:"position"`
	Default     types.Bool           `tfsdk:"default"`
	All         []ViewConditionModel `tfsdk:"all"`
	Any         []ViewConditionModel `tfsdk:"any"`
	Columns     types.List           `tfsdk:"columns"`
	GroupBy     types.String         `tfsdk:"group_by"`
	GroupOrder  types.String         `tfsdk:"group_order"`
	SortBy      types.String         `tfsdk:"sort_by"`
	SortOrder   types.String         `tfsdk:"sort_order"`
	Restriction *ViewRestrictionModel `tfsdk:"restriction"`
	CreatedAt   types.String         `tfsdk:"created_at"`
	UpdatedAt   types.String         `tfsdk:"updated_at"`
}

type viewConditionAPI struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type viewOutputAPI struct {
	Columns    []string `json:"columns,omitempty"`
	GroupBy    string   `json:"group_by,omitempty"`
	GroupOrder string   `json:"group_order,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`
	SortOrder  string   `json:"sort_order,omitempty"`
}

type viewRestrictionAPI struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type viewCreateUpdateAPI struct {
	Title       string              `json:"title"`
	All         []viewConditionAPI  `json:"all,omitempty"`
	Any         []viewConditionAPI  `json:"any,omitempty"`
	Description string              `json:"description,omitempty"`
	Active      *bool               `json:"active,omitempty"`
	Output      *viewOutputAPI      `json:"output,omitempty"`
	Restriction *viewRestrictionAPI `json:"restriction,omitempty"`
}

type viewReadAPI struct {
	ID          int64                  `json:"id"`
	Title       string                 `json:"title"`
	Active      bool                   `json:"active"`
	Description string                 `json:"description"`
	Position    int64                  `json:"position"`
	Default     bool                   `json:"default"`
	Conditions  map[string]interface{} `json:"conditions"`
	Execution   map[string]interface{} `json:"execution"`
	Restriction map[string]interface{} `json:"restriction"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

type viewCreateWrapper struct {
	View viewCreateUpdateAPI `json:"view"`
}

type viewReadWrapper struct {
	View viewReadAPI `json:"view"`
}

func NewViewResource() resource.Resource {
	return &ViewResource{}
}

func (r *ViewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (r *ViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	conditionSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"field": schema.StringAttribute{
				Required:    true,
				Description: "The name of the ticket field to filter on.",
			},
			"operator": schema.StringAttribute{
				Required:    true,
				Description: "The comparison operator (e.g. is, is_not, less_than, greater_than).",
			},
			"value": schema.StringAttribute{
				Optional:    true,
				Description: "The value to compare against.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk view.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the view.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the view is active.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The description of the view.",
			},
			"position": schema.Int64Attribute{
				Computed:    true,
				Description: "The position of the view in the list.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a default system view.",
			},
			"all": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions that must ALL be met (AND logic).",
				NestedObject: conditionSchema,
			},
			"any": schema.ListNestedAttribute{
				Optional:     true,
				Description:  "Conditions where ANY can be met (OR logic).",
				NestedObject: conditionSchema,
			},
			"columns": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Column IDs to display in the view output.",
			},
			"group_by": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The field to group results by.",
			},
			"group_order": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The order of grouped results (asc or desc).",
			},
			"sort_by": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The field to sort results by.",
			},
			"sort_order": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The sort order (asc or desc).",
			},
			"restriction": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Who may access this view.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "The type of restriction (Group or User).",
					},
					"id": schema.Int64Attribute{
						Required:    true,
						Description: "The numeric ID of the group or user.",
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

func (r *ViewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiView := viewCreateUpdateAPI{
		Title:       plan.Title.ValueString(),
		Description: plan.Description.ValueString(),
		All:         buildConditionsAPI(plan.All),
		Any:         buildConditionsAPI(plan.Any),
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		apiView.Active = &v
	}
	apiView.Output = buildViewOutputAPI(ctx, &plan)
	if plan.Restriction != nil {
		apiView.Restriction = &viewRestrictionAPI{
			Type: plan.Restriction.Type.ValueString(),
			ID:   plan.Restriction.ID.ValueInt64(),
		}
	}

	apiReq := viewCreateWrapper{View: apiView}
	var result viewReadWrapper
	if err := r.client.Post("/api/v2/views", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating view", err.Error())
		return
	}

	mapViewToState(ctx, &result.View, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result viewReadWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/views/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading view", err.Error())
		return
	}

	mapViewToState(ctx, &result.View, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ViewResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiView := viewCreateUpdateAPI{
		Title:       plan.Title.ValueString(),
		Description: plan.Description.ValueString(),
		All:         buildConditionsAPI(plan.All),
		Any:         buildConditionsAPI(plan.Any),
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		apiView.Active = &v
	}
	apiView.Output = buildViewOutputAPI(ctx, &plan)
	if plan.Restriction != nil {
		apiView.Restriction = &viewRestrictionAPI{
			Type: plan.Restriction.Type.ValueString(),
			ID:   plan.Restriction.ID.ValueInt64(),
		}
	}

	apiReq := viewCreateWrapper{View: apiView}
	var result viewReadWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/views/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating view", err.Error())
		return
	}

	mapViewToState(ctx, &result.View, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ViewResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/views/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting view", err.Error())
		return
	}
}

func (r *ViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildConditionsAPI(conditions []ViewConditionModel) []viewConditionAPI {
	if conditions == nil {
		return nil
	}
	result := make([]viewConditionAPI, len(conditions))
	for i, c := range conditions {
		result[i] = viewConditionAPI{
			Field:    c.Field.ValueString(),
			Operator: c.Operator.ValueString(),
			Value:    c.Value.ValueString(),
		}
	}
	return result
}

func buildViewOutputAPI(ctx context.Context, plan *ViewResourceModel) *viewOutputAPI {
	output := &viewOutputAPI{}
	hasOutput := false

	if !plan.Columns.IsNull() && !plan.Columns.IsUnknown() {
		var cols []string
		plan.Columns.ElementsAs(ctx, &cols, false)
		output.Columns = cols
		hasOutput = true
	}
	if !plan.GroupBy.IsNull() && !plan.GroupBy.IsUnknown() {
		output.GroupBy = plan.GroupBy.ValueString()
		hasOutput = true
	}
	if !plan.GroupOrder.IsNull() && !plan.GroupOrder.IsUnknown() {
		output.GroupOrder = plan.GroupOrder.ValueString()
		hasOutput = true
	}
	if !plan.SortBy.IsNull() && !plan.SortBy.IsUnknown() {
		output.SortBy = plan.SortBy.ValueString()
		hasOutput = true
	}
	if !plan.SortOrder.IsNull() && !plan.SortOrder.IsUnknown() {
		output.SortOrder = plan.SortOrder.ValueString()
		hasOutput = true
	}

	if hasOutput {
		return output
	}
	return nil
}

func parseConditionsFromAPI(raw interface{}) []ViewConditionModel {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		return nil
	}
	result := make([]ViewConditionModel, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		c := ViewConditionModel{
			Field:    types.StringValue(valueToString(m["field"])),
			Operator: types.StringValue(valueToString(m["operator"])),
			Value:    types.StringValue(valueToString(m["value"])),
		}
		result = append(result, c)
	}
	return result
}

func mapViewToState(ctx context.Context, v *viewReadAPI, m *ViewResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(v.ID, 10))
	m.Title = types.StringValue(v.Title)
	m.Active = types.BoolValue(v.Active)
	m.Description = types.StringValue(v.Description)
	m.Position = types.Int64Value(v.Position)
	m.Default = types.BoolValue(v.Default)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)

	if v.Conditions != nil {
		m.All = parseConditionsFromAPI(v.Conditions["all"])
		m.Any = parseConditionsFromAPI(v.Conditions["any"])
	}

	if v.Execution != nil {
		if cols, ok := v.Execution["columns"].([]interface{}); ok {
			colStrs := make([]attr.Value, 0, len(cols))
			for _, c := range cols {
				if colMap, ok := c.(map[string]interface{}); ok {
					if id, ok := colMap["id"]; ok {
						colStrs = append(colStrs, types.StringValue(valueToString(id)))
					}
				} else {
					colStrs = append(colStrs, types.StringValue(valueToString(c)))
				}
			}
			m.Columns, _ = types.ListValue(types.StringType, colStrs)
		}
		if gb, ok := v.Execution["group_by"].(string); ok {
			m.GroupBy = types.StringValue(gb)
		} else {
			m.GroupBy = types.StringNull()
		}
		if go_, ok := v.Execution["group_order"].(string); ok {
			m.GroupOrder = types.StringValue(go_)
		} else {
			m.GroupOrder = types.StringNull()
		}
		if sb, ok := v.Execution["sort_by"].(string); ok {
			m.SortBy = types.StringValue(sb)
		} else {
			m.SortBy = types.StringNull()
		}
		if so, ok := v.Execution["sort_order"].(string); ok {
			m.SortOrder = types.StringValue(so)
		} else {
			m.SortOrder = types.StringNull()
		}
	}

	if v.Restriction != nil && len(v.Restriction) > 0 {
		rType := valueToString(v.Restriction["type"])
		rID := int64(0)
		if idVal, ok := v.Restriction["id"].(float64); ok {
			rID = int64(idVal)
		}
		m.Restriction = &ViewRestrictionModel{
			Type: types.StringValue(rType),
			ID:   types.Int64Value(rID),
		}
	} else {
		m.Restriction = nil
	}
}
