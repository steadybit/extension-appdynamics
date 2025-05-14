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
	enabled             = ".enabled"
	affectedEntityType  = ".affected_entity_type"
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
			CallInterval: extutil.Ptr("2m"),
		},
	}
}

func (d *healthRuleDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       applicationHealthRuleTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "AppDynamics health-rule", Other: "AppDynamics health-rules"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(appDynamicsTargetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: healthRuleAttribute + ".name"},
				{Attribute: healthRuleAttribute + ".id"},
				{Attribute: healthRuleAttribute + enabled},
				{Attribute: healthRuleAttribute + affectedEntityType},
				{Attribute: healthRuleAttribute + ".application"},
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
				One:   "Health rule",
				Other: "Halth rules",
			},
		}, {
			Attribute: healthRuleAttribute + ".id",
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: healthRuleAttribute + enabled,
			Label: discovery_kit_api.PluralLabel{
				One:   "Status",
				Other: "Status",
			},
		}, {
			Attribute: healthRuleAttribute + affectedEntityType,
			Label: discovery_kit_api.PluralLabel{
				One:   "Affected entity type",
				Other: "Affected entity types",
			},
		},
	}
}

func (d *healthRuleDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllHealthRules(ctx, RestyClient), nil
}

func getAllHealthRules(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	var applications []Application
	var healthrules []HealthRule

	result := make([]discovery_kit_api.Target, 0, 1000)
	res, err := client.R().
		SetContext(ctx).
		SetResult(&applications).
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
		log.Trace().Msgf("AppDynamics response: %v", applications)
	}

	for _, app := range applications {
		res, err := client.R().
			SetContext(ctx).
			SetResult(&healthrules).
			Get("/controller/alerting/rest/v1/applications/" + strconv.Itoa(app.ID) + "/health-rules?output=JSON")

		if err != nil {
			log.Err(err).Msgf("Failed to retrieve health rules from AppDynamics with application %d. Full response: %v", app.ID, res.String())
			return result
		}

		if res.StatusCode() != 200 {
			log.Warn().Msgf("AppDynamics API responded with unexpected status code %d while retrieving alert states. Full response: %v",
				res.StatusCode(),
				res.String())
		} else {
			log.Trace().Msgf("AppDynamics response: %v", applications)
		}
		for _, healthRule := range healthrules {
			result = append(result, discovery_kit_api.Target{
				Id:         strconv.Itoa(app.ID) + "-" + strconv.Itoa(healthRule.ID),
				TargetType: applicationHealthRuleTargetType,
				Label:      healthRule.Name,
				Attributes: map[string][]string{
					healthRuleAttribute + ".name":            {healthRule.Name},
					healthRuleAttribute + ".id":              {strconv.Itoa(healthRule.ID)},
					healthRuleAttribute + enabled:            {strconv.FormatBool(healthRule.Enabled)},
					healthRuleAttribute + affectedEntityType: {healthRule.AffectedEntityType},
					healthRuleAttribute + ".application":     {strconv.Itoa(app.ID)},
				}})
		}
	}

	return result
}
