// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"net/url"
	"strconv"
	"strings"
)

// AppendQuery appends an optional OData query part to query, separating parts
// with "&".
func AppendQuery(query, part *string) {
	if part != nil {
		if len(*query) > 0 {
			*query += "&"
		}
		*query = strings.Join([]string{*query, *part}, "")
	}
}

// BuildTotalRowsCountPart returns the $count query part and adds the Prefer
// header that requests total record count annotations.
func BuildTotalRowsCountPart(headers map[string]string, returnTotalRowsCount *bool) *string {
	if returnTotalRowsCount != nil && *returnTotalRowsCount {
		headers["Prefer"] = "odata.include-annotations=\"Microsoft.Dynamics.CRM.totalrecordcount,Microsoft.Dynamics.CRM.totalrecordcountlimitexceeded\""
		countTrueString := "$count=true"
		return &countTrueString
	}
	return nil
}

// BuildODataSelectPart returns the $select query part.
func BuildODataSelectPart(selectPart []string) *string {
	resultQuery := ""
	if len(selectPart) > 0 {
		resultQuery = "$select=" + url.QueryEscape(strings.Join(selectPart, ","))
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}

// BuildODataSavedQuery returns the savedQuery query part.
func BuildODataSavedQuery(savedQuery *string) *string {
	resultQuery := ""
	if savedQuery != nil {
		resultQuery = strings.Join([]string{"savedQuery=", url.QueryEscape(*savedQuery)}, "")
		return &resultQuery
	}
	return nil
}

// BuildODataUserQuery returns the userQuery query part.
func BuildODataUserQuery(userQuery *string) *string {
	resultQuery := ""
	if userQuery != nil {
		resultQuery = strings.Join([]string{"userQuery=", url.QueryEscape(*userQuery)}, "")
		return &resultQuery
	}
	return nil
}

// BuildODataFilterPart returns the $filter query part.
func BuildODataFilterPart(filter *string) *string {
	resultQuery := ""
	if filter != nil {
		resultQuery = "$filter=" + url.QueryEscape(*filter)
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}

// BuildODataOrderByPart returns the $orderby query part.
func BuildODataOrderByPart(orderBy *string) *string {
	resultQuery := ""
	if orderBy != nil {
		resultQuery = "$orderby=" + url.QueryEscape(*orderBy)
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}

// BuildODataTopPart returns the $top query part.
func BuildODataTopPart(top *int64) *string {
	resultQuery := ""
	if top != nil {
		resultQuery = "$top=" + url.QueryEscape(strconv.Itoa(int(*top)))
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}

// BuildOdataApplyPart returns the $apply query part.
func BuildOdataApplyPart(apply *string) *string {
	resultQuery := ""
	if apply != nil {
		resultQuery = "$apply=" + url.QueryEscape(*apply)
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}
