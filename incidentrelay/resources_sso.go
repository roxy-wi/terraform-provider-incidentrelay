package incidentrelay

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const defaultSSOExtraConfigJSON = `{
	"saml_security": {
		"nameIdEncrypted": false,
		"authnRequestsSigned": false,
		"logoutRequestSigned": false,
		"logoutResponseSigned": false,
		"signMetadata": false,
		"wantMessagesSigned": false,
		"wantAssertionsSigned": false,
		"wantNameId": true,
		"wantNameIdEncrypted": false,
		"wantAssertionsEncrypted": false,
		"wantAttributeStatement": false,
		"requestedAuthnContext": false,
		"signatureAlgorithm": "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
		"digestAlgorithm": "http://www.w3.org/2001/04/xmlenc#sha256"
	}
}`

func resourceSSOProvider() *schema.Resource {
	fields := []fieldDef{
		reqString("slug", "Stable SSO provider slug used in login and callback URLs."),
		reqString("label", "Human-readable SSO provider label."),
		optStringDefault("protocol", "oidc", "SSO protocol: oidc or saml."),
		optBoolDefault("enabled", true, "Whether the SSO provider is enabled."),

		optStringDefault("subject_claim", "sub", "Claim containing the stable external user identifier."),
		optStringDefault("email_claim", "email", "Claim containing the user email address."),
		optStringDefault("username_claim", "preferred_username", "Claim containing the username."),
		optStringDefault("display_name_claim", "name", "Claim containing the display name."),
		optStringDefault("groups_claim", "groups", "Claim containing external group names."),
		optStringDefault("phone_claim", "mobile", "Claim containing the phone number."),
		optStringSet("allowed_domains", "Email domains allowed to authenticate through this provider."),

		optBoolDefault("auto_create_users", false, "Automatically create local users after successful SSO login."),
		optBoolDefault("auto_link_by_email", true, "Link SSO identities to existing users with the same email address."),
		optBoolDefault("require_verified_email", true, "Require the identity provider to report a verified email."),
		optBoolDefault("sync_group_memberships", true, "Apply configured group mappings on SSO login."),
		optBoolDefault("remove_missing_group_memberships", false, "Disable SSO-managed memberships missing from the latest external group claims."),

		optString("client_id", "OIDC client id."),
		optSensitiveString("client_secret", "OIDC client secret. Omit on update to keep the current secret."),
		computedBool("has_client_secret", "Whether an OIDC client secret is configured."),
		optString("oidc_metadata_url", "OIDC discovery metadata URL."),
		optString("oidc_issuer", "Expected OIDC issuer."),
		optString("oidc_authorization_endpoint", "OIDC authorization endpoint."),
		optString("oidc_token_endpoint", "OIDC token endpoint."),
		optString("oidc_userinfo_endpoint", "OIDC userinfo endpoint."),
		optString("oidc_jwks_uri", "OIDC JSON Web Key Set URI."),
		optStringDefault("oidc_scope", "openid email profile", "Space-separated OIDC scopes."),

		optString("saml_idp_entity_id", "SAML identity provider entity id."),
		optString("saml_idp_sso_url", "SAML identity provider SSO URL."),
		optString("saml_idp_slo_url", "SAML identity provider single logout URL."),
		optString("saml_idp_x509_cert", "SAML identity provider x509 certificate."),
		optString("saml_idp_metadata_url", "SAML identity provider metadata URL."),
		optString("saml_sp_entity_id", "IncidentRelay SAML service provider entity id."),
		optString("saml_sp_acs_url", "IncidentRelay SAML assertion consumer service URL."),
		optString("saml_sp_sls_url", "IncidentRelay SAML single logout service URL."),
		optString("saml_sp_x509_cert", "IncidentRelay SAML service provider x509 certificate."),
		optSensitiveString("saml_sp_private_key", "IncidentRelay SAML service provider private key. Omit on update to keep the current key."),
		computedBool("has_saml_sp_private_key", "Whether a SAML service provider private key is configured."),
		optStringDefault("saml_name_id_format", "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress", "SAML NameID format."),
		optJSONDefault("extra_config_json", "extra_config", "{}", "Additional SAML security configuration as JSON."),
	}

	configurableFields := []string{
		"slug",
		"label",
		"protocol",
		"enabled",
		"subject_claim",
		"email_claim",
		"username_claim",
		"display_name_claim",
		"groups_claim",
		"phone_claim",
		"allowed_domains",
		"auto_create_users",
		"auto_link_by_email",
		"require_verified_email",
		"sync_group_memberships",
		"remove_missing_group_memberships",
		"client_id",
		"client_secret",
		"oidc_metadata_url",
		"oidc_issuer",
		"oidc_authorization_endpoint",
		"oidc_token_endpoint",
		"oidc_userinfo_endpoint",
		"oidc_jwks_uri",
		"oidc_scope",
		"saml_idp_entity_id",
		"saml_idp_sso_url",
		"saml_idp_slo_url",
		"saml_idp_x509_cert",
		"saml_idp_metadata_url",
		"saml_sp_entity_id",
		"saml_sp_acs_url",
		"saml_sp_sls_url",
		"saml_sp_x509_cert",
		"saml_sp_private_key",
		"saml_name_id_format",
		"extra_config_json",
	}

	resource := crudResource(resourceSpec{
		Description:  "IncidentRelay OIDC or SAML single sign-on provider.",
		Fields:       fields,
		CreatePath:   createPath("/api/admin/sso/providers"),
		ReadListPath: createPath("/api/admin/sso/providers"),
		UpdatePath:   idPath("/api/admin/sso/providers/%s"),
		DeletePath:   idPath("/api/admin/sso/providers/%s"),
		CreateFields: configurableFields,
		UpdateFields: configurableFields,
	})

	resource.Schema["slug"].ValidateFunc = validation.StringMatch(
		regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,39}$`),
		"must contain 2 to 40 lowercase letters, digits, underscores, or hyphens and start with a letter or digit",
	)
	resource.Schema["label"].ValidateFunc = validation.StringLenBetween(2, 128)
	resource.Schema["protocol"].ValidateFunc = validation.StringInSlice([]string{"oidc", "saml"}, false)
	resource.Schema["allowed_domains"].Elem.(*schema.Schema).StateFunc = func(value interface{}) string {
		return strings.ToLower(strings.TrimSpace(value.(string)))
	}
	resource.Schema["extra_config_json"].StateFunc = normalizeSSOExtraConfigState

	return resource
}

func resourceSSOGroupMapping() *schema.Resource {
	fields := []fieldDef{
		reqIntForceNew("provider_id", "Parent SSO provider id."),
		reqString("external_group", "External identity provider group name."),
		reqInt("group_id", "IncidentRelay group id."),
		optStringDefault("group_role", "viewer", "IncidentRelay group role: viewer, editor, user_admin, or global_admin."),
		optInt("team_id", "Optional IncidentRelay team id. The team must belong to group_id."),
		{
			Name:        "team_role",
			Kind:        kindString,
			Optional:    true,
			Computed:    true,
			Description: "IncidentRelay team role: viewer, responder, or manager. Defaults to viewer when team_id is set.",
		},
		optBoolDefault("active", true, "Whether the group mapping is active."),
		optIntDefault("priority", 100, "Mapping priority from 0 to 10000. Lower values are evaluated first."),
		computedString("group_slug", "IncidentRelay group slug."),
		computedString("group_name", "IncidentRelay group name."),
		computedString("team_slug", "IncidentRelay team slug."),
		computedString("team_name", "IncidentRelay team name."),
	}

	resource := crudResource(resourceSpec{
		Description:  "Maps an external SSO group to an IncidentRelay group and optional team.",
		Fields:       fields,
		CreatePath:   fieldCreatePath("/api/admin/sso/providers/%d/mappings", "provider_id"),
		ReadListPath: fieldListPath("/api/admin/sso/providers/%d/mappings", "provider_id"),
		UpdatePath:   idPath("/api/admin/sso/mappings/%s"),
		DeletePath:   idPath("/api/admin/sso/mappings/%s"),
		CreateFields: []string{"external_group", "group_id", "group_role", "team_id", "team_role", "active", "priority"},
		UpdateFields: []string{"external_group", "group_id", "group_role", "team_id", "team_role", "active", "priority"},
	})

	resource.Schema["provider_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["external_group"].ValidateFunc = validation.StringLenBetween(1, 100)
	resource.Schema["group_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["group_role"].ValidateFunc = validation.StringInSlice(
		[]string{"viewer", "editor", "user_admin", "global_admin"},
		false,
	)
	resource.Schema["team_id"].ValidateFunc = validation.IntAtLeast(1)
	resource.Schema["team_role"].ValidateFunc = validation.StringInSlice(
		[]string{"viewer", "responder", "manager"},
		false,
	)
	resource.Schema["priority"].ValidateFunc = validation.IntBetween(0, 10000)

	return resource
}

func normalizeSSOExtraConfigState(value interface{}) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}

	normalized, err := normalizeSSOExtraConfigJSON(raw)
	if err != nil {
		return raw
	}
	return normalized
}

func normalizeSSOExtraConfigJSON(raw string) (string, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return raw, err
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	var defaults map[string]interface{}
	if err := json.Unmarshal([]byte(defaultSSOExtraConfigJSON), &defaults); err != nil {
		return raw, err
	}

	securityDefaults := defaults["saml_security"].(map[string]interface{})
	security, _ := config["saml_security"].(map[string]interface{})
	for key := range securityDefaults {
		if value, exists := security[key]; exists {
			securityDefaults[key] = value
		}
	}
	config["saml_security"] = securityDefaults

	encoded, err := json.Marshal(config)
	if err != nil {
		return raw, err
	}
	return normalizeJSONString(string(encoded))
}
