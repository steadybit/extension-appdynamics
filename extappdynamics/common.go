/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extappdynamics

import "github.com/go-resty/resty/v2"

var RestyClient *resty.Client

const (
	applicationTargetType           = "com.steadybit.extension_appdynamics.application"
	applicationHealthRuleTargetType = "com.steadybit.extension_appdynamics.health_rule"
	appDynamicsTargetIcon           = "data:image/svg+xml,base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTkuNDkyMzcgMS41QzE1Ljg3NjkgMS41IDIxLjA1MTcgNi42NzQwOSAyMS4wMjE3IDEzLjA1ODZDMjEuMDIxNyAxNi45NjIxIDE5LjA4NDcgMjAuNDEyMSAxNi4xMTkzIDIyLjVMMTQuMzAzOSAxOC42ODc1QzE1LjkwNzYgMTcuMzI1OCAxNi45MDY0IDE1LjI5NzggMTYuOTA2NCAxMy4wNTg2QzE2LjkwNjIgOC45NzM4IDEzLjU3NzIgNS42NDU1MSA5LjQ5MjM3IDUuNjQ1NTFDOS4wMzg1OSA1LjY0NTUyIDguNTg0ODIgNS42NzU4NSA4LjEzMTA0IDUuNzY2Nkw2LjMxNTYxIDEuOTU0MUM3LjMxNDA1IDEuNjUxNTUgOC40MDMxNyAxLjUwMDAzIDkuNDkyMzcgMS41Wk0xMC42NDI4IDIwLjM4MThDMTAuMjQ5NCAyMC40NDI0IDkuODg1NzQgMjAuNDcyNyA5LjQ5MjM3IDIwLjQ3MjdDNS40MDc1IDIwLjQ3MjUgMi4wNzkyOCAxNy4xNDM1IDIuMDc5MjggMTMuMDU4NkMyLjA3OTQxIDEwLjg4MDEgMy4wMTc1NyA4Ljk0MzYxIDQuNTAwMTggNy41ODIwM0wxMC42NDI4IDIwLjM4MThaIiBmaWxsPSJjdXJyZW50Q29sb3IiLz4KPC9zdmc+"
	stateCheckModeAtLeastOnce       = "atLeastOnce"
	stateCheckModeAllTheTime        = "allTheTime"
)
