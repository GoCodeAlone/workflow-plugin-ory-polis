package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type stepConstructor func(name string, config map[string]any) (sdk.StepInstance, error)

var stepRegistry = map[string]stepConstructor{
	"step.ory_polis_auth_provider_describe":       newAuthProviderDescribeStep,
	"step.ory_polis_sso_connection_create":        newPolisStep(oryPolisSSOConnectionCreate),
	"step.ory_polis_sso_connection_get":           newPolisStep(oryPolisSSOConnectionGet),
	"step.ory_polis_sso_connection_list":          newPolisStep(oryPolisSSOConnectionList),
	"step.ory_polis_sso_connection_update":        newPolisStep(oryPolisSSOConnectionUpdate),
	"step.ory_polis_sso_connection_delete":        newPolisStep(oryPolisSSOConnectionDelete),
	"step.ory_polis_directory_create":             newPolisStep(oryPolisDirectoryCreate),
	"step.ory_polis_directory_get":                newPolisStep(oryPolisDirectoryGet),
	"step.ory_polis_directory_list":               newPolisStep(oryPolisDirectoryList),
	"step.ory_polis_directory_update":             newPolisStep(oryPolisDirectoryUpdate),
	"step.ory_polis_directory_delete":             newPolisStep(oryPolisDirectoryDelete),
	"step.ory_polis_directory_event_list":         newPolisStep(oryPolisDirectoryEventList),
	"step.ory_polis_directory_user_list":          newPolisStep(oryPolisDirectoryUserList),
	"step.ory_polis_directory_user_get":           newPolisStep(oryPolisDirectoryUserGet),
	"step.ory_polis_directory_group_list":         newPolisStep(oryPolisDirectoryGroupList),
	"step.ory_polis_directory_group_get":          newPolisStep(oryPolisDirectoryGroupGet),
	"step.ory_polis_directory_group_member_list":  newPolisStep(oryPolisDirectoryGroupMemberList),
	"step.ory_polis_sso_setup_link_create":        newPolisStep(oryPolisSSOSetupLinkCreate),
	"step.ory_polis_sso_setup_link_get":           newPolisStep(oryPolisSSOSetupLinkGet),
	"step.ory_polis_sso_setup_link_delete":        newPolisStep(oryPolisSSOSetupLinkDelete),
	"step.ory_polis_directory_setup_link_create":  newPolisStep(oryPolisDirectorySetupLinkCreate),
	"step.ory_polis_directory_setup_link_get":     newPolisStep(oryPolisDirectorySetupLinkGet),
	"step.ory_polis_directory_setup_link_delete":  newPolisStep(oryPolisDirectorySetupLinkDelete),
	"step.ory_polis_identity_federation_get":      newPolisStep(oryPolisIdentityFederationGet),
	"step.ory_polis_identity_federation_sso_list": newPolisStep(oryPolisIdentityFederationSSOList),
}

func allStepTypes() []string {
	return []string{
		"step.ory_polis_auth_provider_describe",
		"step.ory_polis_sso_connection_create",
		"step.ory_polis_sso_connection_get",
		"step.ory_polis_sso_connection_list",
		"step.ory_polis_sso_connection_update",
		"step.ory_polis_sso_connection_delete",
		"step.ory_polis_directory_create",
		"step.ory_polis_directory_get",
		"step.ory_polis_directory_list",
		"step.ory_polis_directory_update",
		"step.ory_polis_directory_delete",
		"step.ory_polis_directory_event_list",
		"step.ory_polis_directory_user_list",
		"step.ory_polis_directory_user_get",
		"step.ory_polis_directory_group_list",
		"step.ory_polis_directory_group_get",
		"step.ory_polis_directory_group_member_list",
		"step.ory_polis_sso_setup_link_create",
		"step.ory_polis_sso_setup_link_get",
		"step.ory_polis_sso_setup_link_delete",
		"step.ory_polis_directory_setup_link_create",
		"step.ory_polis_directory_setup_link_get",
		"step.ory_polis_directory_setup_link_delete",
		"step.ory_polis_identity_federation_get",
		"step.ory_polis_identity_federation_sso_list",
	}
}

func createStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	constructor, ok := stepRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("ory polis plugin: unknown step type %q", typeName)
	}
	return constructor(name, config)
}

type polisHandler func(context.Context, *OryPolisClient, map[string]any) (map[string]any, error)

type polisStep struct {
	name       string
	moduleName string
	handler    polisHandler
}

func newPolisStep(handler polisHandler) stepConstructor {
	return func(name string, config map[string]any) (sdk.StepInstance, error) {
		moduleName := stringValue(config, "module")
		if moduleName == "" {
			moduleName = "ory_polis"
		}
		return &polisStep{name: name, moduleName: moduleName, handler: handler}, nil
	}
}

func (s *polisStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _, config map[string]any) (*sdk.StepResult, error) {
	client, ok := GetClient(s.moduleName)
	if !ok {
		return &sdk.StepResult{Output: map[string]any{"error": "ory polis client not found: " + s.moduleName}}, nil
	}
	output, err := s.handler(ctx, client, mergeMaps(config, current))
	if err != nil {
		return &sdk.StepResult{Output: errResult(err)}, nil
	}
	return &sdk.StepResult{Output: output}, nil
}

func oryPolisSSOConnectionCreate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	body := requestBody(values, "connection")
	result, err := client.HTTP.do(ctx, http.MethodPost, "/api/v1/sso", nil, body)
	return namedResult("connection", result, err)
}

func oryPolisSSOConnectionGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	query := ssoQuery(values)
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/sso", query, nil)
	return namedResult("connections", result, err)
}

func oryPolisSSOConnectionList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	query := ssoQuery(values)
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/sso", query, nil)
	return namedResult("connections", result, err)
}

func oryPolisSSOConnectionUpdate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	body := requestBody(values, "connection")
	result, err := client.HTTP.do(ctx, http.MethodPatch, "/api/v1/sso", nil, body)
	return namedResult("connection", result, err)
}

func oryPolisSSOConnectionDelete(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	query := ssoQuery(values)
	if query["clientID"] == "" && (query["tenant"] == "" || query["product"] == "") {
		return nil, fmt.Errorf("client_id or tenant/product is required")
	}
	result, err := client.HTTP.do(ctx, http.MethodDelete, "/api/v1/sso", query, nil)
	return namedResult("deleted", result, err)
}

func oryPolisDirectoryCreate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	body := requestBody(values, "directory")
	result, err := client.HTTP.do(ctx, http.MethodPost, "/api/v1/dsync", nil, body)
	return namedResult("directory", result, err)
}

func oryPolisDirectoryGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	if id := firstNonEmpty(values, "directory_id", "directoryId", "id"); id != "" {
		result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/"+url.PathEscape(id), nil, nil)
		return namedResult("directory", result, err)
	}
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync", directoryQuery(values), nil)
	return namedResult("directories", result, err)
}

func oryPolisDirectoryList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync", directoryQuery(values), nil)
	return namedResult("directories", result, err)
}

func oryPolisDirectoryUpdate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	body := requestBody(values, "directory")
	result, err := client.HTTP.do(ctx, http.MethodPatch, "/api/v1/dsync/"+url.PathEscape(id), nil, body)
	return namedResult("directory", result, err)
}

func oryPolisDirectoryDelete(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	result, err := client.HTTP.do(ctx, http.MethodDelete, "/api/v1/dsync/"+url.PathEscape(id), nil, nil)
	return namedResult("deleted", result, err)
}

func oryPolisDirectoryEventList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/events", directoryQuery(values), nil)
	return namedResult("events", result, err)
}

func oryPolisDirectoryUserList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/users", directoryQuery(values), nil)
	return namedResult("users", result, err)
}

func oryPolisDirectoryUserGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "user_id", "userId", "id")
	if err != nil {
		return nil, err
	}
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/users/"+url.PathEscape(id), directoryQuery(values), nil)
	return namedResult("user", result, err)
}

