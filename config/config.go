/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	// Deprecated: AccessToken is no longer supported. Use apiClientName, apiClientSecret, and accountName instead.
	AccessToken                             string   `json:"accessToken" split_words:"true" required:"false"`
	ApiBaseUrl                              string   `json:"apiBaseUrl" split_words:"true" required:"true"`
	ApiClientName                           string   `json:"apiClientName" split_words:"true" required:"false"`
	ApiClientSecret                         string   `json:"apiClientSecret" split_words:"true" required:"false"`
	AccountName                             string   `json:"accountName" split_words:"true" required:"false"`
	EventApplicationID                      string   `json:"eventApplicationID" split_words:"true" required:"false"`
	ActionSuppressionTimezone               string   `json:"actionSuppressionTimezone" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesApplications []string `json:"discoveryAttributesExcludesApplications" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesHealthRules  []string `json:"discoveryAttributesExcludesHealthRules" split_words:"true" required:"false"`
	ApplicationFilter                       []string `json:"applicationFilter" split_words:"true" required:"false"`
}

var (
	Config Specification
)

func ParseConfiguration() {
	err := envconfig.Process("steadybit_extension", &Config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse configuration from environment.")
	}
}

func ValidateConfiguration() {
	if Config.AccessToken != "" {
		log.Warn().Msg("Setting up an access token is deprecated. Please use apiClientName, apiClientSecret and accountName instead.")
	} else if Config.ApiClientName == "" || Config.ApiClientSecret == "" || Config.AccountName == "" {
		log.Fatal().Msg("ApiClientName, ApiClientSecret and AccountName must be set in the configuration.")
	}

	if len(Config.ApplicationFilter) > 0 {
		log.Info().Strs("ApplicationFilter", Config.ApplicationFilter).Msg("Using ApplicationFilter to limit the applications that are discovered. If you want to discover all applications, set ApplicationFilter to an empty list.")
	}
}
