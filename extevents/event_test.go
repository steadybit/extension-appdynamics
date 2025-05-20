// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extevents

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-appdynamics/config"
)

// --- parseBodyToEventRequestBody ---

func TestParseBodyToEventRequestBody_Success(t *testing.T) {
	id := uuid.New()
	input := `{
		"Id":"` + id.String() + `",
		"EventName":"MyEvent",
		"Environment":{"Name":"env1"},
		"Tenant":{"Name":"ten1","Key":"k1"}
	}`
	ev, err := parseBodyToEventRequestBody([]byte(input))
	assert.NoError(t, err)
	assert.Equal(t, id, ev.Id)
	assert.Equal(t, "MyEvent", ev.EventName)
	assert.Equal(t, "env1", ev.Environment.Name)
	assert.Equal(t, "ten1", ev.Tenant.Name)
	assert.Equal(t, "k1", ev.Tenant.Key)
}

func TestParseBodyToEventRequestBody_Error(t *testing.T) {
	_, err := parseBodyToEventRequestBody([]byte(`{invalid json]`))
	assert.Error(t, err)
}

// --- buildOrderedQueryString ---

func TestBuildOrderedQueryString_Success(t *testing.T) {
	kvs := []KeyValue{
		{"eventtype", "CUSTOM"},
		{"customeventtype", "Steadybit"},
		{"severity", "info"},
		{"summary", "Summ"},
		{"propertyvalues", "v1"},
		{"propertynames", "n1"},
	}
	qs, err := buildOrderedQueryString(kvs)
	assert.NoError(t, err)
	parts := strings.Split(qs, "&")
	// first four are the special keys in the right order:
	assert.Equal(t, "customeventtype="+url.QueryEscape("Steadybit"), parts[0])
	assert.Equal(t, "eventtype="+url.QueryEscape("CUSTOM"), parts[1])
	assert.Equal(t, "severity="+url.QueryEscape("info"), parts[2])
	assert.Equal(t, "summary="+url.QueryEscape("Summ"), parts[3])
	// then the propertynames/value
	assert.Contains(t, parts[4:], "propertynames="+url.QueryEscape("n1"))
	assert.Contains(t, parts[4:], "propertyvalues="+url.QueryEscape("v1"))
}

func TestBuildOrderedQueryString_Mismatch(t *testing.T) {
	// one propertynames, zero values
	_, err := buildOrderedQueryString([]KeyValue{
		{"propertynames", "n1"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatched propertynames")
}

// --- getEventBaseTags ---

func TestGetEventBaseTags_WithoutTeam(t *testing.T) {
	id := uuid.New()
	ev := event_kit_api.EventRequestBody{
		Id:          id,
		EventName:   "Ev",
		Environment: &event_kit_api.Environment{Name: "E"},
		Tenant:      event_kit_api.Tenant{Name: "T", Key: "k"},
		Team:        nil,
	}
	tags := getEventBaseTags(ev)
	// should start with the four special keys
	assert.Equal(t, specialKeysOrder[0], tags[0].Key)
	assert.Equal(t, "Steadybit", tags[0].Value)
	assert.Equal(t, "summary", tags[3].Key)
	assert.Contains(t, tags[3].Value, ev.Id.String())
	// should contain Env and Tenant
	foundEnv := false
	foundTen := false
	for _, kv := range tags {
		if kv.Key == "propertyvalues" && kv.Value == "E" {
			foundEnv = true
		}
		if kv.Key == "propertyvalues" && strings.HasPrefix(kv.Value, "T(") {
			foundTen = true
		}
	}
	assert.True(t, foundEnv)
	assert.True(t, foundTen)
}

func TestGetEventBaseTags_WithTeam(t *testing.T) {
	ev := event_kit_api.EventRequestBody{
		Id:          uuid.New(),
		EventName:   "E2",
		Environment: &event_kit_api.Environment{Name: "Env"},
		Tenant:      event_kit_api.Tenant{Name: "Ten", Key: "k2"},
		Team:        &event_kit_api.Team{Name: "Team", Key: "tk"},
	}
	tags := getEventBaseTags(ev)
	// should include Team entries
	found := false
	for i := range tags {
		if tags[i].Key == "propertyvalues" && tags[i].Value == "Team(tk)" {
			found = true
		}
	}
	assert.True(t, found, "expected Team to appear in tags")
}

// --- getExecutionTags ---

func TestGetExecutionTags_NilExecution(t *testing.T) {
	ev := event_kit_api.EventRequestBody{ExperimentExecution: nil}
	tags := getExecutionTags(ev)
	assert.Empty(t, tags)
}

func TestGetExecutionTags_FullExecution(t *testing.T) {
	start := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	ev := event_kit_api.EventRequestBody{
		ExperimentExecution: &event_kit_api.ExperimentExecution{
			ExecutionId:   42,
			ExperimentKey: "expkey",
			Name:          "expname",
			StartedTime:   start,
			EndedTime:     &end,
		},
	}
	tags := getExecutionTags(ev)

	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "exec_id"})
	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "exp_key"})
	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "exp_name"})
}

