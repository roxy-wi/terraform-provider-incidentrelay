---
page_title: "incidentrelay_sso_provider Resource - IncidentRelay"
subcategory: "SSO"
description: |-
  Configures an IncidentRelay OIDC or SAML single sign-on provider.
---

# incidentrelay_sso_provider

Use this resource to configure an OIDC or SAML identity provider. The API
endpoints require an IncidentRelay global administrator.

## OIDC Example

```hcl
variable "oidc_client_secret" {
  type      = string
  sensitive = true
}

resource "incidentrelay_sso_provider" "corporate" {
  slug     = "corporate-oidc"
  label    = "Corporate login"
  protocol = "oidc"

  client_id        = "incidentrelay"
  client_secret    = var.oidc_client_secret
  oidc_metadata_url = "https://idp.example.com/.well-known/openid-configuration"
  oidc_scope        = "openid email profile groups"

  allowed_domains = [
    "example.com",
    "corp.example.com",
  ]

  auto_create_users                = true
  auto_link_by_email               = true
  require_verified_email           = true
  sync_group_memberships           = true
  remove_missing_group_memberships = false
}
```

The callback URL for this example is:

```text
https://incidentrelay.example.com/api/auth/sso/corporate-oidc/callback
```

## SAML Example

```hcl
variable "saml_sp_private_key" {
  type      = string
  sensitive = true
}

resource "incidentrelay_sso_provider" "adfs" {
  slug     = "adfs"
  label    = "Corporate ADFS"
  protocol = "saml"

  subject_claim      = "NameID"
  email_claim        = "email"
  username_claim     = "username"
  display_name_claim = "displayName"
  groups_claim       = "groups"

  saml_idp_entity_id = "https://adfs.example.com/adfs/services/trust"
  saml_idp_sso_url   = "https://adfs.example.com/adfs/ls/"
  saml_idp_x509_cert = file("${path.module}/adfs-signing.pem")

  saml_sp_entity_id   = "https://incidentrelay.example.com/api/auth/sso/adfs/metadata"
  saml_sp_acs_url     = "https://incidentrelay.example.com/api/auth/sso/adfs/callback"
  saml_sp_x509_cert   = file("${path.module}/incidentrelay-saml.pem")
  saml_sp_private_key = var.saml_sp_private_key

  extra_config_json = jsonencode({
    saml_security = {
      authnRequestsSigned   = true
      wantMessagesSigned    = true
      wantAssertionsSigned  = true
      requestedAuthnContext = false
    }
  })
}
```

## Schema

### Required

- `slug` (String) Stable provider slug used in login, callback, and metadata
  URLs.
- `label` (String) Human-readable label shown on the login page.

### Optional

- `protocol` (String) `oidc` or `saml`. Defaults to `oidc`.
- `enabled` (Boolean) Whether the provider is enabled. Defaults to `true`.
- `subject_claim` (String) Stable subject claim. Defaults to `sub`.
- `email_claim` (String) Email claim. Defaults to `email`.
- `username_claim` (String) Username claim. Defaults to
  `preferred_username`.
- `display_name_claim` (String) Display-name claim. Defaults to `name`.
- `groups_claim` (String) External groups claim. Defaults to `groups`.
- `phone_claim` (String) Phone-number claim. Defaults to `mobile`.
- `allowed_domains` (Set of String) Email domains allowed to use the provider.
- `auto_create_users` (Boolean) Automatically create local users. Defaults to
  `false`.
- `auto_link_by_email` (Boolean) Link an identity to an existing user by email.
  Defaults to `true`.
- `require_verified_email` (Boolean) Require a verified email. Defaults to
  `true`.
- `sync_group_memberships` (Boolean) Apply SSO group mappings at login.
  Defaults to `true`.
- `remove_missing_group_memberships` (Boolean) Disable SSO-managed memberships
  that are no longer present in external claims. Defaults to `false`.
- `client_id` (String) OIDC client ID.
- `client_secret` (String, Sensitive) OIDC client secret.
- `oidc_metadata_url` (String) OIDC discovery URL.
- `oidc_issuer` (String) Expected OIDC issuer.
- `oidc_authorization_endpoint` (String) OIDC authorization endpoint.
- `oidc_token_endpoint` (String) OIDC token endpoint.
- `oidc_userinfo_endpoint` (String) OIDC userinfo endpoint.
- `oidc_jwks_uri` (String) OIDC JSON Web Key Set URI.
- `oidc_scope` (String) Space-separated OIDC scopes. Defaults to
  `openid email profile`.
- `saml_idp_entity_id` (String) SAML IdP entity ID.
- `saml_idp_sso_url` (String) SAML IdP SSO URL.
- `saml_idp_slo_url` (String) SAML IdP logout URL.
- `saml_idp_x509_cert` (String) SAML IdP signing certificate.
- `saml_idp_metadata_url` (String) SAML IdP metadata URL.
- `saml_sp_entity_id` (String) IncidentRelay SAML SP entity ID.
- `saml_sp_acs_url` (String) IncidentRelay assertion consumer service URL.
- `saml_sp_sls_url` (String) IncidentRelay single logout service URL.
- `saml_sp_x509_cert` (String) IncidentRelay SAML SP certificate.
- `saml_sp_private_key` (String, Sensitive) IncidentRelay SAML SP private key.
- `saml_name_id_format` (String) SAML NameID format.
- `extra_config_json` (String) Additional SAML security configuration as JSON.

### Read-Only

- `has_client_secret` (Boolean) Whether a client secret is configured.
- `has_saml_sp_private_key` (Boolean) Whether an SP private key is configured.

## Secret State

IncidentRelay never returns `client_secret` or `saml_sp_private_key`. The
provider preserves configured values during refresh and exposes only the
read-only `has_*` flags from the API.

Terraform stores sensitive values in state. Use an encrypted remote backend and
restrict access to state files.

## Import

Import with the numeric SSO provider ID:

```sh
terraform import incidentrelay_sso_provider.corporate 12
```

Secrets are not returned during import. Add them to configuration when
Terraform should manage them.
