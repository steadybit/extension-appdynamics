/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	AccessToken                             string   `json:"accessToken" split_words:"true" required:"true"`
	ApiBaseUrl                              string   `json:"apiBaseUrl" split_words:"true" required:"true"`
	EventApplicationID                      string   `json:"eventApplicationID" split_words:"true" required:"false"`
	ActionSuppressionTimezone               string   `json:"actionSuppressionTimezone" split_words:"true" required:"false"`
	DiscoveryAttributesExcludesApplications []string `json:"discoveryAttributesExcludesApplications" split_words:"true" required:"false"`
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
	// You may optionally validate the configuration here.
}
