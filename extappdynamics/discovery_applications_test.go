// discovery_applications_test.go
package extappdynamics

import (
	"context"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// helper to create a Resty client pointing at our test server
func newTestClient(ts *httptest.Server) *resty.Client {
	return resty.New().SetBaseURL(ts.URL)
}

func TestGetAllApplications_Success(t *testing.T) {
	// prepare a fake AppDynamics JSON payload
	apps := []Application{{ID: 123, Name: "TestApp", Description: "Test Desc", AccountGUID: "GUID-123"}}
	appsbytes, err := json.Marshal(apps)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	// spin up a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/controller/rest/applications" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(appsbytes)
	}))
	defer ts.Close()

	client := resty.New().SetBaseURL(ts.URL).SetHeader("Accept", "application/json")
	RestyClient = client

	appDiscovery := &applicationDiscovery{}
	targets, err := appDiscovery.DiscoverTargets(context.Background())
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}

	got := targets[0]
	if got.Id != "123" {
		t.Errorf("expected Id=\"123\", got %q", got.Id)
	}
	if got.Label != "TestApp" {
		t.Errorf("expected Label=\"TestApp\", got %q", got.Label)
	}
	attrs := got.Attributes
	if attrs[AppAttribute+".description"][0] != "Test Desc" {
		t.Errorf("expected description=\"Test Desc\", got %q", attrs[AppAttribute+".description"][0])
	}
	if attrs[AppAttribute+".account_guid"][0] != "GUID-123" {
		t.Errorf("expected account_guid=\"GUID-123\", got %q", attrs[AppAttribute+".account_guid"][0])
	}
}

func TestGetAllApplications_Non200(t *testing.T) {
	// server returns 500 Internal Server Error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	targets := getAllApplications(context.Background(), client)

	if len(targets) != 0 {
		t.Fatalf("expected 0 targets on non-200, got %d", len(targets))
	}
}

func TestApplicationDiscovery_Describe(t *testing.T) {
	var d applicationDiscovery
	desc := d.Describe()
	if desc.Id != applicationTargetType {
		t.Errorf("Describe().Id = %q; want %q", desc.Id, applicationTargetType)
	}
	if desc.Discover.CallInterval == nil || *desc.Discover.CallInterval != "1m" {
		t.Errorf("Discover.CallInterval = %v; want \"1m\"", desc.Discover.CallInterval)
	}
}

func TestApplicationDiscovery_DescribeTarget(t *testing.T) {
	var d applicationDiscovery
	td := d.DescribeTarget()
	if td.Id != applicationTargetType {
		t.Errorf("DescribeTarget().Id = %q; want %q", td.Id, applicationTargetType)
	}
	if td.Label.One != "AppDynamics application" || td.Label.Other != "AppDynamics applications" {
		t.Errorf("unexpected Label: %+v", td.Label)
	}
	if td.Category == nil || *td.Category != "monitoring" {
		t.Errorf("expected Category=\"monitoring\", got %v", td.Category)
	}
	if len(td.Table.Columns) != 4 {
		t.Errorf("expected 4 table columns, got %d", len(td.Table.Columns))
	}
}

func TestApplicationDiscovery_DescribeAttributes(t *testing.T) {
	var d applicationDiscovery
	attrs := d.DescribeAttributes()
	want := []string{
		AppAttribute + ".name",
		AppAttribute + ".id",
		AppAttribute + ".description",
		AppAttribute + ".account_guid",
	}
	if len(attrs) != len(want) {
		t.Fatalf("DescribeAttributes() len = %d; want %d", len(attrs), len(want))
	}
	for i, a := range attrs {
		if a.Attribute != want[i] {
			t.Errorf("Attribute[%d] = %q; want %q", i, a.Attribute, want[i])
		}
	}
}
