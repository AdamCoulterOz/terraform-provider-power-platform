// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPublisherId   = "12345678-1111-2222-3333-444444444444"
	testPublisherBody = `{"publisherid":"12345678-1111-2222-3333-444444444444","friendlyname":"Contoso Publisher","uniquename":"contoso"}`
)

func responseWithHeaders(header http.Header) *api.Response {
	return &api.Response{
		HttpResponse: &http.Response{
			Header: header,
		},
	}
}

func TestUnitGetPublisherIdFromResponse(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		want    string
		wantErr string
	}{
		{
			name:   "canonical OData-EntityId header",
			header: http.Header{"Odata-Entityid": []string{"https://env.crm.dynamics.com/api/data/v9.2/publishers(" + testPublisherId + ")"}},
			want:   testPublisherId,
		},
		{
			name:   "header name is matched case-insensitively",
			header: http.Header{"ODATA-ENTITYID": []string{"https://env.crm.dynamics.com/api/data/v9.2/publishers(" + testPublisherId + ")"}},
			want:   testPublisherId,
		},
		{
			name:    "missing header",
			header:  http.Header{},
			wantErr: "no publisher id returned from the API",
		},
		{
			name:    "entity id for a different collection",
			header:  http.Header{"Odata-Entityid": []string{"https://env.crm.dynamics.com/api/data/v9.2/solutions(" + testPublisherId + ")"}},
			wantErr: "no publisher id returned from the API",
		},
		{
			name:    "entity id is not a guid",
			header:  http.Header{"Odata-Entityid": []string{"https://env.crm.dynamics.com/api/data/v9.2/publishers(not-a-guid)"}},
			wantErr: "no publisher id returned from the API",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPublisherIdFromResponse(responseWithHeaders(tt.header))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnitEscapeODataString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no quotes unchanged", input: "contoso", want: "contoso"},
		{name: "single quote doubled", input: "o'brien", want: "o''brien"},
		{name: "every quote doubled", input: "a'b'c", want: "a''b''c"},
		{name: "empty string", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeODataString(tt.input))
		})
	}
}

func TestUnitCreatePublisher_PostsThenReadsBack(t *testing.T) {
	tests := []struct {
		name         string
		createStatus int
	}{
		{name: "created", createStatus: http.StatusCreated},
		{name: "no content", createStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t,
				step{status: tt.createStatus, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/publishers(" + testPublisherId + ")")},
				step{status: http.StatusOK, body: testPublisherBody},
			)

			publisher, err := client.CreatePublisher(context.Background(), testEnvironmentHost, map[string]any{
				"uniquename":          "contoso",
				"friendlyname":        "Contoso Publisher",
				"customizationprefix": "ctso",
			})

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{
					method: "POST",
					url:    "https://env.crm.dynamics.com/api/data/v9.2/publishers",
					body:   `{"customizationprefix":"ctso","friendlyname":"Contoso Publisher","uniquename":"contoso"}`,
				},
				{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers%28" + testPublisherId + "%29"},
			})
			assert.Equal(t, testPublisherId, publisher.Id)
			assert.Equal(t, "contoso", publisher.UniqueName)
		})
	}
}

func TestUnitCreatePublisher_MissingEntityIdHeaderFails(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	_, err := client.CreatePublisher(context.Background(), testEnvironmentHost, map[string]any{"uniquename": "contoso"})

	require.Error(t, err)
	assert.EqualError(t, err, "no publisher id returned from the API")
	require.Len(t, *recorded, 1, "no read-back without a created id")
}

func TestUnitGetPublisherById_NotFoundIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusNotFound, body: `{"error":{"code":"0x80040217"}}`})

	_, err := client.GetPublisherById(context.Background(), testEnvironmentHost, testPublisherId)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.Contains(t, err.Error(), "publisher '"+testPublisherId+"' not found")
}

func TestUnitGetPublishers_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testPublisherBody + `]}`})

	publishers, err := client.GetPublishers(context.Background(), testEnvironmentHost)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers"},
	})
	require.Len(t, publishers, 1)
	assert.Equal(t, testPublisherId, publishers[0].Id)
}

func TestUnitGetPublisherByUniqueName_EscapesQuotesInFilter(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[` + testPublisherBody + `]}`})

	publisher, err := client.GetPublisherByUniqueName(context.Background(), testEnvironmentHost, "o'brien")

	require.NoError(t, err)
	// The single quote in the unique name is doubled per OData literal rules
	// before URL encoding.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers?%24filter=uniquename+eq+%27o%27%27brien%27"},
	})
	assert.Equal(t, testPublisherId, publisher.Id)
}

func TestUnitGetPublisherByUniqueName_EmptyResultIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value":[]}`})

	_, err := client.GetPublisherByUniqueName(context.Background(), testEnvironmentHost, "missing")

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.EqualError(t, err, "OBJECT_NOT_FOUND: publisher 'missing' not found")
}

func TestUnitUpdatePublisher_PatchesThenReadsBack(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		step{status: http.StatusNoContent},
		step{status: http.StatusOK, body: testPublisherBody},
	)

	publisher, err := client.UpdatePublisher(context.Background(), testEnvironmentHost, testPublisherId, map[string]any{"friendlyname": "Contoso Publisher"})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers%28" + testPublisherId + "%29", body: `{"friendlyname":"Contoso Publisher"}`},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers%28" + testPublisherId + "%29"},
	})
	assert.Equal(t, testPublisherId, publisher.Id)
}

func TestUnitDeletePublisher_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusNoContent})

	err := client.DeletePublisher(context.Background(), testEnvironmentHost, testPublisherId)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/publishers%28" + testPublisherId + "%29"},
	})
}

func TestUnitDeletePublisher_NotFoundIsProviderError(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusNotFound, body: `{"error":{"code":"0x80040217"}}`})

	err := client.DeletePublisher(context.Background(), testEnvironmentHost, testPublisherId)

	require.Error(t, err)
	assert.True(t, errors.Is(err, customerrors.ErrObjectNotFound))
	assert.EqualError(t, err, "OBJECT_NOT_FOUND: publisher '"+testPublisherId+"' not found")
}
