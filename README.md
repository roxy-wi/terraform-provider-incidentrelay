# Terraform Provider for IncidentRelay

Terraform provider for managing IncidentRelay configuration as code: access
groups, users, teams, notification channels, alert routes, on-call rotations,
escalation and notification policies, service catalog objects, silences,
incident priority policies, maintenance windows, heartbeats, business services,
Event Orchestration, reusable orchestration webhooks, and OIDC/SAML SSO.

The provider talks to the IncidentRelay HTTP API and supports both personal/API
Bearer tokens and username/password login.

## Requirements

- Terraform 1.7+
- Go 1.22.5+ to build locally
- IncidentRelay API access with permissions for the resources you manage

## IncidentRelay Compatibility

The current provider code is tested against IncidentRelay 2.0. It manages Event
Orchestration definitions and reusable webhook actions, lifecycle-aware
silences and maintenance windows, and the Uptime Kuma route source. It also
supports Datadog routes, Slack Bot API channels over HTTP actions or Socket
Mode, SSO providers and group mappings, and incident priority policies. The
provider preserves API-masked Slack, SSO, and orchestration webhook secrets
during refresh so masking does not cause perpetual Terraform drift.

## Quick Start

```hcl
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.6"
    }
  }
}

variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}

resource "incidentrelay_group" "infra" {
  slug = "infra"
  name = "Infrastructure"
}

resource "incidentrelay_team" "platform" {
  group_id = incidentrelay_group.infra.id
  slug     = "platform"
  name     = "Platform"
}
```

Run:

```sh
export INCIDENTRELAY_TOKEN="..."
terraform init
terraform plan
terraform apply
```

## Provider Configuration

Environment variables are also supported:

- `INCIDENTRELAY_BASE_URL`
- `INCIDENTRELAY_TOKEN`
- `INCIDENTRELAY_USERNAME`
- `INCIDENTRELAY_PASSWORD`
- `INCIDENTRELAY_INSECURE_SKIP_TLS_VERIFY`

Prefer `INCIDENTRELAY_TOKEN` for CI and automation. `username` and `password`
are useful for local development.

```hcl
provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}
```

Set `insecure_skip_tls_verify = true` only for local development against a test
IncidentRelay instance with a self-signed certificate.

## Supported Resources

- `incidentrelay_group`
- `incidentrelay_admin_user`
- `incidentrelay_group_membership`
- `incidentrelay_sso_provider`
- `incidentrelay_sso_group_mapping`
- `incidentrelay_team`
- `incidentrelay_team_membership`
- `incidentrelay_channel`
- `incidentrelay_route`
- `incidentrelay_rotation`
- `incidentrelay_rotation_layer`
- `incidentrelay_rotation_layer_member`
- `incidentrelay_rotation_override`
- `incidentrelay_escalation_policy`
- `incidentrelay_escalation_policy_rule`
- `incidentrelay_priority_policy`
- `incidentrelay_priority_policy_rule`
- `incidentrelay_notification_policy`
- `incidentrelay_notification_policy_rule`
- `incidentrelay_service`
- `incidentrelay_service_match_rule`
- `incidentrelay_service_link`
- `incidentrelay_service_runbook`
- `incidentrelay_service_dependency`
- `incidentrelay_silence`
- `incidentrelay_maintenance_window`
- `incidentrelay_heartbeat`
- `incidentrelay_event_orchestration`
- `incidentrelay_orchestration_webhook_action`
- `incidentrelay_business_service`
- `incidentrelay_business_service_component`

## Supported Data Sources

- `incidentrelay_version`
- `incidentrelay_group`
- `incidentrelay_team`
- `incidentrelay_user`
- `incidentrelay_channel`
- `incidentrelay_service`
- `incidentrelay_rotation`
- `incidentrelay_incident_priority`
- `incidentrelay_escalation_policy`
- `incidentrelay_notification_policy`
- `incidentrelay_service_match_rule`

Nested API objects such as channel `config`, route `matchers`, service
`labels`, maintenance `scopes`, heartbeat `metadata`, orchestration `rules`,
and webhook `headers` are represented as validated JSON strings. This keeps the
provider compatible with IncidentRelay's nested DSLs without forcing every API
field into Terraform.

