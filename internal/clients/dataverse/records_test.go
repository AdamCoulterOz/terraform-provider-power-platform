// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package dataverse

import (
	"context"
	"net/http"
	"testing"

	"github.com/microsoft/terraform-provider-power-platform/internal/customerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entityDefinitionStep scripts the standard EntityDefinitions response used by
// the record operations that resolve a table's collection name first.
func entityDefinitionStep(primaryIdAttribute, logicalCollectionName string) step {
	return step{
		status: http.StatusOK,
		body:   `{"PrimaryIdAttribute":"` + primaryIdAttribute + `","LogicalCollectionName":"` + logicalCollectionName + `","MetadataId":"00000000-0000-0000-0000-00000000000d"}`,
	}
}

// entityDefinitionURL is the exact wire URL of a GetEntityDefinition request:
// the parenthesised key is percent-encoded by url.URL and the $select is
// carried in the fragment.
func entityDefinitionURL(entityLogicalName string) string {
	return "https://env.crm.dynamics.com/api/data/v9.2/EntityDefinitions%28LogicalName=%27" + entityLogicalName + "%27%29#$select=PrimaryIdAttribute,LogicalCollectionName"
}

func odataEntityIDHeader(url string) http.Header {
	return http.Header{"OData-EntityId": []string{url}}
}

func TestUnitGetEntityDefinition_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, entityDefinitionStep("accountid", "accounts"))

	definition, err := client.GetEntityDefinition(context.Background(), testEnvironmentHost, "account")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
	})
	assert.Equal(t, "accountid", definition.PrimaryIDAttribute)
	assert.Equal(t, "accounts", definition.LogicalCollectionName)
}

func TestUnitGetEntityDefinition_UnexpectedStatusIsWrapped(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusBadRequest, body: `{"error":"nope"}`})

	_, err := client.GetEntityDefinition(context.Background(), testEnvironmentHost, "account")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get entity definition for account:")
	var statusErr customerrors.UnexpectedHttpStatusCodeError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadRequest, statusErr.StatusCode)
}

func TestUnitGetDataRecordsByODataQuery_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{
		status: http.StatusOK,
		body: `{
			"@odata.context": "https://env.crm.dynamics.com/api/data/v9.2/$metadata#accounts(name)",
			"@Microsoft.Dynamics.CRM.totalrecordcount": 2,
			"@Microsoft.Dynamics.CRM.totalrecordcountlimitexceeded": false,
			"value": [{"name": "a"}, {"name": "b"}]
		}`,
	})

	response, err := client.GetDataRecordsByODataQuery(context.Background(), testEnvironmentHost, "accounts?$select=name", map[string]string{"Prefer": "odata.maxpagesize=2"})

	require.NoError(t, err)
	// The caller's raw OData query is appended verbatim after the api segment.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts?$select=name"},
	})
	assert.Equal(t, "odata.maxpagesize=2", (*recorded)[0].header.Get("Prefer"), "caller headers must reach the wire")

	require.Len(t, response.Records, 2)
	require.NotNil(t, response.TotalRecord)
	assert.Equal(t, int64(2), *response.TotalRecord)
	require.NotNil(t, response.TotalRecordLimitExceeded)
	assert.False(t, *response.TotalRecordLimitExceeded)
	assert.Equal(t, "https://env.crm.dynamics.com/api/data/v9.2/$metadata#accounts(name)", response.TableMetadataUrl)
	assert.Equal(t, "accounts", response.TablePluralName, "plural name is cut at the first '(' of the context")
}

func TestUnitGetDataRecordsByODataQuery_SingleRecordResponse(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{
		status: http.StatusOK,
		body:   `{"@odata.context": "https://env.crm.dynamics.com/api/data/v9.2/$metadata#accounts/$entity", "accountid": "00000000-0000-0000-0000-000000000001"}`,
	})

	response, err := client.GetDataRecordsByODataQuery(context.Background(), testEnvironmentHost, "accounts(00000000-0000-0000-0000-000000000001)", nil)

	require.NoError(t, err)
	require.Len(t, response.Records, 1, "a response without 'value' is a single record")
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", response.Records[0]["accountid"])
	assert.Equal(t, "accounts", response.TablePluralName, "plural name is cut at the first '/' of the context")
	assert.Nil(t, response.TotalRecord)
	assert.Nil(t, response.TotalRecordLimitExceeded)
}

