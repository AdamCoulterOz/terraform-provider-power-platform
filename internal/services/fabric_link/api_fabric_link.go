// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/microsoft/terraform-provider-power-platform/internal/api"
	"github.com/microsoft/terraform-provider-power-platform/internal/constants"
	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
)

// athenaResourceScope is the token resource the "athena" Synapse Link service requires
// (observed aud in the maker-portal capture). It must be requested explicitly because the generic
// client would otherwise guess the scope from the URL host. The provisioning path requires a
// delegated token in the currently verified SCH deployment and is therefore invoked through the
// contained username/password provider alias rather than the default app-only pipeline identity.
const athenaResourceScope = "7f15f9d9-cad0-44f1-bbba-d36650e07765/.default"

// operation-local ceiling so a deterministic service failure does not consume the provider's
// full resource-operation timeout. A shorter caller deadline still wins.
const athenaOrganizationIdHeader = "x-ms-organization-id"

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
	// Every BAP api-version from 2020-10-01 onward returns cluster.uriSuffix, azureRegion and the
	// full linkedEnvironmentMetadata identically (probed 2020-10-01 through 2024-05-01), so this read
	// uses the provider-wide stable version rather than an alpha one.
	values.Add(constants.API_VERSION_PARAM, constants.BAP_API_VERSION)
	apiUrl.RawQuery = values.Encode()

	env := bapEnvironmentDto{}
	resp, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK, http.StatusNotFound}, &env)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment %s: %w", environmentId, err)
	}
	if resp != nil && resp.HttpResponse.StatusCode == http.StatusNotFound {
		return nil, customerrors.WrapIntoProviderError(err, customerrors.ErrorCode(constants.ERROR_OBJECT_NOT_FOUND), fmt.Sprintf("environment %s not found", environmentId))
	}
	return &env, nil
}

// athenaHost builds the Synapse Link ("athena") orchestration host for an environment:
//
//	athenawebservice.{azureRegionPrefix}{cluster.uriSuffix}.powerapps.com
//
// The two fields concatenate with no separator, which is why the result reads as though a stray
// letter had been prepended to the cluster suffix. azureRegionPrefix is the compass-direction
// component of properties.azureRegion with the geography dropped (the geography is already carried
// by the cluster suffix): eastus and australiaeast both give "e", westus and westeurope "w",
// northeurope "n", australiasoutheast "se". Confirmed live against the island gateways for eau-,
// seau-, eus-, wus-, neu- and weu-; a wrong or missing prefix does not produce a useful error, it
// produces NXDOMAIN, so the prefix is derived rather than assumed.
func athenaHost(azureRegion, clusterUriSuffix string) (string, error) {
	if strings.TrimSpace(clusterUriSuffix) == "" {
		return "", errors.New("environment cluster uriSuffix is empty; cannot derive the Link to Fabric (athena) host")
	}
	prefix, err := azureRegionDirectionPrefix(azureRegion)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("athenawebservice.%s%s.powerapps.com", prefix, clusterUriSuffix), nil
}

// compassDirections maps each compass word an Azure region name can carry to the letter it
// contributes to the athena host prefix. A two-word direction contributes both letters in order,
// so "southeast" gives "se" and "northcentral" gives "nc".
var compassDirections = []struct {
	word   string
	letter string
}{
	{"north", "n"},
	{"south", "s"},
	{"east", "e"},
	{"west", "w"},
	{"central", "c"},
}

// azureRegionDirectionPrefix extracts the athena host prefix from an Azure region name.
// Region names carry the direction either last ("australiasoutheast", "southafricanorth") or first
// ("eastus", "northcentralus"), so a trailing direction is matched before a leading one: South
// Africa North is "n", not "s". Both passes take the longest match, so "southcentralus" is "sc".
// A region with no identifiable direction is an error — defaulting to a compass point would build
// a hostname that does not exist.
func azureRegionDirectionPrefix(azureRegion string) (string, error) {
	region := lettersOnly(strings.ToLower(strings.TrimSpace(azureRegion)))
	if region == "" {
		return "", errors.New("environment azureRegion is empty; cannot derive the Link to Fabric (athena) host prefix")
	}
	for i := 1; i < len(region); i++ {
		// starting at 1 keeps a geography in front of the direction: a region name that is nothing
		// but compass words is not a region name.
		if letters, ok := directionLetters(region[i:]); ok {
			return letters, nil
		}
	}
	for i := len(region); i > 0; i-- {
		if letters, ok := directionLetters(region[:i]); ok && i < len(region) {
			return letters, nil
		}
	}
	return "", fmt.Errorf("cannot derive the Link to Fabric (athena) host prefix: no compass direction found in azureRegion %q", azureRegion)
}

