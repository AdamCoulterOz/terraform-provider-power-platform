// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	stdpath "path"
	"regexp"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/microsoft/terraform-provider-power-platform/internal/helpers"
)

var publisherGuidRegex = regexp.MustCompile(helpers.GuidRegex)

// CreatePublisher creates a publisher from an OData attribute map and reads
// the created record back.
func (client *Client) CreatePublisher(ctx context.Context, environmentHost string, publisher map[string]any) (*PublisherDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/publishers", nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodPost, apiUrl, nil, publisher, []int{http.StatusCreated, http.StatusNoContent, http.StatusForbidden}, nil)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}

	publisherId, err := GetPublisherIdFromResponse(resp)
	if err != nil {
		return nil, err
	}

	return client.GetPublisherById(ctx, environmentHost, publisherId)
}

// GetPublisherById reads a publisher by id.
func (client *Client) GetPublisherById(ctx context.Context, environmentHost, publisherId string) (*PublisherDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, fmt.Sprintf("/api/data/v9.2/publishers(%s)", publisherId), nil)
	publisher := PublisherDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodGet, apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &publisher)
	if err != nil {
		if resp != nil && resp.HttpResponse.StatusCode == http.StatusNotFound {
			return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("publisher '%s' not found", publisherId))
		}
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("publisher '%s' not found", publisherId))
	}

	return &publisher, nil
}

// GetPublishers lists the publishers of an environment.
func (client *Client) GetPublishers(ctx context.Context, environmentHost string) ([]PublisherDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/publishers", nil)

	publishers := PublishersDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodGet, apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden}, &publishers)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}

	return publishers.Value, nil
}

// GetPublisherByUniqueName reads a publisher by its unique name.
func (client *Client) GetPublisherByUniqueName(ctx context.Context, environmentHost, uniqueName string) (*PublisherDto, error) {
	values := url.Values{}
	values.Add("$filter", fmt.Sprintf("uniquename eq '%s'", escapeODataString(uniqueName)))
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, "/api/data/v9.2/publishers", values)

	publishers := PublishersDto{}
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodGet, apiUrl, nil, nil, []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound}, &publishers)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}
	if err := client.Api.HandleNotFoundResponse(resp); err != nil {
		return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("publisher '%s' not found", uniqueName))
	}

	if len(publishers.Value) == 0 {
		return nil, customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("publisher '%s' not found", uniqueName))
	}

	return &publishers.Value[0], nil
}

// UpdatePublisher patches a publisher from an OData attribute map and reads
// the record back.
func (client *Client) UpdatePublisher(ctx context.Context, environmentHost, publisherId string, publisher map[string]any) (*PublisherDto, error) {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, fmt.Sprintf("/api/data/v9.2/publishers(%s)", publisherId), nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodPatch, apiUrl, nil, publisher, []int{http.StatusNoContent, http.StatusForbidden}, nil)
	if err != nil {
		return nil, err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return nil, err
	}

	return client.GetPublisherById(ctx, environmentHost, publisherId)
}

// DeletePublisher deletes a publisher.
func (client *Client) DeletePublisher(ctx context.Context, environmentHost, publisherId string) error {
	apiUrl := helpers.BuildDataverseApiUrl(environmentHost, fmt.Sprintf("/api/data/v9.2/publishers(%s)", publisherId), nil)
	resp, err := client.Api.Execute(ctx, scopes(environmentHost), http.MethodDelete, apiUrl, nil, nil, []int{http.StatusNoContent, http.StatusNotFound, http.StatusForbidden}, nil)
	if err != nil {
		return err
	}
	if err := client.Api.HandleForbiddenResponse(resp); err != nil {
		return err
	}
	if resp.HttpResponse.StatusCode == http.StatusNotFound {
		return customerrors.WrapIntoProviderError(nil, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("publisher '%s' not found", publisherId))
	}

	return nil
}

// GetPublisherIdFromResponse extracts the created publisher's id from the
// OData-EntityId header of a create response.
func GetPublisherIdFromResponse(resp *api.Response) (string, error) {
	var entityId string
	for headerName, values := range resp.HttpResponse.Header {
		if strings.EqualFold(headerName, "OData-EntityId") && len(values) > 0 {
			entityId = values[0]
			break
		}
	}
	if entityId == "" {
		return "", errors.New("no publisher id returned from the API")
	}

	parsed, err := url.Parse(entityId)
	if err != nil {
		return "", err
	}

	entitySegment := stdpath.Base(parsed.Path)
	publisherId, found := strings.CutPrefix(entitySegment, "publishers(")
	if !found {
		return "", errors.New("no publisher id returned from the API")
	}

	publisherId = strings.TrimSuffix(publisherId, ")")
	if !publisherGuidRegex.MatchString(publisherId) {
		return "", errors.New("no publisher id returned from the API")
	}

	return publisherId, nil
}

func escapeODataString(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}
