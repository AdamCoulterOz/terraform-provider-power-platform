// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package managedsolution

import "encoding/xml"

type solutionPackageXML struct {
	XMLName          xml.Name                   `xml:"ImportExportXml"`
	SolutionManifest solutionPackageManifestXML `xml:"SolutionManifest"`
}

type solutionPackageManifestXML struct {
	UniqueName          string                         `xml:"UniqueName"`
	Version             string                         `xml:"Version"`
	Managed             int                            `xml:"Managed"`
	LocalizedNames      []solutionLocalizedNameXML     `xml:"LocalizedNames>LocalizedName"`
	MissingDependencies solutionMissingDependenciesXML `xml:"MissingDependencies"`
}

type solutionLocalizedNameXML struct {
	Description string `xml:"description,attr"`
}

type solutionMissingDependenciesXML struct {
	MissingDependencies []solutionMissingDependencyXML `xml:"MissingDependency"`
}

type solutionMissingDependencyXML struct {
	Required []solutionRequiredDependencyXML `xml:"Required"`
}

type solutionRequiredDependencyXML struct {
	Type       string `xml:"type,attr"`
	SchemaName string `xml:"schemaName,attr"`
	Solution   string `xml:"solution,attr"`
}

type solutionCustomizationsXML struct {
	XMLName              xml.Name                         `xml:"ImportExportXml"`
	ConnectionReferences []solutionConnectionReferenceXML `xml:"connectionreferences>connectionreference"`
}

type solutionConnectionReferenceXML struct {
	LogicalName string `xml:"connectionreferencelogicalname,attr"`
	ConnectorID string `xml:"connectorid"`
}

type environmentVariableDefinitionXML struct {
	XMLName      xml.Name `xml:"environmentvariabledefinition"`
	SchemaName   string   `xml:"schemaname,attr"`
	DefaultValue *string  `xml:"defaultvalue"`
}

type environmentVariableValuesJSON struct {
	EnvironmentVariableValues environmentVariableValueContainerJSON `json:"environmentvariablevalues"`
}

type environmentVariableValueContainerJSON struct {
	EnvironmentVariableValue *environmentVariableValueJSON `json:"environmentvariablevalue"`
}

type environmentVariableValueJSON struct {
	Value *string `json:"value"`
}

type environmentVariableDefinitionsResponseDto struct {
	Value []environmentVariableDefinitionDto `json:"value"`
}

type environmentVariableDefinitionDto struct {
	SchemaName string `json:"schemaname"`
	IsManaged  bool   `json:"ismanaged"`
}

type packageEnvironmentVariable struct {
	SchemaName            string
	HasDefaultValue       bool
	ContainsPackagedValue bool
}

type packageConnectionReference struct {
	LogicalName string
	ConnectorID string
}

type solutionPackage struct {
	UniqueName           string
	DisplayName          string
	Version              string
	IsManaged            bool
	ConnectionReferences map[string]packageConnectionReference
	EnvironmentVariables map[string]packageEnvironmentVariable
	Dependencies         map[string]string
}
