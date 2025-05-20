/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extappdynamics

type Application struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AccountGUID string `json:"accountGuid"`
}

type HealthRule struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	AffectedEntityType string `json:"affectedEntityType"`
}

type Violation struct {
	DeepLinkURL          string    `json:"deepLinkUrl"`
	Severity             string    `json:"severity"`
	TriggeredEntity      EntityDef `json:"triggeredEntityDefinition"`
	StartTimeInMillis    int64     `json:"startTimeInMillis"`
	DetectedTimeInMillis int64     `json:"detectedTimeInMillis"`
	EndTimeInMillis      int64     `json:"endTimeInMillis"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	ID                   int64     `json:"id"`
	AffectedEntity       EntityDef `json:"affectedEntityDefinition"`
	IncidentStatus       string    `json:"incidentStatus"`
}

type EntityDef struct {
	EntityType string `json:"entityType"`
	Name       string `json:"name"`
	EntityID   int64  `json:"entityId"`
}

type ActionSuppressionResponse struct {
	ID                      int     `json:"id"`
	Name                    string  `json:"name"`
	DisableAgentReporting   bool    `json:"disableAgentReporting"`
	SuppressionScheduleType string  `json:"suppressionScheduleType"`
	Timezone                string  `json:"timezone"`
	StartTime               string  `json:"startTime"`
	EndTime                 string  `json:"endTime"`
	Affects                 Affects `json:"affects"`
}

type ActionSuppressionRequest struct {
	Name                    string  `json:"name"`
	DisableAgentReporting   bool    `json:"disableAgentReporting"`
	SuppressionScheduleType string  `json:"suppressionScheduleType"`
	Timezone                string  `json:"timezone"`
	StartTime               string  `json:"startTime"`
	EndTime                 string  `json:"endTime"`
	Affects                 Affects `json:"affects"`
}

type Affects struct {
	AffectedInfoType string `json:"affectedInfoType"`
}
