/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extappdynamics

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-appdynamics/config"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"k8s.io/utils/strings/slices"
	"strconv"
	"time"
)

type healthRuleDiscovery struct {
}

const (
	HealthRuleAttribute         = "appdynamics.health-rule"
	AttributeEnabled            = ".enabled"
	AttributeAffectedEntityType = ".affected_entity_type"
	AttributeAppID              = ".application.id"
	AttributeAppName            = ".application.name"
	AttributeOrigin             = ".origin"
)

var (
	_ discovery_kit_sdk.TargetDescriber    = (*healthRuleDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*healthRuleDiscovery)(nil)
)

func NewHealthRuleDiscovery() discovery_kit_sdk.TargetDiscovery {
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
				{Attribute: HealthRuleAttribute + ".name"},
				{Attribute: HealthRuleAttribute + ".id"},
				{Attribute: HealthRuleAttribute + AttributeEnabled},
				{Attribute: HealthRuleAttribute + AttributeAffectedEntityType},
				{Attribute: HealthRuleAttribute + AttributeAppID},
				{Attribute: HealthRuleAttribute + AttributeAppName},
				{Attribute: HealthRuleAttribute + AttributeOrigin},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: HealthRuleAttribute + ".name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *healthRuleDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: HealthRuleAttribute + ".name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Health rule",
				Other: "Halth rules",
			},
		}, {
			Attribute: HealthRuleAttribute + ".id",
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: HealthRuleAttribute + AttributeEnabled,
			Label: discovery_kit_api.PluralLabel{
				One:   "Status",
				Other: "Status",
			},
		}, {
			Attribute: HealthRuleAttribute + AttributeAffectedEntityType,
			Label: discovery_kit_api.PluralLabel{
				One:   "Affected entity type",
				Other: "Affected entity types",
			},
		}, {
			Attribute: HealthRuleAttribute + AttributeAppID,
			Label: discovery_kit_api.PluralLabel{
				One:   "Health rule application id",
				Other: "Health rule application ids",
			},
		}, {
			Attribute: HealthRuleAttribute + AttributeAppName,
			Label: discovery_kit_api.PluralLabel{
				One:   "Health rule application name",
				Other: "Health rule application names",
			},
		}, {
			Attribute: HealthRuleAttribute + AttributeOrigin,
			Label: discovery_kit_api.PluralLabel{
				One:   "Health rule controller url",
				Other: "Health rule controller urls",
			},
		},
	}
}

func (d *healthRuleDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return discovery_kit_commons.ApplyAttributeExcludes(getAllHealthRules(ctx, RestyClient), config.Config.DiscoveryAttributesExcludesHealthRules), nil
}

func getAllHealthRules(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	var applications []Application
	var healthRules []HealthRule

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
		log.Warn().Msgf("AppDynamics API responded with unexpected status code %d while retrieving applications. Full response: %v",
			res.StatusCode(),
			res.String())
	} else {
		log.Trace().Msgf("AppDynamics response: %v", applications)
	}

	for _, app := range applications {
		appId := strconv.Itoa(app.ID)
		if len(config.Config.ApplicationFilter) > 0 && !slices.Contains(config.Config.ApplicationFilter, appId) {
			continue
		}

		res, err := client.R().
			SetContext(ctx).
			SetResult(&healthRules).
			Get("/controller/alerting/rest/v1/applications/" + strconv.Itoa(app.ID) + "/health-rules?output=JSON")

		if err != nil {
			log.Err(err).Msgf("Failed to retrieve health rules from AppDynamics with application %d. Full response: %v", app.ID, res.String())
			return result
		}

		if res.StatusCode() != 200 {
			log.Warn().Msgf("AppDynamics API responded with unexpected status code %d while retrieving health rules. Full response: %v",
				res.StatusCode(),
				res.String())
		} else {
			log.Trace().Msgf("AppDynamics response: %v", applications)
		}
		for _, healthRule := range healthRules {
			result = append(result, discovery_kit_api.Target{
				Id:         strconv.Itoa(app.ID) + "-" + strconv.Itoa(healthRule.ID),
				TargetType: applicationHealthRuleTargetType,
				Label:      healthRule.Name,
				Attributes: map[string][]string{
					HealthRuleAttribute + ".name":                     {healthRule.Name},
					HealthRuleAttribute + ".id":                       {strconv.Itoa(healthRule.ID)},
					HealthRuleAttribute + AttributeEnabled:            {strconv.FormatBool(healthRule.Enabled)},
					HealthRuleAttribute + AttributeAffectedEntityType: {healthRule.AffectedEntityType},
					HealthRuleAttribute + AttributeAppID:              {strconv.Itoa(app.ID)},
					HealthRuleAttribute + AttributeAppName:            {app.Name},
					HealthRuleAttribute + AttributeOrigin:             {config.Config.ApiBaseUrl},
				}})
		}
	}

	return result
}
