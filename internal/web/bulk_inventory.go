package web

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/hendra/manajemen-tpp/internal/domain"
	"github.com/hendra/manajemen-tpp/internal/store"
)

const (
	maxInventoryWorkbookBytes = 6 << 20
	maxInventoryImportRows    = 1000
)

type bulkInventoryColumn struct {
	Key    string
	Header string
}

type bulkInventorySpec struct {
	Kind    domain.InventoryType
	Label   string
	Columns []bulkInventoryColumn
}

type bulkInventoryRowError struct {
	Row     int
	Message string
}

var bulkInventorySpecs = map[domain.InventoryType]bulkInventorySpec{
	domain.InventoryBTD: {
		Kind:  domain.InventoryBTD,
		Label: "Pencatatan BTD",
		Columns: []bulkInventoryColumn{
			{Key: "determination_no", Header: "Nomor BTD *"},
			{Key: "determination_date", Header: "Tanggal BTD *"},
			{Key: "bl_no", Header: "Nomor BL *"},
			{Key: "bl_date", Header: "Tanggal BL *"},
			{Key: "manifest_no", Header: "Nomor Manifest"},
			{Key: "manifest_date", Header: "Tanggal Manifest"},
			{Key: "manifest_position", Header: "Pos Manifest"},
			{Key: "load_type", Header: "Jenis Muatan *"},
			{Key: "origin_warehouse", Header: "TPS Asal *"},
			{Key: "container_no", Header: "Nomor Kontainer (FCL)"},
			{Key: "container_size", Header: "Ukuran Kontainer (FCL)"},
			{Key: "estimated_volume_m3", Header: "Perkiraan Volume m3 (LCL)"},
			{Key: "description", Header: "Uraian Barang *"},
			{Key: "item_kind", Header: "Jenis Barang *"},
			{Key: "goods_value", Header: "Nilai Awal Barang"},
			{Key: "quantity", Header: "Jumlah *"},
			{Key: "unit", Header: "Satuan *"},
			{Key: "at_tpp", Header: "Sudah di TPP? *"},
			{Key: "facility_name", Header: "Nama TPP (jika Ya)"},
			{Key: "location", Header: "Blok/Gudang di TPP"},
			{Key: "owner_name", Header: "Nama Shipper/Consignee"},
			{Key: "owner_address", Header: "Alamat Shipper/Consignee"},
		},
	},
	domain.InventoryBDN: {
		Kind:  domain.InventoryBDN,
		Label: "Penetapan BDN",
		Columns: []bulkInventoryColumn{
			{Key: "determination_no", Header: "Nomor Penetapan *"},
			{Key: "determination_date", Header: "Tanggal Penetapan *"},
			{Key: "category", Header: "Kategori BDN *"},
			{Key: "manifest_no", Header: "Nomor Manifest"},
			{Key: "manifest_date", Header: "Tanggal Manifest"},
			{Key: "manifest_position", Header: "Pos Manifest"},
			{Key: "load_type", Header: "Jenis Muatan *"},
			{Key: "origin_warehouse", Header: "TPS Asal *"},
			{Key: "container_no", Header: "Nomor Kontainer (FCL)"},
			{Key: "container_size", Header: "Ukuran Kontainer (FCL)"},
			{Key: "estimated_volume_m3", Header: "Perkiraan Volume m3 (LCL)"},
			{Key: "description", Header: "Uraian Barang *"},
			{Key: "item_kind", Header: "Jenis Barang *"},
			{Key: "goods_value", Header: "Nilai Awal Barang"},
			{Key: "quantity", Header: "Jumlah *"},
			{Key: "unit", Header: "Satuan *"},
			{Key: "at_tpp", Header: "Sudah di TPP? *"},
			{Key: "facility_name", Header: "Nama TPP (jika Ya)"},
			{Key: "location", Header: "Blok/Gudang di TPP"},
			{Key: "owner_name", Header: "Nama Shipper/Consignee"},
			{Key: "owner_address", Header: "Alamat Shipper/Consignee"},
		},
	},
	domain.InventoryTitipan: {
		Kind:  domain.InventoryTitipan,
		Label: "Pemasukan Barang Titipan",
		Columns: []bulkInventoryColumn{
			{Key: "determination_no", Header: "Nomor Dokumen Dasar Pemasukan *"},
			{Key: "determination_date", Header: "Tanggal Dokumen *"},
			{Key: "entrusted_category", Header: "Kategori Barang *"},
			{Key: "source_office", Header: "Kantor/Unit Penitip *"},
			{Key: "manifest_no", Header: "Nomor Manifest"},
			{Key: "manifest_date", Header: "Tanggal Manifest"},
			{Key: "manifest_position", Header: "Pos Manifest"},
			{Key: "load_type", Header: "Jenis Muatan *"},
			{Key: "container_no", Header: "Nomor Kontainer (FCL)"},
			{Key: "container_size", Header: "Ukuran Kontainer (FCL)"},
			{Key: "estimated_volume_m3", Header: "Perkiraan Volume m3 (LCL)"},
			{Key: "description", Header: "Uraian Barang *"},
			{Key: "item_kind", Header: "Jenis Barang *"},
			{Key: "goods_value", Header: "Nilai Awal Barang"},
			{Key: "quantity", Header: "Jumlah *"},
			{Key: "unit", Header: "Satuan *"},
			{Key: "at_tpp", Header: "Sudah di TPP? *"},
			{Key: "facility_name", Header: "Nama TPP (jika Ya)"},
			{Key: "location", Header: "Blok/Gudang di TPP"},
			{Key: "owner_name", Header: "Nama Shipper/Consignee"},
			{Key: "owner_address", Header: "Alamat Shipper/Consignee"},
		},
	},
}

