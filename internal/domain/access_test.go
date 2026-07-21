package domain

import "testing"

func defaultParameterOptionsForTest() []ParameterOption {
	defaults := make([]ParameterOption, 0, len(BDNCategoryNames)+len(ItemKindNames)+len(GoodsConditionNames)+len(UnitNames)+len(AllocationPurposeNames)+len(TPSNames)+len(LoadTypeOptions)+len(ExitOptions)+len(TransferTypeOptions))
	for index, label := range BDNCategoryNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterBDNCategory, Code: "bdn", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, label := range ItemKindNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterItemKind, Code: "kind", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, label := range GoodsConditionNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterGoodsCondition, Code: "condition", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, label := range UnitNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterUnit, Code: "unit", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, label := range AllocationPurposeNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterAllocationPurpose, Code: "allocation", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, label := range TPSNames {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterOriginTPS, Code: "tps", Label: label, Active: true, SortOrder: index + 1})
	}
	for index, option := range LoadTypeOptions {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterLoadType, Code: option.Code, Label: option.Label, Active: true, SortOrder: index + 1})
	}
	for index, option := range ExitOptions {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterExitType, Code: option.Code, Label: option.Label, AppliesTo: option.Types, Active: true, SortOrder: index + 1})
	}
	for index, option := range TransferTypeOptions {
		defaults = append(defaults, ParameterOption{GroupCode: ParameterTransferType, Code: option.Code, Label: option.Label, AppliesTo: option.Types, Active: true, SortOrder: index + 1})
	}
	return defaults
}

func TestSetRuntimeParametersCanEmptyAGroup(t *testing.T) {
	defaults := defaultParameterOptionsForTest()
	t.Cleanup(func() { SetRuntimeParameters(defaults) })

	SetRuntimeParameters([]ParameterOption{
		{GroupCode: ParameterBDNCategory, Code: "custom", Label: "Kategori Khusus", Active: true},
		{GroupCode: ParameterItemKind, Code: "disabled", Label: "Pilihan nonaktif", Active: false},
		{GroupCode: ParameterExitType, Code: "custom_exit", Label: "KELUAR KHUSUS", AppliesTo: "BMMN", Active: true},
	})
	if len(CurrentItemKinds()) != 0 {
		t.Fatalf("expected item-kind group to be empty, got %v", CurrentItemKinds())
	}
	if !ValidBDNCategory("Kategori Khusus") || ValidBDNCategory(BDNCategoryNames[0]) {
		t.Fatal("runtime BDN category replacement was not applied exactly")
	}
	if !ValidExitType(InventoryBMMN, "custom_exit") || ValidExitType(InventoryBTD, "custom_exit") {
		t.Fatal("runtime exit scope was not applied")
	}
	// Groups absent from an older database retain their built-in defaults.
	if !ValidGoodsCondition(GoodsConditionNames[0]) || !ValidUnit(UnitNames[0]) || !ValidTPS(TPSNames[0]) || !ValidLoadType(LoadTypeOptions[0].Code) {
		t.Fatal("missing groups should retain built-in defaults")
	}
}

func TestNormalizeInventoryGranularPermissionsAddsRequiredReadAndTypeScope(t *testing.T) {
	permissions := NormalizePermissions([]string{
		PermissionInventoryCreateBTD,
		PermissionInventoryActionContainerLoad,
	})
	for _, required := range []string{
		PermissionInventoryCreateBTD,
		PermissionInventoryActionContainerLoad,
		PermissionInventoryView,
		PermissionInventoryBTD,
	} {
		if !contains(permissions, required) {
			t.Fatalf("normalized permissions missing %s: %v", required, permissions)
		}
	}
	if ValidPermission(PermissionInventoryManage) {
		t.Fatal("legacy inventory.manage must not be exposed as a selectable permission")
	}
}
