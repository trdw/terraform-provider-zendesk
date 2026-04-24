package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TicketFieldDataSource{}

type TicketFieldDataSource struct {
	client *ZendeskClient
}

type TicketFieldDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Active      types.Bool   `tfsdk:"active"`
	Position    types.Int64  `tfsdk:"position"`
	Tag         types.String `tfsdk:"tag"`
	System      types.Bool   `tfsdk:"system"`
	Removable   types.Bool   `tfsdk:"removable"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type ticketFieldsListResponse struct {
	TicketFields []ticketFieldAPIObject `json:"ticket_fields"`
	NextPage     *string                `json:"next_page"`
}

func NewTicketFieldDataSource() datasource.DataSource {
	return &TicketFieldDataSource{}
}

func (d *TicketFieldDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ticket_field"
}

func (d *TicketFieldDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk ticket field by ID or title.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the ticket field. Provide either id or title.",
			},
			"title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The title of the ticket field. Provide either id or title.",
			},
			"type":        schema.StringAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"active":      schema.BoolAttribute{Computed: true},
			"position":    schema.Int64Attribute{Computed: true},
			"tag":         schema.StringAttribute{Computed: true},
			"system":      schema.BoolAttribute{Computed: true},
			"removable":   schema.BoolAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TicketFieldDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TicketFieldDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TicketFieldDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasTitle := !config.Title.IsNull() && !config.Title.IsUnknown() && config.Title.ValueString() != ""

	if !hasID && !hasTitle {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id' or 'title' must be provided.")
		return
	}

	var found *ticketFieldAPIObject
	if hasID {
		var result ticketFieldWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/ticket_fields/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading ticket field", err.Error())
			return
		}
		found = &result.TicketField
	} else {
		targetTitle := config.Title.ValueString()
		page := "/api/v2/ticket_fields.json?page[size]=100"
		for page != "" {
			var result ticketFieldsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing ticket fields", err.Error())
				return
			}
			for i := range result.TicketFields {
				if result.TicketFields[i].Title == targetTitle {
					found = &result.TicketFields[i]
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
			resp.Diagnostics.AddError("Ticket field not found", fmt.Sprintf("No ticket field found with title %q", targetTitle))
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.Title = types.StringValue(found.Title)
	config.Type = types.StringValue(found.Type)
	config.Description = types.StringValue(found.Description)
	if found.Active != nil {
		config.Active = types.BoolValue(*found.Active)
	} else {
		config.Active = types.BoolValue(true)
	}
	config.Position = types.Int64Value(found.Position)
	if found.Tag != nil {
		config.Tag = types.StringValue(*found.Tag)
	} else {
		config.Tag = types.StringNull()
	}
	config.System = types.BoolValue(found.System)
	config.Removable = types.BoolValue(found.Removable)
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
