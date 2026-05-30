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
	IDs  types.Set    `tfsdk:"ids"`
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
	// Zendesk macro restrictions distinguish User (single id) from Group (one or
	// more ids). A singular `id` is silently ignored when updating a Group
	// restriction, so Group restrictions must always be sent via `ids`.
	ID  *int64  `json:"id,omitempty"`
	IDs []int64 `json:"ids,omitempty"`
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
						Optional:    true,
						Description: "The numeric ID of the user (for a User restriction). For Group restrictions use `ids`.",
					},
					"ids": schema.SetAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
						Description: "The numeric IDs of the groups (for a Group restriction). Zendesk allows a macro to be restricted to multiple groups.",
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

	// The create response omits restriction.ids for multi-group Group
	// restrictions (it returns only the singular, unstable `id`), so re-read the
	// macro via GET to get the authoritative restriction before saving state.
	if err := r.client.Get(fmt.Sprintf("/api/v2/macros/%d", result.Macro.ID), &result); err != nil {
		resp.Diagnostics.AddError("Error reading macro after create", err.Error())
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

	// The update response omits restriction.ids for multi-group Group
	// restrictions (it returns only the singular, unstable `id`), so re-read the
	// macro via GET to get the authoritative restriction before saving state.
	if err := r.client.Get(fmt.Sprintf("/api/v2/macros/%s", state.ID.ValueString()), &result); err != nil {
		resp.Diagnostics.AddError("Error reading macro after update", err.Error())
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
		r := &macroRestrictionAPI{Type: plan.Restriction.Type.ValueString()}

		// Collect any group ids supplied via the set.
		var ids []int64
		if !plan.Restriction.IDs.IsNull() && !plan.Restriction.IDs.IsUnknown() {
			for _, e := range plan.Restriction.IDs.Elements() {
				if v, ok := e.(types.Int64); ok {
					ids = append(ids, v.ValueInt64())
				}
			}
		}
		hasID := !plan.Restriction.ID.IsNull() && !plan.Restriction.ID.IsUnknown()

		if r.Type == "Group" {
			// Group restrictions must be sent as `ids`; Zendesk ignores a lone
			// `id` on update, so wrap a single configured id into the list.
			if len(ids) > 0 {
				r.IDs = ids
			} else if hasID {
				r.IDs = []int64{plan.Restriction.ID.ValueInt64()}
			}
		} else {
			// User restrictions take a single id.
			if hasID {
				v := plan.Restriction.ID.ValueInt64()
				r.ID = &v
			} else if len(ids) == 1 {
				v := ids[0]
				r.ID = &v
			}
		}

		obj.Restriction = r
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
		restr := &MacroRestrictionModel{
			Type: types.StringValue(rType),
			ID:   types.Int64Null(),
			IDs:  types.SetNull(types.Int64Type),
		}

		// Populate the same representation the configuration used so the
		// post-apply state matches the plan. Zendesk echoes both `id` and `ids`
		// for Group restrictions, and which singular `id` it returns is not
		// stable, so a config using `ids` must read back from `ids`. When the
		// prior model is empty (e.g. after import) fall back to type: Group uses
		// `ids`, User uses `id`.
		usesIDs := rType == "Group"
		if model.Restriction != nil {
			hasIDs := !model.Restriction.IDs.IsNull() && !model.Restriction.IDs.IsUnknown() &&
				len(model.Restriction.IDs.Elements()) > 0
			hasID := !model.Restriction.ID.IsNull() && !model.Restriction.ID.IsUnknown()
			if hasIDs {
				usesIDs = true
			} else if hasID {
				usesIDs = false
			}
		}

		if usesIDs {
			var elems []attr.Value
			if raw, ok := m.Restriction["ids"].([]interface{}); ok {
				for _, it := range raw {
					if f, ok := it.(float64); ok {
						elems = append(elems, types.Int64Value(int64(f)))
					}
				}
			}
			// Older payloads (or single-group restrictions) may only carry `id`.
			if len(elems) == 0 {
				if idVal, ok := m.Restriction["id"].(float64); ok {
					elems = append(elems, types.Int64Value(int64(idVal)))
				}
			}
			restr.IDs = types.SetValueMust(types.Int64Type, elems)
		} else if idVal, ok := m.Restriction["id"].(float64); ok {
			restr.ID = types.Int64Value(int64(idVal))
		}

		model.Restriction = restr
	} else {
		model.Restriction = nil
	}
}
