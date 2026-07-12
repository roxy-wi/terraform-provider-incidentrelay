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

The release workflow runs on semver tags such as `v0.1.2`. It builds
Registry-compatible zip assets, attaches the Terraform Registry manifest,
generates SHA256 checksums, and signs the checksum file with GPG.

## Release Checklist

1. Ensure CI is green on `main`.
2. Update documentation and examples for new resources or fields.
3. Create the next semver tag, for example `v0.1.2`.
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