func (s *Server) importInventoryWorkbook(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, maxInventoryWorkbookBytes+1<<20)
	if err := r.ParseMultipartForm(maxInventoryWorkbookBytes); err != nil {
		redirectMessage(w, r, "/inventory", "error", "File Excel terlalu besar atau form upload tidak valid. Maksimal ukuran file 6 MB.")
		return
	}
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}

	kind := domain.InventoryType(strings.ToUpper(strings.TrimSpace(r.FormValue("item_type"))))
	spec, validKind := bulkInventorySpecs[kind]
	if !validKind || !sessionCanCreateInventory(session, kind) {
		redirectMessage(w, r, "/inventory", "error", "Role Anda tidak memiliki akses untuk jenis upload tersebut.")
		return
	}

	file, header, err := r.FormFile("excel_file")
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", "Pilih file template Excel berformat .xlsx terlebih dahulu.")
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(header.Filename)), ".xlsx") {
		redirectMessage(w, r, "/inventory", "error", "Format file harus .xlsx. Gunakan template yang tersedia pada menu upload.")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxInventoryWorkbookBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxInventoryWorkbookBytes {
		redirectMessage(w, r, "/inventory", "error", "File Excel tidak dapat dibaca atau melebihi batas 6 MB.")
		return
	}

	rows, err := readXLSXRows(data, maxInventoryImportRows+1)
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", "File Excel tidak dapat dibaca. Pastikan file berasal dari template .xlsx dan tidak rusak.")
		return
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	inputs, rowErrors := buildBulkInventoryInputs(spec, rows, facilities, session.DisplayName)
	if len(rowErrors) > 0 {
		redirectMessage(w, r, "/inventory", "error", summarizeBulkInventoryErrors(rowErrors))
		return
	}
	created, err := s.store.CreateInventories(r.Context(), inputs)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			redirectMessage(w, r, "/inventory", "error", "Upload dibatalkan karena terdapat nomor dokumen/referensi atau nomor kontainer yang sudah terdaftar. Tidak ada baris baru yang disimpan.")
			return
		}
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "inventory.import.xlsx", "inventory_batch", string(kind), "success", map[string]any{"rows": len(created), "file_name": filepath.Base(header.Filename)})
	message := fmt.Sprintf("Upload %s berhasil. %d baris tersimpan dan langsung ditampilkan pada inventory.", spec.Label, len(created))
	redirectMessage(w, r, "/inventory?type="+strings.ToLower(string(kind)), "ok", message)
}

