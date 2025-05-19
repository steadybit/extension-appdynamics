package extappdynamics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	actionapitest "github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
)

func TestPrepareSuccess(t *testing.T) {
	a := &ActionSuppressionAction{}
	state := a.NewEmptyState()
	req := actionapitest.PrepareActionRequestBody{
		Target: &actionapitest.Target{Attributes: map[string][]string{
			"appdynamics.application.id": {"app-id-123"},
		}},
		Config: map[string]interface{}{
			"duration":              float64(2000),
			"disableAgentReporting": true,
		},
		ExecutionContext: &actionapitest.ExecutionContext{
			ExperimentUri: extutil.Ptr("exp://example"),
			ExecutionUri:  extutil.Ptr("exec://example"),
		},
	}

	_, err := a.Prepare(context.Background(), &state, req)
	if err != nil {
		t.Fatalf("Prepare returned unexpected error: %v", err)
	}

	if state.ApplicationId != "app-id-123" {
		t.Errorf("expected ApplicationId 'app-id-123', got %q", state.ApplicationId)
	}

	// Check End roughly 2s in the future
	now := time.Now()
	expected := now.Add(2 * time.Second)
	if state.End.Before(expected.Add(-100*time.Millisecond)) || state.End.After(expected.Add(100*time.Millisecond)) {
		t.Errorf("expected End around %v (±100ms), got %v", expected, state.End)
	}

	if !state.DisableAgentReporting {
		t.Errorf("expected DisableAgentReporting true")
	}
}

func TestPrepareMissingApplicationID(t *testing.T) {
	a := &ActionSuppressionAction{}
	state := a.NewEmptyState()
	req := actionapitest.PrepareActionRequestBody{
		Target: &actionapitest.Target{Attributes: map[string][]string{}},
		Config: map[string]interface{}{
			"duration":              float64(1000),
			"disableAgentReporting": false,
		},
	}

	_, err := a.Prepare(context.Background(), &state, req)
	if err == nil {
		t.Fatal("expected error for missing application ID, got nil")
	}
}

func TestActionSuppressionStartSuccess(t *testing.T) {
	// Setup a fake server
	var receivedReq ActionSuppressionRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/controller/alerting/rest/v1/applications/app-123/action-suppressions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 123})
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	old := RestyClient
	RestyClient = client
	defer func() { RestyClient = old }()

	state := ActionSuppressionState{
		ApplicationId:         "app-123",
		DisableAgentReporting: true,
		End:                   time.Now().Add(5 * time.Second),
	}

	res, err := ActionSuppressionStart(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res == nil || res.Messages == nil {
		t.Fatal("expected non-nil StartResult with messages")
	}

	if receivedReq.Name != "Steadybit-app-123" {
		t.Errorf("expected Name 'Steadybit-app-123', got %s", receivedReq.Name)
	}
	if receivedReq.DisableAgentReporting != true {
		t.Errorf("expected DisableAgentReporting true")
	}
}

func TestActionSuppressionStopNoID(t *testing.T) {
	res, err := ActionSuppressionStop(context.Background(), &ActionSuppressionState{}, RestyClient)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result when ActionSuppressionId is nil, got %v", res)
	}
}

func TestActionSuppressionStopSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/controller/alerting/rest/v1/applications/app-123/action-suppressions/123" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	old := RestyClient
	RestyClient = client
	defer func() { RestyClient = old }()

	state := ActionSuppressionState{
		ApplicationId:       "app-123",
		ActionSuppressionId: extutil.Ptr("123"),
	}

	res, err := ActionSuppressionStop(context.Background(), &state, RestyClient)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res == nil || res.Messages == nil {
		t.Fatal("expected non-nil StopResult with messages")
	}
}