// directionLetters decomposes a region fragment into consecutive compass words and returns the
// letters they contribute. It reports false unless the whole fragment is compass words.
func directionLetters(fragment string) (string, bool) {
	if fragment == "" {
		return "", false
	}
	letters := strings.Builder{}
	for fragment != "" {
		matched := false
		for _, direction := range compassDirections {
			if strings.HasPrefix(fragment, direction.word) {
				letters.WriteString(direction.letter)
				fragment = strings.TrimPrefix(fragment, direction.word)
				matched = true
				break
			}
		}
		if !matched {
			return "", false
		}
	}
	return letters.String(), true
}

// lettersOnly drops the ordinals region names carry ("eastus2", "australiacentral2").
func lettersOnly(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		return -1
	}, value)
}

// CreateFabricLink provisions a Link to Fabric and returns the created mirror ids.
// connectionId is the Fabric-to-Dataverse connection to use (see DESIGN.md).
func (client *client) CreateFabricLink(ctx context.Context, environmentId, fabricWorkspaceId, connectionId string, tables []string) (*lakehouseArtifactsResponseDto, error) {
	env, err := client.getBapEnvironment(ctx, environmentId)
	if err != nil {
		return nil, err
	}
	host, err := athenaHost(env.Properties.AzureRegion, env.Properties.Cluster.UriSuffix)
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
	} else {
		tflog.Warn(ctx, fmt.Sprintf("fabric_link: the link was created but its datalakefolder id could not be resolved, so destroy will not be able to unlink it automatically: %s", ferr.Error()))
	}
	return result, nil
}

// updateOrganizationDetails registers/refreshes the organization's record with the athena island.
// Required exactly once per org before its first Link to Fabric: on a never-linked org every
// env-scoped athena call returns a bare 403 until this runs (the maker portal wizard fires it on 403
// and retries; request shape copied verbatim from that flow — both query params AND the JSON body
// echoing the headers).
func (client *client) updateOrganizationDetails(ctx context.Context, environmentId string, env *bapEnvironmentDto) error {
	host, err := athenaHost(env.Properties.AzureRegion, env.Properties.Cluster.UriSuffix)
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

// getDatalakeFolderId resolves the datalakefolder id the unlink DELETE targets.
//
// Picking it is the subtle part. A Dataverse organization carries roughly ten stock datalakefolder
// rows before it has ever been linked to anything — cds2_workspace, cds3_workspace, msdyn_analytics
// and friends are NOT artefacts of a link (an organization with zero synapselinkprofiles was
// observed carrying all of them). Selecting by unique name alone, or falling back to the first row,
// therefore hands DELETE .../lakehouseArtifacts/{id} an unrelated system folder on an organization
// that is not linked, or is linked differently. So the folder is chosen only from those a
// synapselinkprofile actually points at, and no match is an error rather than a guess.
func (client *client) getDatalakeFolderId(ctx context.Context, organizationUrl string) (string, error) {
	host, err := hostFromUrl(organizationUrl)
	if err != nil {
		return "", err
	}

	linkedFolderIds, err := client.getLinkedDatalakeFolderIds(ctx, host)
	if err != nil {
		return "", err
	}
	if len(linkedFolderIds) == 0 {
		return "", errors.New("no synapselinkprofile in this organization references a datalakefolder, so no folder anchors a Link to Fabric; refusing to pick one of the stock datalakefolders (cds2_workspace, cds3_workspace, msdyn_analytics and others exist on every organization and unlinking would delete an unrelated system folder)")
	}

	folders, err := client.listDatalakeFolders(ctx, host)
	if err != nil {
		return "", err
	}

	matches := make([]datalakeFolderDto, 0, len(linkedFolderIds))
	for _, folder := range folders {
		if _, linked := linkedFolderIds[strings.ToLower(strings.TrimSpace(folder.DatalakeFolderId))]; linked {
			matches = append(matches, folder)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("the datalakefolder(s) referenced by this organization's synapselinkprofiles (%s) are not among its datalakefolders (%s); refusing to guess which folder anchors the Link to Fabric",
			strings.Join(sortedFolderIds(linkedFolderIds), ", "), describeDatalakeFolders(folders))
	case 1:
		return matches[0].DatalakeFolderId, nil
	}

	// More than one link exists on the organization (a Fabric link alongside an Azure Synapse one,
	// say). The captured portal unlink of a Link to Fabric named cds3_workspace, so prefer that,
	// then cds2_workspace; anything else is ambiguous and is not guessed at.
	for _, uniqueName := range []string{"cds3_workspace", "cds2_workspace"} {
		for _, folder := range matches {
			if folder.DatalakeFolderUniqueName == uniqueName {
				return folder.DatalakeFolderId, nil
			}
		}
	}
	return "", fmt.Errorf("this organization has more than one linked datalakefolder (%s) and none of them is the Link to Fabric folder (cds3_workspace or cds2_workspace); refusing to guess which one to unlink",
		describeDatalakeFolders(matches))
}

// getLinkedDatalakeFolderIds returns the (lowercased) datalakefolder ids this organization's
// synapselinkprofiles point at. A folder no profile references does not anchor a link.
func (client *client) getLinkedDatalakeFolderIds(ctx context.Context, organizationHost string) (map[string]struct{}, error) {
	apiUrl := &url.URL{
		Scheme: constants.HTTPS,
		Host:   organizationHost,
		Path:   "/api/data/v9.1/synapselinkprofiles",
	}
	// Deliberately no $select: the profile's datalakefolder lookup is read off the row by name, and
	// naming it wrong in $select would fail the whole query with a 400 instead of simply not matching.
	list := synapseLinkProfileListDto{}
	if _, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &list); err != nil {
		return nil, fmt.Errorf("failed to query synapselinkprofiles: %w", err)
	}

	ids := make(map[string]struct{}, len(list.Value))
	for _, profile := range list.Value {
		if id := datalakeFolderIdFromProfile(profile); id != "" {
			ids[strings.ToLower(id)] = struct{}{}
		}
	}
	return ids, nil
}

// datalakeFolderIdFromProfile pulls the datalakefolder lookup out of a synapselinkprofile row.
// OData renders a lookup as "_<name>_value", so keys are normalized before matching.
func datalakeFolderIdFromProfile(profile map[string]any) string {
	for key, value := range profile {
		normalized := strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(key), "_"), "_value")
		if normalized != "datalakefolderid" && normalized != "datalakefolder" {
			continue
		}
		if id, ok := value.(string); ok && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

// listDatalakeFolders reads the organization's datalakefolder rows.
func (client *client) listDatalakeFolders(ctx context.Context, organizationHost string) ([]datalakeFolderDto, error) {
	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     organizationHost,
		Path:     "/api/data/v9.1/datalakefolders",
		RawQuery: "$select=datalakefolderid,datalakefolder_uniquename",
	}
	list := datalakeFolderListDto{}
	if _, err := client.Api.Execute(ctx, nil, "GET", apiUrl.String(), nil, nil, []int{http.StatusOK}, &list); err != nil {
		return nil, fmt.Errorf("failed to query datalakefolders: %w", err)
	}
	return list.Value, nil
}