Channel `config_json` is marked sensitive because it can contain credentials.
Terraform still stores sensitive values in state, so use an encrypted remote
backend and restrict access to state files.

SSO `client_secret` and `saml_sp_private_key` are also marked sensitive. The
IncidentRelay API returns only flags indicating whether those secrets exist;
the provider preserves configured secret values during refresh.

Orchestration webhook `url` and `headers_json` are sensitive as well. Headers
are write-only, and URLs may be returned with embedded credentials redacted;
the provider preserves configured values in both cases.

Use `jsonencode(...)` for those fields:

```hcl
matchers_json = jsonencode({
  labels = {
    service     = "platform-api"
    environment = "production"
  }
})
```

## Examples

- [Provider quickstart](examples/provider/example.tf)
- [Authentication patterns](examples/authentication/main.tf)
- [Core on-call setup](examples/core-oncall/main.tf)
- [Service catalog](examples/service-catalog/main.tf)
- [Maintenance and silences](examples/maintenance/main.tf)
- [Heartbeat monitoring](examples/heartbeat/main.tf)
- [IncidentRelay 1.2: Datadog and Slack Socket Mode](examples/incidentrelay-1.2/main.tf)
- [IncidentRelay 2.0: Event Orchestration and Uptime Kuma](examples/incidentrelay-2.0/main.tf)
- [OIDC SSO and group mapping](examples/sso/main.tf)
- [Data source lookups](examples/data-sources/main.tf)
- [Terraform import blocks](examples/imports/main.tf)

## Documentation

- [Provider docs](docs/index.md)
- [Authentication guide](docs/guides/authentication.md)
- [Import guide](docs/guides/importing.md)
- [JSON fields guide](docs/guides/json-fields.md)
- [Modeling guide](docs/guides/modeling.md)
- [Resource reference](docs/guides/resource-reference.md)
- [Testing and release guide](docs/guides/testing-and-release.md)

## Local Installation

```sh
make install-local
```

This installs the provider under:

```text
~/.terraform.d/plugins/registry.terraform.io/roxy-wi/incidentrelay/0.6.0/<os>_<arch>/
```

## Example

See [examples/provider/example.tf](examples/provider/example.tf).

## Development

```sh
make fmt-check
make test-race
make build
```

The CI workflow runs the same checks on pull requests and pushes to `main`.

Acceptance tests can be run against a live IncidentRelay instance:

```sh
export INCIDENTRELAY_ACC=1
export INCIDENTRELAY_BASE_URL="http://127.0.0.1:8080"
export INCIDENTRELAY_USERNAME="admin"
export INCIDENTRELAY_PASSWORD="change-me-123"
make test-acc
```

GitHub Actions also includes an `Acceptance` workflow that starts
IncidentRelay with Docker and runs this test layer. It tries to pull
`ghcr.io/roxy-wi/incidentrelay:latest` anonymously first, then retries after a
GHCR login with `GITHUB_TOKEN`. If GHCR is still unavailable, it builds the image
from `roxy-wi/IncidentRelay@main` on the runner and uses that local image.

## Publishing to the Terraform Registry

The GitHub repository must be public and named `terraform-provider-incidentrelay`.
The release workflow builds Registry-compatible zip assets, adds
`terraform-provider-incidentrelay_<version>_manifest.json`, generates
`terraform-provider-incidentrelay_<version>_SHA256SUMS`, and signs checksums with
GPG.

The release workflow expects these GitHub Actions secrets, which are already
configured in the `roxy-wi/terraform-provider-incidentrelay` repository:

- `GPG_PRIVATE_KEY`: ASCII-armored private key used only for provider releases.
- `PASSPHRASE`: passphrase for that key.

Add the corresponding ASCII-armored public key in Terraform Registry settings for
the `roxy-wi` namespace. Then create and push a semver tag:

```sh
git tag v0.6.0
git push origin v0.6.0
```

After the GitHub release is published, use Terraform Registry's `Publish >
Provider` flow and select `roxy-wi/terraform-provider-incidentrelay`.
