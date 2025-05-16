// discovery_healthrules_test.go
package extappdynamics

import (
	"context"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

// Test the happy‐path: one application with one health‐rule
func TestGetAllHealthRules_Success(t *testing.T) {
	const appsJSON = `
	[
	  {"ID": 42, "Name": "App42"}
	]`
	const rulesJSON = `
	[
	  {"ID": 100, "Name": "Rule100", "Enabled": true, "AffectedEntityType": "APPLICATION"}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/controller/rest/applications?output=JSON":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(appsJSON))
		case "/controller/alerting/rest/v1/applications/42/health-rules?output=JSON":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(rulesJSON))
		default:
			t.Fatalf("unexpected request URI: %s", r.URL.RequestURI())
		}
	}))
	defer ts.Close()

	client := resty.New().SetHostURL(ts.URL)
	targets := getAllHealthRules(context.Background(), client)

	assert.Len(t, targets, 1)
	hr := targets[0]
	assert.Equal(t, "42-100", hr.Id)
	assert.Equal(t, "Rule100", hr.Label)

	attrs := hr.Attributes
	assert.Equal(t, "Rule100", attrs[HealthRuleAttribute+".name"][0])
	assert.Equal(t, "100", attrs[HealthRuleAttribute+".id"][0])
	assert.Equal(t, "true", attrs[HealthRuleAttribute+AttributeEnabled][0])
	assert.Equal(t, "APPLICATION", attrs[HealthRuleAttribute+AttributeAffectedEntityType][0])
	assert.Equal(t, "42", attrs[HealthRuleAttribute+AttributeAppID][0])
}

// If the applications endpoint fails, we should get zero targets
func TestGetAllHealthRules_ApplicationsNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"oops"}`))
	}))
	defer ts.Close()

	client := resty.New().SetHostURL(ts.URL)
	targets := getAllHealthRules(context.Background(), client)
	assert.Empty(t, targets)
}

// Test Describe()
func TestHealthRuleDiscovery_Describe(t *testing.T) {
	var d healthRuleDiscovery
	desc := d.Describe()
	assert.Equal(t, applicationHealthRuleTargetType, desc.Id)
	assert.NotNil(t, desc.Discover.CallInterval)
	assert.Equal(t, "2m", *desc.Discover.CallInterval)
}

// Test DescribeTarget()
func TestHealthRuleDiscovery_DescribeTarget(t *testing.T) {
	var d healthRuleDiscovery
	td := d.DescribeTarget()

	assert.Equal(t, applicationHealthRuleTargetType, td.Id)
	assert.Equal(t, "AppDynamics health-rule", td.Label.One)
	assert.Equal(t, "AppDynamics health-rules", td.Label.Other)
	assert.NotNil(t, td.Category)
	assert.Equal(t, "monitoring", *td.Category)
	// should list exactly 5 columns
	assert.Len(t, td.Table.Columns, 6)
	// first column should be the rule name
	assert.Equal(t, HealthRuleAttribute+".name", td.Table.Columns[0].Attribute)
	assert.Equal(t, discovery_kit_api.OrderByDirection("ASC"), td.Table.OrderBy[0].Direction)
}

// Test DescribeAttributes()
func TestHealthRuleDiscovery_DescribeAttributes(t *testing.T) {
	var d healthRuleDiscovery
	attrs := d.DescribeAttributes()
	want := []string{
		HealthRuleAttribute + ".name",
		HealthRuleAttribute + ".id",
		HealthRuleAttribute + AttributeEnabled,
		HealthRuleAttribute + AttributeAffectedEntityType,
		HealthRuleAttribute + AttributeAppID,
		HealthRuleAttribute + AttributeAppName,
	}
	var got []string
	for _, a := range attrs {
		got = append(got, a.Attribute)
	}
	assert.Equal(t, want, got)
}
