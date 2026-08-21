// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package ppapi

// GatewayClusterDto represents a gateway cluster in the Power Platform.
type GatewayClusterDto struct {
	ClusterNumber   string `json:"clusterNumber"`
	GeoName         string `json:"geoName"`
	Environment     string `json:"environment"`
	ClusterType     string `json:"clusterType"`
	ClusterCategory string `json:"clusterCategory"`
	ClusterName     string `json:"clusterName"`
	GeoLongName     string `json:"geoLongName"`
}
