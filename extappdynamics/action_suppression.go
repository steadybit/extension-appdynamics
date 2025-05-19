// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extappdynamics

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-appdynamics/config"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ActionSuppressionAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ActionSuppressionState]         = (*ActionSuppressionAction)(nil)
	_ action_kit_sdk.ActionWithStop[ActionSuppressionState] = (*ActionSuppressionAction)(nil)
)

type ActionSuppressionState struct {
	ApplicationId         string
	End                   time.Time
	DisableAgentReporting bool
	ActionSuppressionId   *string
	ExperimentUri         *string
	ExecutionUri          *string
}

func NewActionSuppressionAction() action_kit_sdk.Action[ActionSuppressionState] {
	return &ActionSuppressionAction{}
}
func (m *ActionSuppressionAction) NewEmptyState() ActionSuppressionState {
	return ActionSuppressionState{}
}

func (m *ActionSuppressionAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          fmt.Sprintf("%s.action-suppression", applicationTargetType),
		Label:       "Create Action Suppression",
		Description: "Temporarily suspend the automatic trigger of actions and alerts by a policy in response to an event.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(appDynamicsTargetIcon),
		Technology:  extutil.Ptr("AppDynamics"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          applicationTargetType,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "by application name",
					Query: "appdynamics.application.name=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("monitoring"),
		Kind:        action_kit_api.Other,
		TimeControl: action_kit_api.TimeControlExternal,
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
				Name:         "disableAgentReporting",
				Label:        "Disable Agent Metric Reporting",
				Description:  extutil.Ptr("Should the Agents defined in the scope report any metric data during the time window?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("false"),
				Order:        extutil.Ptr(2),
				Required:     extutil.Ptr(true),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func (m *ActionSuppressionAction) Prepare(_ context.Context, state *ActionSuppressionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	applicationID := request.Target.Attributes["appdynamics.application.id"]
	if len(applicationID) == 0 {
		return nil, extension_kit.ToError("Target is missing the 'appdynamics.application.id' tag.", nil)
	}

	duration := request.Config["duration"].(float64)
	end := time.Now().Add(time.Millisecond * time.Duration(duration))

	state.ApplicationId = extutil.ToString(applicationID[0])
	state.End = end
	state.DisableAgentReporting = request.Config["disableAgentReporting"].(bool)

	return nil, nil
}

func (m *ActionSuppressionAction) Start(ctx context.Context, state *ActionSuppressionState) (*action_kit_api.StartResult, error) {
	return ActionSuppressionStart(ctx, state, RestyClient)
}

func (m *ActionSuppressionAction) Stop(ctx context.Context, state *ActionSuppressionState) (*action_kit_api.StopResult, error) {
	return ActionSuppressionStop(ctx, state, RestyClient)
}

func ActionSuppressionStart(ctx context.Context, state *ActionSuppressionState, client *resty.Client) (*action_kit_api.StartResult, error) {
	// Get configured Time Zone if none is defined
	var timezone string
	var err error
	if config.Config.ActionSuppressionTimezone == "" {
		tz, err := GetLocalTimezone()
		if err != nil {
			return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to get current timezone."), err))
		}
		timezone = tz
	} else {
		timezone = config.Config.ActionSuppressionTimezone
	}

	actionSuppressionRequest := ActionSuppressionRequest{
		Name:                    "Steadybit-" + state.ApplicationId + "-" + uuid.New().String(),
		Affects:                 Affects{AffectedInfoType: "APPLICATION"},
		DisableAgentReporting:   state.DisableAgentReporting,
		StartTime:               time.Now().Format(time.RFC3339),
		EndTime:                 state.End.Format(time.RFC3339),
		SuppressionScheduleType: "ONE_TIME",
		Timezone:                timezone,
	}

	var actionSuppressionResponse ActionSuppressionResponse
	res, err := client.R().
		SetContext(ctx).
		SetBody(actionSuppressionRequest).
		SetResult(&ActionSuppressionResponse{}).
		Post("/controller/alerting/rest/v1/applications/" + state.ApplicationId + "/action-suppressions")

	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to create action suppression in AppDynamics for Application ID %s. Full response: %v", state.ApplicationId, res.String()), err))
	}

	if !res.IsSuccess() {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("AppDynamics API responded with unexpected status code %d while creating action suppression for Application ID %s. Full response: %v", res.StatusCode(), state.ApplicationId, res.String()), err))
	}

	return &action_kit_api.StartResult{
		Messages: &action_kit_api.Messages{
			action_kit_api.Message{Level: extutil.Ptr(action_kit_api.Info), Message: fmt.Sprintf("Action Suppression started. (application ID %s, Action Suppression ID %d)", state.ApplicationId, actionSuppressionResponse.ID)},
		},
	}, nil
}

func ActionSuppressionStop(ctx context.Context, state *ActionSuppressionState, client *resty.Client) (*action_kit_api.StopResult, error) {
	if state.ActionSuppressionId == nil {
		return nil, nil
	}

	res, err := client.R().
		SetContext(ctx).
		Delete("/controller/alerting/rest/v1/applications/" + state.ApplicationId + "/action-suppressions/" + *state.ActionSuppressionId)

	if err != nil {
		return nil, extutil.Ptr(extension_kit.ToError(fmt.Sprintf("Failed to delete action suppression in AppDynamics for Application ID %s. Full response: %v", state.ApplicationId, res.String()), err))
	}

	if !res.IsSuccess() {
		log.Err(err).Msgf("AppDynamics API responded with unexpected status code %d while deleting action suppression for Application ID %s. Full response: %v", res.StatusCode(), state.ApplicationId, res.String())
	}

	return &action_kit_api.StopResult{
		Messages: &action_kit_api.Messages{
			action_kit_api.Message{Level: extutil.Ptr(action_kit_api.Info), Message: fmt.Sprintf("Action Suppression Deleted. (Application ID %s, Action Suppression ID %s)", state.ApplicationId, *state.ActionSuppressionId)},
		},
	}, nil
}

// GetLocalTimezone tries, in order:
// 1) $TZ if set to an IANA name
// 2) /etc/timezone (Debian‐style)
// 3) resolving the /etc/localtime symlink
// 4) falling back to time.Local.String()
func GetLocalTimezone() (string, error) {
	// 1) check TZ env
	if tz := os.Getenv("TZ"); tz != "" && strings.Contains(tz, "/") {
		return tz, nil
	}

	// 2) Debian‐style file
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		tz := strings.TrimSpace(string(data))
		if strings.Contains(tz, "/") {
			return tz, nil
		}
	}

	// 3) /etc/localtime symlink back to zoneinfo
	const localtime = "/etc/localtime"
	if link, err := os.Readlink(localtime); err == nil {
		// e.g. /usr/share/zoneinfo/Asia/Kolkata
		parts := filepath.SplitList(link)
		// filepath.SplitList doesn’t split on “/” on Unix, so:
		parts = strings.Split(link, string(os.PathSeparator))
		for i, p := range parts {
			if p == "zoneinfo" && i+1 < len(parts) {
				return strings.Join(parts[i+1:], "/"), nil
			}
		}
	}

	// 4) fallback
	name := time.Now().Location().String()
	if name == "Local" || name == "" {
		return "", errors.New("could not determine local timezone")
	}
	return name, nil
}
