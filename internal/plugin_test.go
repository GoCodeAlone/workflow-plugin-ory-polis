package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-ory-polis/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestModuleInitRegistersPolisClient(t *testing.T) {
	module, err := newOryPolisModule("ory_polis-test", map[string]any{
		"baseUrl": "https://polis.example.test",
		"apiKey":  "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	client, ok := GetClient("ory_polis-test")
	if !ok || client == nil || client.HTTP == nil {
		t.Fatal("expected registered client")
	}
	if client.BaseURL != "https://polis.example.test" {
		t.Fatalf("base url = %q", client.BaseURL)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetClient("ory_polis-test"); ok {
		t.Fatal("expected client to be unregistered")
	}
}

func TestModuleInitRequiresBaseURL(t *testing.T) {
	module, err := newOryPolisModule("ory_polis-test", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing base url error")
	}
}

func TestContractRegistryIncludesStrictProtoDescriptors(t *testing.T) {
	provider := NewOryPolisPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	registry := provider.ContractRegistry()
	if registry == nil || registry.GetFileDescriptorSet() == nil {
		t.Fatal("missing contract registry file descriptors")
	}
	contractsByType := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.GetContracts() {
		switch contract.GetKind() {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByType["module:"+contract.GetModuleType()] = contract
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByType["step:"+contract.GetStepType()] = contract
		}
	}
	module := contractsByType["module:ory.polis"]
	if module == nil || module.GetConfigMessage() != "workflow.plugins.ory_polis.v1.ProviderConfig" {
		t.Fatalf("unexpected module contract: %#v", module)
	}
	for _, stepType := range allStepTypes() {
		contract := contractsByType["step:"+stepType]
		if contract == nil {
			t.Fatalf("missing step contract %s", stepType)
		}
		if contract.GetMode() != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v", stepType, contract.GetMode())
		}
	}
}

func TestDescriptorAdvertisesOnlyPolisCapabilities(t *testing.T) {
	step, err := newAuthProviderDescribeStep("describe", map[string]any{"baseUrl": "https://polis.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := result.Output["providers"].([]map[string]any)[0]
	categories := stringSet(provider["categories"].([]string))
	for _, category := range []string{"enterprise_sso", "directory_sync"} {
		if !categories[category] {
			t.Fatalf("missing category %q", category)
		}
	}
	for _, absent := range []string{"identity_management", "authentication_method", "oauth2_oidc", "rbac", "mfa"} {
		if categories[absent] {
			t.Fatalf("descriptor must not advertise %s", absent)
		}
	}
	capabilities := provider["capabilities"].([]map[string]any)
	if len(capabilities) != 6 {
		t.Fatalf("capability count = %d", len(capabilities))
	}
	for _, capability := range capabilities {
		if capability["supported"] != true {
			t.Fatalf("%s supported = %#v", capability["key"], capability["supported"])
		}
	}
}

func TestTypedDescriptor(t *testing.T) {
	result, err := typedAuthProviderDescribe(context.Background(), sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]{
		Config: &contracts.AuthProviderDescribeConfig{ProviderId: "ory_polis-admin"},
		Input:  &contracts.AuthProviderDescribeInput{BaseUrl: "https://polis.example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == nil || len(result.Output.GetProviders()) != 1 {
		t.Fatalf("providers = %#v", result.Output)
	}
	if result.Output.GetProviders()[0].GetId() != "ory_polis-admin" {
		t.Fatalf("provider id = %q", result.Output.GetProviders()[0].GetId())
	}
}

func TestSSOConnectionCreateUsesPolisAPI(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/sso" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		body := map[string]any{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["tenant"] != "acme" || body["product"] != "app" {
			t.Fatalf("body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"clientID":"client-1","clientSecret":"secret-1","tenant":"acme","product":"app"}`))
	}))
	defer server.Close()
	module := initTestModule(t, server.URL)
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_polis_sso_connection_create", "create", map[string]any{"module": "ory_polis-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{
		"tenant":             "acme",
		"product":            "app",
		"defaultRedirectUrl": "https://app.example/callback",
		"redirectUrl":        []any{"https://app.example/callback"},
		"rawMetadata":        "<xml/>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["connection"].(map[string]any)["clientID"] != "client-1" {
		t.Fatalf("output = %#v", result.Output)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("authorization header = %q", gotAuth)
	}
}

func TestSSOConnectionListForwardsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/sso" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("tenant") != "acme" || r.URL.Query().Get("product") != "app" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"clientID":"client-1","tenant":"acme","product":"app"}]`))
	}))
	defer server.Close()
	module := initTestModule(t, server.URL)
	defer module.Stop(context.Background())

	step, err := createStep("step.ory_polis_sso_connection_list", "list", map[string]any{"module": "ory_polis-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"tenant": "acme", "product": "app"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["connections"].([]any)[0].(map[string]any)["clientID"] != "client-1" {
		t.Fatalf("output = %#v", result.Output)
	}
}