func oryPolisDirectoryGroupList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/groups", directoryQuery(values), nil)
	return namedResult("groups", result, err)
}

func oryPolisDirectoryGroupGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "group_id", "groupId", "id")
	if err != nil {
		return nil, err
	}
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/groups/"+url.PathEscape(id), directoryQuery(values), nil)
	return namedResult("group", result, err)
}

func oryPolisDirectoryGroupMemberList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	id, err := requiredID(values, "group_id", "groupId", "id")
	if err != nil {
		return nil, err
	}
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/groups/"+url.PathEscape(id)+"/members", directoryQuery(values), nil)
	return namedResult("members", result, err)
}

func oryPolisSSOSetupLinkCreate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodPost, "/api/v1/sso/setuplinks", nil, requestBody(values, "setup_link"))
	return namedResult("setup_link", result, err)
}

func oryPolisSSOSetupLinkGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/sso/setuplinks", setupLinkQuery(values), nil)
	return namedResult("setup_link", result, err)
}

func oryPolisSSOSetupLinkDelete(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodDelete, "/api/v1/sso/setuplinks", setupLinkQuery(values), nil)
	return namedResult("deleted", result, err)
}

func oryPolisDirectorySetupLinkCreate(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodPost, "/api/v1/dsync/setuplinks", nil, requestBody(values, "setup_link"))
	return namedResult("setup_link", result, err)
}

func oryPolisDirectorySetupLinkGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/dsync/setuplinks", setupLinkQuery(values), nil)
	return namedResult("setup_link", result, err)
}

func oryPolisDirectorySetupLinkDelete(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodDelete, "/api/v1/dsync/setuplinks", setupLinkQuery(values), nil)
	return namedResult("deleted", result, err)
}

func oryPolisIdentityFederationGet(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/identity-federation", tenantProductQuery(values), nil)
	return namedResult("identity_federation", result, err)
}

func oryPolisIdentityFederationSSOList(ctx context.Context, client *OryPolisClient, values map[string]any) (map[string]any, error) {
	result, err := client.HTTP.do(ctx, http.MethodGet, "/api/v1/identity-federation/product", tenantProductQuery(values), nil)
	return namedResult("identity_federation_sso", result, err)
}

func requestBody(values map[string]any, key string) map[string]any {
	if payload := mapValue(values, key); payload != nil {
		return payload
	}
	return values
}

func ssoQuery(values map[string]any) map[string]string {
	query := tenantProductQuery(values)
	query["clientID"] = firstNonEmpty(values, "client_id", "clientID", "clientId")
	query["clientSecret"] = firstNonEmpty(values, "client_secret", "clientSecret")
	query["strategy"] = stringValue(values, "strategy")
	query["sort"] = stringValue(values, "sort")
	return query
}

func directoryQuery(values map[string]any) map[string]string {
	query := tenantProductQuery(values)
	query["directoryId"] = firstNonEmpty(values, "directory_id", "directoryId")
	query["pageOffset"] = firstNonEmpty(values, "page_offset", "pageOffset")
	query["pageLimit"] = firstNonEmpty(values, "page_limit", "pageLimit")
	query["pageToken"] = firstNonEmpty(values, "page_token", "pageToken")
	return query
}

func setupLinkQuery(values map[string]any) map[string]string {
	query := tenantProductQuery(values)
	query["token"] = stringValue(values, "token")
	query["id"] = stringValue(values, "id")
	return query
}

func tenantProductQuery(values map[string]any) map[string]string {
	return map[string]string{
		"tenant":  stringValue(values, "tenant"),
		"product": stringValue(values, "product"),
	}
}

func requiredID(values map[string]any, keys ...string) (string, error) {
	id := firstNonEmpty(values, keys...)
	if id == "" {
		return "", fmt.Errorf("%s is required", keys[0])
	}
	return id, nil
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func namedResult(key string, value any, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	return map[string]any{key: value}, nil
}
