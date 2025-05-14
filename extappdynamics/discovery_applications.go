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

type applicationDiscovery struct {
}

const (
	appAttribute   = "appdynamics.application"
	appAccountGUID = ".account_guid"
	appDescription = ".description"
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
				{Attribute: appAttribute + ".name"},
				{Attribute: appAttribute + ".id"},
				{Attribute: appAttribute + appDescription},
				{Attribute: appAttribute + appAccountGUID},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: appAttribute + ".name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *applicationDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: appAttribute + ".name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Application",
				Other: "Applications",
			},
		}, {
			Attribute: appAttribute + ".id",
			Label: discovery_kit_api.PluralLabel{
				One:   "ID",
				Other: "IDs",
			},
		}, {
			Attribute: appAttribute + appDescription,
			Label: discovery_kit_api.PluralLabel{
				One:   "Descriptions",
				Other: "Descriptions",
			},
		}, {
			Attribute: appAttribute + ".account_guid",
			Label: discovery_kit_api.PluralLabel{
				One:   "Account GUID",
				Other: "Account GUIDs",
			},
		},
	}
}

func (d *applicationDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllApplications(ctx, RestyClient), nil
}

func getAllApplications(ctx context.Context, client *resty.Client) []discovery_kit_api.Target {
	var applications []Application
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
		result = append(result, discovery_kit_api.Target{
			Id:         strconv.Itoa(app.ID),
			TargetType: applicationTargetType,
			Label:      app.Name,
			Attributes: map[string][]string{
				appAttribute + appDescription: {app.Description},
				appAttribute + ".name":        {app.Name},
				appAttribute + ".id":          {strconv.Itoa(app.ID)},
				appAttribute + appAccountGUID: {app.AccountGUID},
			}})
	}

	return result
}
