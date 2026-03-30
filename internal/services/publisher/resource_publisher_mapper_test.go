// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package publisher

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnitAddressModelFromDto_IgnoresEmptySlotWithOnlyAddressID(t *testing.T) {
	dto := &publisherDto{
		Address2AddressId: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}

	model := addressModelFromDto(2, dto)
	if model != nil {
		t.Fatalf("expected address slot 2 to be ignored when only address id remains, got %#v", model)
	}
}

func TestUnitAddressModelsFromDto_IgnoresPlaceholderSlotWithoutExistingState(t *testing.T) {
	dto := &publisherDto{
		Address1AddressId:          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Address1AddressTypeCode:    int64Pointer(1),
		Address1ShippingMethodCode: int64Pointer(1),
	}

	models := addressModelsFromDto(dto, nil)
	if models != nil {
		t.Fatalf("expected placeholder address slot to be ignored, got %#v", models)
	}
}

func TestUnitAddressModelsFromDto_PreservesPlaceholderSlotWhenAlreadyTracked(t *testing.T) {
	dto := &publisherDto{
		Address1AddressId:          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Address1AddressTypeCode:    int64Pointer(1),
		Address1ShippingMethodCode: int64Pointer(1),
	}

	existing := []PublisherAddressModel{
		{
			Slot:               types.Int64Value(1),
			AddressTypeCode:    types.Int64Value(1),
			ShippingMethodCode: types.Int64Value(1),
		},
	}

	models := addressModelsFromDto(dto, existing)
	if len(models) != 1 {
		t.Fatalf("expected tracked placeholder address slot to be preserved, got %#v", models)
	}
	if models[0].Slot.ValueInt64() != 1 {
		t.Fatalf("expected preserved address slot 1, got %#v", models[0])
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
