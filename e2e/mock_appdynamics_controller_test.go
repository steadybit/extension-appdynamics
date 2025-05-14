/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

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
	mux.Handle("/controller/rest/applications", handler(mock.viewApplications))
	mux.Handle("/controller/alerting/rest/v1/applications/1/health-rules?output=JSON", handler(mock.viewHealthRules))
	mux.Handle("/controller/alerting/rest/v1/applications/2/health-rules?output=JSON", handler(mock.viewHealthRules))
	return mock
}

func handler[T any](getter func() T) http.Handler {
	return exthttp.PanicRecovery(exthttp.LogRequest(exthttp.GetterAsHandler(getter)))
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
