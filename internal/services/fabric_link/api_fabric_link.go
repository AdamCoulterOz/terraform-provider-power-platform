// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// athenaResourceScope is the token resource the "athena" Synapse Link service requires
// (observed aud in the maker-portal capture). It must be requested explicitly because the generic
// client would otherwise guess the scope from the URL host. NOTE: whether this resource grants
// APP-ONLY (service principal) tokens — the gate for fully unattended Terraform — is unverified;
// the capture only proves delegated (user_impersonation) works.
const athenaResourceScope = "7f15f9d9-cad0-44f1-bbba-d36650e07765/.default"

func newFabricLinkClient(apiClient *api.Client) client {
	return client{Api: apiClient}
}

type client struct {
	Api *api.Client
}

// getBapEnvironment reads the BAP environment to resolve the Dataverse org info and the
// gateway cluster used to build the athena host.
func (client *client) getBapEnvironment(ctx context.Context, environmentId string) (*bapEnvironmentDto, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   client.Api.GetConfig().Urls.BapiUrl,
		Path:   fmt.Sprintf("/providers/Microsoft.BusinessAppPlatform/scopes/admin/environments/%s", environmentId),
	}
	values := url.Values{}
	values.Add(constants.API_VERSION_PARAM, "2020-10-01-alpha")
	apiUrl.RawQuery = values.Encode()

	env := bapEnvironmentDto{}
	if _, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &env); err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", environmentId, err)
	}
	return &env, nil
}

// athenaHost builds the Synapse Link ("athena") orchestration host from the env gateway cluster.
// Observed: cluster.uriSuffix "au-il301.gateway.prod.island" -> athenawebservice.eau-il301.gateway.prod.island.powerapps.com
// (the "athenawebservice." prefix is followed by an "e" then the cluster uriSuffix).
func athenaHost(clusterUriSuffix string) (string, error) {
	if strings.TrimSpace(clusterUriSuffix) == "" {
		return "", fmt.Errorf("environment cluster uriSuffix is empty; cannot derive the Link to Fabric (athena) host")
	}
	return fmt.Sprintf("athenawebservice.e%s.powerapps.com", clusterUriSuffix), nil
}

// CreateFabricLink provisions a Link to Fabric and returns the created mirror ids.
// connectionId is the Dataverse->OneLake connection to use (see upsertConnection / DESIGN.md).
func (client *client) CreateFabricLink(ctx context.Context, environmentId, fabricWorkspaceId, connectionId string, tables []string) (*lakehouseArtifactsResponseDto, error) {
	env, err := client.getBapEnvironment(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	host, err := athenaHost(env.Properties.Cluster.UriSuffix)
	if err != nil {
		return nil, err
	}

	descriptions := make([]entityDescriptionDto, 0, len(tables))
	for _, t := range tables {
		descriptions = append(descriptions, entityDescriptionDto{Type: t, EntitySource: "Dataverse"})
	}

	body := lakehouseArtifactsRequestDto{
		OrganizationId:          env.Properties.LinkedEnvironmentMetadata.ResourceId,
		OrganizationUrl:         env.Properties.LinkedEnvironmentMetadata.InstanceURL,
		EnvironmentFriendlyName: env.Properties.LinkedEnvironmentMetadata.FriendlyName,
		EnvironmentUniqueName:   env.Properties.LinkedEnvironmentMetadata.UniqueName,
		Entities:                tables,
		IsManagedLake:           true,
		WorkspaceId:             fabricWorkspaceId,
		ConnectionId:            connectionId,
		EntityDescriptions:      descriptions,
	}

	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     host,
		Path:     fmt.Sprintf("/environment/%s/lakehouseArtifacts", environmentId),
		RawQuery: "dxt=false",
	}

	// The athena response body is base64-encoded JSON, so read raw bytes and decode.
	resp, err := client.Api.Execute(ctx, []string{athenaResourceScope}, "POST", apiUrl.String(), nil, body, []int{http.StatusOK, http.StatusCreated}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Link to Fabric for environment %s: %w", environmentId, err)
	}

	result, err := decodeArtifactsResponse(resp.BodyAsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode lakehouseArtifacts response: %w", err)
	}

	// Resolve the datalakefolder id now (the athena response omits it) so Delete can target it later.
	// Best-effort: a missing id only weakens destroy, it shouldn't fail a successful create.
	if folderId, ferr := client.getDatalakeFolderId(ctx, env.Properties.LinkedEnvironmentMetadata.InstanceURL); ferr == nil {
		result.DatalakeFolderId = folderId
	}
	return result, nil
}

// getDatalakeFolderId resolves the datalakefolder id the unlink DELETE targets. Link to Fabric
// creates both a cds2_workspace and a cds3_workspace folder; the observed unlink targeted
// cds3_workspace, so that is preferred (cds2_workspace is the fallback).
func (client *client) getDatalakeFolderId(ctx context.Context, organizationUrl string) (string, error) {
	host, err := hostFromUrl(organizationUrl)
	if err != nil {
		return "", err
	}
	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     host,
		Path:     "/api/data/v9.1/datalakefolders",
		RawQuery: "$select=datalakefolderid,datalakefolder_uniquename",
	}
	var list datalakeFolderListDto
	if _, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &list); err != nil {
		return "", fmt.Errorf("failed to query datalakefolders: %w", err)
	}
	var fallback string
	for _, f := range list.Value {
		switch f.DatalakeFolderUniqueName {
		case "cds3_workspace":
			return f.DatalakeFolderId, nil
		case "cds2_workspace":
			fallback = f.DatalakeFolderId
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	if len(list.Value) > 0 {
		return list.Value[0].DatalakeFolderId, nil
	}
	return "", fmt.Errorf("no datalakefolder found for the Link to Fabric")
}

// DeleteFabricLink unlinks the Link to Fabric by deleting its datalakefolder via athena.
func (client *client) DeleteFabricLink(ctx context.Context, environmentId, datalakeFolderId string) error {
	if strings.TrimSpace(datalakeFolderId) == "" {
		return fmt.Errorf("datalake folder id is empty; cannot unlink (re-import or remove the link manually)")
	}
	env, err := client.getBapEnvironment(ctx, environmentId)
	if err != nil {
		return err
	}
	host, err := athenaHost(env.Properties.Cluster.UriSuffix)
	if err != nil {
		return err
	}
	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     host,
		Path:     fmt.Sprintf("/environment/%s/lakehouseArtifacts/%s", environmentId, datalakeFolderId),
		RawQuery: "dxt=false",
	}
	if _, err := client.Api.Execute(ctx, []string{athenaResourceScope}, "DELETE", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound}, nil); err != nil {
		return fmt.Errorf("failed to unlink Link to Fabric for environment %s: %w", environmentId, err)
	}
	return nil
}

func hostFromUrl(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid organization url %q", raw)
	}
	return u.Host, nil
}

func decodeArtifactsResponse(raw []byte) (*lakehouseArtifactsResponseDto, error) {
	trimmed := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	out := lakehouseArtifactsResponseDto{}
	// Response is base64-encoded JSON; fall back to plain JSON if a future API stops encoding it.
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
		if json.Unmarshal(decoded, &out) == nil && out.LakehouseId != "" {
			return &out, nil
		}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
