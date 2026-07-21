package web

import (
	"strings"
	"testing"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestBuildBTDReportRowsGroupsGoodsByDocumentAndContainer(t *testing.T) {
	date := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	items := []domain.InventoryItem{
		{Type: domain.InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, BLNo: "BL-001", BLDate: date.Add(-48 * time.Hour), ManifestNo: "MAN-001", ManifestDate: date.Add(-24 * time.Hour), ManifestPosition: "0101", LoadType: "FCL", ContainerNo: "ABCD 123456-7", ContainerSize: "20'", PhysicalUnitID: "unit-1", Description: "Mesin", ItemKind: "Barang Umum", GoodsCondition: "Baru", Quantity: 12, Unit: "Piece", GoodsValue: 250000000, OriginWarehouse: "PT Agung Raya", FacilityName: "TPP Transporindo", AtTPP: true, OwnerName: "PT Nusantara Machinery", StatusLabel: "Pencacahan", IsActive: true},
		{Type: domain.InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, BLNo: "BL-001", BLDate: date.Add(-48 * time.Hour), ManifestNo: "MAN-001", ManifestDate: date.Add(-24 * time.Hour), ManifestPosition: "0101", LoadType: "FCL", ContainerNo: "ABCD 123456-7", ContainerSize: "20'", PhysicalUnitID: "unit-1", Description: "Suku cadang", ItemKind: "Barang Umum", GoodsCondition: "Bekas", Quantity: 8, Unit: "Box", GoodsValue: 75000000, OriginWarehouse: "PT Agung Raya", FacilityName: "TPP Transporindo", AtTPP: true, OwnerName: "PT Nusantara Machinery", StatusLabel: "Pencacahan", IsActive: true},
		{Type: domain.InventoryBTD, DeterminationNo: "BTD-001", DeterminationDate: date, BLNo: "BL-001", BLDate: date.Add(-48 * time.Hour), ManifestNo: "MAN-001", ManifestDate: date.Add(-24 * time.Hour), ManifestPosition: "0101", LoadType: "FCL", ContainerNo: "EFGH 765432-1", ContainerSize: "40'", PhysicalUnitID: "unit-2", Description: "Peralatan", ItemKind: "Barang Berharga", Quantity: 3, Unit: "Crate", GoodsValue: 100000000, OriginWarehouse: "PT Agung Raya", FacilityName: "TPP Transporindo", AtTPP: true, OwnerName: "PT Nusantara Machinery", StatusLabel: "Pencacahan", IsActive: true},
		{Type: domain.InventoryBDN, DeterminationNo: "BDN-IGNORED", DeterminationDate: date, LoadType: "FCL", ContainerNo: "ZZZZ 999999-9", Description: "Tidak masuk"},
	}

	rows := buildBTDReportRows(items)
	if len(rows) != 1 {
		t.Fatalf("expected one BTD document row, got %d", len(rows))
	}
	row := rows[0]
	if row.DeterminationNo != "BTD-001" {
		t.Fatalf("unexpected document: %+v", row)
	}
	for _, expected := range []string{"ABCD 123456-7", "EFGH 765432-1"} {
		if !strings.Contains(row.ContainerSummary, expected) {
			t.Fatalf("container summary %q does not contain %q", row.ContainerSummary, expected)
		}
	}
	if strings.Contains(row.ContainerSummary, "Kontainer 1") || strings.Contains(row.GoodsSummary, "Kontainer 1") {
		t.Fatalf("BTD report must use container numbers directly without sequence labels: container=%q goods=%q", row.ContainerSummary, row.GoodsSummary)
	}
	for _, expected := range []string{"ABCD 123456-7(Mesin [Barang Umum; Baru]: 12 Piece, Suku cadang [Barang Umum; Bekas]: 8 Box)", "EFGH 765432-1(Peralatan [Barang Berharga]: 3 Crate)"} {
		if !strings.Contains(row.GoodsSummary, expected) {
			t.Fatalf("goods summary %q does not contain %q", row.GoodsSummary, expected)
		}
	}
	if row.BLNo != "BL-001" || row.BLDate != "14/07/2026" || row.ManifestNo != "MAN-001" || row.ManifestDate != "15/07/2026" || row.ManifestPosition != "0101" {
		t.Fatalf("document identity fields were not aggregated correctly: %+v", row)
	}
	if row.OriginWarehouse != "PT Agung Raya" || row.FacilityName != "TPP Transporindo" || row.LocationStatus != "Di TPP" {
		t.Fatalf("location fields were not aggregated correctly: %+v", row)
	}
	if row.OwnerName != "PT Nusantara Machinery" || row.ContainerCount != 2 || row.ItemCount != 3 || row.TotalValue != 425000000 {
		t.Fatalf("owner, item count, or total value is incorrect: %+v", row)
	}
	if row.StatusLabel != "Pencacahan" || row.InventoryStatus != "Aktif" {
		t.Fatalf("status fields were not aggregated correctly: %+v", row)
	}
	if row.LoadType != "FCL" {
		t.Fatalf("load type was not aggregated correctly: %+v", row)
	}
}
