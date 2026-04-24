package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TicketFormDataSource{}

type TicketFormDataSource struct {
	client *ZendeskClient
}

type TicketFormDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Active         types.Bool   `tfsdk:"active"`
	Default        types.Bool   `tfsdk:"default"`
	EndUserVisible types.Bool   `tfsdk:"end_user_visible"`
	Position       types.Int64  `tfsdk:"position"`
	InAllBrands    types.Bool   `tfsdk:"in_all_brands"`
	TicketFieldIDs types.List   `tfsdk:"ticket_field_ids"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

type ticketFormsListResponse struct {
	TicketForms []ticketFormAPIObject `json:"ticket_forms"`
	NextPage    *string               `json:"next_page"`
}

func NewTicketFormDataSource() datasource.DataSource {
	return &TicketFormDataSource{}
}

func (d *TicketFormDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ticket_form"
}

func (d *TicketFormDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk ticket form by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the ticket form. Provide either id or name.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the ticket form. Provide either id or name.",
			},
			"display_name":     schema.StringAttribute{Computed: true},
			"active":           schema.BoolAttribute{Computed: true},
			"default":          schema.BoolAttribute{Computed: true},
			"end_user_visible": schema.BoolAttribute{Computed: true},
			"position":         schema.Int64Attribute{Computed: true},
			"in_all_brands":    schema.BoolAttribute{Computed: true},
			"ticket_field_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TicketFormDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TicketFormDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TicketFormDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id' or 'name' must be provided.")
		return
	}

	var found *ticketFormAPIObject
	if hasID {
		var result ticketFormWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/ticket_forms/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading ticket form", err.Error())
			return
		}
		found = &result.TicketForm
	} else {
		targetName := config.Name.ValueString()
		page := "/api/v2/ticket_forms.json?page[size]=100"
		for page != "" {
			var result ticketFormsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing ticket forms", err.Error())
				return
			}
			for i := range result.TicketForms {
				if result.TicketForms[i].Name == targetName {
					found = &result.TicketForms[i]
					break
				}
			}
			if found != nil {
				break
			}
			if result.NextPage != nil && *result.NextPage != "" {
				page = *result.NextPage
			} else {
				page = ""
			}
		}
		if found == nil {
			resp.Diagnostics.AddError("Ticket form not found", fmt.Sprintf("No ticket form found with name %q", targetName))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Name = types.StringValue(found.Name)
	config.DisplayName = types.StringValue(found.DisplayName)
	if found.Active != nil {
		config.Active = types.BoolValue(*found.Active)
	} else {
		config.Active = types.BoolValue(true)
	}
	if found.Default != nil {
		config.Default = types.BoolValue(*found.Default)
	} else {
		config.Default = types.BoolValue(false)
	}
	if found.EndUserVisible != nil {
		config.EndUserVisible = types.BoolValue(*found.EndUserVisible)
	} else {
		config.EndUserVisible = types.BoolValue(true)
	}
	config.Position = types.Int64Value(found.Position)
	if found.InAllBrands != nil {
		config.InAllBrands = types.BoolValue(*found.InAllBrands)
	} else {
		config.InAllBrands = types.BoolValue(false)
	}
	config.TicketFieldIDs = sliceToInt64List(found.TicketFieldIDs)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