// --- getStepTags ---

func TestGetStepTags_AllFields(t *testing.T) {
	id := uuid.New()
	exec := event_kit_api.ExperimentStepExecution{
		Id:            id,
		ExecutionId:   55,
		ExperimentKey: "ek",
		Type:          event_kit_api.Action,
		ActionId:      ptrString("aid"),
		ActionName:    ptrString("aname"),
		CustomLabel:   ptrString("label"),
	}
	tags := getStepTags(exec)

	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "step_name"})
	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "step_label"})
}

func ptrString(s string) *string { return &s }

// --- getTargetTags ---

func TestGetTargetTags_WithTimes(t *testing.T) {
	start := time.Date(2025, 5, 2, 8, 0, 0, 0, time.UTC)
	end := time.Date(2025, 5, 2, 9, 0, 0, 0, time.UTC)
	tgt := event_kit_api.ExperimentStepTargetExecution{
		ExecutionId:   77,
		ExperimentKey: "tk",
		State:         event_kit_api.Completed,
		StartedTime:   &start,
		EndedTime:     &end,
	}
	tags := getTargetTags(tgt)
	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "execution_id"})
	assert.Contains(t, tags, KeyValue{Key: "propertynames", Value: "execution_key"})
}

// --- getTargetProperties ---

func TestGetTargetProperties_AllAttributes(t *testing.T) {
	attrs := map[string][]string{
		"k8s.cluster-name":      {"clus"},
		"k8s.namespace":         {"ns"},
		"container.host":        {"host1"},
		"host.hostname":         {"hhn"},
		"application.hostname":  {"apphn"},
		"container.id.stripped": {"cid"},
		"aws.region":            {"reg"},
		"aws.zone":              {"zone"},
		"aws.account":           {"acc"},
	}
	tgt := event_kit_api.ExperimentStepTargetExecution{
		ExperimentKey:    "exk",
		ExecutionId:      1,
		State:            event_kit_api.Created,
		TargetAttributes: attrs,
	}
	tags := getTargetProperties(tgt)
	// should include cloud.provider=aws
	foundCloud := false
	for _, kv := range tags {
		if kv.Key == "propertyvalues" && kv.Value == "aws" {
			foundCloud = true
		}
	}
	assert.True(t, foundCloud, "should include AWS cloud.provider")
}

// --- onExperiment, onExperimentCompleted, onExperimentStep, onExperimentTarget ---

func TestOnExperiment(t *testing.T) {
	id := uuid.New()
	ev := event_kit_api.EventRequestBody{
		Id:          id,
		EventName:   "X",
		Environment: &event_kit_api.Environment{Name: "E"},
		Tenant:      event_kit_api.Tenant{Name: "T", Key: "k"},
	}
	tags, err := onExperiment(ev)
	assert.NoError(t, err)
	assert.Equal(t, getEventBaseTags(ev), tags)
}

