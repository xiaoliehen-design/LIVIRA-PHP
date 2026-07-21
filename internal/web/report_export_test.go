package web

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestBuildReportXLSXWorkbook(t *testing.T) {
	data := reportExportData{
		Title:          "Laporan BTD & Inventory",
		Headers:        []string{"Nomor", "Uraian", "Jumlah", "Nilai"},
		Rows:           [][]string{{"=RISIKO", "Mesin & suku cadang", "12", "250000000"}},
		NumericColumns: map[int]string{2: "integer", 3: "currency"},
	}
	payload, err := buildReportXLSXWorkbook(data)
	if err != nil {
		t.Fatalf("build workbook: %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("PK")) {
		t.Fatal("xlsx payload is not a ZIP workbook")
	}
	archive, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("open xlsx archive: %v", err)
	}
	files := map[string]string{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatalf("read %s: %v", file.Name, err)
		}
		files[file.Name] = string(content)
		if strings.HasSuffix(file.Name, ".xml") {
			decoder := xml.NewDecoder(bytes.NewReader(content))
			for {
				if _, err := decoder.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("invalid XML in %s: %v", file.Name, err)
				}
			}
		}
	}
	for _, required := range []string{"[Content_Types].xml", "xl/workbook.xml", "xl/styles.xml", "xl/worksheets/sheet1.xml"} {
		if files[required] == "" {
			t.Fatalf("xlsx file %s is missing", required)
		}
	}
	sheet := files["xl/worksheets/sheet1.xml"]
	for _, expected := range []string{"Laporan BTD &amp; Inventory", "Mesin &amp; suku cadang", `s="6"><v>12.00</v>`, `s="7"><v>250000000.00</v>`, `<autoFilter ref="A4:D5"/>`} {
		if !strings.Contains(sheet, expected) {
			t.Fatalf("sheet did not contain %q", expected)
		}
	}
	if !strings.Contains(sheet, "=RISIKO") {
		t.Fatal("text beginning with equals was not preserved as a text cell")
	}
	for _, required := range []string{`<dimension ref="A1:D5"/>`, `<selection pane="bottomLeft" activeCell="A5" sqref="A5"/>`} {
		if !strings.Contains(sheet, required) {
			t.Fatalf("sheet did not contain required workbook metadata %q", required)
		}
	}
	for _, ordered := range [][2]string{{"<autoFilter", "<mergeCells"}, {"<mergeCells", "<pageMargins"}, {"<pageMargins", "<pageSetup"}} {
		left, right := strings.Index(sheet, ordered[0]), strings.Index(sheet, ordered[1])
		if left < 0 || right < 0 || left >= right {
			t.Fatalf("invalid worksheet element order: %s must appear before %s", ordered[0], ordered[1])
		}
	}
}

func TestBuildLegacyExcelWorkbook(t *testing.T) {
	payload := buildLegacyExcelWorkbook(reportExportData{
		Title:          "Laporan Rekonsiliasi",
		Headers:        []string{"Referensi", "Nilai"},
		Rows:           [][]string{{"INV-001", "1000"}},
		NumericColumns: map[int]string{1: "currency"},
	})
	if err := validateLegacyExcelWorkbook(payload); err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, expected := range []string{"application progid=", "Laporan Rekonsiliasi", `ss:StyleID="Currency"`, `ss:Type="Number">1000`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("legacy Excel workbook did not contain %q", expected)
		}
	}
}
