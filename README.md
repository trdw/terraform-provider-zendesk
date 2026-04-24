# terraform-provider-zendesk

Manage Zendesk Support resources (groups, views, agents, triggers, trigger categories, automations, macros, custom roles, brands, ticket forms, custom ticket fields, custom user fields) via a custom Terraform provider.

## Project Structure

```
├── main.go                          # Provider entry point
├── go.mod / go.sum                  # Go module
├── generate.go                      # //go:generate directive for tfplugindocs
├── internal/provider/               # Resource & data source implementations
├── templates/                       # tfplugindocs source templates (e.g. index.md.tmpl)
├── docs/                            # Registry documentation (generated; index.md is rendered from templates/)
├── scripts/
│   ├── assimilate.sh                # Import existing Zendesk resources into Terraform
│   └── generators/                  # Bash generators that convert API JSON to .tf files
│       ├── automation.sh            # zendesk_automation
│       ├── brand.sh                 # zendesk_brand
│       ├── custom_role.sh           # zendesk_custom_role
│       ├── group.sh                 # zendesk_group + membership + user data sources
│       ├── macro.sh                 # zendesk_macro
│       ├── ticket_field.sh          # zendesk_ticket_field (custom ticket fields)
│       ├── ticket_form.sh           # zendesk_ticket_form
│       ├── trigger.sh               # zendesk_trigger
│       ├── trigger_category.sh      # zendesk_trigger_category
│       ├── user.sh                  # zendesk_user
│       ├── user_field.sh            # zendesk_user_field (custom user fields)
│       └── view.sh                  # zendesk_view
├── zendesk-api-spec.yaml            # Zendesk API specification (OpenAPI)
├── goreleaser.yml                   # GoReleaser configuration for registry releases
└── .github/workflows/
    └── release.yml                  # Tag-triggered release workflow (GoReleaser)
```

`infrastructure/` is not included in this repository — you create it locally as a sibling of `scripts/`. Each subdirectory under `infrastructure/` is a self-contained Terraform root module with its own `provider.tf`, state backend, and resource files. A common convention is `<resource_type>_<name>.tf` (e.g. `group_support.tf`, `trigger_autodispatch.tf`).

Example layout:

```
infrastructure/
├── prod/
├── sandbox/
└── staging/
```

## Prerequisites