func buildBulkInventoryInputs(spec bulkInventorySpec, rows [][]string, facilities []domain.Facility, actor string) ([]domain.NewInventoryInput, []bulkInventoryRowError) {
	if len(rows) == 0 {
		return nil, []bulkInventoryRowError{{Row: 1, Message: "sheet data kosong"}}
	}
	headerIndexes := make(map[string]int, len(rows[0]))
	for index, value := range rows[0] {
		normalized := normalizeWorkbookHeader(value)
		if normalized != "" {
			headerIndexes[normalized] = index
		}
	}
	missing := make([]string, 0)
	columnIndexes := make(map[string]int, len(spec.Columns))
	for _, column := range spec.Columns {
		index, ok := headerIndexes[normalizeWorkbookHeader(column.Header)]
		if !ok {
			missing = append(missing, column.Header)
			continue
		}
		columnIndexes[column.Key] = index
	}
	if len(missing) > 0 {
		return nil, []bulkInventoryRowError{{Row: 1, Message: "kolom template tidak lengkap: " + strings.Join(missing, ", ")}}
	}

	facilityByName := make(map[string]domain.Facility, len(facilities)*2)
	for _, facility := range facilities {
		if !facility.Active {
			continue
		}
		facilityByName[normalizeWorkbookValue(facility.Name)] = facility
		facilityByName[normalizeWorkbookValue(facility.ID)] = facility
	}

	var inputs []domain.NewInventoryInput
	var inputRows []int
	var rowErrors []bulkInventoryRowError
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		if workbookRowBlank(row) {
			continue
		}
		if len(inputs) >= maxInventoryImportRows {
			rowErrors = append(rowErrors, bulkInventoryRowError{Row: rowIndex + 1, Message: fmt.Sprintf("jumlah data melebihi batas %d baris", maxInventoryImportRows)})
			break
		}
		get := func(key string) string {
			index := columnIndexes[key]
			if index < 0 || index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}

		input := domain.NewInventoryInput{Type: spec.Kind, Actor: actor}
		var problems []string
		input.DeterminationNo = get("determination_no")
		if input.DeterminationNo == "" {
			problems = append(problems, "nomor dokumen wajib diisi")
		}
		input.DeterminationDate, _ = parseWorkbookDate(get("determination_date"))
		if input.DeterminationDate.IsZero() {
			problems = append(problems, "tanggal dokumen wajib diisi dengan format dd/mm/yyyy")
		}
		if spec.Kind == domain.InventoryBTD {
			input.BLNo = get("bl_no")
			if input.BLNo == "" {
				problems = append(problems, "nomor BL wajib diisi untuk BTD")
			}
			input.BLDate, _ = parseWorkbookDate(get("bl_date"))
			if input.BLDate.IsZero() {
				problems = append(problems, "tanggal BL wajib diisi dengan format dd/mm/yyyy untuk BTD")
			}
		}
		input.ManifestNo = get("manifest_no")
		if raw := get("manifest_date"); raw != "" {
			input.ManifestDate, _ = parseWorkbookDate(raw)
			if input.ManifestDate.IsZero() {
				problems = append(problems, "tanggal manifest tidak valid")
			}
		}
		input.ManifestPosition = get("manifest_position")

		switch spec.Kind {
		case domain.InventoryBDN:
			if value, ok := canonicalWorkbookOption(get("category"), domain.CurrentBDNCategories()); ok {
				input.Category = value
			} else {
				problems = append(problems, "kategori BDN tidak sesuai pilihan aplikasi")
			}
		case domain.InventoryTitipan:
			if value, ok := canonicalWorkbookOption(get("entrusted_category"), domain.EntrustedCategoryNames); ok {
				input.EntrustedCategory = value
			} else {
				problems = append(problems, "kategori barang titipan tidak sesuai pilihan template")
			}
			input.SourceOffice = get("source_office")
			if input.SourceOffice == "" {
				problems = append(problems, "kantor/unit penitip wajib diisi")
			}
		}

		loadType := strings.ToUpper(strings.TrimSpace(get("load_type")))
		if value, ok := canonicalLoadType(loadType); ok {
			input.LoadType = value
		} else {
			problems = append(problems, "jenis muatan harus FCL atau LCL")
		}
		if spec.Kind != domain.InventoryTitipan {
			if value, ok := canonicalWorkbookOption(get("origin_warehouse"), domain.CurrentOriginTPS()); ok {
				input.OriginWarehouse = value
			} else {
				problems = append(problems, "TPS asal tidak sesuai pilihan aplikasi")
			}
		}

		if input.LoadType == "FCL" {
			number, valid := normalizeContainerNumber(get("container_no"))
			if !valid {
				problems = append(problems, "nomor kontainer FCL harus terdiri dari 4 huruf dan 7 angka")
			} else {
				input.ContainerNo = number
			}
			if size, ok := normalizeWorkbookContainerSize(get("container_size")); ok {
				input.ContainerSize = size
			} else {
				problems = append(problems, "ukuran kontainer harus 20', 40', 40' HC, atau 45' HC")
			}
		} else if input.LoadType == "LCL" {
			volume, err := parseWorkbookNumber(get("estimated_volume_m3"))
			if err != nil || volume <= 0 {
				problems = append(problems, "perkiraan volume LCL harus lebih dari 0 m3")
			} else {
				input.EstimatedVolumeM3 = volume
			}
		}

		input.Description = get("description")
		if input.Description == "" {
			problems = append(problems, "uraian barang wajib diisi")
		}
		if value, ok := canonicalWorkbookOption(get("item_kind"), domain.CurrentItemKinds()); ok {
			input.ItemKind = value
		} else {
			problems = append(problems, "jenis barang tidak sesuai pilihan aplikasi")
		}
		quantity, quantityErr := parseWorkbookNumber(get("quantity"))
		if quantityErr != nil || quantity <= 0 {
			problems = append(problems, "jumlah harus berupa angka lebih dari 0")
		} else {
			input.Quantity = quantity
		}
		if value, ok := canonicalWorkbookOption(get("unit"), domain.CurrentUnits()); ok {
			input.Unit = value
		} else {
			problems = append(problems, "satuan tidak sesuai pilihan aplikasi")
		}
		if raw := get("goods_value"); raw != "" {
			value, err := parseWorkbookMoney(raw)
			if err != nil || value < 0 {
				problems = append(problems, "nilai awal barang harus berupa angka tidak negatif")
			} else {
				input.GoodsValue = value
			}
		}

		atTPP, ok := parseWorkbookYesNo(get("at_tpp"))
		if !ok {
			problems = append(problems, "kolom Sudah di TPP? harus Ya atau Tidak")
		} else {
			input.AtTPP = atTPP
		}
		if input.AtTPP {
			facility, found := facilityByName[normalizeWorkbookValue(get("facility_name"))]
			if !found {
				problems = append(problems, "nama TPP tidak ditemukan atau tidak aktif")
			} else {
				input.FacilityID = facility.ID
			}
		}
		input.Location = get("location")
		input.OwnerName = get("owner_name")
		input.OwnerAddress = get("owner_address")

		if len(problems) > 0 {
			rowErrors = append(rowErrors, bulkInventoryRowError{Row: rowIndex + 1, Message: strings.Join(problems, "; ")})
			continue
		}
		inputs = append(inputs, input)
		inputRows = append(inputRows, rowIndex+1)
	}
	if len(inputs) == 0 && len(rowErrors) == 0 {
		rowErrors = append(rowErrors, bulkInventoryRowError{Row: 2, Message: "tidak ada baris data yang dapat diimpor"})
	}
	if len(rowErrors) > 0 {
		return nil, rowErrors
	}

	type seenPhysicalUnit struct {
		row               int
		determinationNo   string
		determinationDate string
		blNo              string
		blDate            string
		manifestNo        string
		originWarehouse   string
		facilityID        string
		containerSize     string
		volume            float64
		atTPP             bool
	}
	fclUnits := make(map[string]seenPhysicalUnit)
	lclUnits := make(map[string]seenPhysicalUnit)
	for index := range inputs {
		input := &inputs[index]
		row := inputRows[index]
		docDate := input.DeterminationDate.Format("2006-01-02")
		if input.LoadType == "FCL" {
			key := strings.ToUpper(strings.ReplaceAll(input.ContainerNo, " ", ""))
			unitID := fmt.Sprintf("%s|%s|%s", input.Type, input.DeterminationNo, key)
			if previous, found := fclUnits[key]; found {
				blDate := input.BLDate.Format("2006-01-02")
				consistent := previous.determinationNo == input.DeterminationNo && previous.determinationDate == docDate && previous.blNo == input.BLNo && previous.blDate == blDate && previous.manifestNo == input.ManifestNo && previous.originWarehouse == input.OriginWarehouse && previous.facilityID == input.FacilityID && previous.containerSize == input.ContainerSize && previous.atTPP == input.AtTPP
				if !consistent {
					rowErrors = append(rowErrors, bulkInventoryRowError{Row: row, Message: fmt.Sprintf("nomor kontainer sama dengan baris %d tetapi dokumen, ukuran, manifest, nomor/tanggal BL, TPS/TPP, atau status lokasinya tidak konsisten", previous.row)})
					continue
				}
				input.PhysicalUnitID = unitID
				input.OccupancyPrimary = false
			} else {
				fclUnits[key] = seenPhysicalUnit{row: row, determinationNo: input.DeterminationNo, determinationDate: docDate, blNo: input.BLNo, blDate: input.BLDate.Format("2006-01-02"), manifestNo: input.ManifestNo, originWarehouse: input.OriginWarehouse, facilityID: input.FacilityID, containerSize: input.ContainerSize, atTPP: input.AtTPP}
				input.PhysicalUnitID = unitID
				input.OccupancyPrimary = true
			}
			continue
		}
		key := fmt.Sprintf("%s|%s|%s", input.Type, input.DeterminationNo, docDate)
		if previous, found := lclUnits[key]; found {
			consistent := previous.blNo == input.BLNo && previous.blDate == input.BLDate.Format("2006-01-02") && previous.manifestNo == input.ManifestNo && previous.originWarehouse == input.OriginWarehouse && previous.facilityID == input.FacilityID && previous.atTPP == input.AtTPP && math.Abs(previous.volume-input.EstimatedVolumeM3) < 0.000001
			if !consistent {
				rowErrors = append(rowErrors, bulkInventoryRowError{Row: row, Message: fmt.Sprintf("baris LCL satu dokumen dengan baris %d memiliki volume, manifest, nomor/tanggal BL, TPS/TPP, atau status lokasi yang tidak konsisten", previous.row)})
				continue
			}
			input.PhysicalUnitID = key
			input.OccupancyPrimary = false
		} else {
			lclUnits[key] = seenPhysicalUnit{row: row, blNo: input.BLNo, blDate: input.BLDate.Format("2006-01-02"), manifestNo: input.ManifestNo, originWarehouse: input.OriginWarehouse, facilityID: input.FacilityID, volume: input.EstimatedVolumeM3, atTPP: input.AtTPP}
			input.PhysicalUnitID = key
			input.OccupancyPrimary = true
		}
	}
	if len(rowErrors) > 0 {
		return nil, rowErrors
	}

	counts := make(map[string]int, len(inputs))
	for _, input := range inputs {
		counts[input.DeterminationNo]++
	}
	positions := make(map[string]int, len(counts))
	for index := range inputs {
		key := inputs[index].DeterminationNo
		positions[key]++
		inputs[index].ReferenceNo = key
		if counts[key] > 1 {
			inputs[index].ReferenceNo = fmt.Sprintf("%s/%02d", key, positions[key])
		}
	}
	return inputs, nil
}

