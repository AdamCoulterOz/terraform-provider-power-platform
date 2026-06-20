// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package fabric_link

// Link to Fabric provisioning, reverse-engineered from the maker-portal wizard (HAR capture).
// It is NOT a direct `synapselinkprofile` Web API create — the portal drives an orchestration
// service ("athena") plus a Power Query connection. The synapselinkprofile/datalakefolder rows
// are created server-side by athena.
//
// Flow:
//  1. GET  api.bap.microsoft.com .../environments/{environmentId}?api-version=2020-10-01-alpha
//     -> org id/url/unique/friendly name + cluster.uriSuffix (for the athena host).
//  2. PUT  {geo}.prod.powerquery.microsoft.com/api/connections/upsertConnection
//     -> a CommonDataService connection with hostContext type "TridentDataverseOneLake";
//        returns the ConnectionId. (SP-auth variant is a TODO — capture showed a user account.)
//  3. POST athenawebservice.e{cluster.uriSuffix}.powerapps.com/environment/{environmentId}/lakehouseArtifacts?dxt=false
//     body = lakehouseArtifactsRequestDto; response = BASE64-encoded lakehouseArtifactsResponseDto
//     ({WorkspaceId, LakehouseId, ConnectionId}).

// bapEnvironmentDto is the subset of the BAP environment GET we need.
type bapEnvironmentDto struct {
	Properties struct {
		AzureRegion string `json:"azureRegion"`
		Cluster     struct {
			UriSuffix    string `json:"uriSuffix"`    // e.g. "au-il301.gateway.prod.island"
			GeoShortName string `json:"geoShortName"` // e.g. "AU"
		} `json:"cluster"`
		LinkedEnvironmentMetadata struct {
			ResourceId   string `json:"resourceId"`   // Dataverse OrganizationId
			InstanceURL  string `json:"instanceUrl"`  // OrganizationUrl
			FriendlyName string `json:"friendlyName"` // EnvironmentFriendlyName
			UniqueName   string `json:"uniqueName"`   // EnvironmentUniqueName
		} `json:"linkedEnvironmentMetadata"`
	} `json:"properties"`
}

// lakehouseArtifactsRequestDto is the POST body that provisions the link.
type lakehouseArtifactsRequestDto struct {
	OrganizationId          string                 `json:"OrganizationId"`
	OrganizationUrl         string                 `json:"OrganizationUrl"`
	EnvironmentFriendlyName string                 `json:"EnvironmentFriendlyName"`
	EnvironmentUniqueName   string                 `json:"EnvironmentUniqueName"`
	Entities                []string               `json:"Entities"`
	IsManagedLake           bool                   `json:"IsManagedLake"`
	WorkspaceId             string                 `json:"WorkspaceId"` // target Fabric workspace
	ConnectionId            string                 `json:"ConnectionId"`
	EntityDescriptions      []entityDescriptionDto `json:"EntityDescriptions"`
}

type entityDescriptionDto struct {
	Type         string `json:"Type"`
	EntitySource string `json:"EntitySource"` // "Dataverse"
}

// lakehouseArtifactsResponseDto is the (base64-wrapped) response.
type lakehouseArtifactsResponseDto struct {
	WorkspaceId  string `json:"WorkspaceId"`
	LakehouseId  string `json:"LakehouseId"`
	ConnectionId string `json:"ConnectionId"`

	// Resolved after create (not in the athena response). The unlink DELETE targets the
	// datalakefolder id, so it is captured here and persisted for Delete.
	DatalakeFolderId string `json:"-"`
}

// datalakeFolderListDto is the Dataverse Web API datalakefolders query used to resolve the
// datalakefolder id that the unlink DELETE targets.
type datalakeFolderListDto struct {
	Value []struct {
		DatalakeFolderId         string `json:"datalakefolderid"`
		DatalakeFolderUniqueName string `json:"datalakefolder_uniquename"`
	} `json:"value"`
}

// The Dataverse->OneLake connection is NOT created here — it is a standard CommonDataService
// connection created by the existing `powerplatform_connection` resource (connector
// `shared_commondataserviceforapps`, service-principal connection parameters), whose `id` is passed
// to this resource as `connection_id`. athena's lakehouseArtifacts call binds that connection to the
// target Fabric workspace via the ConnectionId + WorkspaceId it receives.
