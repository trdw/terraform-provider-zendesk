package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GroupMembershipDataSource{}

type GroupMembershipDataSource struct {
	client *ZendeskClient
}

type GroupMembershipDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	GroupID   types.Int64  `tfsdk:"group_id"`
	Default   types.Bool   `tfsdk:"default"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type groupMembershipsListResponse struct {
	GroupMemberships []groupMembershipAPIObject `json:"group_memberships"`
	NextPage         *string                    `json:"next_page"`
}

func NewGroupMembershipDataSource() datasource.DataSource {
	return &GroupMembershipDataSource{}
}

func (d *GroupMembershipDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (d *GroupMembershipDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Zendesk group membership by ID, or by the (user_id, group_id) pair.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the group membership. Provide either id, or the (user_id, group_id) pair.",
			},
			"user_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the user.",
			},
			"group_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the group.",
			},
			"default":    schema.BoolAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *GroupMembershipDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupMembershipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupMembershipDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasUserID := !config.UserID.IsNull() && !config.UserID.IsUnknown()
	hasGroupID := !config.GroupID.IsNull() && !config.GroupID.IsUnknown()

	if !hasID && !(hasUserID && hasGroupID) {
		resp.Diagnostics.AddError("Missing attribute", "Either 'id', or both 'user_id' and 'group_id' must be provided.")
		return
	}

	var found *groupMembershipAPIObject
	if hasID {
		var result groupMembershipWrapper
		if err := d.client.Get(fmt.Sprintf("/api/v2/group_memberships/%s", config.ID.ValueString()), &result); err != nil {
			resp.Diagnostics.AddError("Error reading group membership", err.Error())
			return
		}
		found = &result.GroupMembership
	} else {
		userID := config.UserID.ValueInt64()
		groupID := config.GroupID.ValueInt64()
		page := fmt.Sprintf("/api/v2/users/%d/group_memberships.json?page[size]=100", userID)
		for page != "" {
			var result groupMembershipsListResponse
			if err := d.client.Get(page, &result); err != nil {
				resp.Diagnostics.AddError("Error listing group memberships", err.Error())
				return
			}
			for i := range result.GroupMemberships {
				if result.GroupMemberships[i].GroupID == groupID {
					found = &result.GroupMemberships[i]
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
			resp.Diagnostics.AddError(
				"Group membership not found",
				fmt.Sprintf("No membership found for user_id=%d, group_id=%d", userID, groupID),
			)
			return
		}
	}

	config.ID = types.StringValue(strconv.FormatInt(found.ID, 10))
	config.UserID = types.Int64Value(found.UserID)
	config.GroupID = types.Int64Value(found.GroupID)
	if found.Default != nil {
		config.Default = types.BoolValue(*found.Default)
	} else {
		config.Default = types.BoolValue(false)
	}
	config.CreatedAt = types.StringValue(found.CreatedAt)
	config.UpdatedAt = types.StringValue(found.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