func summarizeBulkInventoryErrors(rowErrors []bulkInventoryRowError) string {
	if len(rowErrors) == 0 {
		return "Upload tidak dapat diproses."
	}
	limit := len(rowErrors)
	if limit > 6 {
		limit = 6
	}
	parts := make([]string, 0, limit+1)
	for _, item := range rowErrors[:limit] {
		parts = append(parts, fmt.Sprintf("Baris %d: %s", item.Row, item.Message))
	}
	if len(rowErrors) > limit {
		parts = append(parts, fmt.Sprintf("dan %d kesalahan baris lainnya", len(rowErrors)-limit))
	}
	return "Upload dibatalkan. " + strings.Join(parts, " | ") + ". Tidak ada data yang disimpan."
}

func normalizeWorkbookHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	spacePending := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if spacePending && builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteRune(r)
			spacePending = false
		} else {
			spacePending = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func normalizeWorkbookValue(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func workbookRowBlank(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func canonicalWorkbookOption(value string, options []string) (string, bool) {
	normalized := normalizeWorkbookValue(value)
	if normalized == "" {
		return "", false
	}
	for _, option := range options {
		if normalizeWorkbookValue(option) == normalized {
			return option, true
		}
	}
	return "", false
}

func canonicalLoadType(value string) (string, bool) {
	for _, option := range domain.CurrentLoadTypes() {
		if strings.EqualFold(strings.TrimSpace(option.Code), value) || strings.EqualFold(strings.TrimSpace(option.Label), value) {
			return option.Code, true
		}
	}
	return "", false
}

func normalizeWorkbookContainerSize(value string) (string, bool) {
	compact := strings.ToUpper(strings.TrimSpace(value))
	compact = strings.NewReplacer("’", "", "'", "", "\"", "", " ", "", "-", "").Replace(compact)
	switch compact {
	case "20", "20FT":
		return "20", true
	case "40", "40FT":
		return "40", true
	case "40HC", "40H", "40HIGHCUBE":
		return "40HC", true
	case "45", "45HC", "45H", "45HIGHCUBE":
		return "45HC", true
	default:
		return "", false
	}
}

func parseWorkbookYesNo(value string) (bool, bool) {
	switch normalizeWorkbookValue(value) {
	case "ya", "y", "yes", "sudah", "true", "1":
		return true, true
	case "tidak", "n", "no", "belum", "false", "0":
		return false, true
	default:
		return false, false
	}
}

func parseWorkbookDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}
	layouts := []string{"02/01/2006", "2/1/2006", "2006-01-02", "02-01-2006", "2-1-2006", time.RFC3339}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	serial, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	if err == nil && serial > 1 {
		return excelSerialTime(serial), nil
	}
	return time.Time{}, errors.New("invalid date")
}

func parseWorkbookNumber(value string) (float64, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, " ", ""))
	if value == "" {
		return 0, errors.New("empty number")
	}
	if strings.Contains(value, ",") && !strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ",", ".")
	} else if strings.Contains(value, ",") && strings.Contains(value, ".") {
		if strings.LastIndex(value, ",") > strings.LastIndex(value, ".") {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	}
	return strconv.ParseFloat(value, 64)
}

