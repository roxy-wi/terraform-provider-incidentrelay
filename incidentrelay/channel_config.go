package incidentrelay

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/idna"
	"golang.org/x/text/unicode/norm"
)

// suppressEquivalentEmailConfig recognizes the normalization performed by IR
// 2.4's normalize_email_recipient_config. It is deliberately scoped to email:
// recipient-like keys in other channel types must remain ordinary JSON values.
func suppressEquivalentEmailConfig(_ string, old, new string, d *schema.ResourceData) bool {
	if d.Get("channel_type") != "email" {
		return false
	}
	oldType, newType := d.GetChange("channel_type")
	if oldType != "" && oldType != newType {
		return false
	}
	oldConfig, oldOK := comparableEmailConfig(old)
	newConfig, newOK := comparableEmailConfig(new)
	// Invalid or unknown values must reach API validation, never disappear from
	// the plan merely because both sides failed to normalize.
	return oldOK && newOK && reflect.DeepEqual(oldConfig, newConfig)
}

func comparableEmailConfig(raw string) (map[string]interface{}, bool) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil || config == nil {
		return nil, false
	}
	mode, present := config["recipient_mode"]
	if !present {
		mode = "assignee"
	}
	switch mode {
	case "assignee":
		// The API removes both fields, including a stale fixed address.
		delete(config, "recipient_mode")
		delete(config, "recipient_email")
	case "fixed", "fallback":
		address, ok := config["recipient_email"].(string)
		if !ok {
			return nil, false
		}
		local, domain, ok := strings.Cut(strings.TrimSpace(address), "@")
		if !ok || local == "" || domain == "" || strings.Contains(domain, "@") {
			return nil, false
		}
		// email-validator uses NFC for the local part and non-transitional
		// UTS-46/IDNA for the domain. Local-part case is significant, except
		// for the RFC 2142 role mailboxes normalized by email-validator.
		local = norm.NFC.String(local)
		switch strings.ToLower(local) {
		case "info", "marketing", "sales", "support", "abuse", "noc", "security",
			"postmaster", "hostmaster", "usenet", "news", "webmaster", "www", "uucp", "ftp":
			local = strings.ToLower(local)
		}
		domain, err := idna.Lookup.ToUnicode(strings.ToLower(domain))
		if err != nil {
			return nil, false
		}
		config["recipient_email"] = local + "@" + domain
	default:
		return nil, false
	}
	return config, true
}