func TestDirectoryConnectionAndResourcesUsePolisAPI(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/dsync":
			seen["create"] = true
			_, _ = w.Write([]byte(`{"id":"dir-1","tenant":"acme","product":"app","type":"okta-scim-v2"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dsync/dir-1":
			seen["get"] = true
			_, _ = w.Write([]byte(`{"id":"dir-1","tenant":"acme","product":"app"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dsync/events":
			seen["events"] = true
			_, _ = w.Write([]byte(`{"data":[{"id":"event-1","directory_id":"dir-1"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dsync/users":
			seen["users"] = true
			_, _ = w.Write([]byte(`{"data":[{"id":"user-1","email":"user@example.com"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dsync/groups/group-1/members":
			seen["members"] = true
			_, _ = w.Write([]byte(`{"data":[{"id":"user-1"}]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	module := initTestModule(t, server.URL)
	defer module.Stop(context.Background())

	executeStep(t, "step.ory_polis_directory_create", map[string]any{"tenant": "acme", "product": "app", "type": "okta-scim-v2"})
	executeStep(t, "step.ory_polis_directory_get", map[string]any{"directory_id": "dir-1"})
	executeStep(t, "step.ory_polis_directory_event_list", map[string]any{"directory_id": "dir-1"})
	executeStep(t, "step.ory_polis_directory_user_list", map[string]any{"directory_id": "dir-1"})
	executeStep(t, "step.ory_polis_directory_group_member_list", map[string]any{"directory_id": "dir-1", "group_id": "group-1"})
	for _, key := range []string{"create", "get", "events", "users", "members"} {
		if !seen[key] {
			t.Fatalf("missing request %s", key)
		}
	}
}

func TestSetupLinkAndFederationStepsUsePolisAPI(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sso/setuplinks":
			seen["sso_link"] = true
			_, _ = w.Write([]byte(`{"id":"link-1","tenant":"acme","product":"app"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dsync/setuplinks":
			seen["directory_link"] = true
			_, _ = w.Write([]byte(`{"id":"link-2","tenant":"acme","product":"app"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/identity-federation":
			seen["federation"] = true
			_, _ = w.Write([]byte(`{"tenant":"acme","product":"app","connections":[]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	module := initTestModule(t, server.URL)
	defer module.Stop(context.Background())

	executeStep(t, "step.ory_polis_sso_setup_link_create", map[string]any{"tenant": "acme", "product": "app"})
	executeStep(t, "step.ory_polis_directory_setup_link_get", map[string]any{"tenant": "acme", "product": "app"})
	executeStep(t, "step.ory_polis_identity_federation_get", map[string]any{"tenant": "acme", "product": "app"})
	for _, key := range []string{"sso_link", "directory_link", "federation"} {
		if !seen[key] {
			t.Fatalf("missing request %s", key)
		}
	}
}

func TestMissingClientReturnsStepError(t *testing.T) {
	step, err := createStep("step.ory_polis_sso_connection_get", "get", map[string]any{"module": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"tenant": "acme", "product": "app"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] == nil {
		t.Fatalf("expected error output, got %#v", result.Output)
	}
}

func initTestModule(t *testing.T, baseURL string) *oryPolisModule {
	t.Helper()
	module, err := newOryPolisModule("ory_polis-test", map[string]any{"baseUrl": baseURL, "apiKey": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	return module
}

func executeStep(t *testing.T, stepType string, values map[string]any) *sdk.StepResult {
	t.Helper()
	step, err := createStep(stepType, "test", map[string]any{"module": "ory_polis-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, values)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] != nil {
		t.Fatalf("step %s returned error: %#v", stepType, result.Output)
	}
	return result
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
