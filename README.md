# Terraform Provider for IncidentRelay

Terraform provider for managing IncidentRelay configuration as code.

The provider talks to the IncidentRelay HTTP API and supports both personal/API
Bearer tokens and username/password login.

## Requirements

- Terraform 1.7+
- Go 1.22.5+ to build locally
- IncidentRelay API access with permissions for the resources you manage

## Build

```sh
make build
```

The binary is written to `bin/terraform-provider-incidentrelay`.

## Provider Configuration

```hcl
terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "0.1.0"
    }
  }
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

Environment variables are also supported:

- `INCIDENTRELAY_BASE_URL`
- `INCIDENTRELAY_TOKEN`
- `INCIDENTRELAY_USERNAME`
- `INCIDENTRELAY_PASSWORD`
- `INCIDENTRELAY_INSECURE_SKIP_TLS_VERIFY`

Prefer `INCIDENTRELAY_TOKEN` for CI and automation. `username` and `password`
are useful for local development.

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
git tag v0.1.0
git push origin v0.1.0
```

After the GitHub release is published, use Terraform Registry's `Publish >
Provider` flow and select `roxy-wi/terraform-provider-incidentrelay`.
