---
page_title: "Authentication - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Authentication patterns for the IncidentRelay Terraform provider.
---

# Authentication

The provider supports two authentication modes:

- Bearer token authentication with `token` or `INCIDENTRELAY_TOKEN`.
- Username/password login against `/api/auth/login`.

Token authentication is recommended for CI, Terraform Cloud, and automation.
Username/password authentication is useful for local development or bootstrap
workflows.

## Token Authentication

```hcl
variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

For CI, prefer environment variables:

```sh
export INCIDENTRELAY_BASE_URL="https://incidentrelay.example.com"
export INCIDENTRELAY_TOKEN="..."
terraform plan
```

### IncidentRelay 2.3 token scopes

Personal API tokens in IncidentRelay 2.3 use granular read/write scopes. A
Terraform plan needs the matching `:read` scopes, and apply/destroy also need
the matching `:write` scopes:

| Terraform configuration | IncidentRelay scopes |
| --- | --- |
| Groups and memberships | `groups:read`, `groups:write` |
| Admin users | `users:read`, `users:write` |
| Teams and memberships | `teams:read`, `teams:write` |
| Channels | `channels:read`, `channels:write` |
| Routes | `routes:read`, `routes:write` |
| Rotations and overrides | `rotations:read`, `rotations:write` |
| Escalation, notification, priority, and matcher policies | `policies:read`, `policies:write` |
| Services and business services | `services:read`, `services:write` |
| Silences and maintenance windows | `maintenance:read`, `maintenance:write` |
| Heartbeats | `heartbeats:read`, `heartbeats:write` |
| Event Orchestration and webhook actions | `orchestrations:read`, `orchestrations:write` |
| SSO providers and mappings | `sso:read`, `sso:write` |
| Incident priority data source | `incidents:read` |

The legacy `resources:read` and `resources:write` aggregate scopes remain
compatible with these configuration domains. Prefer the granular scopes for a
least-privilege automation token.

## Username And Password Authentication

```hcl
variable "incidentrelay_username" {
  type = string
}

variable "incidentrelay_password" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}
```

The provider exchanges the credentials for an access token during provider
configuration. The password is marked sensitive in Terraform.

## SSO Configuration And Provider Authentication

`incidentrelay_sso_provider` configures how users sign in to IncidentRelay.
It does not authenticate Terraform itself. The Terraform provider still needs
an API token or a local IncidentRelay username and password with global
administrator access to manage SSO providers and group mappings.

## Change Integrations (IR 2.5)

IR 2.5.0 uses a JWT for a global administrator or an authorized Group
`user_admin`. Set the login username and password so the provider obtains a
JWT, or supply a valid JWT as the provider token.

IR 2.5.1+ also accepts personal API tokens with both
`change-integrations:read` and `change-integrations:write`. The token owner
must have the administrator role required for the integration. Group-bound
tokens can manage integrations owned by that Group. Central integrations
require a global administrator token without a Group or Team bound.
Team-bound tokens are denied for integration administration.

Change integration machine credentials with `changes:ingest` authenticate
native Change Events and GitLab deployment ingestion. Use a JWT or personal
API token for Terraform administration.

## System settings (IR 2.5)

IR 2.5.0 requires a global administrator JWT. On IR 2.5.1+, the resource also
accepts that administrator's personal token with `system-settings:read` and
`system-settings:write`. The data source needs only `system-settings:read`.
System settings tokens must have no Group or Team bound.

## Directory Sync (IR 2.5.1+)

Use a global administrator or destination Group `user_admin`, authenticated
by JWT or a personal token with `directory-sync:read` and
`directory-sync:write`. A Group-bound token can manage that Group's sources.
Use a token without a Team bound.

The same `directory-sync` scopes cover Jira CMDB configuration and source
deletion on IR 2.5.2. The Jira PAT belongs in the source's sensitive
`headers_json`, separately from the IR token used by the provider.

## Personal token configuration

```hcl
provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  token    = var.incidentrelay_personal_token
}
```

You can also set `INCIDENTRELAY_TOKEN`. The provider sends the credential as
a Bearer token. Terraform refresh, import and revision checks require read
scopes in addition to write scopes. Legacy `resources:read/write` scopes
cover their established resource domains. Add the dedicated
`change-integrations`, `system-settings` and `directory-sync` scopes for
these administrative features. Server-side Group, Team and role restrictions
apply to personal tokens on IR 2.5.1.

## Local Development With Self-Signed TLS

```hcl
provider "incidentrelay" {
  base_url                 = "https://localhost:5000"
  token                    = var.incidentrelay_token
  insecure_skip_tls_verify = true
}
```

Do not use `insecure_skip_tls_verify` in production automation.

## GitHub Actions Example

```yaml
name: Terraform

on:
  pull_request:

jobs:
  plan:
    runs-on: ubuntu-latest
    env:
      INCIDENTRELAY_BASE_URL: ${{ secrets.INCIDENTRELAY_BASE_URL }}
      INCIDENTRELAY_TOKEN: ${{ secrets.INCIDENTRELAY_TOKEN }}
    steps:
      - uses: actions/checkout@v6
      - uses: hashicorp/setup-terraform@v3
      - run: terraform init
      - run: terraform validate
      - run: terraform plan -input=false
```

## Secret Handling

- Keep tokens in the secret store of your CI platform.
- Do not commit `.tfvars` files containing secrets.
- Use sensitive Terraform variables for tokens and passwords.
- Rotate tokens used by CI separately from human user credentials.