func parseWorkbookMoney(value string) (int64, error) {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.NewReplacer("Rp", "", "RP", "", "rp", "", " ", "").Replace(cleaned)
	if cleaned == "" {
		return 0, nil
	}
	// Excel numeric cells normally arrive without separators. For formatted text,
	// Indonesian and international thousands separators are both accepted.
	if strings.Count(cleaned, ".") > 1 || strings.Count(cleaned, ",") > 1 {
		cleaned = strings.NewReplacer(".", "", ",", "").Replace(cleaned)
		return strconv.ParseInt(cleaned, 10, 64)
	}
	if strings.Contains(cleaned, ".") && strings.Contains(cleaned, ",") {
		if strings.LastIndex(cleaned, ",") > strings.LastIndex(cleaned, ".") {
			cleaned = strings.ReplaceAll(cleaned, ".", "")
			cleaned = strings.ReplaceAll(cleaned, ",", ".")
		} else {
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		}
	}
	if strings.Count(cleaned, ".") == 1 || strings.Count(cleaned, ",") == 1 {
		separator := "."
		if strings.Contains(cleaned, ",") {
			separator = ","
		}
		parts := strings.Split(cleaned, separator)
		if len(parts) == 2 && len(parts[1]) == 3 {
			cleaned = parts[0] + parts[1]
		} else {
			cleaned = strings.ReplaceAll(cleaned, ",", ".")
			parsed, err := strconv.ParseFloat(cleaned, 64)
			if err != nil {
				return 0, err
			}
			return int64(math.Round(parsed)), nil
		}
	}
	return strconv.ParseInt(cleaned, 10, 64)
}

