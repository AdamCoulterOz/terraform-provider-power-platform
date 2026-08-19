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
	"time"

	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
)

// athenaResourceScope is the token resource the "athena" Synapse Link service requires
// (observed aud in the maker-portal capture). It must be requested explicitly because the generic
// client would otherwise guess the scope from the URL host. NOTE: whether this resource grants
// APP-ONLY (service principal) tokens — the gate for fully unattended Terraform — is unverified;
// the capture only proves delegated (user_impersonation) works.
const athenaResourceScope = "7f15f9d9-cad0-44f1-bbba-d36650e07765/.default"

const athenaOrganizationIdHeader = "x-ms-organization-id"

// The athena unlink endpoint is synchronous. Keep its transport retries inside a short,
// operation-local ceiling so a deterministic service failure does not consume the provider's
// full resource-operation timeout. A shorter caller deadline still wins.
const fabricLinkDeleteRetryTimeout = 2 * time.Minute

const maxFabricLinkErrorBodyBytes = 2048

func newFabricLinkClient(apiClient *api.Client) client {
	return client{
		Api:                apiClient,
		deleteRetryTimeout: fabricLinkDeleteRetryTimeout,
	}
}

type client struct {
	Api                *api.Client
	deleteRetryTimeout time.Duration
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
// The prefix before the cluster uriSuffix is the env's AZURE REGION code: "e" = Australia East,
// "se" = Australia Southeast (same cluster uriSuffix "au-il301...", different region). acdev's env
// is Australia East -> "e" (athenawebservice.eau-il301...). Verified against the live athena POST
// as the hatch user. TODO: derive the prefix from env.Properties.AzureRegion (or make it
// overridable) so non-East-AU regions resolve correctly instead of hardcoding "e".
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

	// athena establishes the Dataverse organization context from the x-ms-organization-id HEADER (not the
	// body's OrganizationId). Without it the POST fails 404 "DatalakefolderNotFoundException ... the
	// organization context does not have an OrganizationId". The maker portal sends this header; mirror it.
	headers := make(http.Header)
	headers.Set(athenaOrganizationIdHeader, env.Properties.LinkedEnvironmentMetadata.ResourceId)
	// A 403 (empty body) on an org that has NEVER been linked means the org isn't registered with the
	// athena island yet. The maker portal wizard handles this exact case: 403 -> POST
	// updateorganizationdetails -> retry (HAR-verified on a virgin org). Accept 403 here so we can
	// replicate that self-heal instead of failing the create.
	resp, err := client.Api.Execute(ctx, []string{athenaResourceScope}, "POST", apiUrl.String(), headers, body, []int{http.StatusOK, http.StatusCreated, http.StatusForbidden}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Link to Fabric for environment %s: %w", environmentId, err)
	}
	if resp.HttpResponse.StatusCode == http.StatusForbidden {
		if regErr := client.updateOrganizationDetails(ctx, environmentId, env); regErr != nil {
			return nil, fmt.Errorf("Link to Fabric returned 403 for environment %s (organization not yet registered with the athena island) and registering it failed: %w", environmentId, regErr)
		}
		resp, err = client.Api.Execute(ctx, []string{athenaResourceScope}, "POST", apiUrl.String(), headers, body, []int{http.StatusOK, http.StatusCreated}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Link to Fabric for environment %s after registering the organization with the athena island: %w", environmentId, err)
		}
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

// updateOrganizationDetails registers/refreshes the organization's record with the athena island.
// Required exactly once per org before its first Link to Fabric: on a never-linked org every
// env-scoped athena call returns a bare 403 until this runs (the maker portal wizard fires it on 403
// and retries; request shape copied verbatim from that flow — both query params AND the JSON body
// echoing the headers).
func (client *client) updateOrganizationDetails(ctx context.Context, environmentId string, env *bapEnvironmentDto) error {
	host, err := athenaHost(env.Properties.Cluster.UriSuffix)
	if err != nil {
		return err
	}
	orgId := env.Properties.LinkedEnvironmentMetadata.ResourceId
	values := url.Values{}
	values.Add("organizationUrl", env.Properties.LinkedEnvironmentMetadata.InstanceURL)
	values.Add("organizationId", orgId)
	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     host,
		Path:     fmt.Sprintf("/environment/%s/updateorganizationdetails", environmentId),
		RawQuery: values.Encode(),
	}
	headers := make(http.Header)
	headers.Set(athenaOrganizationIdHeader, orgId)
	body := map[string]map[string]string{"headers": {"x-ms-organization-id": orgId}}
	if _, err := client.Api.Execute(ctx, []string{athenaResourceScope}, "POST", apiUrl.String(), headers, body, []int{http.StatusOK, http.StatusNoContent}, nil); err != nil {
		return fmt.Errorf("failed to register organization %s with the athena island: %w", orgId, err)
	}
	return nil
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

	// athena requires the organization context on every environment-scoped call. Omitting this
	// header leaves the unlink request unable to resolve the Dataverse organization even though the
	// environment id is present in the path.
	headers := make(http.Header)
	headers.Set(athenaOrganizationIdHeader, env.Properties.LinkedEnvironmentMetadata.ResourceId)

	retryTimeout := client.deleteRetryTimeout
	if retryTimeout <= 0 {
		retryTimeout = fabricLinkDeleteRetryTimeout
	}
	deleteCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	// A 404 is not sufficient proof that the link is absent: athena also uses not-found responses
	// when it cannot resolve organization/folder context. Until Read can verify live absence, fail
	// loudly instead of silently dropping Terraform state.
	resp, err := client.Api.Execute(deleteCtx, []string{athenaResourceScope}, "DELETE", apiUrl.String(), headers, nil, []int{http.StatusOK, http.StatusNoContent}, nil)
	if err != nil {
		return fmt.Errorf(
			"failed to unlink Link to Fabric for environment %s after retrying within %s%s: %w",
			environmentId,
			retryTimeout,
			formatLastFabricLinkHttpFailure(resp),
			err,
		)
	}
	return nil
}

func formatLastFabricLinkHttpFailure(resp *api.Response) string {
	if resp == nil || resp.HttpResponse == nil {
		return ""
	}

	body := strings.TrimSpace(string(resp.BodyAsBytes))
	if len(body) > maxFabricLinkErrorBodyBytes {
		body = body[:maxFabricLinkErrorBodyBytes] + "..."
	}
	if body == "" {
		return fmt.Sprintf("; last HTTP status %d (%s)", resp.HttpResponse.StatusCode, resp.HttpResponse.Status)
	}
	return fmt.Sprintf("; last HTTP status %d (%s), body %q", resp.HttpResponse.StatusCode, resp.HttpResponse.Status, body)
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