func TestUnitGetDataRecordsByODataQuery_MissingContextFails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, step{status: http.StatusOK, body: `{"value": []}`})

	_, err := client.GetDataRecordsByODataQuery(context.Background(), testEnvironmentHost, "accounts", nil)

	require.Error(t, err)
	assert.EqualError(t, err, "@odata.context field missing from response")
}

func TestUnitGetDataRecord_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusOK, body: `{"accountid":"00000000-0000-0000-0000-000000000001","name":"Contoso"}`},
	)

	record, err := client.GetDataRecord(context.Background(), "00000000-0000-0000-0000-000000000001", testEnvironmentHost, "account")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2800000000-0000-0000-0000-000000000001%29"},
	})
	assert.Equal(t, "Contoso", record["name"])
}

func TestUnitGetRelationData_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusOK, body: `{"value":[{"contactid":"00000000-0000-0000-0000-000000000002"}]}`},
	)

	value, err := client.GetRelationData(context.Background(), testEnvironmentHost, "account", "00000000-0000-0000-0000-000000000001", "contact_customer_accounts")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2800000000-0000-0000-0000-000000000001%29/contact_customer_accounts?$select=createdon"},
	})
	require.Len(t, value, 1)
}

func TestUnitGetRelationData_MissingValueFails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusOK, body: `{}`},
	)

	_, err := client.GetRelationData(context.Background(), testEnvironmentHost, "account", "00000000-0000-0000-0000-000000000001", "contact_customer_accounts")

	require.Error(t, err)
	assert.EqualError(t, err, "value field not found in result when retrieving relational data")
}