// Minimal XLSX reader using only the Go standard library. It intentionally reads
// the first worksheet because all official upload templates put the data sheet first.
func readXLSXRows(data []byte, maxDataRows int) ([][]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[path.Clean(file.Name)] = file
	}
	worksheetPath := firstXLSXWorksheetPath(files)
	worksheetFile := files[worksheetPath]
	if worksheetFile == nil {
		return nil, errors.New("worksheet not found")
	}
	sharedStrings, _ := readXLSXSharedStrings(files["xl/sharedStrings.xml"])
	dateStyles, _ := readXLSXDateStyles(files["xl/styles.xml"])
	return readXLSXWorksheet(worksheetFile, sharedStrings, dateStyles, maxDataRows)
}

type xlsxWorkbook struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
		RID  string `xml:"id,attr"`
	} `xml:"sheets>sheet"`
}

type xlsxRelationships struct {
	Items []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

func firstXLSXWorksheetPath(files map[string]*zip.File) string {
	workbookFile := files["xl/workbook.xml"]
	relsFile := files["xl/_rels/workbook.xml.rels"]
	if workbookFile != nil && relsFile != nil {
		var workbook xlsxWorkbook
		var relationships xlsxRelationships
		if readXLSXXML(workbookFile, &workbook) == nil && readXLSXXML(relsFile, &relationships) == nil && len(workbook.Sheets) > 0 {
			targetByID := make(map[string]string, len(relationships.Items))
			for _, item := range relationships.Items {
				targetByID[item.ID] = item.Target
			}
			if target := targetByID[workbook.Sheets[0].RID]; target != "" {
				target = strings.TrimPrefix(target, "/")
				if !strings.HasPrefix(target, "xl/") {
					target = path.Join("xl", target)
				}
				return path.Clean(target)
			}
		}
	}
	return "xl/worksheets/sheet1.xml"
}

type xlsxSharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func readXLSXSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	var document xlsxSharedStrings
	if err := readXLSXXML(file, &document); err != nil {
		return nil, err
	}
	values := make([]string, len(document.Items))
	for index, item := range document.Items {
		if item.Text != "" {
			values[index] = item.Text
			continue
		}
		var builder strings.Builder
		for _, run := range item.Runs {
			builder.WriteString(run.Text)
		}
		values[index] = builder.String()
	}
	return values, nil
}

