---
page_title: "Testing And Release - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Local testing, GitHub Actions, and Terraform Registry release workflow for the IncidentRelay provider.
---

# Testing And Release

## Local Checks

```sh
make fmt-check
make test-race
make build
```

The test suite uses Go unit tests and `httptest`; it does not require a running
IncidentRelay instance.

## GitHub Actions

The CI workflow runs on pull requests and pushes to `main`:

```sh
test -z "$(gofmt -l .)"
go test -race ./...
go build -o /tmp/terraform-provider-incidentrelay .
```

The acceptance workflow starts a real IncidentRelay container with Docker,
creates a temporary admin user, and runs the provider acceptance tests with
`INCIDENTRELAY_ACC=1`.

Acceptance also installs Terraform CLI and runs a real CLI smoke flow with a
locally built provider binary through Terraform's development override
mechanism. That test covers `terraform validate`, `import`, `apply`,
idempotent `plan -detailed-exitcode`, and `destroy`.

The acceptance matrix retains IR 2.3 coverage and adds IR 2.4:

- IR 2.3 first pulls `ghcr.io/roxy-wi/incidentrelay:2.3` anonymously, then retries
  with GHCR authentication and falls back to building source tag `v2.3.0`.
- IR 2.4 builds the final source tag `v2.4.0`.

The 2.4-specific tests skip on older servers. On IR 2.4+, they cover OpenSearch
and Kibana routes; legacy, assignee, fallback and fixed email configurations;
and publication/update of `reopen_window_seconds`. The CLI flow additionally
imports the new configuration, changes recipient modes, and requires an empty
plan after each apply and after import.
CLI writes use `-parallelism=1` because the disposable server uses SQLite;
this avoids API transaction upgrade conflicts and does not test concurrent
production writes.

Acceptance tests are intentionally separate from the fast CI workflow because
they pull a Docker image, run migrations, and exercise the live HTTP API.

Run the same test layer locally against an already running IncidentRelay:

```sh
export INCIDENTRELAY_ACC=1
export INCIDENTRELAY_BASE_URL="http://127.0.0.1:8080"
export INCIDENTRELAY_USERNAME="admin"
export INCIDENTRELAY_PASSWORD="change-me-123"
export INCIDENTRELAY_EXPECTED_VERSION="2.4"
make test-acc
```

The release workflow runs on semver tags such as `v0.8.0`. It builds
Registry-compatible zip assets, attaches the Terraform Registry manifest,
generates SHA256 checksums, and signs the checksum file with GPG.

## Release Checklist

1. Ensure CI is green on `main`.
2. Update documentation and examples for new resources or fields.
3. Create the next semver tag, for example `v0.8.0`.
4. Push the tag.
5. Verify GitHub Release assets:
   - platform zip files
   - `terraform-provider-incidentrelay_<version>_manifest.json`
   - `terraform-provider-incidentrelay_<version>_SHA256SUMS`
   - `terraform-provider-incidentrelay_<version>_SHA256SUMS.sig`
6. Confirm the signing key is registered in Terraform Registry.

## Registry Signing Key

The public key is committed at:

```text
registry/terraform-registry-gpg-public-key.asc
```

Fingerprint:

```text
A552E788E097A7E8692DD20ACF2F18BA743AC8EC
```
