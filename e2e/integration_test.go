/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	actValidate "github.com/steadybit/action-kit/go/action_kit_test/validate"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-appdynamics/extappdynamics"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	server := createMockAppDynamicsController()
	defer server.http.Close()
	split := strings.SplitAfter(server.http.URL, ":")
	port := split[len(split)-1]

	extlogging.InitZeroLog()

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-appdynamics",
		Port: 8083,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", fmt.Sprintf("appdynamics.apiBaseUrl=http://host.minikube.internal:%s", port),
				"--set", "logging.level=trace",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "validate discovery",
			Test: validateDiscovery,
		},
		{
			Name: "test discovery",
			Test: testDiscovery,
		},
		{
			Name: "validate Actions",
			Test: validateActions,
		},
		{
			Name: "health rule check meets expectations",
			Test: testHealthRuleCheck(true, "1", ""),
		},
		{
			Name: "health rule check fails expectations",
			Test: testHealthRuleCheck(true, "2", "[failed] HealthRule 'health rule name' has violations 'false' whereas 'Violations Expected :true'."),
		},
	})
}

func testHealthRuleCheck(hasViolation bool, appID string, wantedActionStatus action_kit_api.ActionKitErrorStatus) func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
	return func(t *testing.T, minikube *e2e.Minikube, e *e2e.Extension) {
		target := &action_kit_api.Target{
			Name: "dynamic_health_rule",
			Attributes: map[string][]string{
				extappdynamics.HealthRuleAttribute + ".application.id": {appID},
				extappdynamics.HealthRuleAttribute + ".id":             {"1"},
				extappdynamics.HealthRuleAttribute + ".name":           {"health rule name"},
			},
		}

		config := struct {
			Duration       int    `json:"duration"`
			Violation      bool   `json:"violation"`
			StateCheckMode string `json:"stateCheckMode"`
		}{Duration: 1_000, Violation: hasViolation, StateCheckMode: extappdynamics.StateCheckModeAllTheTime}

		action, err := e.RunAction("com.steadybit.extension_appdynamics.health-rule.check", target, config, &action_kit_api.ExecutionContext{})
		require.NoError(t, err)
		defer func() { _ = action.Cancel() }()

		err = action.Wait()
		if wantedActionStatus == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, string(wantedActionStatus))
		}
	}
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func testDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	app, err := e2e.PollForTarget(ctx, e, "com.steadybit.extension_appdynamics.application", func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "appdynamics.application.id", "1")
	})
	require.NoError(t, err)
	healthrule, err := e2e.PollForTarget(ctx, e, "com.steadybit.extension_appdynamics.health-rule", func(target discovery_kit_api.Target) bool {
		return e2e.HasAttribute(target, "appdynamics.health-rule.id", "1")
	})
	require.NoError(t, err)
	assert.Equal(t, app.TargetType, "com.steadybit.extension_appdynamics.application")
	assert.Equal(t, app.Attributes[extappdynamics.AppAttribute+".description"], []string{"test"})
	assert.Equal(t, app.Attributes[extappdynamics.AppAttribute+extappdynamics.AppAccountGUID], []string{"test"})
	assert.Equal(t, app.Attributes[extappdynamics.AppAttribute+".name"], []string{"test"})

	assert.Equal(t, healthrule.TargetType, "com.steadybit.extension_appdynamics.health-rule")
	assert.Equal(t, healthrule.Attributes[extappdynamics.HealthRuleAttribute+extappdynamics.AttributeAffectedEntityType], []string{"Node"})
	assert.Equal(t, healthrule.Attributes[extappdynamics.HealthRuleAttribute+extappdynamics.AttributeEnabled], []string{"true"})
	assert.Equal(t, healthrule.Attributes[extappdynamics.HealthRuleAttribute+".application.id"], []string{"1"})
}

func validateActions(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, actValidate.ValidateEndpointReferences("/", e.Client))
}
