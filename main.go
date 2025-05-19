/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-appdynamics/config"
	"github.com/steadybit/extension-appdynamics/extappdynamics"
	"github.com/steadybit/extension-appdynamics/extevents"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kit/extsignals"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
	_ "net/http/pprof"           //allow pprof
	"strings"
)

func main() {
	extlogging.InitZeroLog()

	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	exthealth.SetReady(false)
	exthealth.StartProbes(8084)

	config.ParseConfiguration()
	config.ValidateConfiguration()
	initRestyClient()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	discovery_kit_sdk.Register(extappdynamics.NewApplicationDiscovery())
	discovery_kit_sdk.Register(extappdynamics.NewHealthRuleDiscovery())
	action_kit_sdk.RegisterAction(extappdynamics.NewHealthRuleStateCheckAction())

	if config.Config.EventApplicationID != "" {
		extevents.RegisterEventListenerHandlers()
	}

	extsignals.ActivateSignalHandlers()

	action_kit_sdk.RegisterCoverageEndpoints()

	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8083,
	})
}

func initRestyClient() {
	extappdynamics.RestyClient = resty.New()
	extappdynamics.RestyClient.SetBaseURL(strings.TrimRight(config.Config.ApiBaseUrl, "/"))
	extappdynamics.RestyClient.SetHeader("Authorization", "Bearer "+config.Config.AccessToken)
	extappdynamics.RestyClient.SetHeader("Content-Type", "application/json")

	extevents.RestyClient = resty.New()
	extevents.RestyClient.SetBaseURL(strings.TrimRight(config.Config.ApiBaseUrl, "/"))
	extevents.RestyClient.SetHeader("Authorization", "Bearer "+config.Config.AccessToken)
	extevents.RestyClient.SetHeader("Content-Type", "application/json")
}

type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	event_kit_api.EventListenerList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	extList := ExtensionListResponse{
		ActionList:    action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
	}
	if config.Config.EventApplicationID != "" {
		extList.EventListenerList = event_kit_api.EventListenerList{
			EventListeners: []event_kit_api.EventListener{
				{
					Method:   "POST",
					Path:     "/events/experiment-started",
					ListenTo: []string{"experiment.execution.created"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-completed",
					ListenTo: []string{"experiment.execution.completed", "experiment.execution.failed", "experiment.execution.canceled", "experiment.execution.errored"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-step-started",
					ListenTo: []string{"experiment.execution.step-started"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-target-started",
					ListenTo: []string{"experiment.execution.target-started"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-target-completed",
					ListenTo: []string{"experiment.execution.target-completed", "experiment.execution.target-canceled", "experiment.execution.target-errored", "experiment.execution.target-failed"},
				},
			},
		}

	}
	return extList
}