func TestOnExperimentStep_StoresAndReturns(t *testing.T) {
	stepExecutions = sync.Map{} // clear
	id := uuid.New()
	exec := event_kit_api.ExperimentStepExecution{
		Id:            id,
		ExecutionId:   10,
		ExperimentKey: "ek",
	}
	ev := event_kit_api.EventRequestBody{
		Id:                      uuid.New(),
		EventName:               "S",
		Environment:             &event_kit_api.Environment{Name: "E"},
		Tenant:                  event_kit_api.Tenant{Name: "T", Key: "k"},
		ExperimentStepExecution: &exec,
	}
	tags, err := onExperimentStep(ev)
	assert.NoError(t, err)
	// should have stored
	v, ok := stepExecutions.Load(id)
	assert.True(t, ok)
	assert.Equal(t, exec, v.(event_kit_api.ExperimentStepExecution))
	// tags should include step_exec_id
	found := false
	for _, kv := range tags {
		if kv.Key == "propertyvalues" && kv.Value == "10" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestOnExperimentTarget_Various(t *testing.T) {
	id := uuid.New()
	stepExecutions = sync.Map{}
	ev := event_kit_api.EventRequestBody{
		Id:          uuid.New(),
		EventName:   "S",
		Environment: &event_kit_api.Environment{Name: "E"},
		Tenant:      event_kit_api.Tenant{Name: "T", Key: "k"},
	}
	// nil target → nil
	tags, err := onExperimentTarget(ev)
	assert.NoError(t, err)
	assert.Nil(t, tags)

	// no stored step → nil
	tags, err = onExperimentTarget(ev)
	assert.NoError(t, err)
	assert.Nil(t, tags)

	// stored but wrong kind → nil
	actionKind := event_kit_api.Attack
	step := event_kit_api.ExperimentStepExecution{ActionKind: &actionKind}
	stepExecutions.Store(id, step)
	ev.ExperimentStepTargetExecution = &event_kit_api.ExperimentStepTargetExecution{}
	ev.ExperimentStepTargetExecution.State = event_kit_api.Completed
	tags, err = onExperimentTarget(ev)
	assert.NoError(t, err)
	assert.Nil(t, tags)

	// stored and Attack → some tags
	stepExecutions.Store(id, step)
	ev.ExperimentStepTargetExecution.StepExecutionId = id
	tags, err = onExperimentTarget(ev)
	assert.NoError(t, err)
	assert.True(t, len(tags) > 0)
}

// --- handlePostEvent ---

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newResty(fn roundTripperFunc) *resty.Client {
	return resty.NewWithClient(&http.Client{Transport: fn})
}

func TestHandlePostEvent_Success(t *testing.T) {
	config.Config.EventApplicationID = "theApp"
	called := false
	var capturedURL *url.URL

	RestyClient = newResty(func(req *http.Request) (*http.Response, error) {
		called = true
		capturedURL = req.URL
		// return OK
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	})

	// simple kv
	kv := []KeyValue{{Key: "customeventtype", Value: "Steadybit"}}
	handlePostEvent(context.Background(), RestyClient, kv)
	assert.True(t, called)
	assert.Equal(t, "/controller/rest/applications/theApp/events", capturedURL.Path)
	assert.Equal(t, "customeventtype=Steadybit", capturedURL.RawQuery)
}

func TestHandlePostEvent_BuildQueryError(t *testing.T) {
	// mismatched so buildOrderedQueryString fails
	RestyClient = newResty(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("should not be called when query building fails")
		return nil, errors.New("boom")
	})
	// one propertynames, zero values → error
	handlePostEvent(context.Background(), RestyClient, []KeyValue{{Key: "propertynames", Value: "n1"}})
	// no panic, nothing else to assert
}