func TestUnitGetTableSingularNameFromPlural_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"LogicalName":"account","PrimaryIdAttribute":"accountid","LogicalCollectionName":"accounts"}]}`})

	singular, err := client.GetTableSingularNameFromPlural(context.Background(), testEnvironmentHost, "accounts")

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/EntityDefinitions?%24filter=LogicalCollectionName+eq+%27accounts%27&%24select=PrimaryIdAttribute%2CLogicalCollectionName%2CLogicalName"},
	})
	require.NotNil(t, singular)
	assert.Equal(t, "account", *singular)
}

func TestUnitGetEntityAttributesDefinition_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t, step{status: http.StatusOK, body: `{"value":[{"LogicalName":"name","MetadataId":"00000000-0000-0000-0000-00000000000a"}]}`})

	attributes, err := client.GetEntityAttributesDefinition(context.Background(), testEnvironmentHost, "account")

	require.NoError(t, err)
	// This operation formats the URL as a raw string, so the parentheses and
	// quotes stay unescaped on the wire (unlike the url.URL-built requests).
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/EntityDefinitions(LogicalName='account')/Attributes?$select=LogicalName"},
	})
	require.Len(t, attributes, 1)
	assert.Equal(t, "name", attributes[0].LogicalName)
}

func TestUnitGetEntityRelationDefinitionInfo_ResolvesRelationKinds(t *testing.T) {
	tests := []struct {
		name         string
		entity       string
		relation     string
		responseBody string
		wantTable    string
	}{
		{
			name:     "one-to-many referencing navigation property",
			entity:   "account",
			relation: "contact_customer_accounts",
			responseBody: `{
				"OneToManyRelationships": [{"ReferencingEntityNavigationPropertyName": "contact_customer_accounts", "Entity1LogicalName": "contact", "ReferencedEntity": "contact"}],
				"ManyToOneRelationships": [],
				"ManyToManyRelationships": []
			}`,
			wantTable: "contact",
		},
		{
			name:     "many-to-one referencing navigation property",
			entity:   "contact",
			relation: "parentcustomerid_account",
			responseBody: `{
				"OneToManyRelationships": [],
				"ManyToOneRelationships": [{"ReferencingEntityNavigationPropertyName": "parentcustomerid_account", "ReferencedEntity": "account"}],
				"ManyToManyRelationships": []
			}`,
			wantTable: "account",
		},
		{
			name:     "many-to-many second entity navigation property",
			entity:   "account",
			relation: "accountleads_association",
			responseBody: `{
				"OneToManyRelationships": [],
				"ManyToOneRelationships": [],
				"ManyToManyRelationships": [{"Entity2NavigationPropertyName": "accountleads_association", "Entity2LogicalName": "lead"}]
			}`,
			wantTable: "lead",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			recorded := activateRecorder(t, step{status: http.StatusOK, body: tt.responseBody})

			table, err := client.GetEntityRelationDefinitionInfo(context.Background(), testEnvironmentHost, tt.entity, tt.relation)

			require.NoError(t, err)
			assertRequests(t, recorded, []recordedRequest{
				{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/EntityDefinitions(LogicalName='" + tt.entity + "')?$expand=OneToManyRelationships,ManyToManyRelationships,ManyToOneRelationships"},
			})
			assert.Equal(t, tt.wantTable, table)
		})
	}
}

func TestUnitApplyDataRecord_Create_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/accounts(11111111-2222-3333-4444-555555555555)")},
	)

	record, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", map[string]any{"name": "Contoso"})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts", body: `{"name":"Contoso"}`},
	})
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", record.Id, "id must be parsed out of the OData-EntityId header")
}

func TestUnitApplyDataRecord_UpdateByRecordId_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/accounts(33333333-3333-3333-3333-333333333333)")},
	)

	record, err := client.ApplyDataRecord(context.Background(), "33333333-3333-3333-3333-333333333333", testEnvironmentHost, "account", map[string]any{"name": "Contoso"})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2833333333-3333-3333-3333-333333333333%29", body: `{"name":"Contoso"}`},
	})
	assert.Equal(t, "33333333-3333-3333-3333-333333333333", record.Id)
}

func TestUnitApplyDataRecord_UpdateByPrimaryIdColumn_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/accounts(44444444-4444-4444-4444-444444444444)")},
	)

	columns := map[string]any{
		"accountid": "44444444-4444-4444-4444-444444444444",
		"name":      "Contoso",
	}
	record, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", columns)

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2844444444-4444-4444-4444-444444444444%29", body: `{"accountid":"44444444-4444-4444-4444-444444444444","name":"Contoso"}`},
	})
	assert.Equal(t, "44444444-4444-4444-4444-444444444444", record.Id)
}

func TestUnitApplyDataRecord_SingleValuedRelationBecomesODataBind(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("contactid", "contacts"),
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/accounts(11111111-2222-3333-4444-555555555555)")},
	)

	columns := map[string]any{
		"name": "Contoso",
		"primarycontactid": map[string]any{
			"table_logical_name": "contact",
			"data_record_id":     "55555555-5555-5555-5555-555555555555",
		},
	}
	record, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", columns)

	require.NoError(t, err)
	// The nested map is resolved against the related table's entity definition
	// first, then rewritten into a single-valued @odata.bind column.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("contact")},
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts", body: `{"name":"Contoso","primarycontactid@odata.bind":"/contacts(55555555-5555-5555-5555-555555555555)"}`},
	})
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", record.Id)
}

func TestUnitApplyDataRecord_CollectionRelationSyncsRefs(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent, header: odataEntityIDHeader("https://env.crm.dynamics.com/api/data/v9.2/accounts(77777777-7777-7777-7777-777777777777)")},
		step{status: http.StatusOK, body: `{"value":[{"@odata.id":"https://env.crm.dynamics.com/api/data/v9.2/contacts(88888888-8888-8888-8888-888888888888)"}]}`},
		entityDefinitionStep("contactid", "contacts"),
		step{status: http.StatusNoContent},
		entityDefinitionStep("contactid", "contacts"),
		step{status: http.StatusNoContent},
	)

	columns := map[string]any{
		"contact_customer_accounts": []any{
			map[string]any{
				"table_logical_name": "contact",
				"data_record_id":     "66666666-6666-6666-6666-666666666666",
			},
		},
	}
	record, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", columns)

	require.NoError(t, err)
	// The collection-valued relation is reconciled after the create: read the
	// existing $ref list, detach the stale relation by its raw @odata.id, then
	// attach the desired one.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts", body: `{}`},
		{method: "GET", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2877777777-7777-7777-7777-777777777777%29/contact_customer_accounts/$ref"},
		{method: "GET", url: entityDefinitionURL("contact")},
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/contacts(88888888-8888-8888-8888-888888888888)"},
		{method: "GET", url: entityDefinitionURL("contact")},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%2877777777-7777-7777-7777-777777777777%29/contact_customer_accounts/$ref", body: `{"@odata.id":"https://env.crm.dynamics.com/api/data/v9.2/contacts(66666666-6666-6666-6666-666666666666)"}`},
	})
	assert.Equal(t, "77777777-7777-7777-7777-777777777777", record.Id)
}

func TestUnitApplyDataRecord_PreconditionFailedRetriesAsPatch(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusPreconditionFailed, body: `{"error":"record exists"}`},
		step{status: http.StatusOK, body: `{"id":"99999999-9999-9999-9999-999999999999"}`},
	)

	record, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", map[string]any{"name": "Contoso"})

	require.NoError(t, err)
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "POST", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts", body: `{"name":"Contoso"}`},
		{method: "PATCH", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts", body: `{"name":"Contoso"}`},
	})
	assert.Equal(t, "99999999-9999-9999-9999-999999999999", record.Id, "id must be read from the response body when one is returned")
}

func TestUnitApplyDataRecord_PreconditionFailedThenMethodNotAllowed(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusPreconditionFailed, body: `{"error":"record exists"}`},
		step{status: http.StatusMethodNotAllowed, body: `{"error":"PATCH on a collection"}`},
	)

	_, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", map[string]any{"name": "Contoso"})

	require.Error(t, err)
	assert.EqualError(t, err, "record already exists. To update an existing record, primaryId must be provided in the columns attribute")
}

func TestUnitApplyDataRecord_NoEntityIdReturnedFails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent},
	)

	_, err := client.ApplyDataRecord(context.Background(), "", testEnvironmentHost, "account", map[string]any{"name": "Contoso"})

	require.Error(t, err)
	assert.EqualError(t, err, "no entity record id returned from the API")
}

func TestUnitDeleteDataRecord_RequestShape(t *testing.T) {
	client := newTestClient(t)
	recorded := activateRecorder(t,
		entityDefinitionStep("accountid", "accounts"),
		step{status: http.StatusNoContent},
		step{status: http.StatusNoContent},
	)

	columns := map[string]any{
		"name": "Contoso",
		"contact_customer_accounts": []any{
			map[string]any{
				"table_logical_name": "contact",
				"data_record_id":     "bbbb2222-bbbb-2222-bbbb-222222222222",
			},
		},
	}
	err := client.DeleteDataRecord(context.Background(), "aaaa1111-aaaa-1111-aaaa-111111111111", testEnvironmentHost, "account", columns)

	require.NoError(t, err)
	// Collection-valued relations are detached one by one before the record
	// itself is deleted; the remaining scalar columns ride along as the body.
	assertRequests(t, recorded, []recordedRequest{
		{method: "GET", url: entityDefinitionURL("account")},
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%28aaaa1111-aaaa-1111-aaaa-111111111111%29/contact_customer_accounts%28bbbb2222-bbbb-2222-bbbb-222222222222%29/$ref"},
		{method: "DELETE", url: "https://env.crm.dynamics.com/api/data/v9.2/accounts%28aaaa1111-aaaa-1111-aaaa-111111111111%29", body: `{"name":"Contoso"}`},
	})
}

func TestUnitDeleteDataRecord_MissingRelationRecordIdFails(t *testing.T) {
	client := newTestClient(t)
	activateRecorder(t, entityDefinitionStep("accountid", "accounts"))

	columns := map[string]any{
		"contact_customer_accounts": []any{
			map[string]any{"table_logical_name": "contact"},
		},
	}
	err := client.DeleteDataRecord(context.Background(), "aaaa1111-aaaa-1111-aaaa-111111111111", testEnvironmentHost, "account", columns)

	require.Error(t, err)
	assert.EqualError(t, err, "data_record_id field is missing or not a string")
}
