package web

import (
	"strings"
	"testing"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestBulkInventoryTemplatesParseAndValidate(t *testing.T) {
	facilities := []domain.Facility{
		{ID: "tpp-transporindo", Name: "TPP Transporindo", Active: true},
		{ID: "tpp-multi-sejahtera", Name: "TPP Multi Sejahtera", Active: true},
		{ID: "tpp-kbn-marunda", Name: "TPP KBN Marunda", Active: true},
		{ID: "tpp-graha-segara", Name: "TPP Graha Segara", Active: true},
	}
	cases := []struct {
		name     string
		file     string
		kind     domain.InventoryType
		expected int
	}{
		{name: "BTD", file: "static/templates/template_upload_btd.xlsx", kind: domain.InventoryBTD, expected: 1},
		{name: "BDN", file: "static/templates/template_upload_bdn.xlsx", kind: domain.InventoryBDN, expected: 1},
		{name: "barang titipan", file: "static/templates/template_upload_barang_titipan.xlsx", kind: domain.InventoryTitipan, expected: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := assets.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("read embedded template: %v", err)
			}
			rows, err := readXLSXRows(data, maxInventoryImportRows+1)
			if err != nil {
				t.Fatalf("parse template: %v", err)
			}
			if len(rows) < 2 {
				t.Fatalf("expected header and example row, got %d rows", len(rows))
			}
			inputs, rowErrors := buildBulkInventoryInputs(bulkInventorySpecs[tc.kind], rows, facilities, "Tester")
			if len(rowErrors) > 0 {
				t.Fatalf("template example should validate, got %+v", rowErrors)
			}
			if len(inputs) != tc.expected {
				t.Fatalf("expected %d input(s), got %d", tc.expected, len(inputs))
			}
			if inputs[0].Type != tc.kind || inputs[0].ReferenceNo == "" || inputs[0].DeterminationDate.IsZero() {
				t.Fatalf("unexpected parsed input: %+v", inputs[0])
			}
			if tc.kind == domain.InventoryBTD && (inputs[0].BLNo == "" || inputs[0].BLDate.IsZero()) {
				t.Fatalf("BTD template must include mandatory BL number and date: %+v", inputs[0])
			}
		})
	}
}

func TestBuildBulkInventoryInputsAssignsReferencesAndOnePhysicalUnit(t *testing.T) {
	data, err := assets.ReadFile("static/templates/template_upload_btd.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readXLSXRows(data, maxInventoryImportRows+1)
	if err != nil {
		t.Fatal(err)
	}
	// Template hanya menyediakan satu baris contoh. Gandakan baris tersebut
	// secara in-memory untuk tetap menguji banyak uraian dalam satu kontainer.
	rows = append(rows[:2], append([]string(nil), rows[1]...))
	rows[2][12] = "Suku cadang mesin dalam peti"
	facilities := []domain.Facility{{ID: "tpp-graha-segara", Name: "TPP Graha Segara", Active: true}}
	inputs, rowErrors := buildBulkInventoryInputs(bulkInventorySpecs[domain.InventoryBTD], rows, facilities, "Tester")
	if len(rowErrors) > 0 {
		t.Fatalf("unexpected validation errors: %+v", rowErrors)
	}
	if len(inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(inputs))
	}
	if inputs[0].ReferenceNo != inputs[0].DeterminationNo+"/01" || inputs[1].ReferenceNo != inputs[1].DeterminationNo+"/02" {
		t.Fatalf("unexpected references: %q, %q", inputs[0].ReferenceNo, inputs[1].ReferenceNo)
	}
	if inputs[0].PhysicalUnitID == "" || inputs[0].PhysicalUnitID != inputs[1].PhysicalUnitID {
		t.Fatalf("same container must share physical unit: %q vs %q", inputs[0].PhysicalUnitID, inputs[1].PhysicalUnitID)
	}
	if !inputs[0].OccupancyPrimary || inputs[1].OccupancyPrimary {
		t.Fatalf("only first goods row should count occupancy: %+v", inputs)
	}
}

func TestBuildBulkInventoryInputsRejectsSameContainerAcrossDifferentDocuments(t *testing.T) {
	data, err := assets.ReadFile("static/templates/template_upload_btd.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := readXLSXRows(data, maxInventoryImportRows+1)
	if err != nil {
		t.Fatal(err)
	}
	determinationIndex := -1
	for index, header := range rows[0] {
		if normalizeWorkbookHeader(header) == normalizeWorkbookHeader("Nomor BTD *") {
			determinationIndex = index
			break
		}
	}
	if determinationIndex < 0 {
		t.Fatal("determination column not found")
	}
	rows = append(rows[:2], append([]string(nil), rows[1]...))
	rows[2][determinationIndex] = "BTD-002/KPU.01/2026"
	facilities := []domain.Facility{{ID: "tpp-graha-segara", Name: "TPP Graha Segara", Active: true}}
	inputs, rowErrors := buildBulkInventoryInputs(bulkInventorySpecs[domain.InventoryBTD], rows, facilities, "Tester")
	if len(inputs) != 0 || len(rowErrors) == 0 {
		t.Fatalf("expected all-or-nothing validation failure, inputs=%d errors=%+v", len(inputs), rowErrors)
	}
	if !strings.Contains(strings.ToLower(rowErrors[0].Message), "tidak konsisten") {
		t.Fatalf("unexpected error: %+v", rowErrors)
	}
}
