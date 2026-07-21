package web

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

type reportExportFormat string

const (
	reportExportCSV  reportExportFormat = "csv"
	reportExportXLSX reportExportFormat = "xlsx"
	reportExportXLS  reportExportFormat = "xls"
)

var errReportExportForbidden = errors.New("report export forbidden")

type reportExportData struct {
	FilenameBase   string
	Title          string
	Headers        []string
	Rows           [][]string
	NumericColumns map[int]string
	AuditAction    string
	AuditFields    map[string]any
}

func (s *Server) exportReport(w http.ResponseWriter, r *http.Request, format reportExportFormat) {
	data, err := s.buildReportExportData(r)
	if errors.Is(err, errReportExportForbidden) {
		http.Error(w, "akses ditolak", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, friendlyError(err), http.StatusInternalServerError)
		return
	}

	var payload []byte
	switch format {
	case reportExportXLSX:
		payload, err = buildReportXLSXWorkbook(data)
		if err != nil {
			http.Error(w, "File Excel belum dapat dibuat.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="`+data.FilenameBase+`.xlsx"`)
	case reportExportXLS:
		payload = buildLegacyExcelWorkbook(data)
		w.Header().Set("Content-Type", "application/vnd.ms-excel; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+data.FilenameBase+`.xls"`)
	default:
		payload, err = buildCSVReport(data)
		if err != nil {
			http.Error(w, "File CSV belum dapat dibuat.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+data.FilenameBase+`.csv"`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	_, _ = w.Write(payload)

	fields := data.AuditFields
	if fields == nil {
		fields = map[string]any{}
	}
	fields["format"] = string(format)
	s.writeAudit(r, data.AuditAction, "report", data.FilenameBase, "success", fields)
}

func (s *Server) buildReportExportData(r *http.Request) (reportExportData, error) {
	session, _ := sessionFromContext(r.Context())
	preset := strings.TrimSpace(r.URL.Query().Get("preset"))
	now := time.Now()

	if preset == "reconciliation" || preset == "data_correction" {
		if !session.Can(domain.PermissionReconciliationView) {
			return reportExportData{}, errReportExportForbidden
		}
		records, err := s.store.ListReconciliations(r.Context(), 20000)
		if err != nil {
			return reportExportData{}, err
		}
		records = filterReconciliationsForSession(session, records)
		reconciliations, dataCorrections := splitReconciliationRecords(records)
		if preset == "data_correction" {
			changeRows := flattenDataCorrectionRows(dataCorrections)
			rows := make([][]string, 0, len(changeRows))
			for _, row := range changeRows {
				record := row.Record
				if row.Legacy {
					rows = append(rows, []string{record.CreatedAt.Format("2006-01-02 15:04"), record.InventoryReference, string(record.InventoryType), "Audit lama", "", "Rincian perubahan belum tersedia", "", "", record.CorrectionReason, record.Actor})
					continue
				}
				change := row.Change
				rows = append(rows, []string{
					record.CreatedAt.Format("2006-01-02 15:04"), record.InventoryReference, string(record.InventoryType),
					correctionSectionLabel(change.Section), change.Context, correctionFieldLabel(change.Field),
					correctionDisplayValue(change.Field, change.Before), correctionDisplayValue(change.Field, change.After),
					record.CorrectionReason, record.Actor,
				})
			}
			return reportExportData{
				FilenameBase: "livira-perubahan-data-barang-" + now.Format("20060102"),
				Title:        "Rekap Perubahan Data Barang",
				Headers:      []string{"Tanggal", "Referensi Inventory", "Jenis Inventory", "Bagian Data", "Konteks", "Data yang Diubah", "Nilai Sebelum", "Nilai Sesudah", "Alasan Perubahan", "Petugas"},
				Rows:         rows,
				AuditAction:  "report.data_correction.export",
				AuditFields:  map[string]any{"records": len(dataCorrections), "rows": len(changeRows)},
			}, nil
		}

		rows := make([][]string, 0, len(reconciliations))
		for _, record := range reconciliations {
			typeLabel := "Tercatat di aplikasi tetapi tidak ada di lapangan"
			if record.Type == "found_not_recorded" {
				typeLabel = "Tidak tercatat di aplikasi tetapi ada di lapangan"
			}
			actionLabel := "Dikeluarkan dari inventory aktif"
			if record.Action == "added" {
				actionLabel = "Ditambahkan ke inventory"
			}
			rows = append(rows, []string{record.CreatedAt.Format("2006-01-02 15:04"), typeLabel, actionLabel, record.InventoryReference, string(record.InventoryType), record.PreviousStatusLabel, record.ResultStatusLabel, record.Notes, record.Actor})
		}
		return reportExportData{
			FilenameBase: "livira-rekonsiliasi-" + now.Format("20060102"),
			Title:        "Rekap Rekonsiliasi",
			Headers:      []string{"Tanggal", "Jenis Rekonsiliasi", "Tindakan", "Referensi Inventory", "Jenis Barang", "Status Sebelumnya", "Status Hasil", "Catatan", "Petugas"},
			Rows:         rows,
			AuditAction:  "report.reconciliation.export",
			AuditFields:  map[string]any{"rows": len(reconciliations)},
		}, nil
	}

	if preset == "btd" {
		items, _, report, err := s.reportItems(r, 20000)
		if err != nil {
			return reportExportData{}, err
		}
		btdRows := buildBTDReportRows(items)
		rows := make([][]string, 0, len(btdRows))
		for _, row := range btdRows {
			rows = append(rows, []string{
				row.DeterminationNo, row.DeterminationDate.Format("2006-01-02"), row.BLNo, row.BLDate,
				row.ManifestNo, row.ManifestDate, row.ManifestPosition, row.LoadType,
				row.OriginWarehouse, row.FacilityName, row.LocationStatus, row.ContainerSummary,
				strconv.Itoa(row.ContainerCount), row.GoodsSummary, strconv.Itoa(row.ItemCount), strconv.FormatInt(row.TotalValue, 10),
				row.OwnerName, row.StatusLabel, row.InventoryStatus,
			})
		}
		return reportExportData{
			FilenameBase:   reportFilename(report),
			Title:          "Laporan BTD",
			Headers:        []string{"Nomor BTD", "Tanggal BTD", "Nomor BL", "Tanggal BL", "Nomor Manifest", "Tanggal Manifest", "Pos Manifest", "Jenis Muatan", "TPS Asal", "TPP", "Status Lokasi", "Kontainer / LCL", "Jumlah Kontainer", "Uraian, Jenis, Kondisi, dan Jumlah Barang", "Jumlah Rincian Barang", "Total Nilai Barang", "Pemilik / Shipper / Consignee", "Status Barang", "Status Inventory"},
			Rows:           rows,
			NumericColumns: map[int]string{12: "integer", 14: "integer", 15: "currency"},
			AuditAction:    "report.btd.export",
			AuditFields:    map[string]any{"documents": len(btdRows), "items": len(items)},
		}, nil
	}

	items, _, report, err := s.reportItems(r, 20000)
	if err != nil {
		return reportExportData{}, err
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "Selesai"
		if item.IsActive {
			state = "Aktif"
		}
		rows = append(rows, []string{item.DeterminationNo, string(item.Type), string(item.OriginType), item.Category, item.AllocationPurpose, item.DeterminationDate.Format("2006-01-02"), strconv.Itoa(item.AgeDays(now)), item.ContainerNo, item.ManifestNo, item.Description, item.ItemKind, item.GoodsCondition, strconv.FormatFloat(item.Quantity, 'f', -1, 64), item.Unit, strconv.FormatInt(item.GoodsValue, 10), item.OwnerName, item.OriginWarehouse, item.LocationStatus, item.Location, item.FacilityName, item.StatusLabel, processLabel(item.CurrentDisposition), state})
	}
	return reportExportData{
		FilenameBase:   reportFilename(report),
		Title:          report.Title,
		Headers:        []string{"Nomor Penetapan", "Jenis Saat Ini", "Jenis Asal", "Kategori BDN", "Peruntukan BMMN", "Tanggal Penetapan", "Umur (Hari)", "Nomor Kontainer", "Nomor Manifest", "Uraian Barang", "Jenis Barang", "Kondisi Barang", "Jumlah", "Satuan", "Nilai Barang", "Pemilik", "TPS Asal", "Status Lokasi", "Lokasi", "TPP", "Status Barang", "Proses", "Status Inventory"},
		Rows:           rows,
		NumericColumns: map[int]string{6: "integer", 12: "decimal", 14: "currency"},
		AuditAction:    "report.inventory.export",
		AuditFields:    map[string]any{"rows": len(items), "scope": report.Scope, "date_from": report.DateFrom, "date_to": report.DateTo},
	}, nil
}

func buildCSVReport(data reportExportData) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(csvSafeRow(data.Headers)); err != nil {
		return nil, err
	}
	for _, row := range data.Rows {
		if err := writer.Write(csvSafeRow(row)); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func buildReportXLSXWorkbook(data reportExportData) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
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
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><bookViews><workbookView xWindow="0" yWindow="0" windowWidth="24000" windowHeight="12000"/></bookViews><sheets><sheet name="Data Laporan" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/styles.xml":            reportXLSXStylesXML(),
		"xl/worksheets/sheet1.xml": reportXLSXSheetXML(data),
		"docProps/core.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:creator>LIVIRA</dc:creator><cp:lastModifiedBy>LIVIRA</cp:lastModifiedBy><dc:title>` + xmlEscape(data.Title) + `</dc:title><dcterms:created xsi:type="dcterms:W3CDTF">` + time.Now().UTC().Format(time.RFC3339) + `</dcterms:created></cp:coreProperties>`,
		"docProps/app.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>LIVIRA</Application></Properties>`,
	}
	order := []string{"[Content_Types].xml", "_rels/.rels", "docProps/core.xml", "docProps/app.xml", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/worksheets/sheet1.xml"}
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

func reportXLSXStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<numFmts count="3"><numFmt numFmtId="164" formatCode="#,##0.00"/><numFmt numFmtId="165" formatCode="#,##0"/><numFmt numFmtId="166" formatCode="&quot;Rp&quot; #,##0"/></numFmts>
<fonts count="3"><font><sz val="10"/><name val="Aptos"/></font><font><b/><sz val="10"/><color rgb="FFFFFFFF"/><name val="Aptos"/></font><font><b/><sz val="15"/><color rgb="FF17384B"/><name val="Aptos Display"/></font></fonts>
<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF0F766E"/><bgColor indexed="64"/></patternFill></fill></fills>
<borders count="2"><border/><border><left style="thin"><color rgb="FFDDE5EA"/></left><right style="thin"><color rgb="FFDDE5EA"/></right><top style="thin"><color rgb="FFDDE5EA"/></top><bottom style="thin"><color rgb="FFDDE5EA"/></bottom></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="8"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf><xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf><xf numFmtId="164" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyAlignment="1"><alignment horizontal="right" vertical="top"/></xf><xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment wrapText="1"/></xf><xf numFmtId="165" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyAlignment="1"><alignment horizontal="right" vertical="top"/></xf><xf numFmtId="166" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyAlignment="1"><alignment horizontal="right" vertical="top"/></xf></cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`
}

func reportXLSXSheetXML(data reportExportData) string {
	columnCount := len(data.Headers)
	var rows strings.Builder
	lastColumn := xlsxColumnLetters(columnCount)
	rows.WriteString(`<row r="1" ht="24" customHeight="1">` + xlsxStringCell("A1", data.Title, 4) + `</row>`)
	rows.WriteString(`<row r="2">` + xlsxStringCell("A2", "Dibuat oleh LIVIRA pada "+time.Now().Format("02-01-2006 15:04"), 5) + `</row>`)
	rows.WriteString(`<row r="3"></row>`)
	headerCells := make([]string, 0, columnCount)
	for column, header := range data.Headers {
		headerCells = append(headerCells, xlsxStringCell(xlsxCellRef(column+1, 4), header, 1))
	}
	rows.WriteString(`<row r="4" ht="34" customHeight="1">` + strings.Join(headerCells, "") + `</row>`)
	for index, values := range data.Rows {
		rowNumber := index + 5
		cells := make([]string, 0, columnCount)
		for column := 0; column < columnCount; column++ {
			value := ""
			if column < len(values) {
				value = values[column]
			}
			reference := xlsxCellRef(column+1, rowNumber)
			kind := data.NumericColumns[column]
			if kind != "" && strings.TrimSpace(value) != "" {
				if number, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
					style := 3
					if kind == "integer" {
						style = 6
					} else if kind == "currency" {
						style = 7
					}
					cells = append(cells, xlsxNumberCell(reference, number, style))
					continue
				}
			}
			cells = append(cells, xlsxStringCell(reference, value, 2))
		}
		rows.WriteString(xlsxRow(rowNumber, cells))
	}
	lastRow := len(data.Rows) + 4
	if lastRow < 4 {
		lastRow = 4
	}
	var columns strings.Builder
	for column := 0; column < columnCount; column++ {
		width := float64(legacyExcelColumnWidth(data, column)) / 7.0
		columns.WriteString(`<col min="` + strconv.Itoa(column+1) + `" max="` + strconv.Itoa(column+1) + `" width="` + strconv.FormatFloat(width, 'f', 1, 64) + `" customWidth="1"/>`)
	}
	merge := ""
	if columnCount > 1 {
		merge = `<mergeCells count="2"><mergeCell ref="A1:` + lastColumn + `1"/><mergeCell ref="A2:` + lastColumn + `2"/></mergeCells>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1:` + lastColumn + strconv.Itoa(lastRow) + `"/><sheetViews><sheetView workbookViewId="0" showGridLines="0"><pane ySplit="4" topLeftCell="A5" activePane="bottomLeft" state="frozen"/><selection pane="bottomLeft" activeCell="A5" sqref="A5"/></sheetView></sheetViews><sheetFormatPr defaultRowHeight="15"/><cols>` + columns.String() + `</cols><sheetData>` + rows.String() + `</sheetData><autoFilter ref="A4:` + lastColumn + strconv.Itoa(lastRow) + `"/>` + merge + `<pageMargins left="0.3" right="0.3" top="0.5" bottom="0.5" header="0.2" footer="0.2"/><pageSetup orientation="landscape" fitToWidth="1" fitToHeight="0"/></worksheet>`
}

func xlsxColumnLetters(column int) string {
	if column < 1 {
		return "A"
	}
	var letters string
	for column > 0 {
		column--
		letters = string(rune('A'+column%26)) + letters
		column /= 26
	}
	return letters
}

func buildLegacyExcelWorkbook(data reportExportData) []byte {
	columnCount := len(data.Headers)
	rowCount := len(data.Rows) + 4
	var table strings.Builder
	for column := 0; column < columnCount; column++ {
		width := legacyExcelColumnWidth(data, column)
		table.WriteString(`<Column ss:AutoFitWidth="0" ss:Width="` + strconv.Itoa(width) + `"/>`)
	}
	table.WriteString(`<Row ss:Height="24"><Cell ss:StyleID="Title" ss:MergeAcross="` + strconv.Itoa(maxInt(columnCount-1, 0)) + `"><Data ss:Type="String">` + legacyExcelEscape(data.Title) + `</Data></Cell></Row>`)
	table.WriteString(`<Row><Cell ss:StyleID="Subtitle" ss:MergeAcross="` + strconv.Itoa(maxInt(columnCount-1, 0)) + `"><Data ss:Type="String">Dibuat oleh LIVIRA pada ` + legacyExcelEscape(time.Now().Format("02-01-2006 15:04")) + `</Data></Cell></Row>`)
	table.WriteString(`<Row/>`)
	table.WriteString(`<Row ss:Height="34">`)
	for _, header := range data.Headers {
		table.WriteString(`<Cell ss:StyleID="Header"><Data ss:Type="String">` + legacyExcelEscape(header) + `</Data></Cell>`)
	}
	table.WriteString(`</Row>`)
	for _, row := range data.Rows {
		table.WriteString(`<Row>`)
		for column := 0; column < columnCount; column++ {
			value := ""
			if column < len(row) {
				value = row[column]
			}
			kind := data.NumericColumns[column]
			if kind != "" && strings.TrimSpace(value) != "" {
				if number, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
					style := "Decimal"
					if kind == "integer" {
						style = "Integer"
					} else if kind == "currency" {
						style = "Currency"
					}
					table.WriteString(`<Cell ss:StyleID="` + style + `"><Data ss:Type="Number">` + strconv.FormatFloat(number, 'f', -1, 64) + `</Data></Cell>`)
					continue
				}
			}
			table.WriteString(`<Cell ss:StyleID="Text"><Data ss:Type="String">` + legacyExcelEscape(value) + `</Data></Cell>`)
		}
		table.WriteString(`</Row>`)
	}

	lastRow := rowCount
	if lastRow < 4 {
		lastRow = 4
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet" xmlns:html="http://www.w3.org/TR/REC-html40">
<DocumentProperties xmlns="urn:schemas-microsoft-com:office:office"><Author>LIVIRA</Author><Created>` + time.Now().UTC().Format(time.RFC3339) + `</Created><Title>` + legacyExcelEscape(data.Title) + `</Title></DocumentProperties>
<ExcelWorkbook xmlns="urn:schemas-microsoft-com:office:excel"><WindowHeight>12345</WindowHeight><WindowWidth>24000</WindowWidth><ProtectStructure>False</ProtectStructure><ProtectWindows>False</ProtectWindows></ExcelWorkbook>
<Styles>
<Style ss:ID="Default" ss:Name="Normal"><Alignment ss:Vertical="Top"/><Borders/><Font ss:FontName="Aptos" ss:Size="10"/><Interior/><NumberFormat/><Protection/></Style>
<Style ss:ID="Title"><Font ss:FontName="Aptos Display" ss:Size="15" ss:Bold="1" ss:Color="#17384B"/><Alignment ss:Vertical="Center"/></Style>
<Style ss:ID="Subtitle"><Font ss:FontName="Aptos" ss:Size="9" ss:Color="#64788A"/><Alignment ss:Vertical="Center"/></Style>
<Style ss:ID="Header"><Alignment ss:Horizontal="Center" ss:Vertical="Center" ss:WrapText="1"/><Borders><Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1"/><Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1"/><Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1"/><Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1"/></Borders><Font ss:FontName="Aptos" ss:Size="10" ss:Bold="1" ss:Color="#FFFFFF"/><Interior ss:Color="#0F766E" ss:Pattern="Solid"/></Style>
<Style ss:ID="Text"><Alignment ss:Vertical="Top" ss:WrapText="1"/><Borders><Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#DDE5EA"/><Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#DDE5EA"/><Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#DDE5EA"/><Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#DDE5EA"/></Borders></Style>
<Style ss:ID="Integer" ss:Parent="Text"><Alignment ss:Horizontal="Right" ss:Vertical="Top"/><NumberFormat ss:Format="#,##0"/></Style>
<Style ss:ID="Decimal" ss:Parent="Text"><Alignment ss:Horizontal="Right" ss:Vertical="Top"/><NumberFormat ss:Format="#,##0.00"/></Style>
<Style ss:ID="Currency" ss:Parent="Text"><Alignment ss:Horizontal="Right" ss:Vertical="Top"/><NumberFormat ss:Format="&quot;Rp&quot; #,##0"/></Style>
</Styles>
<Worksheet ss:Name="Data Laporan"><Table ss:ExpandedColumnCount="` + strconv.Itoa(columnCount) + `" ss:ExpandedRowCount="` + strconv.Itoa(rowCount) + `" x:FullColumns="1" x:FullRows="1">` + table.String() + `</Table>
<AutoFilter x:Range="R4C1:R` + strconv.Itoa(lastRow) + `C` + strconv.Itoa(columnCount) + `" xmlns="urn:schemas-microsoft-com:office:excel"/>
<WorksheetOptions xmlns="urn:schemas-microsoft-com:office:excel"><Selected/><FreezePanes/><FrozenNoSplit/><SplitHorizontal>4</SplitHorizontal><TopRowBottomPane>4</TopRowBottomPane><ActivePane>2</ActivePane><DoNotDisplayGridlines/><PageSetup><Layout x:Orientation="Landscape" x:FitToPage="1"/><PageMargins x:Bottom="0.5" x:Left="0.3" x:Right="0.3" x:Top="0.5"/></PageSetup><ProtectObjects>False</ProtectObjects><ProtectScenarios>False</ProtectScenarios></WorksheetOptions>
</Worksheet></Workbook>`)
}

func legacyExcelColumnWidth(data reportExportData, column int) int {
	longest := 10
	if column < len(data.Headers) {
		longest = len([]rune(data.Headers[column]))
	}
	limit := len(data.Rows)
	if limit > 250 {
		limit = 250
	}
	for row := 0; row < limit; row++ {
		if column >= len(data.Rows[row]) {
			continue
		}
		length := len([]rune(strings.ReplaceAll(data.Rows[row][column], "\n", " ")))
		if length > longest {
			longest = length
		}
	}
	if longest > 42 {
		longest = 42
	}
	width := longest*6 + 18
	if width < 72 {
		width = 72
	}
	if width > 270 {
		width = 270
	}
	return width
}

func legacyExcelEscape(value string) string {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func validateLegacyExcelWorkbook(payload []byte) error {
	if !bytes.Contains(payload, []byte("<Workbook")) || !bytes.Contains(payload, []byte("<Worksheet")) {
		return fmt.Errorf("invalid legacy Excel workbook")
	}
	return nil
}
