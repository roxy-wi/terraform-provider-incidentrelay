# Terraform Registry Publishing Checklist

This repository is prepared for the public Terraform Registry as:

```text
roxy-wi/incidentrelay
```

The GitHub repository must remain public and named:

```text
terraform-provider-incidentrelay
```

Release signing key fingerprint:

```text
A552E788E097A7E8692DD20ACF2F18BA743AC8EC
```

## 1. Release Signing Key

Terraform Registry provider releases must be signed. This repository already has
a dedicated RSA GPG key configured in GitHub Actions secrets:

- `GPG_PRIVATE_KEY`
- `PASSPHRASE`

If the key ever needs to be rotated, use a dedicated RSA or DSA GPG key for
provider releases. The Registry does not accept the default ECC key type.

Example RSA key generation for rotation:

```sh
gpg --full-generate-key
```

Choose:

- key type: RSA and RSA
- key size: 4096
- expiration: your normal release-key policy
- user ID: `IncidentRelay Terraform Provider <release@incidentrelay.io>`

## 2. GitHub Actions Secrets

The required GitHub Actions secrets have already been created. If rotating the
key, export the private key:

Export the private key:

```sh
gpg --armor --export-secret-keys "<KEY_ID_OR_EMAIL>" > gpg-private-key.asc
```

Add repository secrets:

```sh
gh secret set GPG_PRIVATE_KEY -R roxy-wi/terraform-provider-incidentrelay < gpg-private-key.asc
printf '%s' "<key passphrase>" | gh secret set PASSPHRASE -R roxy-wi/terraform-provider-incidentrelay
```

Do not commit `gpg-private-key.asc`.

## 3. Add Public Key to Terraform Registry

Export the public key:

```sh
gpg --armor --export "<KEY_ID_OR_EMAIL>" > terraform-registry-gpg-public-key.asc
```

In Terraform Registry, open `User Settings > Signing Keys` and add this public
key for the `roxy-wi` namespace.

For this repository, the public key is committed at:

```text
registry/terraform-registry-gpg-public-key.asc
```

## 4. Create the First Release

Create and push a semver tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The GitHub Actions release workflow runs GoReleaser and publishes release
assets.

## 5. Verify Release Assets

The release must contain:

- `terraform-provider-incidentrelay_0.1.0_<os>_<arch>.zip`
- `terraform-provider-incidentrelay_0.1.0_manifest.json`
- `terraform-provider-incidentrelay_0.1.0_SHA256SUMS`
- `terraform-provider-incidentrelay_0.1.0_SHA256SUMS.sig`

Each zip contains a binary named:

```text
terraform-provider-incidentrelay_v0.1.0
```

## 6. Publish in Terraform Registry

Open Terraform Registry and choose:

```text
Publish > Provider
```

Select:

```text
roxy-wi/terraform-provider-incidentrelay
```

After publishing, Registry webhooks ingest future GitHub releases automatically.
