// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-appdynamics/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func RegisterEventListenerHandlers() {
	exthttp.RegisterHttpHandler("/events/experiment-started", handle(onExperiment))
	exthttp.RegisterHttpHandler("/events/experiment-completed", handle(onExperimentCompleted))
	exthttp.RegisterHttpHandler("/events/experiment-step-started", handle(onExperimentStep))
	exthttp.RegisterHttpHandler("/events/experiment-target-started", handle(onExperimentTarget))
	exthttp.RegisterHttpHandler("/events/experiment-target-completed", handle(onExperimentTarget))
}

// These keys must always come first in the query string
var specialKeysOrder = []string{
	"customeventtype",
	"eventtype",
	"severity",
	"summary",
}

var (
	stepExecutions = sync.Map{}
)

type KeyValue struct {
	Key   string
	Value string
}

var RestyClient *resty.Client

type eventHandler func(event event_kit_api.EventRequestBody) ([]KeyValue, error)

func handle(handler eventHandler) func(w http.ResponseWriter, r *http.Request, body []byte) {
	return func(w http.ResponseWriter, r *http.Request, body []byte) {

		event, err := parseBodyToEventRequestBody(body)
		if err != nil {
			exthttp.WriteError(w, extension_kit.ToError("Failed to decode event request body", err))
			return
		}

		if request, err := handler(event); err == nil {
			if request != nil {
				handlePostEvent(r.Context(), RestyClient, request)
			}
		} else {
			exthttp.WriteError(w, extension_kit.ToError(err.Error(), err))
			return
		}

		exthttp.WriteBody(w, "{}")
	}
}

func onExperiment(event event_kit_api.EventRequestBody) ([]KeyValue, error) {
	tags := getEventBaseTags(event)
	tags = append(tags, getExecutionTags(event)...)

	return tags, nil
}

func onExperimentCompleted(event event_kit_api.EventRequestBody) ([]KeyValue, error) {
	stepExecutions.Range(func(key, value interface{}) bool {
		stepExecution := value.(event_kit_api.ExperimentStepExecution)
		if stepExecution.ExecutionId == event.ExperimentExecution.ExecutionId {
			log.Debug().Msgf("Delete step execution data for id %.0f", stepExecution.ExecutionId)
			stepExecutions.Delete(key)
		}
		return true
	})

	return onExperiment(event)
}

func onExperimentStep(event event_kit_api.EventRequestBody) ([]KeyValue, error) {
	tags := getEventBaseTags(event)
	tags = append(tags, getExecutionTags(event)...)
	tags = append(tags, getStepTags(*event.ExperimentStepExecution)...)

	stepExecutions.Store(event.ExperimentStepExecution.Id, *event.ExperimentStepExecution)

	return tags, nil
}

func getEventBaseTags(event event_kit_api.EventRequestBody) []KeyValue {
	tags := make([]KeyValue, 0)
	tags = append(tags, KeyValue{Key: "customeventtype", Value: "Steadybit"})
	tags = append(tags, KeyValue{Key: "eventtype", Value: "CUSTOM"})
	tags = append(tags, KeyValue{Key: "severity", Value: "info"})
	tags = append(tags, KeyValue{Key: "summary", Value: "Steadybit Event " + event.Id.String() + " " + event.EventName})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "Environment"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.Environment.Name})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "Tenant"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.Tenant.Name + "(" + event.Tenant.Key + ")"})
	if event.Team != nil {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "Team"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.Team.Name + "(" + event.Team.Key + ")"})
	}

	return tags
}

func getExecutionTags(event event_kit_api.EventRequestBody) []KeyValue {
	tags := make([]KeyValue, 0)
	if event.ExperimentExecution == nil {
		return tags
	}
	tags = append(tags, KeyValue{Key: "propertynames", Value: "exec_id"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: fmt.Sprintf("%g", event.ExperimentExecution.ExecutionId)})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "exp_key"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.ExperimentExecution.ExperimentKey})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "exp_name"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.ExperimentExecution.Name})

	if event.ExperimentExecution.StartedTime.IsZero() {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "started_time"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: time.Now().Format(time.RFC3339)})
	} else {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "started_time"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.ExperimentExecution.StartedTime.Format(time.RFC3339)})
	}

	if event.ExperimentExecution.EndedTime != nil && !(*event.ExperimentExecution.EndedTime).IsZero() {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "ended_time"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: event.ExperimentExecution.EndedTime.Format(time.RFC3339)})
	}

	return tags
}

func getStepTags(step event_kit_api.ExperimentStepExecution) []KeyValue {
	tags := make([]KeyValue, 0)

	if step.Type == event_kit_api.Action {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "step_action_id"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: *step.ActionId})
	}
	if step.ActionName != nil {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "step_name"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: *step.ActionName})
	}
	if step.CustomLabel != nil {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "step_label"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: *step.CustomLabel})
	}
	tags = append(tags, KeyValue{Key: "propertynames", Value: "step_exec_id"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: fmt.Sprintf("%.0f", step.ExecutionId)})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "step_exp_key"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: step.ExperimentKey})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "step_id"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: step.Id.String()})

	return tags
}

func getTargetTags(target event_kit_api.ExperimentStepTargetExecution) []KeyValue {
	tags := make([]KeyValue, 0)

	tags = append(tags, KeyValue{Key: "propertynames", Value: "execution_id"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: fmt.Sprintf("%.0f", target.ExecutionId)})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "execution_key"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: target.ExperimentKey})
	tags = append(tags, KeyValue{Key: "propertynames", Value: "execution_state"})
	tags = append(tags, KeyValue{Key: "propertyvalues", Value: string(target.State)})

	if target.StartedTime != nil {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "started_time"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: target.StartedTime.Format(time.RFC3339)})
	}

	if target.EndedTime != nil {
		tags = append(tags, KeyValue{Key: "propertynames", Value: "ended_time"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: target.EndedTime.Format(time.RFC3339)})
	}

	return tags
}

