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
	_ resource.Resource                = &MacroResource{}
	_ resource.ResourceWithImportState = &MacroResource{}
)

type MacroResource struct {
	client *ZendeskClient
}

type MacroActionModel struct {
	Field types.String `tfsdk:"field"`
	Value types.String `tfsdk:"value"`
}

type MacroRestrictionModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.Int64  `tfsdk:"id"`
}

type MacroResourceModel struct {
	ID          types.String           `tfsdk:"id"`
	Title       types.String           `tfsdk:"title"`
	Active      types.Bool             `tfsdk:"active"`
	Description types.String           `tfsdk:"description"`
	Position    types.Int64            `tfsdk:"position"`
	Default     types.Bool             `tfsdk:"default"`
	Actions     []MacroActionModel     `tfsdk:"actions"`
	Restriction *MacroRestrictionModel `tfsdk:"restriction"`
	CreatedAt   types.String           `tfsdk:"created_at"`
	UpdatedAt   types.String           `tfsdk:"updated_at"`
}

type macroActionAPI struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type macroRestrictionAPI struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type macroCreateUpdateAPI struct {
	Title       string               `json:"title"`
	Active      *bool                `json:"active,omitempty"`
	Description *string              `json:"description,omitempty"`
	Actions     []macroActionAPI     `json:"actions"`
	Restriction *macroRestrictionAPI `json:"restriction,omitempty"`
}

type macroReadAPI struct {
	ID          int64                  `json:"id"`
	Title       string                 `json:"title"`
	Active      bool                   `json:"active"`
	Description *string                `json:"description"`
	Position    int64                  `json:"position"`
	Default     bool                   `json:"default"`
	Actions     []macroActionAPI       `json:"actions"`
	Restriction map[string]interface{} `json:"restriction"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

type macroCreateWrapper struct {
	Macro macroCreateUpdateAPI `json:"macro"`
}

type macroReadWrapper struct {
	Macro macroReadAPI `json:"macro"`
}

func NewMacroResource() resource.Resource {
	return &MacroResource{}
}

func (r *MacroResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_macro"
}

func (r *MacroResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Zendesk macro.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the macro.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the macro is active.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The description of the macro.",
			},
			"position": schema.Int64Attribute{
				Computed:    true,
				Description: "The position of the macro.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a default system macro.",
			},
			"actions": schema.ListNestedAttribute{
				Required:    true,
				Description: "Actions the macro performs on tickets.",
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
			"restriction": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Who may access this macro.",
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

func (r *MacroResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MacroResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MacroResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildMacroAPI(&plan)
	apiReq := macroCreateWrapper{Macro: apiObj}

	var result macroReadWrapper
	if err := r.client.Post("/api/v2/macros", apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error creating macro", err.Error())
		return
	}

	mapMacroToState(&result.Macro, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MacroResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MacroResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result macroReadWrapper
	err := r.client.Get(fmt.Sprintf("/api/v2/macros/%s", state.ID.ValueString()), &result)
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading macro", err.Error())
		return
	}

	mapMacroToState(&result.Macro, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MacroResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MacroResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MacroResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj := buildMacroAPI(&plan)
	apiReq := macroCreateWrapper{Macro: apiObj}

	var result macroReadWrapper
	if err := r.client.Put(fmt.Sprintf("/api/v2/macros/%s", state.ID.ValueString()), apiReq, &result); err != nil {
		resp.Diagnostics.AddError("Error updating macro", err.Error())
		return
	}

	mapMacroToState(&result.Macro, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MacroResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MacroResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(fmt.Sprintf("/api/v2/macros/%s", state.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting macro", err.Error())
		return
	}
}

func (r *MacroResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildMacroAPI(plan *MacroResourceModel) macroCreateUpdateAPI {
	obj := macroCreateUpdateAPI{
		Title: plan.Title.ValueString(),
	}

	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		v := plan.Active.ValueBool()
		obj.Active = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		obj.Description = &v
	}

	if plan.Actions != nil {
		obj.Actions = make([]macroActionAPI, len(plan.Actions))
		for i, a := range plan.Actions {
			obj.Actions[i] = macroActionAPI{
				Field: a.Field.ValueString(),
				Value: stringToValue(a.Value.ValueString()),
			}
		}
	}

	if plan.Restriction != nil {
		obj.Restriction = &macroRestrictionAPI{
			Type: plan.Restriction.Type.ValueString(),
			ID:   plan.Restriction.ID.ValueInt64(),
		}
	}

	return obj
}

func mapMacroToState(m *macroReadAPI, model *MacroResourceModel) {
	model.ID = types.StringValue(strconv.FormatInt(m.ID, 10))
	model.Title = types.StringValue(m.Title)
	model.Active = types.BoolValue(m.Active)
	if m.Description != nil {
		model.Description = types.StringValue(*m.Description)
	} else {
		model.Description = types.StringNull()
	}
	model.Position = types.Int64Value(m.Position)
	model.Default = types.BoolValue(m.Default)
	model.CreatedAt = types.StringValue(m.CreatedAt)
	model.UpdatedAt = types.StringValue(m.UpdatedAt)

	if m.Actions != nil {
		model.Actions = make([]MacroActionModel, len(m.Actions))
		for i, a := range m.Actions {
			model.Actions[i] = MacroActionModel{
				Field: types.StringValue(a.Field),
				Value: types.StringValue(valueToString(a.Value)),
			}
		}
	}

	if m.Restriction != nil && len(m.Restriction) > 0 {
		rType := valueToString(m.Restriction["type"])
		rID := int64(0)
		if idVal, ok := m.Restriction["id"].(float64); ok {
			rID = int64(idVal)
		}
		model.Restriction = &MacroRestrictionModel{
			Type: types.StringValue(rType),
			ID:   types.Int64Value(rID),
		}
	} else {
		model.Restriction = nil
	}
}
