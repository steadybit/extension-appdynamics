/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extappdynamics

type Application struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AccountGUID string `json:"accountGuid"`
}
