/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extappdynamics

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"strconv"
	"time"
)

type healthRuleDiscovery struct {
}

const (
	healthRuleAttribute = "appdynamics.health_rule"
)

var (
	_ discovery_kit_sdk.TargetDescriber    = (*healthRuleDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*healthRuleDiscovery)(nil)
)

func NewhealthRuleDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &healthRuleDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *healthRuleDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: applicationHealthRuleTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *healthRuleDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       applicationHealthRuleTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "AppDynamics health-rule", Other: "AppDynamics health-rules"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(applicationTargetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: healthRuleAttribute + ".name"},
				{Attribute: healthRuleAttribute + ".id"},
				{Attribute: healthRuleAttribute + ".description"},
				{Attribute: healthRuleAttribute + ".account_guid"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: healthRuleAttribute + ".name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *healthRuleDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: healthRuleAttribute + ".name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Application",
				Other: "Applications",
			},
		}, {
			Attribute: healthRuleAttribute + ".id",
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: healthRuleAttribute + ".description",
			Label: discovery_kit_api.PluralLabel{
				One:   "Descriptions",
				Other: "Descriptions",
			},
		}, {
			Attribute: healthRuleAttribute + ".account_guid",
			Label: discovery_kit_api.PluralLabel{
				One:   "Account GUID",
				Other: "Account GUIDs",
			},
		},
	}
}

func (d *healthRuleDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllApplications(ctx, RestyClient), nil
}

func getAllHealthRules(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	var appDynamicsResponse []Application
	result := make([]discovery_kit_api.Target, 0, 1000)
	res, err := client.R().
		SetContext(ctx).
		SetResult(&appDynamicsResponse).
		Get("/controller/rest/applications?output=JSON")

	if err != nil {
		log.Err(err).Msgf("Failed to retrieve applications from AppDynamics. Full response: %v", res.String())
		return result
	}

	if res.StatusCode() != 200 {
		log.Warn().Msgf("AppDynamics API responded with unexpected status code %d while retrieving alert states. Full response: %v",
			res.StatusCode(),
			res.String())
	} else {
		log.Trace().Msgf("AppDynamics response: %v", appDynamicsResponse)
	}

	for _, app := range appDynamicsResponse {
		result = append(result, discovery_kit_api.Target{
			Id:         strconv.Itoa(app.ID),
			TargetType: applicationTargetType,
			Label:      app.Name,
			Attributes: map[string][]string{
				healthRuleAttribute + ".description":  {app.Description},
				healthRuleAttribute + ".name":         {app.Name},
				healthRuleAttribute + ".id":           {strconv.Itoa(app.ID)},
				healthRuleAttribute + ".account_guid": {app.AccountGUID},
			}})
	}

	return result
}