type xlsxStyles struct {
	NumFmts []struct {
		ID   int    `xml:"numFmtId,attr"`
		Code string `xml:"formatCode,attr"`
	} `xml:"numFmts>numFmt"`
	CellXfs []struct {
		NumFmtID int `xml:"numFmtId,attr"`
	} `xml:"cellXfs>xf"`
}

func readXLSXDateStyles(file *zip.File) (map[int]bool, error) {
	styles := make(map[int]bool)
	if file == nil {
		return styles, nil
	}
	var document xlsxStyles
	if err := readXLSXXML(file, &document); err != nil {
		return styles, err
	}
	custom := make(map[int]string, len(document.NumFmts))
	for _, item := range document.NumFmts {
		custom[item.ID] = item.Code
	}
	for index, xf := range document.CellXfs {
		if builtinXLSXDateFormat(xf.NumFmtID) || customXLSXDateFormat(custom[xf.NumFmtID]) {
			styles[index] = true
		}
	}
	return styles, nil
}

func builtinXLSXDateFormat(id int) bool {
	if id >= 14 && id <= 22 {
		return true
	}
	if id >= 27 && id <= 36 {
		return true
	}
	if id >= 45 && id <= 47 {
		return true
	}
	return id >= 50 && id <= 58
}

func customXLSXDateFormat(code string) bool {
	code = strings.ToLower(code)
	code = strings.ReplaceAll(code, "\\", "")
	return (strings.Contains(code, "yy") || strings.Contains(code, "dd")) && strings.Contains(code, "m")
}

