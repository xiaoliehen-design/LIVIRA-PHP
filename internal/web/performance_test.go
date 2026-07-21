package web

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"testing"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestBuildPerformanceReportUsesOriginalDeterminationAndGroupsDocuments(t *testing.T) {
	date := func(value string) time.Time {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	items := []domain.InventoryItem{
		{ID: "inv-a", Type: domain.InventoryBTD, OriginType: domain.InventoryBTD, DeterminationNo: "KEP-BTD-1", DeterminationDate: date("2026-01-01")},
		{ID: "inv-b", Type: domain.InventoryBMMN, OriginType: domain.InventoryBDN, DeterminationNo: "KEP-BMMN-1", DeterminationDate: date("2026-02-01"), OriginDocumentNo: "KEP-BDN-1", OriginDocumentDate: date("2026-01-01")},
		{ID: "inv-c", Type: domain.InventoryBDN, OriginType: domain.InventoryBDN, DeterminationNo: "KEP-BDN-2", DeterminationDate: date("2026-02-01")},
		{ID: "inv-d", Type: domain.InventoryBMMN, OriginType: domain.InventoryBTD, DeterminationNo: "KEP-BMMN-2", DeterminationDate: date("2026-02-10"), OriginDocumentNo: "KEP-BTD-2", OriginDocumentDate: date("2026-01-10")},
	}
	events := []domain.TimelineEvent{
		{ID: "e1", InventoryID: "inv-a", Code: "pencacahan", DocumentNo: "BA-CACAH-1", DocumentDate: date("2026-01-11"), CreatedAt: date("2026-01-11")},
		{ID: "e2", InventoryID: "inv-a", Code: "request_penelitian_pfpd", DocumentNo: "REQ-1", DocumentDate: date("2026-01-12"), CreatedAt: date("2026-01-12")},
		{ID: "e3", InventoryID: "inv-a", Code: "penelitian_pfpd", DocumentNo: "PFPD-1", DocumentDate: date("2026-01-15"), CreatedAt: date("2026-01-15")},
		{ID: "e4", InventoryID: "inv-a", DispositionID: "proc-a", Code: "selesai_lelang", DocumentNo: "RISALAH-1", DocumentDate: date("2026-03-01"), CreatedAt: date("2026-03-01")},
		{ID: "e5", InventoryID: "inv-b", DispositionID: "proc-b", Code: "selesai_lelang", DocumentNo: "RISALAH-1", DocumentDate: date("2026-03-01"), CreatedAt: date("2026-03-01")},
		{ID: "e6", InventoryID: "inv-b", Code: "penetapan_bmmn", DocumentNo: "KEP-BMMN-1", DocumentDate: date("2026-02-01"), CreatedAt: date("2026-02-01")},
		{ID: "e7", InventoryID: "inv-c", DispositionID: "proc-c", Code: "ba_musnah", DocumentNo: "BA-MUSNAH-1", DocumentDate: date("2026-03-03"), CreatedAt: date("2026-03-03")},
		{ID: "e8", InventoryID: "inv-d", DispositionID: "proc-d", Code: "ba_serah_terima", DocumentNo: "BA-HIBAH-1", DocumentDate: date("2026-03-10"), CreatedAt: date("2026-03-10")},
	}

	report := buildPerformanceReport(items, events, date("2026-01-01"), date("2026-12-31"))
	if report.TotalCompleted != 6 {
		t.Fatalf("expected 6 grouped completions, got %d", report.TotalCompleted)
	}
	metrics := make(map[string]PerformanceMetric)
	for _, metric := range report.Metrics {
		metrics[metric.Code] = metric
	}
	if metrics[performanceAuction].Count != 1 {
		t.Fatalf("same auction document across two goods must count once, got %d", metrics[performanceAuction].Count)
	}
	if math.Abs(metrics[performanceAuction].AverageDays-59) > 0.001 {
		t.Fatalf("auction duration must use original determination, got %.2f days", metrics[performanceAuction].AverageDays)
	}
	if math.Abs(metrics[performanceCensus].AverageDays-10) > 0.001 {
		t.Fatalf("census average mismatch: %.2f", metrics[performanceCensus].AverageDays)
	}
	if math.Abs(metrics[performancePFPD].AverageDays-3) > 0.001 {
		t.Fatalf("PFPD duration must start from request, got %.2f", metrics[performancePFPD].AverageDays)
	}
	if math.Abs(metrics[performanceBMMN].AverageDays-31) > 0.001 {
		t.Fatalf("BMMN conversion duration must use origin date, got %.2f", metrics[performanceBMMN].AverageDays)
	}
}

func TestBuildPerformanceWorkbookProducesTwoReadableSheets(t *testing.T) {
	report := PerformanceReport{
		DateFrom:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DateTo:         time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		PeriodLabel:    "Tahun 2026",
		TotalCompleted: 1,
		Metrics:        []PerformanceMetric{{Code: performanceAuction, Label: "Performa lelang", Count: 1, DurationSamples: 1, AverageHours: 48, AverageDays: 2, Description: "Uji"}},
		Details:        []PerformanceDetail{{MetricCode: performanceAuction, MetricLabel: "Performa lelang", CompletionDocument: "RISALAH-1", CompletionDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), StartDocument: "KEP-1", StartDate: time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC), DurationHours: 48, DurationDays: 2, DurationValid: true, InventoryCount: 2}},
	}
	payload, err := buildPerformanceWorkbook(report)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("invalid xlsx zip: %v", err)
	}
	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		files[file.Name] = file
	}
	for _, name := range []string{"xl/workbook.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml", "xl/styles.xml"} {
		file := files[name]
		if file == nil {
			t.Fatalf("missing workbook part %s", name)
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil || len(content) == 0 {
			t.Fatalf("empty workbook part %s", name)
		}
	}
}

