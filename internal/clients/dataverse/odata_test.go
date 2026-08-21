// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func TestUnitAppendQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		part  *string
		want  string
	}{
		{name: "nil part leaves query untouched", query: "$select=name", part: nil, want: "$select=name"},
		{name: "first part gets no separator", query: "", part: stringPtr("$select=name"), want: "$select=name"},
		{name: "subsequent part is ampersand separated", query: "$select=name", part: stringPtr("$top=5"), want: "$select=name&$top=5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.query
			AppendQuery(&query, tt.part)
			assert.Equal(t, tt.want, query)
		})
	}
}

func TestUnitBuildTotalRowsCountPart(t *testing.T) {
	t.Run("true adds count part and Prefer annotation header", func(t *testing.T) {
		headers := map[string]string{}
		part := BuildTotalRowsCountPart(headers, boolPtr(true))
		require.NotNil(t, part)
		assert.Equal(t, "$count=true", *part)
		assert.Equal(t, `odata.include-annotations="Microsoft.Dynamics.CRM.totalrecordcount,Microsoft.Dynamics.CRM.totalrecordcountlimitexceeded"`, headers["Prefer"])
	})
	t.Run("false returns nil and sets no header", func(t *testing.T) {
		headers := map[string]string{}
		assert.Nil(t, BuildTotalRowsCountPart(headers, boolPtr(false)))
		assert.Empty(t, headers)
	})
	t.Run("nil returns nil and sets no header", func(t *testing.T) {
		headers := map[string]string{}
		assert.Nil(t, BuildTotalRowsCountPart(headers, nil))
		assert.Empty(t, headers)
	})
}

func TestUnitBuildODataParts(t *testing.T) {
	tests := []struct {
		name string
		got  *string
		want *string
	}{
		{name: "select escapes comma-joined columns", got: BuildODataSelectPart([]string{"name", "createdon"}), want: stringPtr("$select=name%2Ccreatedon")},
		{name: "select single column", got: BuildODataSelectPart([]string{"name"}), want: stringPtr("$select=name")},
		{name: "select empty returns nil", got: BuildODataSelectPart([]string{}), want: nil},
		{name: "select nil returns nil", got: BuildODataSelectPart(nil), want: nil},
		{name: "savedQuery escapes value", got: BuildODataSavedQuery(stringPtr("00000000-0000-0000-0000-000000000001")), want: stringPtr("savedQuery=00000000-0000-0000-0000-000000000001")},
		{name: "savedQuery nil returns nil", got: BuildODataSavedQuery(nil), want: nil},
		{name: "userQuery escapes value", got: BuildODataUserQuery(stringPtr("a b")), want: stringPtr("userQuery=a+b")},
		{name: "userQuery nil returns nil", got: BuildODataUserQuery(nil), want: nil},
		{name: "filter escapes spaces and quotes", got: BuildODataFilterPart(stringPtr("name eq 'contoso'")), want: stringPtr("$filter=name+eq+%27contoso%27")},
		{name: "filter nil returns nil", got: BuildODataFilterPart(nil), want: nil},
		{name: "orderby escapes spaces", got: BuildODataOrderByPart(stringPtr("createdon desc")), want: stringPtr("$orderby=createdon+desc")},
		{name: "orderby nil returns nil", got: BuildODataOrderByPart(nil), want: nil},
		{name: "top formats the number", got: BuildODataTopPart(int64Ptr(42)), want: stringPtr("$top=42")},
		{name: "top nil returns nil", got: BuildODataTopPart(nil), want: nil},
		{name: "apply escapes the aggregation", got: BuildOdataApplyPart(stringPtr("groupby((statuscode))")), want: stringPtr("$apply=groupby%28%28statuscode%29%29")},
		{name: "apply nil returns nil", got: BuildOdataApplyPart(nil), want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == nil {
				assert.Nil(t, tt.got)
				return
			}
			require.NotNil(t, tt.got)
			assert.Equal(t, *tt.want, *tt.got)
		})
	}
}

// TestUnitBuildODataQuery_ComposedOrdering pins the composition contract the
// services rely on: parts are appended in call order with "&" separators, and
// each part arrives already URL-encoded.
func TestUnitBuildODataQuery_ComposedOrdering(t *testing.T) {
	headers := map[string]string{}
	query := ""
	AppendQuery(&query, BuildODataSelectPart([]string{"name", "accountid"}))
	AppendQuery(&query, BuildODataFilterPart(stringPtr("name eq 'contoso'")))
	AppendQuery(&query, BuildODataOrderByPart(stringPtr("createdon desc")))
	AppendQuery(&query, BuildODataTopPart(int64Ptr(3)))
	AppendQuery(&query, BuildOdataApplyPart(nil))
	AppendQuery(&query, BuildTotalRowsCountPart(headers, boolPtr(true)))

	assert.Equal(t, "$select=name%2Caccountid&$filter=name+eq+%27contoso%27&$orderby=createdon+desc&$top=3&$count=true", query)
	assert.Contains(t, headers, "Prefer")
}
