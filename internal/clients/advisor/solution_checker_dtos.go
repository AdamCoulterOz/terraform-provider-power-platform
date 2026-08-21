// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package advisor

// RuleDto is a single solution checker rule as returned by the Advisor rule
// API.
type RuleDto struct {
	Code            string `json:"code"`
	Description     string `json:"description"`
	Summary         string `json:"summary"`
	HowToFix        string `json:"howToFix"`
	GuidanceUrl     string `json:"guidanceUrl"`
	ComponentType   int    `json:"componentType"`
	PrimaryCategory int    `json:"primaryCategory"`
	Include         bool   `json:"include"`
	Severity        int    `json:"severity"`
}