// describeDatalakeFolders renders folders as "uniquename (id)" for an error message.
func describeDatalakeFolders(folders []datalakeFolderDto) string {
	described := make([]string, 0, len(folders))
	for _, folder := range folders {
		described = append(described, fmt.Sprintf("%s (%s)", folder.DatalakeFolderUniqueName, folder.DatalakeFolderId))
	}
	if len(described) == 0 {
		return "none"
	}
	slices.Sort(described)
	return strings.Join(described, ", ")
}

// sortedFolderIds renders a folder-id set deterministically for an error message.
func sortedFolderIds(set map[string]struct{}) []string {
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// DeleteFabricLink unlinks the Link to Fabric by deleting its datalakefolder via athena.
func (client *client) DeleteFabricLink(ctx context.Context, environmentId, datalakeFolderId string) error {
	if strings.TrimSpace(datalakeFolderId) == "" {
		return errors.New("datalake folder id is empty; cannot unlink (re-import or remove the link manually)")
	}
	env, err := client.getBapEnvironment(ctx, environmentId)
	if err != nil {
		return err
	}
	host, err := athenaHost(env.Properties.AzureRegion, env.Properties.Cluster.UriSuffix)
	if err != nil {
		return err
	}
	apiUrl := &url.URL{
		Scheme:   constants.HTTPS,
		Host:     host,
		Path:     fmt.Sprintf("/environment/%s/lakehouseArtifacts/%s", environmentId, datalakeFolderId),
		RawQuery: "dxt=false",
	}
	retryTimeout := client.deleteRetryTimeout
	if retryTimeout <= 0 {
		retryTimeout = fabricLinkDeleteRetryTimeout
	}
	deleteCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	// athena resolves the Dataverse organization from this header on every environment-scoped call.
	// Without it the unlink cannot resolve the organization even though the environment id is in the path.
	headers := make(http.Header)
	headers.Set(athenaOrganizationIdHeader, env.Properties.LinkedEnvironmentMetadata.ResourceId)

	resp, err := client.Api.Execute(deleteCtx, []string{athenaResourceScope}, "DELETE", apiUrl.String(), headers, nil, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound}, nil)
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
