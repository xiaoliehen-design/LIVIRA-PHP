package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestMemoryStoreExcludesTPPL4(t *testing.T) {
	data := NewMemoryStore()
	facilities, err := data.Facilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(facilities) != 4 {
		t.Fatalf("expected 4 facilities, got %d", len(facilities))
	}
	for _, facility := range facilities {
		if facility.Name == "TPP L4" {
			t.Fatal("TPP L4 must not be present")
		}
	}
}

func TestBMMNCanOnlyComeFromBTDOrBDN(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	_, err := data.CreateInventory(ctx, domain.NewInventoryInput{Type: domain.InventoryBMMN, Description: "Tidak boleh", FacilityID: "tpp-transporindo", DeterminationNo: "KEP-X", Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, Description: "Peralatan pengujian", ItemKind: "Barang Umum", Quantity: 2, Unit: "Piece", AtTPP: true, FacilityID: "tpp-transporindo", OriginWarehouse: "PT Agung Raya",
		DeterminationNo: "KEP-TEST/KPU.01/2026", DeterminationDate: time.Now(), Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := data.AddInventoryEvent(ctx, created.ID, domain.NewEventInput{Code: "penetapan_bmmn", DocumentNo: "SKEP-BMMN-TEST", DocumentDate: time.Now(), Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Type != domain.InventoryBMMN || updated.OriginType != domain.InventoryBTD {
		t.Fatalf("expected BMMN from BTD, got type=%s origin=%s", updated.Type, updated.OriginType)
	}
	if updated.OriginDocumentType != "BCF 1.5" || updated.OriginDocumentNo != created.DeterminationNo {
		t.Fatalf("BMMN origin document was not preserved: %+v", updated)
	}
}

func TestOneActiveDispositionAndCompletion(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBDN, Category: domain.BDNCategoryNames[0], Description: "Barang uji proses", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", AtTPP: true, FacilityID: "tpp-graha-segara", OriginWarehouse: "PT Multi Terminal Indonesia (CDC Banda)",
		DeterminationNo: "KEP-PROCESS/KPU.01/2026", DeterminationDate: time.Now(), Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	process, err := data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: created.ID, Type: domain.DispositionAuction, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: created.ID, Type: domain.DispositionGrant, Actor: "Tester"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	now := time.Now()
	steps := []domain.NewEventInput{
		{Code: "kep_lelang", DocumentNo: "KEP-L-01", DocumentDate: now, Actor: "Tester"},
		{Code: "kep_htl", DocumentNo: "KEP-HTL-01", DocumentDate: now, HTLValue: 100000000, Actor: "Tester"},
		{Code: "jadwal_lelang", DocumentNo: "ND-JADWAL-01", DocumentDate: now, ExecutionStartDate: now.AddDate(0, 0, 7), Actor: "Tester"},
		{Code: "selesai_lelang", DocumentNo: "RISALAH-01", DocumentDate: now, AuctionOutcome: "laku", SaleValue: 125000000, Actor: "Tester"},
		{Code: "alokasi_hasil_lelang", DocumentNo: "KEP-ALOKASI-01", DocumentDate: now, AllocationTarget: "Kas negara", Actor: "Tester"},
	}
	for _, step := range steps {
		if _, err = data.AddDispositionEvent(ctx, process.ID, step); err != nil {
			t.Fatalf("step %s failed: %v", step.Code, err)
		}
	}
	item, err := data.GetInventory(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !item.IsActive || item.CurrentDisposition != "" {
		t.Fatal("completed auction must return item to inventory for documented release")
	}
	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pengeluaran_barang", DocumentNo: "OUT-LELANG-01", DocumentDate: now, ExitType: "lelang", Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive {
		t.Fatal("released item must leave active inventory")
	}
	active, err := data.ListInventory(ctx, domain.InventoryFilter{Query: created.ReferenceNo})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatal("completed item appeared in active inventory")
	}
	reportRows, err := data.ListInventory(ctx, domain.InventoryFilter{Query: created.ReferenceNo, IncludeInactive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(reportRows) != 1 {
		t.Fatal("completed item must remain available for reporting")
	}
	completedOnly, err := data.ListInventory(ctx, domain.InventoryFilter{Query: created.ReferenceNo, OnlyInactive: true})
	if err != nil || len(completedOnly) != 1 {
		t.Fatalf("completed-only report filter failed: rows=%d err=%v", len(completedOnly), err)
	}
}

func TestLocationPFPDAndExitWorkflow(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-TPS-TEST", DeterminationDate: now,
		Description: "Uraian awal", ItemKind: "Barang Umum", Quantity: 10, Unit: "Piece",
		OriginWarehouse: "PT Agung Raya", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.AtTPP || item.LocationStatus != "PT Agung Raya" || item.Location != "PT Agung Raya" {
		t.Fatalf("expected item at origin TPS, got at_tpp=%v status=%q location=%q", item.AtTPP, item.LocationStatus, item.Location)
	}

	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pemindahan", TargetFacilityID: "tpp-kbn-marunda", DocumentNo: "SPRIN-01", DocumentDate: now, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if !item.AtTPP || item.FacilityID != "tpp-kbn-marunda" || item.LocationStatus != "TPP KBN Marunda" {
		t.Fatalf("movement did not update location: %+v", item)
	}

	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pencacahan", DocumentNo: "BA-CACAH-01", DocumentDate: now, Description: "Uraian hasil cacah", ItemKind: "Barang Berharga", Quantity: 12, Unit: "Piece", GoodsCondition: "Bekas", PFPDRequired: true, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Description != "Uraian hasil cacah" || item.ItemKind != "Barang Berharga" || item.Quantity != 12 {
		t.Fatal("census result did not update inventory fields")
	}

	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "request_penelitian_pfpd", DocumentNo: "ND-REQ-01", DocumentDate: now, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "penelitian_pfpd", DocumentNo: "ND-PFPD-01", DocumentDate: now, HSCode: "8471.30.90", RestrictionStatus: "ya", IsRestricted: true, RestrictionRule: "Persetujuan teknis", GoodsValue: 250000000, Actor: "PFPD"})
	if err != nil {
		t.Fatal(err)
	}
	if item.HSCode != "8471.30.90" || !item.IsRestricted || item.GoodsValue != 250000000 {
		t.Fatal("PFPD result did not update HS, lartas, and goods value")
	}

	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pengeluaran_barang", DocumentNo: "OUT-01", DocumentDate: now, ExitType: "reekspor", Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive || item.ExitType != "reekspor" || item.LocationStatus != "Barang telah dikeluarkan" {
		t.Fatal("exit action did not close active inventory")
	}
}

