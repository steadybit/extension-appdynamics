package extappdynamics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

// TestHasViolations verifies the hasViolations helper.
func TestHasViolations(t *testing.T) {
	violations := []Violation{
		{Name: "foo"},
		{Name: "bar"},
	}
	hasViolationsFoo, _ := hasViolations(violations, "foo")
	hasViolationsBaz, _ := hasViolations(violations, "baz")
	if !hasViolationsFoo {
		t.Error(`expected hasViolations to return true for name "foo"`)
	}
	if hasViolationsBaz {
		t.Error(`expected hasViolations to return false for name "baz"`)
	}
}

// TestHealthRuleCheckStatus_NoViolations_AllTheTime tests AllTheTime mode when no violations occur and none are expected.
func TestHealthRuleCheckStatus_NoViolations_AllTheTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	state := HealthRuleCheckState{
		HealthRuleName:        "foo",
		HealthRuleApplication: "app",
		End:                   time.Now().Add(-time.Second),
		IsViolationExpected:   false,
		StateCheckMode:        StateCheckModeAllTheTime,
	}

	res, err := HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Completed {
		t.Error("expected Completed to be true")
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %v", res.Error)
	}
	if res.Metrics == nil || len(*res.Metrics) == 0 {
		t.Error("expected at least one metric")
	}
	if (*res.Metrics)[0].Metric["state"] != "success" {
		t.Errorf("expected metric state \"success\", got %q", (*res.Metrics)[0].Metric["state"])
	}
}

// TestHealthRuleCheckStatus_ViolationsUnexpected_AllTheTime tests AllTheTime mode when a violation occurs but none are expected.
func TestHealthRuleCheckStatus_ViolationsUnexpected_AllTheTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"name":"foo"}]`))
	}))
	defer ts.Close()

	client := resty.New().SetHostURL(ts.URL)
	state := HealthRuleCheckState{
		HealthRuleName:        "foo",
		HealthRuleApplication: "app",
		End:                   time.Now().Add(-time.Second),
		IsViolationExpected:   false,
		StateCheckMode:        StateCheckModeAllTheTime,
		FailEarly:             true,
	}

	res, err := HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Completed {
		t.Error("expected Completed to be true")
	}
	if res.Error == nil {
		t.Error("expected an error because an unexpected violation occurred")
	}
}

// TestHealthRuleCheckStatus_AllTheTime_FailAtEnd verifies that with fail early disabled the deviation is
// reported only once the step ends, using the past-tense message.
func TestHealthRuleCheckStatus_AllTheTime_FailAtEnd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"name":"foo"}]`))
	}))
	defer ts.Close()

	client := resty.New().SetHostURL(ts.URL)

	// Not yet completed: a deviation is observed but must not fail early.
	state := HealthRuleCheckState{
		HealthRuleName:        "foo",
		HealthRuleApplication: "app",
		End:                   time.Now().Add(time.Minute),
		IsViolationExpected:   false,
		StateCheckMode:        StateCheckModeAllTheTime,
		FailEarly:             false,
	}
	res, err := HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error != nil {
		t.Error("expected no error before the step ends (fail early disabled)")
	}
	if !state.DeviationSeen {
		t.Error("expected the deviation to be remembered")
	}

	// Completed: the remembered deviation is reported with the past-tense message.
	state.End = time.Now().Add(-time.Second)
	res, err = HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error == nil {
		t.Fatal("expected an error at the end of the step")
	}
	if !strings.Contains(res.Error.Title, "had violations") {
		t.Errorf("expected past-tense message, got %q", res.Error.Title)
	}
}

// TestHealthRuleCheckStatus_AtLeastOnce_Success tests AtLeastOnce mode when a violation occurs as expected.
func TestHealthRuleCheckStatus_AtLeastOnce_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"name":"foo"}]`))
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL)
	state := HealthRuleCheckState{
		HealthRuleName:        "foo",
		HealthRuleApplication: "app",
		End:                   time.Now().Add(-time.Second),
		IsViolationExpected:   false,
		StateCheckMode:        StateCheckModeAtLeastOnce,
	}

	res, err := HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Completed {
		t.Error("expected Completed to be true")
	}
	if res.Error != nil {
		t.Errorf("expected no error, got %v", res.Error)
	}
}

// TestHealthRuleCheckStatus_AtLeastOnce_Failure tests AtLeastOnce mode when no violation occurs but one is expected.
func TestHealthRuleCheckStatus_AtLeastOnce_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := resty.New().SetHostURL(ts.URL)
	state := HealthRuleCheckState{
		HealthRuleName:        "foo",
		HealthRuleApplication: "app",
		End:                   time.Now().Add(-time.Second),
		IsViolationExpected:   true,
		StateCheckMode:        StateCheckModeAtLeastOnce,
	}

	res, err := HealthRuleCheckStatus(context.Background(), &state, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Completed {
		t.Error("expected Completed to be true")
	}
	if res.Error == nil {
		t.Error("expected an error due to missing expected violation, got nil")
	}
}