func TestBuildPerformanceReportAvoidsLegacyCompletionDuplicates(t *testing.T) {
	date := func(value string) time.Time {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	items := []domain.InventoryItem{
		{ID: "auction-item", Type: domain.InventoryBDN, DeterminationNo: "KEP-BDN-1", DeterminationDate: date("2026-01-01")},
		{ID: "legacy-auction-item", Type: domain.InventoryBDN, DeterminationNo: "KEP-BDN-3", DeterminationDate: date("2026-01-02")},
		{ID: "grant-item", Type: domain.InventoryBMMN, OriginType: domain.InventoryBDN, OriginDocumentNo: "KEP-BDN-2", OriginDocumentDate: date("2026-01-05")},
	}
	events := []domain.TimelineEvent{
		// Rekaman lama: status hasil lelang tersimpan di samping event resmi.
		{ID: "auction-official", InventoryID: "auction-item", Code: "selesai_lelang", DocumentNo: "RISALAH-1", DocumentDate: date("2026-02-01"), CreatedAt: date("2026-02-01")},
		{ID: "auction-legacy", InventoryID: "auction-item", DispositionID: "proc-1", Code: "tidak_laku", CreatedAt: date("2026-02-02")},
		// Jika hanya kode lama tersedia, event yang terhubung ke proses dipakai.
		{ID: "legacy-copy", InventoryID: "legacy-auction-item", Code: "tidak_laku", CreatedAt: date("2026-02-03")},
		{ID: "legacy-process", InventoryID: "legacy-auction-item", DispositionID: "proc-legacy", Code: "tidak_laku", CreatedAt: date("2026-02-03")},
		// Rekaman lama: BA serah terima tersalin tanpa referensi proses.
		{ID: "grant-copy", InventoryID: "grant-item", Code: "ba_serah_terima", CreatedAt: date("2026-03-01")},
		{ID: "grant-official", InventoryID: "grant-item", DispositionID: "proc-2", Code: "ba_serah_terima", DocumentNo: "BA-HIBAH-1", DocumentDate: date("2026-03-01"), CreatedAt: date("2026-03-01")},
	}

	report := buildPerformanceReport(items, events, date("2026-01-01"), date("2026-12-31"))
	metrics := make(map[string]PerformanceMetric)
	for _, metric := range report.Metrics {
		metrics[metric.Code] = metric
	}
	if metrics[performanceAuction].Count != 2 {
		t.Fatalf("legacy auction status must not duplicate official completion, got %d", metrics[performanceAuction].Count)
	}
	if metrics[performanceGrant].Count != 1 {
		t.Fatalf("copied grant status must not duplicate disposition event, got %d", metrics[performanceGrant].Count)
	}
}