type xlsxWorksheet struct {
	Rows []struct {
		Number int `xml:"r,attr"`
		Cells  []struct {
			Reference string `xml:"r,attr"`
			Type      string `xml:"t,attr"`
			Style     int    `xml:"s,attr"`
			Value     string `xml:"v"`
			Inline    struct {
				Text string `xml:"t"`
				Runs []struct {
					Text string `xml:"t"`
				} `xml:"r"`
			} `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func readXLSXWorksheet(file *zip.File, sharedStrings []string, dateStyles map[int]bool, maxDataRows int) ([][]string, error) {
	var document xlsxWorksheet
	if err := readXLSXXML(file, &document); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, minInt(len(document.Rows), maxDataRows+1))
	for _, sourceRow := range document.Rows {
		if len(rows) >= maxDataRows+1 {
			break
		}
		maxColumn := -1
		values := make(map[int]string, len(sourceRow.Cells))
		for _, cell := range sourceRow.Cells {
			column := xlsxColumnIndex(cell.Reference)
			if column < 0 {
				continue
			}
			if column > maxColumn {
				maxColumn = column
			}
			value := cell.Value
			switch cell.Type {
			case "s":
				index, _ := strconv.Atoi(cell.Value)
				if index >= 0 && index < len(sharedStrings) {
					value = sharedStrings[index]
				}
			case "inlineStr":
				value = cell.Inline.Text
				if value == "" {
					var builder strings.Builder
					for _, run := range cell.Inline.Runs {
						builder.WriteString(run.Text)
					}
					value = builder.String()
				}
			case "b":
				if cell.Value == "1" {
					value = "Ya"
				} else {
					value = "Tidak"
				}
			default:
				if dateStyles[cell.Style] && cell.Value != "" {
					if serial, err := strconv.ParseFloat(cell.Value, 64); err == nil {
						value = excelSerialTime(serial).Format("02/01/2006")
					}
				}
			}
			values[column] = value
		}
		if maxColumn < 0 {
			continue
		}
		row := make([]string, maxColumn+1)
		for column, value := range values {
			row[column] = value
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, errors.New("empty worksheet")
	}
	return rows, nil
}

func readXLSXXML(file *zip.File, target any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(reader).Decode(target)
}

func xlsxColumnIndex(reference string) int {
	index := 0
	letters := 0
	for _, r := range reference {
		if r < 'A' || r > 'Z' {
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			} else {
				break
			}
		}
		index = index*26 + int(r-'A'+1)
		letters++
	}
	if letters == 0 {
		return -1
	}
	return index - 1
}

func excelSerialTime(serial float64) time.Time {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	whole, fraction := math.Modf(serial)
	return base.AddDate(0, 0, int(whole)).Add(time.Duration(fraction * float64(24*time.Hour)))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
