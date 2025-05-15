/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extappdynamics

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-appdynamics/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"strconv"
	"time"
)

type HealthRuleStateCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[HealthRuleCheckState]           = (*HealthRuleStateCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[HealthRuleCheckState] = (*HealthRuleStateCheckAction)(nil)
)

type HealthRuleCheckState struct {
	HealthRuleId          string
	HealthRuleName        string
	HealthRuleApplication string
	End                   time.Time
	IsViolationExpected   bool
	StateCheckMode        string
	StateCheckSuccess     bool
}

func NewHealthRuleStateCheckAction() action_kit_sdk.Action[HealthRuleCheckState] {
	return &HealthRuleStateCheckAction{}
}

func (m *HealthRuleStateCheckAction) NewEmptyState() HealthRuleCheckState {
	return HealthRuleCheckState{}
}

func (m *HealthRuleStateCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.check", applicationHealthRuleTargetType),
		Label:       "Health Rule Check",
		Description: "Verify if an health rule is observing violations.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(appDynamicsTargetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          applicationHealthRuleTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find health rule by name"),
					Query:       "appdynamics.health-rule.name=\"\"",
				},
			}),
		}),
		Technology:  extutil.Ptr("AppDynamics"),
		Category:    extutil.Ptr("AppDynamics"), //Can be removed in Q1/24 - support for backward compatibility of old sidebar
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "violation",
				Label:        "Is Any Violations Expected?",
				Description:  extutil.Ptr("Does the health rule will observe some violations of critical or warning conditions?"),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(2),
			},
			{
				Name:         "stateCheckMode",
				Label:        "State Check Mode",
				Description:  extutil.Ptr("How often should the state be checked ?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr(stateCheckModeAllTheTime),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "All the time",
						Value: stateCheckModeAllTheTime,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least once",
						Value: stateCheckModeAtLeastOnce,
					},
				}),
				Required: extutil.Ptr(true),
				Order:    extutil.Ptr(3),
			},
		},
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "AppDynamics Health Rule State",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: HealthRuleAttribute + ".id",
				},
				Label: action_kit_api.StateOverTimeWidgetLabelConfig{
					From: HealthRuleAttribute + ".name",
				},
				State: action_kit_api.StateOverTimeWidgetStateConfig{
					From: "state",
				},
				Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
					From: "tooltip",
				},
				Url: extutil.Ptr(action_kit_api.StateOverTimeWidgetUrlConfig{
					From: extutil.Ptr("url"),
				}),
				Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: extutil.Ptr(true),
				}),
			},
		}),
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1s"),
		}),
	}
}

func (m *HealthRuleStateCheckAction) Prepare(_ context.Context, state *HealthRuleCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	now := time.Now()
	HealthRuleId := request.Target.Attributes[HealthRuleAttribute+".id"]
	if len(HealthRuleId) == 0 {
		return nil, extutil.Ptr(extension_kit.ToError("Target is missing the 'appdynamics.health-rule.id' attribute.", nil))
	}
	state.HealthRuleId = HealthRuleId[0]

	duration := request.Config["duration"].(float64)
	end := now.Add(time.Millisecond * time.Duration(duration))

	var expectedViolation bool
	if request.Config["violation"] != nil {
		expectedViolation = extutil.ToBool(request.Config["violation"])
	}

	var stateCheckMode string
	if request.Config["stateCheckMode"] != nil {
		stateCheckMode = fmt.Sprintf("%v", request.Config["stateCheckMode"])
	}

	state.HealthRuleName = request.Target.Attributes["appdynamics.health-rule.name"][0]
	state.HealthRuleApplication = request.Target.Attributes["appdynamics.health-rule.application.id"][0]
	state.End = end
	state.IsViolationExpected = expectedViolation
	state.StateCheckMode = stateCheckMode

	return nil, nil
}

func (m *HealthRuleStateCheckAction) Start(_ context.Context, _ *HealthRuleCheckState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (m *HealthRuleStateCheckAction) Status(ctx context.Context, state *HealthRuleCheckState) (*action_kit_api.StatusResult, error) {
	return HealthRuleCheckStatus(ctx, state, RestyClient)
}

func HealthRuleCheckStatus(ctx context.Context, state *HealthRuleCheckState, client *resty.Client) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	nowStr := strconv.FormatInt(now.UnixMilli(), 10) // base 10
	endStr := strconv.FormatInt(state.End.UnixMilli(), 10)
	completed := time.Now().After(state.End)
	if completed {
		nowStr = strconv.FormatInt(state.End.UnixMilli(), 10)
	}
	var violations []Violation

	uri := "/controller/rest/applications/" + state.HealthRuleApplication + "/problems/healthrule-violations?output=JSON&time-range-type=BETWEEN_TIMES&start-time=" + nowStr + "&end-time=" + endStr
	res, err := client.R().
		SetContext(ctx).
		SetResult(&violations).
		Get(uri)

	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to retrieve health rules from AppDynamics for Application ID %s. Full response: %v", state.HealthRuleApplication, res.String()), err))
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("AppDynamics API responded with unexpected status code %d while retrieving health rule violations for Application ID %s. Full response: %v", res.StatusCode(), state.HealthRuleApplication, res.String())
	}

	var checkError *action_kit_api.ActionKitError
	healthRuleHasViolations := hasViolations(violations, state.HealthRuleName)

	if state.StateCheckMode == stateCheckModeAllTheTime {
		if !state.IsViolationExpected == healthRuleHasViolations {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title: fmt.Sprintf("HealthRule '%s' has violations '%t' whereas 'Violations Expected :%t'.",
					state.HealthRuleName,
					healthRuleHasViolations,
					state.IsViolationExpected),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}
	} else if state.StateCheckMode == stateCheckModeAtLeastOnce {
		if state.IsViolationExpected == healthRuleHasViolations {
			state.StateCheckSuccess = true
		}
		if completed && !state.StateCheckSuccess {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title: fmt.Sprintf("HealthRule '%s' has violations '%t' whereas 'Violations Expected :%t' was expected once.",
					state.HealthRuleName,
					healthRuleHasViolations,
					state.IsViolationExpected),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}
	}

	metrics := []action_kit_api.Metric{
		*toMetric(state.HealthRuleId, state.HealthRuleName, state.HealthRuleApplication, healthRuleHasViolations, now, state.End),
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics:   extutil.Ptr(metrics),
	}, nil
}

func toMetric(HealthRuleID string, HealthRuleName string, AppID string, hasViolations bool, now time.Time, end time.Time) *action_kit_api.Metric {
	var tooltip string
	var state string

	tooltip = fmt.Sprintf("Health rule has violations: %t", hasViolations)
	if !hasViolations {
		state = "success"
	} else {
		state = "danger"
	}

	return extutil.Ptr(action_kit_api.Metric{
		Name: extutil.Ptr("appdynamics_health_rule_state"),
		Metric: map[string]string{
			HealthRuleAttribute + ".id":   HealthRuleID,
			HealthRuleAttribute + ".name": HealthRuleName,
			"state":                       state,
			"tooltip":                     tooltip,
			"url":                         fmt.Sprintf("%s/controller/#/location=ALERT_RESPOND_HEALTH_RULES&timeRange=Custom_Time_Range.BETWEEN_TIMES.%d.%d.120&application=%s", config.Config.ApiBaseUrl, now.UnixMilli(), end.UnixMilli(), AppID),
		},
		Timestamp: now,
		Value:     0,
	})
}

func hasViolations(violations []Violation, healthRuleName string) bool {
	for _, violation := range violations {
		if violation.Name == healthRuleName {
			return true
		}
	}
	return false
}
