package domain

import (
	"testing"
	"time"
)

func TestSummarizeDashboardInventoryDeduplicatesDocumentsAndPhysicalUnits(t *testing.T) {
	date := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	items := []InventoryItem{
		{Type: InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, LoadType: "FCL", ContainerNo: "ABCD 123456-7", PhysicalUnitID: "BTD-001|ABCD1234567", IsActive: true},
		{Type: InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, LoadType: "FCL", ContainerNo: "ABCD1234567", PhysicalUnitID: "BTD-001|ABCD1234567", IsActive: true},
		{Type: InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, LoadType: "FCL", ContainerNo: "EFGH 765432-1", PhysicalUnitID: "BTD-001|EFGH7654321", IsActive: true},
		{Type: InventoryBTD, DeterminationNo: "BTD-002", DeterminationDate: date, LoadType: "LCL", PhysicalUnitID: "BTD|BTD-002|2026-07-16", IsActive: true},
		{Type: InventoryBTD, DeterminationNo: "BTD-002", DeterminationDate: date, LoadType: "LCL", PhysicalUnitID: "BTD|BTD-002|2026-07-16", IsActive: true},
		{Type: InventoryBTD, DeterminationNo: "BTD-003", DeterminationDate: date, LoadType: "FCL", ContainerNo: "IJKL 111111-1", IsActive: false},
	}

	got := SummarizeDashboardInventory(items)
	if got.Documents != 2 || got.FCL != 2 || got.LCL != 1 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}
