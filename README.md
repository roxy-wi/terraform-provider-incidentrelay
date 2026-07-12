# Terraform Provider for IncidentRelay

Terraform provider for managing IncidentRelay configuration as code: access
groups, users, teams, notification channels, alert routes, on-call rotations,
escalation and notification policies, service catalog objects, silences,
maintenance windows, heartbeats, and business services.

The provider talks to the IncidentRelay HTTP API and supports both personal/API
Bearer tokens and username/password login.

## Requirements

- Terraform 1.7+
- Go 1.22.5+ to build locally
- IncidentRelay API access with permissions for the resources you manage

## Quick Start

```hcl
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.1"
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
- `incidentrelay_team`
- `incidentrelay_team_membership`
- `incidentrelay_channel`
- `incidentrelay_route`
- `incidentrelay_rotation`
- `incidentrelay_rotation_layer`
- `incidentrelay_rotation_layer_member`
- `incidentrelay_escalation_policy`
- `incidentrelay_escalation_policy_rule`
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
- `incidentrelay_business_service`
- `incidentrelay_business_service_component`

## Supported Data Sources

- `incidentrelay_version`
- `incidentrelay_group`
- `incidentrelay_team`
- `incidentrelay_user`
- `incidentrelay_service`

Nested API objects such as channel `config`, route `matchers`, service
`labels`, maintenance `scopes`, and heartbeat `metadata` are represented as
validated JSON strings. This keeps the provider compatible with IncidentRelay's
matcher and integration DSL without forcing every nested field into Terraform.

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
~/.terraform.d/plugins/registry.terraform.io/roxy-wi/incidentrelay/0.1.0/<os>_<arch>/
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
`ghcr.io/roxy-wi/incidentrelay:1.1` with Docker and runs this test layer. The
workflow authenticates to GHCR with the repository `GITHUB_TOKEN` and
`packages: read` permission before pulling the image.

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
git tag v0.1.2
git push origin v0.1.2
```

After the GitHub release is published, use Terraform Registry's `Publish >
Provider` flow and select `roxy-wi/terraform-provider-incidentrelay`.
