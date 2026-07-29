package incidentrelay

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	ProviderBaseURL  = "base_url"
	ProviderToken    = "token"
	ProviderUsername = "username"
	ProviderPassword = "password"
	ProviderInsecure = "insecure_skip_tls_verify"
)

type Config struct {
	Client *Client
}

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			ProviderBaseURL: {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("INCIDENTRELAY_BASE_URL", nil),
				Description: "IncidentRelay base URL, for example https://incidentrelay.example.com.",
			},
			ProviderToken: {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("INCIDENTRELAY_TOKEN", nil),
				Description: "Bearer API token or JWT token. If omitted, username/password authentication is used.",
			},
			ProviderUsername: {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("INCIDENTRELAY_USERNAME", nil),
				Description: "IncidentRelay username for /api/auth/login.",
			},
			ProviderPassword: {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("INCIDENTRELAY_PASSWORD", nil),
				Description: "IncidentRelay password for /api/auth/login.",
			},
			ProviderInsecure: {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: boolEnvDefaultFunc("INCIDENTRELAY_INSECURE_SKIP_TLS_VERIFY", false),
				Description: "Skip TLS certificate verification. Intended only for local development.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"incidentrelay_group":                      resourceGroup(),
			"incidentrelay_admin_user":                 resourceAdminUser(),
			"incidentrelay_group_membership":           resourceGroupMembership(),
			"incidentrelay_sso_provider":               resourceSSOProvider(),
			"incidentrelay_sso_group_mapping":          resourceSSOGroupMapping(),
			"incidentrelay_team":                       resourceTeam(),
			"incidentrelay_team_membership":            resourceTeamMembership(),
			"incidentrelay_channel":                    resourceChannel(),
			"incidentrelay_route":                      resourceRoute(),
			"incidentrelay_rotation":                   resourceRotation(),
			"incidentrelay_rotation_layer":             resourceRotationLayer(),
			"incidentrelay_rotation_layer_member":      resourceRotationLayerMember(),
			"incidentrelay_rotation_override":          resourceRotationOverride(),
			"incidentrelay_escalation_policy":          resourceEscalationPolicy(),
			"incidentrelay_escalation_policy_rule":     resourceEscalationPolicyRule(),
			"incidentrelay_notification_policy":        resourceNotificationPolicy(),
			"incidentrelay_notification_policy_rule":   resourceNotificationPolicyRule(),
			"incidentrelay_service":                    resourceService(),
			"incidentrelay_service_match_rule":         resourceServiceMatchRule(),
			"incidentrelay_service_link":               resourceServiceLink(),
			"incidentrelay_service_runbook":            resourceServiceRunbook(),
			"incidentrelay_service_dependency":         resourceServiceDependency(),
			"incidentrelay_silence":                    resourceSilence(),
			"incidentrelay_maintenance_window":         resourceMaintenanceWindow(),
			"incidentrelay_heartbeat":                  resourceHeartbeat(),
			"incidentrelay_business_service":           resourceBusinessService(),
			"incidentrelay_business_service_component": resourceBusinessServiceComponent(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"incidentrelay_version": datasourceVersion(),
			"incidentrelay_group":   datasourceGroup(),
			"incidentrelay_team":    datasourceTeam(),
			"incidentrelay_user":    datasourceUser(),
			"incidentrelay_service": datasourceService(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			terraformVersion = "1.0+compatible"
		}

		return providerConfigure(ctx, d, terraformVersion)
	}

	return p
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	client, err := NewClient(ClientConfig{
		BaseURL:   d.Get(ProviderBaseURL).(string),
		Token:     d.Get(ProviderToken).(string),
		Username:  d.Get(ProviderUsername).(string),
		Password:  d.Get(ProviderPassword).(string),
		UserAgent: fmt.Sprintf("terraform/%s terraform-provider-incidentrelay", terraformVersion),
		Insecure:  d.Get(ProviderInsecure).(bool),
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if err := client.Configure(ctx); err != nil {
		return nil, diag.FromErr(err)
	}

	return &Config{Client: client}, nil
}

func boolEnvDefaultFunc(key string, defaultValue bool) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		value := os.Getenv(key)
		if value == "" {
			return defaultValue, nil
		}
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid boolean value for %s: %w", key, err)
		}
		return parsed, nil
	}
}