func TestApplyInventoryCensusCreatesMultipleGoodsRowsButCountsOneContainer(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	beforeStats, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	beforeYard := 0.0
	for _, row := range beforeStats.FacilityBreakdown {
		if row.FacilityID == "tpp-graha-segara" {
			beforeYard = row.YardUsed
			break
		}
	}

	created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-MULTI-CENSUS", DeterminationDate: now,
		Description: "Uraian awal", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
		AtTPP: true, FacilityID: "tpp-graha-segara", OriginWarehouse: "PT Agung Raya",
		LoadType: "FCL", ContainerNo: "TEST 123456-7", ContainerSize: "40", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := data.ApplyInventoryCensus(ctx, created.ID, []domain.InventoryGoodsLine{
		{InventoryID: created.ID, Description: "Komputer industri", ItemKind: "Barang Umum", Quantity: 4, QuantityDetail: "2 peti @ 2 unit", Unit: "Piece", GoodsCondition: "Bekas"},
		{Description: "Baterai lithium cadangan", ItemKind: "Barang Berbahaya (B3)", Quantity: 8, Unit: "Piece", GoodsCondition: "Baru"},
		{Description: "Modul kendali", ItemKind: "Barang Berharga", Quantity: 12, Unit: "Piece", GoodsCondition: "Rusak"},
	}, domain.NewEventInput{Code: "pencacahan", DocumentNo: "BA-CACAH-MULTI", DocumentDate: now, PFPDRequired: true, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("created=%+v rows=%+v", created, rows)
	if len(rows) != 3 {
		t.Fatalf("expected 3 inventory rows, got %d", len(rows))
	}
	physicalID := rows[0].PhysicalUnitID
	primary := 0
	for _, row := range rows {
		if row.ContainerNo != created.ContainerNo || row.PhysicalUnitID != physicalID {
			t.Fatalf("multi-goods rows did not retain one physical container: %+v", row)
		}
		if row.OccupancyPrimary {
			primary++
		}
		if !row.PFPDRequired {
			t.Fatal("PFPD requirement was not copied to every goods row")
		}
	}
	if primary != 1 {
		t.Fatalf("expected exactly one occupancy-primary row, got %d", primary)
	}
	if rows[0].QuantityDetail != "2 peti @ 2 unit" {
		t.Fatalf("quantity detail was not persisted: %+v", rows[0])
	}

	stats, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var facility domain.FacilityBreakdown
	for _, row := range stats.FacilityBreakdown {
		if row.FacilityID == "tpp-graha-segara" {
			facility = row
			break
		}
	}
	if delta := facility.YardUsed - beforeYard; delta != 2 {
		t.Fatalf("40-foot container should add exactly 2 TEU once; delta=%v total=%v", delta, facility.YardUsed)
	}
}

func TestInitialMultiGoodsRowsCountOnePhysicalUnit(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	before, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fclPhysicalID := "KEP-MULTI-GOODS|FCLU1234567"
	fclRows, err := data.CreateInventories(ctx, []domain.NewInventoryInput{
		{
			ReferenceNo: "KEP-MULTI-GOODS/C01/G01", Type: domain.InventoryBTD,
			DeterminationNo: "KEP-MULTI-GOODS", DeterminationDate: now,
			Description: "Mesin produksi", ItemKind: "Barang Umum", GoodsValue: 500_000_000, Quantity: 1, Unit: "Set",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "FCL", ContainerNo: "FCLU 123456-7", ContainerSize: "40", PhysicalUnitID: fclPhysicalID, OccupancyPrimary: true, Actor: "Tester",
		},
		{
			ReferenceNo: "KEP-MULTI-GOODS/C01/G02", Type: domain.InventoryBTD,
			DeterminationNo: "KEP-MULTI-GOODS", DeterminationDate: now,
			Description: "Panel kendali", ItemKind: "Barang Berharga", GoodsValue: 125_000_000, Quantity: 4, Unit: "Piece",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "FCL", ContainerNo: "FCLU 123456-7", ContainerSize: "40", PhysicalUnitID: fclPhysicalID, OccupancyPrimary: false, Actor: "Tester",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fclRows) != 2 || fclRows[0].PhysicalUnitID != fclRows[1].PhysicalUnitID {
		t.Fatalf("expected two inventory rows sharing one FCL physical unit: %+v", fclRows)
	}

	lclPhysicalID := "KEP-LCL-MULTI|LCL"
	lclRows, err := data.CreateInventories(ctx, []domain.NewInventoryInput{
		{
			ReferenceNo: "KEP-LCL-MULTI/LCL/G01", Type: domain.InventoryBDN, Category: domain.BDNCategoryNames[0],
			DeterminationNo: "KEP-LCL-MULTI", DeterminationDate: now,
			Description: "Suku cadang A", ItemKind: "Barang Umum", GoodsValue: 25_000_000, Quantity: 10, Unit: "Box",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "LCL", EstimatedVolumeM3: 18.5, PhysicalUnitID: lclPhysicalID, OccupancyPrimary: true, Actor: "Tester",
		},
		{
			ReferenceNo: "KEP-LCL-MULTI/LCL/G02", Type: domain.InventoryBDN, Category: domain.BDNCategoryNames[0],
			DeterminationNo: "KEP-LCL-MULTI", DeterminationDate: now,
			Description: "Suku cadang B", ItemKind: "Barang Umum", GoodsValue: 15_000_000, Quantity: 6, Unit: "Box",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "LCL", EstimatedVolumeM3: 18.5, PhysicalUnitID: lclPhysicalID, OccupancyPrimary: false, Actor: "Tester",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lclRows) != 2 || lclRows[0].PhysicalUnitID != lclRows[1].PhysicalUnitID {
		t.Fatalf("expected two inventory rows sharing one LCL physical unit: %+v", lclRows)
	}

	after, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := after.Occupancy.YardUsed - before.Occupancy.YardUsed; delta != 2 {
		t.Fatalf("two goods rows in one 40-foot container must add 2 TEU once, got %.2f", delta)
	}
	if delta := after.Occupancy.ShedUsed - before.Occupancy.ShedUsed; delta != 18.5 {
		t.Fatalf("two LCL goods rows must add their shared 18.5 m3 once, got %.2f", delta)
	}
}

func TestTimelineIsChronological(t *testing.T) {
	data := NewMemoryStore()
	events, err := data.Timeline(context.Background(), "inv-003")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	for index := 1; index < len(events); index++ {
		if events[index].CreatedAt.Before(events[index-1].CreatedAt) {
			t.Fatal("timeline is not chronological")
		}
	}
}

func TestInventoryReportFiltersAndValueSort(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	for index, value := range []int64{175000000, 425000000} {
		_, err := data.CreateInventory(ctx, domain.NewInventoryInput{
			Type: domain.InventoryBTD, DeterminationNo: "KEP-REPORT-" + string(rune('A'+index)), DeterminationDate: now.AddDate(0, 0, -10-index),
			Description: "Barang laporan", ItemKind: "Barang Berbahaya (B3)", Quantity: 1, Unit: "Piece", GoodsValue: value,
			AtTPP: true, FacilityID: "tpp-transporindo", OriginWarehouse: domain.TPSNames[0], Actor: "Tester",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	items, err := data.ListInventory(ctx, domain.InventoryFilter{
		DateFrom: now.AddDate(0, 0, -30), DateTo: now, ItemKind: "Barang Berbahaya (B3)", LocationScope: "tpp", MinValue: 100000000, Sort: "value_desc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].GoodsValue != 425000000 || items[1].GoodsValue != 175000000 {
		t.Fatalf("report filters/value sort returned unexpected rows: %+v", items)
	}
}

func TestMasterValidationAndTypeSpecificExit(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	_, err := data.CreateInventory(ctx, domain.NewInventoryInput{Type: domain.InventoryBDN, DeterminationNo: "KEP-BDN-NO-CATEGORY", DeterminationDate: now, Description: "Barang uji", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", OriginWarehouse: domain.TPSNames[0], Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("BDN without category must be rejected, got %v", err)
	}

	_, err = data.CreateInventory(ctx, domain.NewInventoryInput{Type: domain.InventoryBTD, DeterminationNo: "KEP-INVALID-KIND", DeterminationDate: now, Description: "Barang uji", ItemKind: "Jenis bebas", Quantity: 1, Unit: "Piece", OriginWarehouse: domain.TPSNames[0], Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("free-text item kind must be rejected, got %v", err)
	}

	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{Type: domain.InventoryBTD, DeterminationNo: "BCF-1.5-EXIT", DeterminationDate: now, Description: "Barang uji pengeluaran", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", OriginWarehouse: domain.TPSNames[0], Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pengeluaran_barang", DocumentNo: "OUT-WRONG", DocumentDate: now, ExitType: "pembatalan_bdn", Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("BTD must reject BDN-only exit type, got %v", err)
	}
}

func TestAuctionAdjustmentSkipsNewHTL(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	before, err := data.GetDisposition(ctx, "proc-003")
	if err != nil {
		t.Fatal(err)
	}
	adjusted, err := data.AddDispositionEvent(ctx, before.ID, domain.NewEventInput{Code: "lelang_penyesuaian", DocumentNo: "KEP-PENYESUAIAN-01", DocumentDate: now, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if adjusted.Round != before.Round+1 || adjusted.HTLValue != before.HTLValue || adjusted.StatusCode != "lelang_penyesuaian" {
		t.Fatalf("adjustment did not preserve HTL/increment round: %+v", adjusted)
	}
	_, err = data.AddDispositionEvent(ctx, before.ID, domain.NewEventInput{Code: "kep_htl", DocumentNo: "KEP-HTL-TIDAK-BOLEH", DocumentDate: now, HTLValue: 1, Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("adjustment round must skip new HTL, got %v", err)
	}
	_, err = data.AddDispositionEvent(ctx, before.ID, domain.NewEventInput{Code: "jadwal_lelang", DocumentNo: "ND-JADWAL-PENYESUAIAN", DocumentDate: now, ExecutionStartDate: now.AddDate(0, 0, 3), Actor: "Tester"})
	if err != nil {
		t.Fatalf("adjustment must proceed directly to scheduling: %v", err)
	}
}

func TestRejectsTamperedWorkflowAction(t *testing.T) {
	data := NewMemoryStore()
	_, err := data.AddInventoryEvent(context.Background(), "inv-001", domain.NewEventInput{Code: "status_buatan", Label: "Status buatan", Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	_, err = data.AddDispositionEvent(context.Background(), "proc-001", domain.NewEventInput{Code: "ba_musnah", Label: "Berita acara pemusnahan", Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for cross-process action, got %v", err)
	}
}

func TestAuctionCanBeReleasedImmediatelyAfterSuccessfulSale(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "BCF-1.5-AUCTION-EXIT", DeterminationDate: now,
		Description: "Mesin untuk pengujian pelelangan", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", GoodsValue: 300000000,
		AtTPP: true, FacilityID: "tpp-transporindo", OriginWarehouse: domain.TPSNames[0], Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	process, err := data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: item.ID, Type: domain.DispositionAuction, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	steps := []domain.NewEventInput{
		{Code: "kep_lelang", DocumentNo: "KEP-L-EXIT", DocumentDate: now, Actor: "Tester"},
		{Code: "kep_htl", DocumentNo: "KEP-HTL-EXIT", DocumentDate: now, HTLValue: 225000000, Actor: "Tester"},
		{Code: "jadwal_lelang", DocumentNo: "ND-JADWAL-EXIT", DocumentDate: now, ExecutionStartDate: now, Actor: "Tester"},
		{Code: "selesai_lelang", DocumentNo: "RISALAH-EXIT", DocumentDate: now, AuctionOutcome: "laku", SaleValue: 250000000, Actor: "Tester"},
	}
	for _, step := range steps {
		if _, err := data.AddDispositionEvent(ctx, process.ID, step); err != nil {
			t.Fatalf("step %s failed: %v", step.Code, err)
		}
	}
	item, err = data.GetInventory(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.StatusCode != "laku" || item.CurrentDisposition != domain.DispositionAuction {
		t.Fatalf("expected successful auction awaiting release, got %+v", item)
	}
	if _, err := data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pemindahan", DocumentNo: "MOVE-AFTER-SALE", DocumentDate: now, TargetFacilityID: "tpp-kbn-marunda", Actor: "Tester"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("completed auction must only allow release, got %v", err)
	}
	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "pengeluaran_barang", DocumentNo: "OUT-AUCTION", DocumentDate: now, ExitType: "lelang", Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive || item.CurrentDisposition != "" {
		t.Fatalf("released auction item must be inactive: %+v", item)
	}
	activeProcesses, err := data.ListDispositions(ctx, domain.DispositionFilter{Type: domain.DispositionAuction})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range activeProcesses {
		if candidate.InventoryID == item.ID {
			t.Fatal("released auction item still appeared in active auction menu")
		}
	}
	history, err := data.ListDispositions(ctx, domain.DispositionFilter{Type: domain.DispositionAuction, OnlyInactiveInventory: true})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, candidate := range history {
		if candidate.InventoryID == item.ID {
			found = true
			if candidate.IsActive {
				t.Fatal("release must close the auction process")
			}
		}
	}
	if !found {
		t.Fatal("released auction item was not available in auction history")
	}
}

func TestAllocationPurposeFilterAndDeterminationSort(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	var expectedNewest string
	for index, purpose := range []string{"Lelang", "PSP"} {
		item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
			Type: domain.InventoryBTD, DeterminationNo: "BCF-ALLOC-" + string(rune('A'+index)), DeterminationDate: now.AddDate(0, 0, -10+index),
			Description: "Barang BMMN untuk " + purpose, ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", GoodsValue: int64(100000000 + index*50000000),
			AtTPP: true, FacilityID: "tpp-multi-sejahtera", OriginWarehouse: domain.TPSNames[1], Actor: "Tester",
		})
		if err != nil {
			t.Fatal(err)
		}
		item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "penetapan_bmmn", DocumentNo: "SKEP-BMMN-" + purpose, DocumentDate: now.Add(time.Duration(index) * time.Hour), Actor: "Tester"})
		if err != nil {
			t.Fatal(err)
		}
		item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{Code: "usulan_peruntukan_bmmn", DocumentNo: "USUL-" + purpose, DocumentDate: now, AllocationType: purpose, Actor: "Tester"})
		if err != nil {
			t.Fatal(err)
		}
		if purpose == "PSP" {
			expectedNewest = item.ID
		}
	}
	psp, err := data.ListInventory(ctx, domain.InventoryFilter{AllocationPurpose: "psp", Sort: "determination_newest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(psp) != 1 || psp[0].AllocationPurpose != "PSP" {
		t.Fatalf("allocation purpose filter returned unexpected rows: %+v", psp)
	}
	all, err := data.ListInventory(ctx, domain.InventoryFilter{Query: "Barang BMMN untuk", Sort: "determination_newest"})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != expectedNewest {
		t.Fatalf("determination newest sort failed: %+v", all)
	}
	_, err = data.AddInventoryEvent(ctx, all[0].ID, domain.NewEventInput{Code: "persetujuan_peruntukan_bmmn", DocumentNo: "APPROVAL-BAD", DocumentDate: now, AllocationType: "Jenis bebas", Actor: "Tester"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("non-master allocation purpose must be rejected, got %v", err)
	}
}

func TestRegistrationApprovalRequiresVerifiedEmailAndActiveRole(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	if err := data.ApproveUser(ctx, "user-pending-2", "role-auction", "Administrator"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("unverified applicant must not be approved, got %v", err)
	}
	if err := data.ApproveUser(ctx, "user-pending-1", "role-auction", "Administrator"); err != nil {
		t.Fatalf("verified applicant should be approved: %v", err)
	}
	account, err := data.UserByAuthID(ctx, "demo-pending-1")
	if err != nil {
		t.Fatal(err)
	}
	if account.ApprovalStatus != "approved" || account.RoleName != "Petugas Lelang" {
		t.Fatalf("approval did not persist role: %+v", account)
	}
	if !containsPermission(account.Permissions, domain.PermissionAuctionManage) || !containsPermission(account.Permissions, domain.PermissionAuctionView) {
		t.Fatalf("auction permissions were not assigned: %v", account.Permissions)
	}
}

func TestCustomRoleAndDynamicParameterLifecycle(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	role, err := data.CreateRole(ctx, domain.NewRoleInput{
		Name:        "Petugas Musnah BMMN",
		Description: "Khusus pemusnahan BMMN",
		Permissions: []string{domain.PermissionDestructionManage, domain.PermissionInventoryBMMN},
		Actor:       "Administrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsPermission(role.Permissions, domain.PermissionDestructionView) {
		t.Fatalf("manage permission must include matching view permission: %v", role.Permissions)
	}

	parameter, err := data.CreateParameter(ctx, domain.NewParameterInput{
		GroupCode: domain.ParameterItemKind,
		Label:     "Barang Uji Khusus",
		SortOrder: 60,
		Actor:     "Administrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := data.ParameterOptions(ctx, domain.ParameterItemKind, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, option := range active {
		if option.ID == parameter.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("new parameter was not available in active options")
	}
	if err := data.SetParameterActive(ctx, parameter.ID, false); err != nil {
		t.Fatal(err)
	}
	active, err = data.ParameterOptions(ctx, domain.ParameterItemKind, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range active {
		if option.ID == parameter.ID {
			t.Fatal("inactive parameter still appeared in active dropdown options")
		}
	}
}

func TestRoleDeletionRequiresNoAssignedUsers(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	emptyRole, err := data.CreateRole(ctx, domain.NewRoleInput{
		Name:        "Role Sementara",
		Description: "Role tanpa pengguna",
		Permissions: []string{domain.PermissionDashboardView},
		Actor:       "Administrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := data.DeleteRole(ctx, emptyRole.ID)
	if err != nil {
		t.Fatalf("empty role should be deletable: %v", err)
	}
	if deleted.ID != emptyRole.ID || deleted.Name != emptyRole.Name {
		t.Fatalf("unexpected deleted role: %+v", deleted)
	}
	if _, err := data.DeleteRole(ctx, emptyRole.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted role should no longer exist, got %v", err)
	}

	if err := data.ApproveUser(ctx, "user-pending-1", "role-auction", "Administrator"); err != nil {
		t.Fatal(err)
	}
	roles, err := data.ListRoles(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	assignedUsers := -1
	for _, role := range roles {
		if role.ID == "role-auction" {
			assignedUsers = role.AssignedUsers
			break
		}
	}
	if assignedUsers != 1 {
		t.Fatalf("assigned user count = %d, want 1", assignedUsers)
	}
	if _, err := data.DeleteRole(ctx, "role-auction"); !errors.Is(err, ErrRoleInUse) {
		t.Fatalf("assigned role must not be deleted, got %v", err)
	}
}

func TestExpandedParametersAndTPPMaster(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	unit, err := data.CreateParameter(ctx, domain.NewParameterInput{
		GroupCode: domain.ParameterUnit,
		Label:     "Container",
		SortOrder: 500,
		Actor:     "Administrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	all, err := data.ParameterOptions(ctx, domain.ParameterUnit, false)
	if err != nil {
		t.Fatal(err)
	}
	foundUnit := false
	for _, option := range all {
		if option.ID == unit.ID && option.Label == "Container" {
			foundUnit = true
		}
	}
	if !foundUnit {
		t.Fatal("custom unit was not returned by parameter options")
	}

	facility, err := data.CreateParameter(ctx, domain.NewParameterInput{
		GroupCode: domain.ParameterTPP,
		Code:      "tpp-uji",
		Label:     "TPP Uji",
		SortOrder: 500,
		Actor:     "Administrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	facilities, err := data.Facilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	foundFacility := false
	for _, item := range facilities {
		if item.ID == facility.Code && item.Name == "TPP Uji" {
			foundFacility = true
		}
	}
	if !foundFacility {
		t.Fatal("new TPP was not added to the facility dropdown")
	}
	if err := data.SetParameterActive(ctx, facility.ID, false); err != nil {
		t.Fatal(err)
	}
	facilities, err = data.Facilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range facilities {
		if item.ID == facility.Code {
			t.Fatal("inactive TPP still appeared in active facilities")
		}
	}
}

func containsPermission(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestMemoryStoreDeleteInventoryRemovesRelatedData(t *testing.T) {
	store := NewMemoryStore()
	if err := store.DeleteInventory(context.Background(), "inv-003", "Administrator TPP"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetInventory(context.Background(), "inv-003"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted inventory to be missing, got %v", err)
	}
	if _, err := store.GetDisposition(context.Background(), "proc-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected related disposition to be missing, got %v", err)
	}
	if _, err := store.Timeline(context.Background(), "inv-003"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected related timeline to be missing, got %v", err)
	}
}

func TestMultiContainerCreationAndOccupancy(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	before, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}

	created, err := data.CreateInventories(ctx, []domain.NewInventoryInput{
		{
			ReferenceNo: "KEP-MULTI/KPU.01/2026/01", Type: domain.InventoryBDN,
			Category: domain.BDNCategoryNames[0], DeterminationNo: "KEP-MULTI/KPU.01/2026", DeterminationDate: now,
			Description: "Kontainer pertama", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "FCL", ContainerNo: "TEMU 123456-7", ContainerSize: "20", Actor: "Tester",
		},
		{
			ReferenceNo: "KEP-MULTI/KPU.01/2026/02", Type: domain.InventoryBDN,
			Category: domain.BDNCategoryNames[0], DeterminationNo: "KEP-MULTI/KPU.01/2026", DeterminationDate: now,
			Description: "Kontainer kedua", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
			OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
			LoadType: "FCL", ContainerNo: "MSCU 765432-1", ContainerSize: "40", Actor: "Tester",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 || created[0].ID == created[1].ID || created[0].ReferenceNo == created[1].ReferenceNo {
		t.Fatalf("multi-container creation returned invalid rows: %+v", created)
	}
	if created[0].DeterminationNo != created[1].DeterminationNo {
		t.Fatal("all container rows must retain the same determination number")
	}

	afterFCL, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := afterFCL.Occupancy.YardUsed - before.Occupancy.YardUsed; got != 3 {
		t.Fatalf("expected YOR usage to increase by 3 TEU, got %.2f", got)
	}
	if got := afterFCL.Occupancy.ShedUsed - before.Occupancy.ShedUsed; got != 0 {
		t.Fatalf("FCL must not increase SOR usage, got %.2f m3", got)
	}

	_, err = data.CreateInventory(ctx, domain.NewInventoryInput{
		ReferenceNo: "BCF-LCL/KPU.01/2026", Type: domain.InventoryBTD,
		DeterminationNo: "BCF-LCL/KPU.01/2026", DeterminationDate: now,
		Description: "Barang LCL", ItemKind: "Barang Umum", Quantity: 10, Unit: "Piece",
		OriginWarehouse: domain.TPSNames[0], AtTPP: true, FacilityID: "tpp-transporindo",
		LoadType: "LCL", EstimatedVolumeM3: 12.5, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	afterLCL, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := afterLCL.Occupancy.ShedUsed - afterFCL.Occupancy.ShedUsed; got != 12.5 {
		t.Fatalf("expected SOR usage to increase by 12.5 m3, got %.2f", got)
	}
}

func TestUpdateFacilityCapacity(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	updated, err := data.UpdateFacilityCapacity(ctx, "tpp-transporindo", 275.5, 1400.75)
	if err != nil {
		t.Fatal(err)
	}
	if updated.YardCapacity != 275.5 || updated.ShedCapacity != 1400.75 {
		t.Fatalf("capacity was not updated: %+v", updated)
	}

	stats, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, facility := range stats.FacilityBreakdown {
		if facility.FacilityID == updated.ID {
			found = true
			if facility.YardCapacity != 275.5 || facility.ShedCapacity != 1400.75 {
				t.Fatalf("dashboard did not use edited capacity: %+v", facility)
			}
		}
	}
	if !found {
		t.Fatal("updated facility was not present in dashboard")
	}

	if _, err := data.UpdateFacilityCapacity(ctx, updated.ID, -1, 10); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("negative capacity must be rejected, got %v", err)
	}
}

func TestEntrustedInventoryUsesDedicatedExitAndDirectLocationName(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	before, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryTitipan, DeterminationNo: "ND-TITIPAN-001", DeterminationDate: now,
		EntrustedCategory: "BDN", SourceOffice: "Kantor Wilayah DJBC Jakarta",
		Description: "Perangkat pemeriksaan titipan", ItemKind: "Barang Umum", Quantity: 3, Unit: "Piece",
		LoadType: "LCL", EstimatedVolumeM3: 2.5, AtTPP: true, FacilityID: "tpp-graha-segara", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Type != domain.InventoryTitipan || item.EntrustedCategory != "BDN" || item.SourceOffice == "" {
		t.Fatalf("entrusted metadata was not stored: %+v", item)
	}
	if item.LocationStatus != "TPP Graha Segara" || strings.Contains(item.LocationStatus, "Berada di") {
		t.Fatalf("location status must be the direct TPP name, got %q", item.LocationStatus)
	}
	afterCreate, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if afterCreate.TitipanTotal-before.TitipanTotal != 1 || afterCreate.ActiveTotal-before.ActiveTotal != 1 {
		t.Fatalf("dashboard must count entrusted inventory in its dedicated KPI and active total: before=%+v after=%+v", before, afterCreate)
	}
	if afterCreate.TitipanSummary.Documents-before.TitipanSummary.Documents != 1 || afterCreate.TitipanSummary.LCL-before.TitipanSummary.LCL != 1 {
		t.Fatalf("entrusted document and LCL summary were not counted: before=%+v after=%+v", before.TitipanSummary, afterCreate.TitipanSummary)
	}
	if _, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{
		Code: "pengeluaran_barang", DocumentNo: "OUT-TITIPAN-WRONG", DocumentDate: now, ExitType: "reekspor", Actor: "Tester",
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("entrusted item must reject non-dedicated exit, got %v", err)
	}
	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{
		Code: "pengeluaran_barang", DocumentNo: "OUT-TITIPAN-001", DocumentDate: now, ExitType: "pengeluaran_barang_titipan", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive || item.ExitType != "pengeluaran_barang_titipan" {
		t.Fatalf("entrusted release did not close inventory: %+v", item)
	}
}

func TestInventoryLocationStatusUsesDirectTPSName(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-LOKASI-TPS-001", DeterminationDate: time.Now(),
		Description: "Barang yang masih berada di TPS", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
		OriginWarehouse: "PT Agung Raya", LoadType: "LCL", EstimatedVolumeM3: 1, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.LocationStatus != "PT Agung Raya" {
		t.Fatalf("location status must show the direct TPS name, got %q", item.LocationStatus)
	}
	if item.StatusCode != "masih_di_tps" || item.StatusLabel != "Masih di TPS" {
		t.Fatalf("workflow status must remain separate from the location name: %+v", item)
	}
}

func TestDestructionCanExitAfterKEPAndRemainOpenUntilBA(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-MUSNAH-EXIT", DeterminationDate: now,
		Description: "Barang rusak untuk dimusnahkan", ItemKind: "Barang Umum", Quantity: 4, Unit: "Piece",
		OriginWarehouse: "PT Agung Raya", LoadType: "LCL", EstimatedVolumeM3: 1.5,
		AtTPP: true, FacilityID: "tpp-graha-segara", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	process, err := data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: item.ID, Type: domain.DispositionDestruction, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	process, err = data.AddDispositionEvent(ctx, process.ID, domain.NewEventInput{
		Code: "kep_musnah", DocumentNo: "KEP-MUSNAH-001", DocumentDate: now, DestructionCost: 1000000, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err = data.AddInventoryEvent(ctx, item.ID, domain.NewEventInput{
		Code: "pengeluaran_barang", DocumentNo: "OUT-MUSNAH-001", DocumentDate: now, ExitType: "musnah", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive || item.CurrentDisposition != domain.DispositionDestruction {
		t.Fatalf("physical exit must hide inventory but keep destruction linked: %+v", item)
	}
	activeProcesses, err := data.ListDispositions(ctx, domain.DispositionFilter{Type: domain.DispositionDestruction, IncludeInactiveInventory: true})
	if err != nil {
		t.Fatal(err)
	}
	var processAfterExit domain.Disposition
	for _, candidate := range activeProcesses {
		if candidate.ID == process.ID {
			processAfterExit = candidate
			break
		}
	}
	if processAfterExit.ID == "" || !processAfterExit.IsActive || processAfterExit.Inventory.IsActive {
		t.Fatalf("destruction process must remain visible after physical exit: %+v", processAfterExit)
	}
	process, err = data.AddDispositionEvent(ctx, process.ID, domain.NewEventInput{
		Code: "ba_musnah", DocumentNo: "BA-MUSNAH-001", DocumentDate: now, DestructionCost: 1250000, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if process.IsActive {
		t.Fatal("BA Musnah must close the destruction process")
	}
	item, err = data.GetInventory(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive || item.CurrentDisposition != "" || item.StatusCode != "ba_musnah" {
		t.Fatalf("closed destruction must remain outside active inventory with final status: %+v", item)
	}
}

func TestReconciliationRemovesMissingItemAndAddsActualProcessStatus(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBDN, Category: domain.BDNCategoryNames[0], DeterminationNo: "KEP-REC-REMOVE", DeterminationDate: now,
		Description: "Barang tidak ditemukan", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
		OriginWarehouse: "PT Agung Raya", LoadType: "LCL", EstimatedVolumeM3: 1,
		AtTPP: true, FacilityID: "tpp-graha-segara", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	process, err := data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: item.ID, Type: domain.DispositionAuction, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	record, removed, err := data.ReconcileInventory(ctx, domain.NewReconciliationInput{
		Type: "recorded_not_found", InventoryID: item.ID, Notes: "Tidak ditemukan pada stock opname bersama TPP.", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Action != "removed" || removed.IsActive || removed.StatusCode != "rekonsiliasi_tidak_ditemukan" {
		t.Fatalf("recorded-not-found reconciliation was not applied: record=%+v item=%+v", record, removed)
	}
	process, err = data.GetDisposition(ctx, process.ID)
	if err != nil {
		t.Fatal(err)
	}
	if process.IsActive {
		t.Fatal("active related process must be closed when reconciliation removes inventory")
	}

	addedRecord, added, err := data.ReconcileInventory(ctx, domain.NewReconciliationInput{
		Type: "found_not_recorded", Notes: "Ditemukan saat rekonsiliasi fisik dan sudah memiliki ND jadwal lelang.", Actor: "Tester",
		NewItem: domain.NewInventoryInput{
			Type: domain.InventoryBMMN, DeterminationNo: "ND-JADWAL-REC-001", DeterminationDate: now,
			Description: "Barang BMMN ditemukan di lapangan", ItemKind: "Barang Berharga", Quantity: 2, Unit: "Piece",
			LoadType: "LCL", EstimatedVolumeM3: 2, AtTPP: true, FacilityID: "tpp-kbn-marunda",
			InitialStatusCode: "jadwal_lelang", InitialStatusLabel: "Jadwal lelang", InitialDispositionType: domain.DispositionAuction,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if addedRecord.Action != "added" || !added.IsActive || added.Type != domain.InventoryBMMN || added.CurrentDisposition != domain.DispositionAuction {
		t.Fatalf("found-not-recorded reconciliation was not applied: record=%+v item=%+v", addedRecord, added)
	}
	processes, err := data.ListDispositions(ctx, domain.DispositionFilter{Type: domain.DispositionAuction, IncludeInactiveInventory: true})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, candidate := range processes {
		if candidate.InventoryID == added.ID {
			found = true
			if candidate.StatusCode != "jadwal_lelang" || candidate.ScheduleDocumentNo != added.DeterminationNo || !candidate.IsActive {
				t.Fatalf("reconciled process did not reflect actual status: %+v", candidate)
			}
		}
	}
	if !found {
		t.Fatal("reconciliation-added item was not added to the related auction menu")
	}
	records, err := data.ListReconciliations(ctx, 10)
	if err != nil || len(records) != 2 {
		t.Fatalf("expected two reconciliation records, got %d err=%v", len(records), err)
	}
}

func TestAuctionSortingUsesHTLInsteadOfPFPDGoodsValue(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	createAuction := func(reference string, goodsValue, htl int64) domain.Disposition {
		t.Helper()
		item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
			Type: domain.InventoryBTD, DeterminationNo: reference, DeterminationDate: now,
			Description: reference, ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", GoodsValue: goodsValue,
			OriginWarehouse: domain.TPSNames[0], LoadType: "LCL", EstimatedVolumeM3: 1,
			AtTPP: true, FacilityID: "tpp-graha-segara", Actor: "Tester",
		})
		if err != nil {
			t.Fatal(err)
		}
		process, err := data.CreateDisposition(ctx, domain.NewDispositionInput{InventoryID: item.ID, Type: domain.DispositionAuction, Actor: "Tester"})
		if err != nil {
			t.Fatal(err)
		}
		process, err = data.AddDispositionEvent(ctx, process.ID, domain.NewEventInput{Code: "kep_lelang", DocumentNo: "KEP-" + reference, DocumentDate: now, Actor: "Tester"})
		if err != nil {
			t.Fatal(err)
		}
		process, err = data.AddDispositionEvent(ctx, process.ID, domain.NewEventInput{Code: "kep_htl", DocumentNo: "HTL-" + reference, DocumentDate: now, HTLValue: htl, Actor: "Tester"})
		if err != nil {
			t.Fatal(err)
		}
		return process
	}

	highPFPDLowHTL := createAuction("BTD-HIGH-PFPD", 900_000_000, 100_000_000)
	lowPFPDHighHTL := createAuction("BTD-HIGH-HTL", 200_000_000, 500_000_000)

	processes, err := data.ListDispositions(ctx, domain.DispositionFilter{Type: domain.DispositionAuction, Sort: "value_desc", IncludeInactiveInventory: true})
	if err != nil {
		t.Fatal(err)
	}
	positions := map[string]int{}
	for index, process := range processes {
		positions[process.ID] = index
	}
	if positions[lowPFPDHighHTL.ID] >= positions[highPFPDLowHTL.ID] {
		t.Fatalf("auction value_desc must prioritize HTL: high-HTL index=%d high-PFPD index=%d", positions[lowPFPDHighHTL.ID], positions[highPFPDLowHTL.ID])
	}
}

func TestDocumentAttachmentRoleAndParameterUpdates(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	document, err := data.CreateDocument(ctx, domain.NewDocumentInput{
		FileName: "kep-test.pdf", MIMEType: "application/pdf", SizeBytes: 8,
		Content: []byte("%PDF-1.7"), UploadedBy: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-DOC-001", DeterminationDate: now,
		Description: "Barang dengan dokumen", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
		LoadType: "LCL", EstimatedVolumeM3: 1, AtTPP: true, FacilityID: "tpp-transporindo",
		OriginWarehouse: domain.TPSNames[0], Actor: "Tester", DocumentID: document.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	events, err := data.Timeline(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || len(events[0].Attachments) != 1 || events[0].Attachments[0].FileName != "kep-test.pdf" {
		t.Fatalf("document was not attached to timeline: %+v", events)
	}
	meta, content, err := data.GetDocument(ctx, document.ID)
	if err != nil || meta.MIMEType != "application/pdf" || string(content) != "%PDF-1.7" {
		t.Fatalf("stored document mismatch: meta=%+v content=%q err=%v", meta, string(content), err)
	}
	if _, _, err := data.ReconcileInventory(ctx, domain.NewReconciliationInput{Type: "recorded_not_found", InventoryID: item.ID, Notes: "Tidak ditemukan saat pemeriksaan", Actor: "Tester", DocumentID: document.ID}); err != nil {
		t.Fatal(err)
	}
	events, err = data.Timeline(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	lastEvent := events[len(events)-1]
	if len(lastEvent.Attachments) != 1 || lastEvent.Attachments[0].ID != document.ID {
		t.Fatalf("reconciliation document was not attached to timeline: %+v", lastEvent)
	}

	account, err := data.CreateUserApplication(ctx, domain.NewUserApplicationInput{AuthUserID: "auth-role-test", Name: "Role Test", Email: "role.test@example.go.id"})
	if err != nil {
		t.Fatal(err)
	}
	if err := data.MarkUserEmailVerified(ctx, "auth-role-test", account.Email); err != nil {
		t.Fatal(err)
	}
	if err := data.ApproveUser(ctx, account.ID, "role-inventory", "Admin"); err != nil {
		t.Fatal(err)
	}
	if err := data.UpdateUserRole(ctx, account.ID, "role-auction", "Admin"); err != nil {
		t.Fatal(err)
	}
	updatedAccount, err := data.UserByAuthID(ctx, "auth-role-test")
	if err != nil || updatedAccount.RoleID != "role-auction" || updatedAccount.RoleName != "Petugas Lelang" {
		t.Fatalf("approved user role was not updated: account=%+v err=%v", updatedAccount, err)
	}

	updatedParameter, err := data.UpdateParameter(ctx, "param-unit-01", domain.NewParameterInput{Label: "Karton Uji", SortOrder: 7, Actor: "Admin"})
	if err != nil {
		t.Fatal(err)
	}
	if updatedParameter.Label != "Karton Uji" || updatedParameter.SortOrder != 7 {
		t.Fatalf("parameter update was not persisted: %+v", updatedParameter)
	}
	updatedTPP, err := data.UpdateParameter(ctx, "facility--tpp-transporindo", domain.NewParameterInput{Label: "TPP Transporindo Baru", SortOrder: 99, Actor: "Admin"})
	if err != nil {
		t.Fatal(err)
	}
	if updatedTPP.Label != "TPP Transporindo Baru" || updatedTPP.SortOrder != 99 {
		t.Fatalf("TPP parameter update was not persisted: %+v", updatedTPP)
	}
}

func TestDispositionStatusFiltersSeparateActivePageAndHistory(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	auction, err := data.AddDispositionEvent(ctx, "proc-001", domain.NewEventInput{
		Code: "selesai_lelang", DocumentNo: "RISALAH-HISTORY-001", DocumentDate: now,
		AuctionOutcome: "laku", SaleValue: 500000000, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	activeAuction, err := data.ListDispositions(ctx, domain.DispositionFilter{
		Type: domain.DispositionAuction, ExcludeStatusCodes: []string{"laku", "alokasi_hasil_lelang"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, process := range activeAuction {
		if process.ID == auction.ID {
			t.Fatal("sold auction still appeared on the main auction page")
		}
	}
	auctionHistory, err := data.ListDispositions(ctx, domain.DispositionFilter{
		Type: domain.DispositionAuction, IncludeInactiveInventory: true,
		IncludeStatusCodes: []string{"laku", "alokasi_hasil_lelang"},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundAuction := false
	for _, process := range auctionHistory {
		if process.ID == auction.ID {
			foundAuction = true
		}
	}
	if !foundAuction {
		t.Fatal("sold auction was not available in auction history")
	}

	destruction, err := data.AddDispositionEvent(ctx, "proc-002", domain.NewEventInput{
		Code: "ba_musnah", DocumentNo: "BA-MUSNAH-HISTORY-001", DocumentDate: now,
		DestructionCost: 20000000, Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	activeDestruction, err := data.ListDispositions(ctx, domain.DispositionFilter{
		Type: domain.DispositionDestruction, IncludeInactiveInventory: true,
		ExcludeStatusCodes: []string{"ba_musnah"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, process := range activeDestruction {
		if process.ID == destruction.ID {
			t.Fatal("completed destruction still appeared on the main destruction page")
		}
	}
	destructionHistory, err := data.ListDispositions(ctx, domain.DispositionFilter{
		Type: domain.DispositionDestruction, IncludeInactiveInventory: true,
		IncludeStatusCodes: []string{"ba_musnah"},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundDestruction := false
	for _, process := range destructionHistory {
		if process.ID == destruction.ID {
			foundDestruction = true
		}
	}
	if !foundDestruction {
		t.Fatal("completed destruction was not available in destruction history")
	}
}

func TestFailedAuctionCanTransferToDestruction(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()

	before, err := data.GetInventory(ctx, "inv-008")
	if err != nil {
		t.Fatal(err)
	}
	if before.StatusCode != "tidak_laku" || before.CurrentDisposition != domain.DispositionAuction {
		t.Fatalf("unexpected failed auction seed: %+v", before)
	}

	created, err := data.CreateDisposition(ctx, domain.NewDispositionInput{
		InventoryID: before.ID,
		Type:        domain.DispositionDestruction,
		Actor:       "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Type != domain.DispositionDestruction || !created.IsActive {
		t.Fatalf("destruction process was not created: %+v", created)
	}

	oldAuction, err := data.GetDisposition(ctx, "proc-003")
	if err != nil {
		t.Fatal(err)
	}
	if oldAuction.IsActive || oldAuction.StatusCode != "dialihkan_musnah" {
		t.Fatalf("failed auction was not archived as transferred: %+v", oldAuction)
	}
	after, err := data.GetInventory(ctx, before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.CurrentDisposition != domain.DispositionDestruction || after.StatusCode != "proses_musnah" {
		t.Fatalf("inventory was not moved to destruction: %+v", after)
	}
}

func TestCorrectInventoryDataUpdatesBusinessDataAndDocuments(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	item, err := data.GetInventory(ctx, "inv-001")
	if err != nil {
		t.Fatal(err)
	}
	events, err := data.Timeline(ctx, item.ID)
	if err != nil || len(events) == 0 {
		t.Fatalf("timeline unavailable: %v", err)
	}
	item.ReferenceNo = "BTD-2026-0041-REV"
	item.DeterminationNo = "BCF-REV-001"
	item.Description = "Tekstil dan produk tekstil hasil koreksi"
	item.GoodsValue = 987654321
	corrections := []domain.EventCorrection{{
		ID: events[0].ID, Label: events[0].Label, DocumentNo: "SURAT-REV-001",
		DocumentDate: time.Now(), Notes: "Nomor surat dikoreksi",
	}}
	record, updated, err := data.CorrectInventoryData(ctx, domain.InventoryCorrectionInput{
		InventoryID: item.ID, Reason: "Kesalahan input", Actor: "Tester", Item: item, Events: corrections,
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Type != "data_correction" || record.Action != "updated" {
		t.Fatalf("unexpected reconciliation record: %+v", record)
	}
	if record.CorrectionReason != "Kesalahan input" || len(record.ChangeDetails) == 0 {
		t.Fatalf("correction audit details were not recorded: %+v", record)
	}
	fields := map[string]bool{}
	for _, change := range record.ChangeDetails {
		fields[change.Section+":"+change.Field] = true
	}
	if !fields["inventory:reference_no"] || !fields["timeline:notes"] {
		t.Fatalf("expected inventory and timeline changes in audit: %+v", record.ChangeDetails)
	}
	if updated.ReferenceNo != item.ReferenceNo || updated.DeterminationNo != item.DeterminationNo || updated.GoodsValue != item.GoodsValue {
		t.Fatalf("inventory correction was not persisted: %+v", updated)
	}
	updatedEvents, err := data.Timeline(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	foundDocument, foundAudit := false, false
	for _, event := range updatedEvents {
		if event.ID == events[0].ID && event.DocumentNo == "SURAT-REV-001" {
			foundDocument = true
		}
		if event.Code == "perubahan_data_barang" && strings.Contains(event.Notes, "Kesalahan input") {
			foundAudit = true
		}
	}
	if !foundDocument || !foundAudit {
		t.Fatalf("document correction or audit event missing: %+v", updatedEvents)
	}
}

func TestFailedAuctionCanTransferToGrant(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	before, err := data.GetInventory(ctx, "inv-008")
	if err != nil {
		t.Fatal(err)
	}
	created, err := data.CreateDisposition(ctx, domain.NewDispositionInput{
		InventoryID: before.ID,
		Type:        domain.DispositionGrant,
		Actor:       "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Type != domain.DispositionGrant || !created.IsActive {
		t.Fatalf("grant process was not created: %+v", created)
	}
	oldAuction, err := data.GetDisposition(ctx, "proc-003")
	if err != nil {
		t.Fatal(err)
	}
	if oldAuction.IsActive || oldAuction.StatusCode != "dialihkan_hibah" {
		t.Fatalf("failed auction was not archived as transferred: %+v", oldAuction)
	}
	after, err := data.GetInventory(ctx, before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.CurrentDisposition != domain.DispositionGrant || after.StatusCode != "proses_hibah" {
		t.Fatalf("inventory was not moved to grant/PSP: %+v", after)
	}
}

func TestCorrectInventoryDataIsAtomicWhenProcessCorrectionIsInvalid(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	before, err := data.GetInventory(ctx, "inv-001")
	if err != nil {
		t.Fatal(err)
	}
	corrected := before
	corrected.Description = "Uraian yang tidak boleh tersimpan sebagian"
	_, _, err = data.CorrectInventoryData(ctx, domain.InventoryCorrectionInput{
		InventoryID: before.ID,
		Reason:      "Kesalahan input",
		Actor:       "Tester",
		Item:        corrected,
		Processes: []domain.DispositionCorrection{{
			ID: "process-yang-tidak-milik-barang",
		}},
	})
	if err == nil {
		t.Fatal("expected invalid process correction to fail")
	}
	after, getErr := data.GetInventory(ctx, before.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if after.Description != before.Description {
		t.Fatalf("inventory was partially changed after failed correction: before=%q after=%q", before.Description, after.Description)
	}
}

func TestCreateInventoriesAllowsMultipleGoodsInSameContainer(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	physicalUnitID := "BTD|BTD-UPLOAD-MULTI|ABCD1234567"
	base := domain.NewInventoryInput{
		Type: domain.InventoryBTD, BLNo: "BL-UPLOAD-001", BLDate: now.Add(-24 * time.Hour), DeterminationNo: "BTD-UPLOAD-MULTI", DeterminationDate: now,
		ItemKind: "Barang Umum", Unit: "Piece", OriginWarehouse: domain.TPSNames[0],
		LoadType: "FCL", ContainerNo: "ABCD 123456-7", ContainerSize: "20", PhysicalUnitID: physicalUnitID, Actor: "Tester",
	}
	first := base
	first.ReferenceNo = "BTD-UPLOAD-MULTI/01"
	first.Description = "Mesin"
	first.Quantity = 2
	first.OccupancyPrimary = true
	second := base
	second.ReferenceNo = "BTD-UPLOAD-MULTI/02"
	second.Description = "Suku cadang"
	second.Quantity = 8
	second.OccupancyPrimary = false

	created, err := data.CreateInventories(ctx, []domain.NewInventoryInput{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 || created[0].PhysicalUnitID != created[1].PhysicalUnitID {
		t.Fatalf("same-container goods were not stored as one physical unit: %+v", created)
	}
	primary := 0
	for _, item := range created {
		if item.OccupancyPrimary {
			primary++
		}
	}
	if primary != 1 {
		t.Fatalf("expected one occupancy primary row, got %d", primary)
	}
}

func TestRelocateInventoryLoadSplitsQuantityValueAndPhysicalPlacement(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type:              domain.InventoryBTD,
		BLNo:              "BL-RELOCATE-001",
		BLDate:            now.Add(-24 * time.Hour),
		DeterminationNo:   "BTD-RELOCATE-001",
		DeterminationDate: now,
		Description:       "Sepuluh peti suku cadang",
		ItemKind:          "Barang Umum",
		Quantity:          10,
		Unit:              "Piece",
		GoodsValue:        1001,
		AtTPP:             true,
		FacilityID:        "tpp-transporindo",
		OriginWarehouse:   domain.TPSNames[0],
		LoadType:          "FCL",
		ContainerNo:       "ABCD 123456-7",
		ContainerSize:     "20",
		PhysicalUnitID:    "FCL:ABCD1234567",
		OccupancyPrimary:  true,
		Actor:             "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := data.RelocateInventoryLoad(ctx, domain.InventoryLoadRelocationInput{
		InventoryID: created.ID,
		Allocations: []domain.InventoryLoadAllocation{
			{LoadType: "FCL", ContainerNo: "EFGH1234567", ContainerSize: "20", Quantity: 3},
			{LoadType: "FCL", ContainerNo: "IJKL 765432-1", ContainerSize: "40HC", Quantity: 4},
			{LoadType: "LCL", EstimatedVolumeM3: 2.75, Quantity: 3},
		},
		DocumentNo:   "BA-PINDAH-001",
		DocumentDate: now,
		Notes:        "Dibagi ke dua kontainer dan satu lot LCL.",
		Actor:        "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected three destination rows, got %d", len(rows))
	}
	if rows[0].ID != created.ID {
		t.Fatalf("the first allocation must retain the source inventory id: got %s want %s", rows[0].ID, created.ID)
	}

	var totalQuantity float64
	var totalValue int64
	seenUnits := make(map[string]int)
	for _, row := range rows {
		totalQuantity += row.Quantity
		totalValue += row.GoodsValue
		seenUnits[row.PhysicalUnitID]++
		if row.StatusCode != created.StatusCode || row.StatusLabel != created.StatusLabel {
			t.Fatalf("relocation must preserve the existing inventory status: got %s/%s want %s/%s", row.StatusCode, row.StatusLabel, created.StatusCode, created.StatusLabel)
		}
		if !row.OccupancyPrimary {
			t.Fatalf("each independent destination should be occupancy primary: %+v", row)
		}
		timeline, timelineErr := data.Timeline(ctx, row.ID)
		if timelineErr != nil {
			t.Fatal(timelineErr)
		}
		actionCount := 0
		for _, event := range timeline {
			if event.Code == "pindah_bongkar_kontainer" {
				actionCount++
			}
		}
		if actionCount != 1 {
			t.Fatalf("each result row must contain one relocation event, got %d for %s", actionCount, row.ID)
		}
	}
	if totalQuantity != 10 {
		t.Fatalf("quantity conservation failed: got %.2f", totalQuantity)
	}
	if totalValue != 1001 {
		t.Fatalf("goods value conservation failed: got %d", totalValue)
	}
	if len(seenUnits) != 3 {
		t.Fatalf("expected three independent physical placements, got %v", seenUnits)
	}
	if rows[0].ContainerNo != "EFGH 123456-7" || rows[1].ContainerNo != "IJKL 765432-1" {
		t.Fatalf("container normalization failed: %+v", rows)
	}
	if rows[2].LoadType != "LCL" || rows[2].ContainerNo != "" || rows[2].ContainerSize != "" || rows[2].EstimatedVolumeM3 != 2.75 {
		t.Fatalf("LCL allocation was not stored correctly: %+v", rows[2])
	}
}

func TestRelocateInventoryLoadAllowsProcessedInventoryWithoutSplitting(t *testing.T) {
	data := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	createProcessed := func(reference, container string) domain.InventoryItem {
		created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
			Type:              domain.InventoryBTD,
			BLNo:              "BL-" + reference,
			BLDate:            now.Add(-24 * time.Hour),
			DeterminationNo:   reference,
			DeterminationDate: now,
			Description:       "Barang dalam proses lelang",
			ItemKind:          "Barang Umum",
			Quantity:          10,
			Unit:              "Piece",
			GoodsValue:        1000,
			AtTPP:             true,
			FacilityID:        "tpp-transporindo",
			OriginWarehouse:   domain.TPSNames[0],
			LoadType:          "FCL",
			ContainerNo:       container,
			ContainerSize:     "20",
			PhysicalUnitID:    "FCL:" + strings.NewReplacer(" ", "", "-", "").Replace(container),
			OccupancyPrimary:  true,
			Actor:             "Tester",
		})
		if err != nil {
			t.Fatal(err)
		}
		data.mu.Lock()
		processed := data.items[created.ID]
		processed.CurrentDisposition = domain.DispositionAuction
		processed.StatusCode = "jadwal_lelang"
		processed.StatusLabel = "Penjadwalan Lelang"
		data.items[created.ID] = processed
		data.mu.Unlock()
		return processed
	}

	processed := createProcessed("BTD-PROCESS-001", "PROC 123456-7")
	rows, err := data.RelocateInventoryLoad(ctx, domain.InventoryLoadRelocationInput{
		InventoryID: processed.ID,
		Allocations: []domain.InventoryLoadAllocation{{LoadType: "FCL", ContainerNo: "MOVE 123456-7", ContainerSize: "40", Quantity: 10}},
		DocumentNo:  "BA-MOVE-PROCESS-001", DocumentDate: now, Actor: "Tester",
	})
	if err != nil {
		t.Fatalf("processed inventory should remain physically relocatable: %v", err)
	}
	if len(rows) != 1 || rows[0].CurrentDisposition != domain.DispositionAuction || rows[0].StatusCode != "jadwal_lelang" || rows[0].StatusLabel != "Penjadwalan Lelang" {
		t.Fatalf("relocation must preserve the active process and its status: %+v", rows)
	}

	processedSplit := createProcessed("BTD-PROCESS-002", "LOCK 123456-7")
	_, err = data.RelocateInventoryLoad(ctx, domain.InventoryLoadRelocationInput{
		InventoryID: processedSplit.ID,
		Allocations: []domain.InventoryLoadAllocation{
			{LoadType: "FCL", ContainerNo: "SPLT 123456-7", ContainerSize: "20", Quantity: 5},
			{LoadType: "LCL", EstimatedVolumeM3: 2, Quantity: 5},
		},
		DocumentNo: "BA-MOVE-PROCESS-002", DocumentDate: now, Actor: "Tester",
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("processed inventory must not be split across destinations, got %v", err)
	}
}
