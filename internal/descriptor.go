package internal

import (
	"context"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type authProviderDescribeStep struct {
	name   string
	config map[string]any
}

func newAuthProviderDescribeStep(name string, config map[string]any) (sdk.StepInstance, error) {
	return &authProviderDescribeStep{name: name, config: config}, nil
}

func (s *authProviderDescribeStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current, _, _ map[string]any) (*sdk.StepResult, error) {
	values := mergeMaps(s.config, current)
	providerID := firstNonEmpty(values, "provider_id", "providerId")
	if providerID == "" {
		providerID = "ory_polis"
	}
	baseURL := firstNonEmpty(values, "base_url", "baseUrl", "admin_url", "adminUrl")
	return &sdk.StepResult{Output: map[string]any{
		"providers": []map[string]any{oryPolisProviderDescriptor(providerID, baseURL)},
	}}, nil
}

func oryPolisProviderDescriptor(providerID, baseURL string) map[string]any {
	return map[string]any{
		"id":             providerID,
		"label":          "Ory Polis",
		"description":    "Ory Polis enterprise SSO and Directory Sync administration integration.",
		"categories":     []string{"enterprise_sso", "directory_sync"},
		"implementation": "workflow-plugin-ory-polis",
		"version":        Version,
		"docs_url":       "https://github.com/ory/polis",
		"support_level":  "management",
		"capabilities": []map[string]any{
			oryPolisCapability("ory_polis_sso_connections", "SSO connections", "enterprise_sso", "Create, read, list, update, and delete SAML/OIDC SSO connections through the Polis API.", []string{"polis.sso.connections.read", "polis.sso.connections.write"}, oryPolisFields(baseURL)),
			oryPolisCapability("ory_polis_sso_setup_links", "SSO setup links", "enterprise_sso", "Create and manage customer setup links for delegated SSO configuration.", []string{"polis.sso.setup_links.read", "polis.sso.setup_links.write"}, oryPolisFields(baseURL)),
			oryPolisCapability("ory_polis_directory_connections", "Directory Sync connections", "directory_sync", "Create, read, list, update, and delete SCIM/Directory Sync connections.", []string{"polis.directory.connections.read", "polis.directory.connections.write"}, oryPolisFields(baseURL)),
			oryPolisCapability("ory_polis_directory_resources", "Directory users and groups", "directory_sync", "Read synchronized directory users, groups, group members, and event logs.", []string{"polis.directory.resources.read"}, oryPolisFields(baseURL)),
			oryPolisCapability("ory_polis_directory_setup_links", "Directory setup links", "directory_sync", "Create and manage delegated setup links for Directory Sync configuration.", []string{"polis.directory.setup_links.read", "polis.directory.setup_links.write"}, oryPolisFields(baseURL)),
			oryPolisCapability("ory_polis_identity_federation", "Identity federation", "enterprise_sso", "Read identity federation configuration and SSO mappings exposed by Polis.", []string{"polis.identity_federation.read"}, oryPolisFields(baseURL)),
		},
	}
}

func oryPolisCapability(key, label, category, description string, appScopes []string, fields []map[string]any) map[string]any {
	return map[string]any{
		"key":                key,
		"label":              label,
		"category":           category,
		"description":        description,
		"supported":          true,
		"app_scopes":         appScopes,
		"admin_read_scopes":  []string{"admin.auth.providers.read"},
		"admin_write_scopes": []string{"admin.auth.providers.write"},
		"config_fields":      fields,
	}
}

func oryPolisFields(baseURL string) []map[string]any {
	return []map[string]any{
		oryPolisField("ory_polis_base_url", "Polis API URL", "url", "Base URL for the Ory Polis service.", "Keep the management API private and call it only from trusted server-side workflow steps.", false, true, optionIfSet(strings.TrimRight(baseURL, "/"))),
		oryPolisField("ory_polis_api_key", "Polis API key", "secret", "API key sent as Authorization: Bearer <token>.", "Write-only secret. Use least-privilege keys and rotate them regularly.", true, false, nil),
	}
}

func oryPolisField(key, label, inputType, description, helpText string, secret, required bool, options []map[string]any) map[string]any {
	return map[string]any{
		"key":         key,
		"label":       label,
		"input_type":  inputType,
		"description": description,
		"help_text":   helpText,
		"secret":      secret,
		"required":    required,
		"options":     options,
	}
}

func optionIfSet(value string) []map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []map[string]any{{"value": value, "label": value}}
}