- [Go](https://go.dev/dl/) 1.22+
- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.5+
- A Zendesk API token (Admin > Channels > API)
- `jq` and `curl` (for the assimilate script)

## Building the Provider

```bash
go build -o terraform-provider-zendesk .
```

## Local Development Setup

1. **Build the provider** (see above).

2. **Create or update `~/.terraformrc`** to point Terraform at the local binary:

   ```hcl
   provider_installation {
     dev_overrides {
       "yourorg/zendesk" = "/path/to/terraform-provider-zendesk"
     }
     direct {}
   }
   ```

   The address must match the `Address` field in `main.go`.

3. **Export the Zendesk API token** (subdomain and email are read from each environment's `provider.tf`):

   ```bash
   export TF_VAR_zendesk_api_token="your-api-token"
   ```

4. **Run Terraform** against an environment (no `terraform init` needed with dev overrides):

   ```bash
   cd infrastructure/sandbox
   terraform plan
   terraform apply
   ```

## Assimilating Existing Resources

The `scripts/assimilate.sh` script automates importing a Zendesk resource into Terraform. It:

1. Reads the Zendesk subdomain and email from the target environment's `provider.tf`
2. Parses a Zendesk admin URL to determine the resource type and ID
3. Fetches the resource JSON from the Zendesk API
4. Runs the appropriate generator to produce a `.tf` file
5. Writes the file to the target infrastructure directory
6. Runs `terraform import` to bring the resource into state
7. Runs `terraform plan` to verify a clean import

The only required environment variable is `TF_VAR_zendesk_api_token` — the same variable used by Terraform itself. The subdomain and email are extracted automatically from the target environment's `provider.tf`.

### Usage

```bash
./scripts/assimilate.sh <infra-dir> <zendesk-admin-url>
```

The first argument is the path to the target Terraform root module (its `provider.tf` must exist). The second argument is the full Zendesk admin URL for the resource.

### Examples

```bash
# Import a trigger into the sandbox environment
./scripts/assimilate.sh ../infrastructure/sandbox https://my-organization.zendesk.com/admin/objects-rules/rules/triggers/360226671071

# Import a group into production
./scripts/assimilate.sh /path/to/infrastructure/prod https://my-organization.zendesk.com/admin/people/team/groups/12345678

# Import a view
./scripts/assimilate.sh ../infrastructure/staging https://my-organization.zendesk.com/admin/workspaces/agent-workspace/views/98765432
```

### Supported URL Patterns

Hyphen or underscore variants in the path (e.g. `ticket-forms` vs `ticket_forms`) are both accepted.

| URL Path Pattern                                            | Resource Type     |
|-------------------------------------------------------------|-------------------|
| `/admin/objects-rules/rules/triggers/<id>`                  | trigger           |
| `/admin/objects-rules/rules/trigger-categories/<id>`        | trigger_category  |
| `/admin/objects-rules/rules/automations/<id>`               | automation        |
| `/admin/workspaces/agent-workspace/views/<id>`              | view              |
| `/admin/objects-rules/rules/macros/<id>`                    | macro             |
| `/admin/objects-rules/tickets/ticket-forms/edit/<id>`       | ticket_form       |
| `/admin/objects-rules/tickets/ticket-fields/<id>`           | ticket_field      |
| `/admin/people/configuration/user-fields/<id>`              | user_field        |
| `/admin/people/team/groups/<id>`                            | group             |
| `/admin/people/team/roles/<id>`                             | custom_role       |
| `/admin/people/team/members/<id>`                           | user              |
| `/admin/account/brand_management/brands/<id>`               | brand             |

When importing a **group**, the script will prompt to also import group members and their memberships.

## Generators

The `scripts/generators/` directory contains bash scripts that convert Zendesk API JSON responses into Terraform resource blocks. Each generator is sourced by `assimilate.sh` and exposes a `generate_<type>()` function.

| Generator               | Function(s)                                                                 |
|-------------------------|-----------------------------------------------------------------------------|
| `automation.sh`         | `generate_automation`                                                        |
| `brand.sh`              | `generate_brand`                                                             |
| `custom_role.sh`        | `generate_custom_role`                                                       |
| `group.sh`              | `generate_group`, `generate_group_membership`, `generate_user_data_sources` |
| `macro.sh`              | `generate_macro`                                                             |
| `ticket_field.sh`       | `generate_ticket_field`                                                      |
| `ticket_form.sh`        | `generate_ticket_form`                                                       |
| `trigger.sh`            | `generate_trigger`                                                           |
| `trigger_category.sh`   | `generate_trigger_category`                                                  |
| `user.sh`               | `generate_user`                                                              |
| `user_field.sh`         | `generate_user_field`                                                        |
| `view.sh`               | `generate_view`                                                              |

## Adding New Resources Manually

Create a new `.tf` file in the target environment directory with the appropriate prefix:

- **Group:** `group_<name>.tf`
- **View:** `view_<name>.tf`
- **Trigger:** `trigger_<name>.tf`
- **Trigger Category:** `trigger_category_<name>.tf`
- **Automation:** `automation_<name>.tf`
- **Macro:** `macro_<name>.tf`
- **Custom role:** `custom_role_<name>.tf`
- **User/Agent:** `<role>_<name>.tf` (e.g. `admins.tf`, `service_agents.tf`)
- **Brand:** `brand_<name>.tf`
- **Ticket Form:** `ticket_form_<name>.tf`
- **Ticket Field:** `ticket_field_<name>.tf`
- **User Field:** `user_field_<name>.tf`

Then import the resource into state:

```bash
cd infrastructure/sandbox
terraform import zendesk_group.support 43777127244173
```

## Supported Resource Types

Every resource has a matching data source for looking up existing objects by id or natural key (name, title, email, subdomain, key, etc.).

| Resource                   | Terraform Type                | Data Source Lookup By         |
|----------------------------|-------------------------------|-------------------------------|
| Groups                     | `zendesk_group`               | id, name                      |
| Group Memberships          | `zendesk_group_membership`    | id, (user_id + group_id)      |
| Views                      | `zendesk_view`                | id, title                     |
| Ticket Triggers            | `zendesk_trigger`             | id, title                     |
| Trigger Categories         | `zendesk_trigger_category`    | id, name                      |
| Automations                | `zendesk_automation`          | id, title                     |
| Macros                     | `zendesk_macro`               | id, title                     |
| Custom Agent Roles         | `zendesk_custom_role`         | id, name                      |
| Users / Agents             | `zendesk_user`                | id, email                     |
| Brands                     | `zendesk_brand`               | id, name, subdomain           |
| Ticket Forms               | `zendesk_ticket_form`         | id, name                      |
| Custom Ticket Fields       | `zendesk_ticket_field`        | id, title                     |
| Custom User Fields         | `zendesk_user_field`          | id, key, title                |
