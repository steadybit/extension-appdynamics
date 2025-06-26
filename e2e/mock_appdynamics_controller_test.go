/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package e2e

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-appdynamics/extappdynamics"
	"github.com/steadybit/extension-kit/exthttp"
	"net"
	"net/http"
	"net/http/httptest"
)

type mockServer struct {
	http  *httptest.Server
	state string
}

func createMockAppDynamicsController() *mockServer {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen: %v", err))
	}
	mux := http.NewServeMux()

	server := httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	log.Info().Str("url", server.URL).Msg("Started Mock-Server")

	mock := &mockServer{http: &server, state: "CLEAR"}
	mux.Handle("/controller/api/oauth/access_token", handler(mock.getToken))
	mux.Handle("/controller/rest/applications", handler(mock.viewApplications))
	mux.Handle("/controller/alerting/rest/v1/applications/1/health-rules", handler(mock.viewHealthRules))
	mux.Handle("/controller/alerting/rest/v1/applications/2/health-rules", handler(mock.viewHealthRulesForApp2))
	mux.Handle("/controller/rest/applications/1/problems/healthrule-violations", handler(mock.viewHealthRuleViolationsForApp1))
	mux.Handle("/controller/rest/applications/2/problems/healthrule-violations", handler(mock.viewHealthRuleViolationsForApp2))
	return mock
}

func handler[T any](getter func() T) http.Handler {
	return exthttp.PanicRecovery(exthttp.LogRequest(exthttp.GetterAsHandler(getter)))
}

func (m *mockServer) getToken() extappdynamics.TokenResponse {
	return extappdynamics.TokenResponse{
		AccessToken: "eyJraWQiOiJhOGU4OTBlNS0zMjMwLTRlNGEtYTU2YS04NTg4Yjk0YjlhNTIiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJBcHBEeW5hbWljcyIsImF1ZCI6IkFwcERfQVBJcyIsImp0aSI6IktkS3hNd3UyUS1oZzNoNWtCeFZSOFEiLCJzdWIiOiJ0ZXN0IiwiaWRUeXBlIjoiQVBJX0NMSUVOVCIsImlkIjoiYzYzNDQwZjQtODY5YS00YmRkLWJkZWEtZTJkMzlkZDc2ZmIxIiwiYWNjdElkIjoiYThlODkwZTUtMzIzMC00ZTRhLWE1NmEtODU4OGI5NGI5YTUyIiwidG50SWQiOiJhOGU4OTBlNS0zMjMwLTRlNGEtYTU2YS04NTg4Yjk0YjlhNTIiLCJhY2N0TmFtZSI6InR3aW4yMDI1MDYyMzIxNDY1MzUiLCJ0ZW5hbnROYW1lIjoidHdpbjIwMjUwNjIzMjE0NjUzNSIsImZtbVRudElkIjpudWxsLCJhY2N0UGVybSI6W10sInJvbGVJZHMiOltdLCJpYXQiOjE3NTA5MzY5NzAsIm5iZiI6MTc1MDkzNjg1MCwiZXhwIjoxNzUwOTU0OTcwLCJ0b2tlblR5cGUiOiJBQ0NFU1MifQ._lEFk3Wmuo7ch_3MqcBaQj2cDjrs-8r8nBOxMXZMkhE",
		ExpiresIn:   18000,
	}
}

func (m *mockServer) viewApplications() []extappdynamics.Application {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extappdynamics.Application{{ID: 1, Name: "test", Description: "test", AccountGUID: "test"}, {ID: 2, Name: "test2", Description: "test", AccountGUID: "test"}}
}

func (m *mockServer) viewHealthRules() []extappdynamics.HealthRule {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extappdynamics.HealthRule{{
		ID: 1, Name: "Health", AffectedEntityType: "Node", Enabled: true,
	}}
}

func (m *mockServer) viewHealthRulesForApp2() []extappdynamics.HealthRule {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extappdynamics.HealthRule{{
		ID: 2, Name: "CPU", AffectedEntityType: "Node", Enabled: true,
	}}
}

func (m *mockServer) viewHealthRuleViolationsForApp1() []extappdynamics.Violation {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extappdynamics.Violation{
		{
			ID:          int64(32422),
			Name:        "health rule name",
			Description: "test",
		},
	}
}

func (m *mockServer) viewHealthRuleViolationsForApp2() []extappdynamics.Violation {
	if m.state == "STATUS-500" {
		panic("status 500")
	}
	return []extappdynamics.Violation{}
}
