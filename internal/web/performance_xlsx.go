package web

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) performanceXLSX(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	from, to := performanceRange(r.URL.Query().Get("date_from"), r.URL.Query().Get("date_to"), time.Now())
	report, err := s.performanceReport(r.Context(), session, from, to)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	payload, err := buildPerformanceWorkbook(report)
	if err != nil {
		http.Error(w, "File Excel performa belum dapat dibuat.", http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("livira-performa-%s-%s.xlsx", from.Format("20060102"), to.Format("20060102"))
	s.writeAudit(r, "report.performance.export", "report", filename, "success", map[string]any{"date_from": from.Format("2006-01-02"), "date_to": to.Format("2006-01-02"), "details": len(report.Details)})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	_, _ = w.Write(payload)
}

func buildPerformanceWorkbook(report PerformanceReport) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Ringkasan" sheetId="1" r:id="rId1"/><sheet name="Rincian" sheetId="2" r:id="rId2"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`,
		"xl/styles.xml":            performanceStylesXML(),
		"xl/worksheets/sheet1.xml": performanceSummarySheetXML(report),
		"xl/worksheets/sheet2.xml": performanceDetailSheetXML(report),
		"docProps/core.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:creator>LIVIRA</dc:creator><cp:lastModifiedBy>LIVIRA</cp:lastModifiedBy><dc:title>Laporan Performa Kinerja</dc:title><dcterms:created xsi:type="dcterms:W3CDTF">` + time.Now().UTC().Format(time.RFC3339) + `</dcterms:created></cp:coreProperties>`,
		"docProps/app.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>LIVIRA</Application></Properties>`,
	}
	order := []string{"[Content_Types].xml", "_rels/.rels", "docProps/core.xml", "docProps/app.xml", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml"}
	for _, name := range order {
		writer, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write([]byte(files[name])); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func performanceStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<numFmts count="1"><numFmt numFmtId="164" formatCode="0.00"/></numFmts>
<fonts count="3"><font><sz val="11"/><name val="Aptos"/></font><font><b/><sz val="11"/><color rgb="FFFFFFFF"/><name val="Aptos"/></font><font><b/><sz val="14"/><name val="Aptos Display"/></font></fonts>
<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF16857A"/><bgColor indexed="64"/></patternFill></fill></fills>
<borders count="2"><border/><border><left style="thin"><color rgb="FFD8E3EA"/></left><right style="thin"><color rgb="FFD8E3EA"/></right><top style="thin"><color rgb="FFD8E3EA"/></top><bottom style="thin"><color rgb="FFD8E3EA"/></bottom></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="6"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf><xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf><xf numFmtId="164" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1"/><xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment wrapText="1"/></xf></cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`
}

func performanceSummarySheetXML(report PerformanceReport) string {
	var rows strings.Builder
	rows.WriteString(xlsxRow(1, []string{xlsxStringCell("A1", "Laporan Performa Kinerja LIVIRA", 4)}))
	rows.WriteString(xlsxRow(2, []string{xlsxStringCell("A2", "Periode: "+report.PeriodLabel, 5)}))
	rows.WriteString(xlsxRow(3, []string{xlsxStringCell("A3", fmt.Sprintf("Total penyelesaian terukur: %d", report.TotalCompleted), 5)}))
	headers := []string{"Kategori", "Jumlah selesai", "Rata-rata (hari)", "Rata-rata (jam)", "Sampel durasi valid", "Dasar penghitungan"}
	cells := make([]string, 0, len(headers))
	for index, header := range headers {
		cells = append(cells, xlsxStringCell(xlsxCellRef(index+1, 5), header, 1))
	}
	rows.WriteString(xlsxRow(5, cells))
	for index, metric := range report.Metrics {
		row := index + 6
		rows.WriteString(xlsxRow(row, []string{
			xlsxStringCell(xlsxCellRef(1, row), metric.Label, 2),
			xlsxNumberCell(xlsxCellRef(2, row), float64(metric.Count), 2),
			xlsxNumberCell(xlsxCellRef(3, row), metric.AverageDays, 3),
			xlsxNumberCell(xlsxCellRef(4, row), metric.AverageHours, 3),
			xlsxNumberCell(xlsxCellRef(5, row), float64(metric.DurationSamples), 2),
			xlsxStringCell(xlsxCellRef(6, row), metric.Description, 2),
		}))
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="5" topLeftCell="A6" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols><col min="1" max="1" width="27" customWidth="1"/><col min="2" max="5" width="19" customWidth="1"/><col min="6" max="6" width="58" customWidth="1"/></cols><sheetData>` + rows.String() + `</sheetData><autoFilter ref="A5:F` + strconv.Itoa(len(report.Metrics)+5) + `"/><pageMargins left="0.3" right="0.3" top="0.5" bottom="0.5" header="0.2" footer="0.2"/></worksheet>`
}

func performanceDetailSheetXML(report PerformanceReport) string {
	var rows strings.Builder
	headers := []string{"Kategori", "Dokumen selesai", "Tanggal selesai", "Dokumen awal/request", "Tanggal awal/request", "Durasi (hari)", "Durasi (jam)", "Jumlah barang"}
	cells := make([]string, 0, len(headers))
	for index, header := range headers {
		cells = append(cells, xlsxStringCell(xlsxCellRef(index+1, 1), header, 1))
	}
	rows.WriteString(xlsxRow(1, cells))
	for index, detail := range report.Details {
		row := index + 2
		completionDoc := detail.CompletionDocument
		if completionDoc == "" {
			completionDoc = "(tanpa nomor dokumen)"
		}
		startDoc := detail.StartDocument
		if startDoc == "" {
			startDoc = "—"
		}
		startDate := "—"
		if !detail.StartDate.IsZero() {
			startDate = detail.StartDate.Format("2006-01-02")
		}
		durationDaysCell := xlsxStringCell(xlsxCellRef(6, row), "—", 2)
		durationHoursCell := xlsxStringCell(xlsxCellRef(7, row), "—", 2)
		if detail.DurationValid {
			durationDaysCell = xlsxNumberCell(xlsxCellRef(6, row), detail.DurationDays, 3)
			durationHoursCell = xlsxNumberCell(xlsxCellRef(7, row), detail.DurationHours, 3)
		}
		rows.WriteString(xlsxRow(row, []string{
			xlsxStringCell(xlsxCellRef(1, row), detail.MetricLabel, 2),
			xlsxStringCell(xlsxCellRef(2, row), completionDoc, 2),
			xlsxStringCell(xlsxCellRef(3, row), detail.CompletionDate.Format("2006-01-02"), 2),
			xlsxStringCell(xlsxCellRef(4, row), startDoc, 2),
			xlsxStringCell(xlsxCellRef(5, row), startDate, 2),
			durationDaysCell,
			durationHoursCell,
			xlsxNumberCell(xlsxCellRef(8, row), float64(detail.InventoryCount), 2),
		}))
	}
	lastRow := len(report.Details) + 1
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><cols><col min="1" max="1" width="28" customWidth="1"/><col min="2" max="2" width="30" customWidth="1"/><col min="3" max="3" width="16" customWidth="1"/><col min="4" max="4" width="30" customWidth="1"/><col min="5" max="7" width="18" customWidth="1"/><col min="8" max="8" width="15" customWidth="1"/></cols><sheetData>` + rows.String() + `</sheetData><autoFilter ref="A1:H` + strconv.Itoa(lastRow) + `"/><pageMargins left="0.3" right="0.3" top="0.5" bottom="0.5" header="0.2" footer="0.2"/></worksheet>`
}

func xlsxRow(row int, cells []string) string {
	return `<row r="` + strconv.Itoa(row) + `">` + strings.Join(cells, "") + `</row>`
}

func xlsxStringCell(reference, value string, style int) string {
	return `<c r="` + reference + `" t="inlineStr" s="` + strconv.Itoa(style) + `"><is><t xml:space="preserve">` + xmlEscape(value) + `</t></is></c>`
}

func xlsxNumberCell(reference string, value float64, style int) string {
	return `<c r="` + reference + `" s="` + strconv.Itoa(style) + `"><v>` + strconv.FormatFloat(value, 'f', 2, 64) + `</v></c>`
}

func xlsxCellRef(column, row int) string {
	var letters string
	for column > 0 {
		column--
		letters = string(rune('A'+column%26)) + letters
		column /= 26
	}
	return letters + strconv.Itoa(row)
}

func xmlEscape(value string) string {
	var cleaned strings.Builder
	for _, char := range value {
		if char == '\t' || char == '\n' || char == '\r' || char >= 0x20 {
			cleaned.WriteRune(char)
		}
	}
	var output bytes.Buffer
	_ = xml.EscapeText(&output, []byte(cleaned.String()))
	return output.String()
}
