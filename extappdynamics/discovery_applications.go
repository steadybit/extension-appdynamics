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

type applicationDiscovery struct {
}

const (
	AppAttribute   = "appdynamics.application"
	AppAccountGUID = ".account-guid"
	AppDescription = ".description"
	AppOrigin      = ".origin"
)

var (
	_ discovery_kit_sdk.TargetDescriber    = (*applicationDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*applicationDiscovery)(nil)
)

func NewApplicationDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &applicationDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}

func (d *applicationDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: applicationTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *applicationDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       applicationTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "AppDynamics application", Other: "AppDynamics applications"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(appDynamicsTargetIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: AppAttribute + ".name"},
				{Attribute: AppAttribute + ".id"},
				{Attribute: AppAttribute + AppDescription},
				{Attribute: AppAttribute + AppAccountGUID},
				{Attribute: AppAttribute + AppOrigin},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: AppAttribute + ".name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *applicationDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: AppAttribute + ".name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Application",
				Other: "Applications",
			},
		}, {
			Attribute: AppAttribute + ".id",
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: AppAttribute + AppDescription,
			Label: discovery_kit_api.PluralLabel{
				One:   "Descriptions",
				Other: "Descriptions",
			},
		}, {
			Attribute: AppAttribute + AppAccountGUID,
			Label: discovery_kit_api.PluralLabel{
				One:   "Account GUID",
				Other: "Account GUIDs",
			},
		}, {
			Attribute: AppAttribute + AppOrigin,
			Label: discovery_kit_api.PluralLabel{
				One:   "Controller Url",
				Other: "Controller Urls",
			},
		},
	}
}

func (d *applicationDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return discovery_kit_commons.ApplyAttributeExcludes(getAllApplications(ctx, RestyClient), config.Config.DiscoveryAttributesExcludesApplications), nil
}

func getAllApplications(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	if client == nil {
		log.Error().Msg("Client is nil.")
	}
	applications := []Application{}
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
		result = append(result, discovery_kit_api.Target{
			Id:         appId,
			TargetType: applicationTargetType,
			Label:      app.Name,
			Attributes: map[string][]string{
				AppAttribute + AppDescription: {app.Description},
				AppAttribute + ".name":        {app.Name},
				AppAttribute + ".id":          {appId},
				AppAttribute + AppAccountGUID: {app.AccountGUID},
				AppAttribute + AppOrigin:      {config.Config.ApiBaseUrl},
			}})
	}

	return result
}