func getTargetProperties(target event_kit_api.ExperimentStepTargetExecution) []KeyValue {
	tags := make([]KeyValue, 0)
	const clusterNameSteadybitAttribute = "k8s.cluster-name"

	if _, ok := target.TargetAttributes[clusterNameSteadybitAttribute]; ok {
		getTargetAttributeToKeyValue(tags, target, clusterNameSteadybitAttribute)
		getTargetAttributeToKeyValue(tags, target, "k8s.namespace")
		getTargetAttributeToKeyValue(tags, target, "k8s.deployment")
		getTargetAttributeToKeyValue(tags, target, "k8s.pod.name")
		getTargetAttributeToKeyValue(tags, target, "k8s.container.name")
	}

	getTargetAttributeToKeyValue(tags, target, "container.host")
	getTargetAttributeToKeyValue(tags, target, "container.host")
	getTargetAttributeToKeyValue(tags, target, "host.hostname")
	getTargetAttributeToKeyValue(tags, target, "application.hostname")

	getTargetAttributeToKeyValue(tags, target, "container.id.stripped")

	if _, ok := target.TargetAttributes["aws.region"]; ok {
		//AWS tags
		tags = append(tags, KeyValue{Key: "propertynames", Value: "cloud.provider"})
		tags = append(tags, KeyValue{Key: "propertyvalues", Value: "aws"})
		getTargetAttributeToKeyValue(tags, target, "aws.region")
		getTargetAttributeToKeyValue(tags, target, "aws.zone")
		getTargetAttributeToKeyValue(tags, target, "aws.account")
	}

	return tags
}

func getTargetAttributeToKeyValue(tags []KeyValue, target event_kit_api.ExperimentStepTargetExecution, steadybitAttribute string) {
	if values, ok := target.TargetAttributes[steadybitAttribute]; ok {
		if (len(values)) == 1 {
			tags = append(tags, KeyValue{Key: "propertynames", Value: steadybitAttribute})
			tags = append(tags, KeyValue{Key: "propertyvalues", Value: values[0]})
		}
	}
}

func parseBodyToEventRequestBody(body []byte) (event_kit_api.EventRequestBody, error) {
	var event event_kit_api.EventRequestBody
	err := json.Unmarshal(body, &event)
	return event, err
}

func handlePostEvent(ctx context.Context, client *resty.Client, queryParameters []KeyValue) {
	query, err := buildOrderedQueryString(queryParameters)
	if err != nil {
		log.Err(err).Msgf("Failed to create query string for the custom event: %v", err)
		return
	}
	req := client.R().
		SetContext(ctx).
		SetQueryString(query)

	res, err := req.
		Post("/controller/rest/applications/" + config.Config.EventApplicationID + "/events")

	if err != nil {
		log.Err(err).Msgf("Failed to post custom event. Full response: %v", res.String())
		return
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("AppDynamics API responded with unexpected status code %d while posting events. Full response: %v", res.StatusCode(), res.String())
	}
}

func onExperimentTarget(event event_kit_api.EventRequestBody) ([]KeyValue, error) {
	if event.ExperimentStepTargetExecution == nil {
		return nil, nil
	}

	var v, ok = stepExecutions.Load(event.ExperimentStepTargetExecution.StepExecutionId)
	if !ok {
		log.Warn().Msgf("Could not find step infos for step execution id %s", event.ExperimentStepTargetExecution.StepExecutionId)
		return nil, nil
	}
	stepExecution := v.(event_kit_api.ExperimentStepExecution)

	if stepExecution.ActionKind != nil && *stepExecution.ActionKind == event_kit_api.Attack {
		tags := getEventBaseTags(event)
		tags = append(tags, getExecutionTags(event)...)
		tags = append(tags, getTargetTags(*event.ExperimentStepTargetExecution)...)
		tags = append(tags, getTargetProperties(*event.ExperimentStepTargetExecution)...)

		return tags, nil
	}

	return nil, nil
}

func buildOrderedQueryString(kvs []KeyValue) (string, error) {
	var parts []string
	seenSpecial := make(map[string]bool)

	// First: append special keys in the defined order
	for _, specialKey := range specialKeysOrder {
		for _, kv := range kvs {
			if kv.Key == specialKey && !seenSpecial[kv.Key] {
				parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(kv.Key), url.QueryEscape(kv.Value)))
				seenSpecial[kv.Key] = true
			}
		}
	}

	// Then: collect and append propertynames and propertyvalues in order
	var propertyNames, propertyValues []string
	for _, kv := range kvs {
		switch kv.Key {
		case "propertynames":
			propertyNames = append(propertyNames, kv.Value)
		case "propertyvalues":
			propertyValues = append(propertyValues, kv.Value)
		}
	}

	if len(propertyNames) != len(propertyValues) {
		return "", fmt.Errorf("mismatched propertynames (%d) and propertyvalues (%d)", len(propertyNames), len(propertyValues))
	}

	for _, name := range propertyNames {
		parts = append(parts, "propertynames="+url.QueryEscape(name))
	}

	for _, value := range propertyValues {
		parts = append(parts, "propertyvalues="+url.QueryEscape(value))
	}

	return strings.Join(parts, "&"), nil
}
