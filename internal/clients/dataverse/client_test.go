// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEnvironmentHost is the per-environment Dataverse host used by every
// request-shape test in this package.
const testEnvironmentHost = "env.crm.dynamics.com"

// newTestClient returns a dataverse client over the shared transport in
// TestMode: token acquisition returns a mock token and SleepWithContext is a
// no-op, so retry/poll loops run instantly.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	cfg := &config.ProviderConfig{
		TestMode: true,
		Urls: config.ProviderConfigUrls{
			BapiUrl:               "api.bap.microsoft.com",
			PowerAppsUrl:          "api.powerapps.com",
			PowerAppsScope:        "https://service.powerapps.com/.default",
			PowerPlatformUrl:      "api.powerplatform.com",
			PowerPlatformScope:    "https://api.powerplatform.com/.default",
			AdminPowerPlatformUrl: "api.admin.powerplatform.microsoft.com",
			PowerAppsAdvisor:      "api.advisor.powerapps.com",
			PowerAppsAdvisorScope: "https://advisor.powerapps.com/.default",
			AnalyticsScope:        "https://adminanalytics.powerplatform.microsoft.com/.default",
		},
	}
	return NewClient(api.NewApiClientBase(cfg, api.NewAuthBase(cfg)))
}

// recordedRequest is one HTTP request observed on the wire, exactly as the
// transport sent it: method, the full URL string (scheme, host, escaped path,
// raw query and fragment), the raw request body, and the request headers.
type recordedRequest struct {
	method string
	url    string
	body   string
	header http.Header
}

// step scripts one HTTP response returned by the recorder, in order.
type step struct {
	status int
	body   string
	header http.Header
}

// activateRecorder activates httpmock and installs a catch-all responder that
// records every outgoing request and answers the n-th request with the n-th
// scripted step. Extra unscripted requests fail the calling operation with a
// descriptive transport error, so tests pin the exact request sequence.
func activateRecorder(t *testing.T, steps ...step) *[]recordedRequest {
	t.Helper()
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	recorded := &[]recordedRequest{}
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		var body string
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			body = string(bodyBytes)
		}
		requestIndex := len(*recorded)
		*recorded = append(*recorded, recordedRequest{method: req.Method, url: req.URL.String(), body: body, header: req.Header.Clone()})
		if requestIndex >= len(steps) {
			return nil, fmt.Errorf("unscripted request #%d: %s %s", requestIndex+1, req.Method, req.URL.String())
		}
		scripted := steps[requestIndex]
		resp := httpmock.NewStringResponse(scripted.status, scripted.body)
		resp.Header.Set("Content-Type", "application/json")
		for name, values := range scripted.header {
			for _, value := range values {
				resp.Header.Add(name, value)
			}
		}
		// api.Client.HandleNotFoundResponse formats the request URL, so the
		// response must reference its request like a real transport does.
		resp.Request = req
		return resp, nil
	})
	return recorded
}

// assertRequests pins the exact sequence of wire requests: method, full URL,
// and body (JSON bodies are compared structurally, others byte-for-byte).
func assertRequests(t *testing.T, got *[]recordedRequest, want []recordedRequest) {
	t.Helper()
	require.Len(t, *got, len(want), "wire request count")
	for i, wantReq := range want {
		gotReq := (*got)[i]
		assert.Equal(t, wantReq.method, gotReq.method, "request #%d method", i+1)
		assert.Equal(t, wantReq.url, gotReq.url, "request #%d url", i+1)
		if wantReq.body == "" {
			assert.Empty(t, gotReq.body, "request #%d body", i+1)
		} else {
			assert.JSONEq(t, wantReq.body, gotReq.body, "request #%d body", i+1)
		}
	}
}

// TestUnitScopes pins the package-level scope mapping: every Dataverse call
// authenticates against the environment host it targets, nothing else.
func TestUnitScopes(t *testing.T) {
	tests := []struct {
		name            string
		environmentHost string
		want            []string
	}{
		{
			name:            "public cloud org host",
			environmentHost: "org42.crm.dynamics.com",
			want:            []string{"https://org42.crm.dynamics.com/.default"},
		},
		{
			name:            "regional org host",
			environmentHost: "org.crm4.dynamics.com",
			want:            []string{"https://org.crm4.dynamics.com/.default"},
		},
		{
			name:            "test fixture host",
			environmentHost: testEnvironmentHost,
			want:            []string{"https://env.crm.dynamics.com/.default"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scopes(tt.environmentHost))
		})
	}
}
