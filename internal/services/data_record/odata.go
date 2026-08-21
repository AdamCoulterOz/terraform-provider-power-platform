// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.
package data_record

import (
	"fmt"
	"strings"

	"github.com/microsoft/terraform-provider-power-platform/internal/clients/dataverse"
)

func BuildODataQueryFromModel(model *DataRecordListDataSourceModel) (string, map[string]string, error) {
	var resultQuery = ""
	var headers = make(map[string]string)

	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataSavedQuery(model.SavedQuery.ValueStringPointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataUserQuery(model.UserQuery.ValueStringPointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataSelectPart(model.Select))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataFilterPart(model.Filter.ValueStringPointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildOdataApplyPart(model.Apply.ValueStringPointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataOrderByPart(model.OrderBy.ValueStringPointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildODataTopPart(model.Top.ValueInt64Pointer()))
	dataverse.AppendQuery(&resultQuery, dataverse.BuildTotalRowsCountPart(headers, model.ReturnTotalRowsCount.ValueBoolPointer()))
	dataverse.AppendQuery(&resultQuery, buildExpandODataQueryPart(model.Expand))

	if len(resultQuery) > 0 {
		return fmt.Sprintf("%s?%s", model.EntityCollection.ValueString(), resultQuery), headers, nil
	}
	return model.EntityCollection.ValueString(), headers, nil
}

func buildExpandODataQueryPart(model []ExpandModel) *string {
	if model == nil {
		return nil
	}

	expandQueryStrings := make([]string, 0)
	for _, m := range model {
		expandString := buildExpandODataQueryPart(m.Expand)
		expandQueryFilterString := buildExpandFilterQueryPart(&m, expandString)

		if expandQueryFilterString != nil {
			expandQueryStrings = append(expandQueryStrings, fmt.Sprintf("%s(%s)", m.NavigationProperty.ValueString(), *expandQueryFilterString))
		} else {
			expandQueryStrings = append(expandQueryStrings, m.NavigationProperty.ValueString())
		}
	}

	if len(expandQueryStrings) > 0 {
		result := strings.Join(expandQueryStrings, ",")
		result = "$expand=" + strings.TrimSuffix(result, ",")
		return &result
	}
	return nil
}

func buildExpandFilterQueryPart(model *ExpandModel, subExpandValueString *string) *string {
	resultQuery := ""

	selectString := dataverse.BuildODataSelectPart(model.Select)
	if selectString != nil {
		resultQuery = strings.Join([]string{resultQuery, *selectString}, "")
	}
	filterString := dataverse.BuildODataFilterPart(model.Filter.ValueStringPointer())
	if filterString != nil {
		if len(resultQuery) > 0 {
			resultQuery += ";"
		}
		resultQuery = strings.Join([]string{resultQuery, *filterString}, "")
	}
	orderByString := dataverse.BuildODataOrderByPart(model.OrderBy.ValueStringPointer())
	if orderByString != nil {
		if len(resultQuery) > 0 {
			resultQuery += ";"
		}
		resultQuery = strings.Join([]string{resultQuery, *orderByString}, "")
	}

	topString := dataverse.BuildODataTopPart(model.Top.ValueInt64Pointer())
	if topString != nil {
		if len(resultQuery) > 0 {
			resultQuery += ";"
		}
		resultQuery = strings.Join([]string{resultQuery, *topString}, "")
	}
	if subExpandValueString != nil {
		if len(resultQuery) > 0 {
			resultQuery += ";"
		}
		resultQuery = strings.Join([]string{resultQuery, *subExpandValueString}, "")
	}
	if resultQuery == "" {
		return nil
	}
	return &resultQuery
}
