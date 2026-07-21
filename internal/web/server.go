package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/auth"
	"github.com/hendra/manajemen-tpp/internal/config"
	"github.com/hendra/manajemen-tpp/internal/domain"
	"github.com/hendra/manajemen-tpp/internal/store"
)

//go:embed templates/*.html static/* static/templates/*
var assets embed.FS

type contextKey string

const sessionKey contextKey = "session"

type Server struct {
	cfg         config.Config
	store       store.Store
	auth        *auth.Manager
	captcha     *captchaManager
	logger      *slog.Logger
	templates   map[string]*template.Template
	authLimiter *rateLimiter
	parameters  parameterCache
}

type ReportOptions struct {
	Preset            string
	Scope             string
	DateFrom          string
	DateTo            string
	Location          string
	ItemKind          string
	GoodsCondition    string
	Category          string
	AllocationPurpose string
	MinValue          string
	MaxValue          string
	MinAge            string
	Title             string
	Description       string
	ExportURL         string
	CSVExportURL      string
	ExcelExportURL    string
}

type SearchOptions struct {
	Scope             string
	DateFrom          string
	DateTo            string
	ItemKind          string
	GoodsCondition    string
	Category          string
	AllocationPurpose string
	LocationScope     string
	MinValue          string
	MaxValue          string
}

const maxDocumentUploadBytes int64 = 8 << 20

type goodsLineDraft struct {
	InventoryID    string  `json:"inventory_id,omitempty"`
	Description    string  `json:"description"`
	ItemKind       string  `json:"item_kind"`
	GoodsValue     string  `json:"goods_value,omitempty"`
	Quantity       float64 `json:"quantity"`
	QuantityDetail string  `json:"quantity_detail,omitempty"`
	Unit           string  `json:"unit"`
	GoodsCondition string  `json:"goods_condition,omitempty"`
}

type containerDraft struct {
	Number string           `json:"number"`
	Size   string           `json:"size"`
	Goods  []goodsLineDraft `json:"goods"`
}

type censusTargetDraft struct {
	TargetID string           `json:"target_id"`
	LoadType string           `json:"load_type"`
	Lines    []goodsLineDraft `json:"lines"`
}

type htlResultDraft struct {
	ProcessID string `json:"process_id"`
	HTLValue  string `json:"htl_value"`
}

type containerRelocationOperationDraft struct {
	InventoryID string                           `json:"inventory_id"`
	Allocations []domain.InventoryLoadAllocation `json:"allocations"`
}

type containerRelocationDraft struct {
	Mode        string                              `json:"mode"`
	Operations  []containerRelocationOperationDraft `json:"operations"`
	InventoryID string                              `json:"inventory_id"`
	Allocations []domain.InventoryLoadAllocation    `json:"allocations"`
}

func paginationForTotal(total int, r *http.Request) (int, int, PaginationData) {
	allowedSizes := []int{10, 20, 50, 100}
	pageSize, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page_size")))
	valid := false
	for _, v := range allowedSizes {
		if pageSize == v {
			valid = true
			break
		}
	}
	if !valid {
		pageSize = 20
	}
	page, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}
	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > total {
		end = total
	}
	makeURL := func(targetPage, targetSize int) string {
		values := r.URL.Query()
		values.Set("page", strconv.Itoa(targetPage))
		values.Set("page_size", strconv.Itoa(targetSize))
		return r.URL.Path + "?" + values.Encode()
	}
	p := PaginationData{Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages, HasPrevious: page > 1, HasNext: page < totalPages}
	if total > 0 {
		p.StartItem = offset + 1
		p.EndItem = end
	}
	if p.HasPrevious {
		p.PreviousURL = makeURL(page-1, pageSize)
	}
	if p.HasNext {
		p.NextURL = makeURL(page+1, pageSize)
	}
	for _, size := range allowedSizes {
		p.Sizes = append(p.Sizes, PageSizeOption{Value: size, URL: makeURL(1, size), Selected: size == pageSize})
	}
	return offset, pageSize, p
}
func paginate[T any](items []T, r *http.Request) ([]T, PaginationData) {
	offset, pageSize, p := paginationForTotal(len(items), r)
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	if offset > len(items) {
		offset = len(items)
	}
	return items[offset:end], p
}

func groupCensusTargets(items []domain.InventoryItem) []CensusTargetGroup {
	groups := make(map[string]*CensusTargetGroup)
	order := make([]string, 0)
	for _, item := range items {
		if !item.IsActive || item.CurrentDisposition != "" || completedInventoryProcessStatus(item.StatusCode) {
			continue
		}
		key := item.ID
		if strings.EqualFold(item.LoadType, "FCL") {
			key = domain.InventoryPhysicalUnitKey(item)
		}
		group := groups[key]
		if group == nil {
			group = &CensusTargetGroup{
				TargetID: item.ID, PhysicalUnitID: domain.InventoryPhysicalUnitKey(item), LoadType: item.LoadType,
				ContainerNo: item.ContainerNo, ContainerSize: item.ContainerSize, DeterminationNo: item.DeterminationNo,
				InventoryType: item.Type, StatusCode: item.StatusCode, StatusLabel: item.StatusLabel,
			}
			groups[key] = group
			order = append(order, key)
		}
		if item.OccupancyPrimary {
			group.TargetID = item.ID
		}
		group.Items = append(group.Items, item)
	}
	result := make([]CensusTargetGroup, 0, len(order))
	for _, key := range order {
		group := groups[key]
		sort.SliceStable(group.Items, func(i, j int) bool {
			if group.Items[i].OccupancyPrimary != group.Items[j].OccupancyPrimary {
				return group.Items[i].OccupancyPrimary
			}
			return group.Items[i].CreatedAt.Before(group.Items[j].CreatedAt)
		})
		result = append(result, *group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].LoadType != result[j].LoadType {
			return result[i].LoadType < result[j].LoadType
		}
		if result[i].ContainerNo != result[j].ContainerNo {
			return result[i].ContainerNo < result[j].ContainerNo
		}
		return result[i].DeterminationNo < result[j].DeterminationNo
	})
	return result
}

func relocationTargetKey(item domain.InventoryItem) string {
	if strings.EqualFold(item.LoadType, "FCL") {
		containerNo := strings.ToUpper(strings.NewReplacer(" ", "", "-", "", ".", "").Replace(strings.TrimSpace(item.ContainerNo)))
		if containerNo != "" {
			return "FCL:" + containerNo
		}
	}
	return domain.InventoryPhysicalUnitKey(item)
}

func groupRelocationTargets(items []domain.InventoryItem) []RelocationTargetGroup {
	type targetAccumulator struct {
		group    RelocationTargetGroup
		eligible bool
	}

	groups := make(map[string]*targetAccumulator)
	order := make([]string, 0)
	for _, item := range items {
		if !item.IsActive {
			continue
		}
		loadType := strings.ToUpper(strings.TrimSpace(item.LoadType))
		key := item.ID
		if loadType == "FCL" {
			key = relocationTargetKey(item)
		}
		entry := groups[key]
		if entry == nil {
			entry = &targetAccumulator{
				eligible: true,
				group: RelocationTargetGroup{
					TargetKey: key, PhysicalUnitID: domain.InventoryPhysicalUnitKey(item), LoadType: loadType,
					ContainerNo: item.ContainerNo, ContainerSize: item.ContainerSize, DeterminationNo: item.DeterminationNo,
					InventoryType: item.Type, StatusCode: item.StatusCode, StatusLabel: item.StatusLabel,
				},
			}
			groups[key] = entry
			order = append(order, key)
		}
		if item.OccupancyPrimary {
			entry.group.DeterminationNo = item.DeterminationNo
			entry.group.InventoryType = item.Type
			entry.group.StatusCode = item.StatusCode
			entry.group.StatusLabel = item.StatusLabel
		}
		if item.Quantity <= 0 {
			entry.eligible = false
		}
		entry.group.Items = append(entry.group.Items, item)
	}

	result := make([]RelocationTargetGroup, 0, len(order))
	for _, key := range order {
		entry := groups[key]
		if entry == nil || !entry.eligible || len(entry.group.Items) == 0 {
			continue
		}
		sort.SliceStable(entry.group.Items, func(i, j int) bool {
			if entry.group.Items[i].OccupancyPrimary != entry.group.Items[j].OccupancyPrimary {
				return entry.group.Items[i].OccupancyPrimary
			}
			return entry.group.Items[i].CreatedAt.Before(entry.group.Items[j].CreatedAt)
		})
		result = append(result, entry.group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].LoadType != result[j].LoadType {
			return result[i].LoadType < result[j].LoadType
		}
		if result[i].ContainerNo != result[j].ContainerNo {
			return result[i].ContainerNo < result[j].ContainerNo
		}
		return result[i].DeterminationNo < result[j].DeterminationNo
	})
	return result
}

func groupResearchRequests(items []domain.InventoryItem) []ResearchRequestGroup {
	byRequest := make(map[string]*ResearchRequestGroup)
	var order []string
	for _, item := range items {
		if !item.IsActive || strings.TrimSpace(item.ResearchRequestNo) == "" || item.StatusCode != "request_penelitian_pfpd" {
			continue
		}
		key := strings.TrimSpace(item.ResearchRequestNo)
		group := byRequest[key]
		if group == nil {
			group = &ResearchRequestGroup{RequestNo: key, RequestDate: item.ResearchRequestDate}
			byRequest[key] = group
			order = append(order, key)
		}
		group.Items = append(group.Items, item)
	}
	sort.Slice(order, func(i, j int) bool {
		return byRequest[order[i]].RequestDate.After(byRequest[order[j]].RequestDate)
	})
	result := make([]ResearchRequestGroup, 0, len(order))
	for _, key := range order {
		group := byRequest[key]
		sort.Slice(group.Items, func(i, j int) bool {
			if group.Items[i].ContainerNo == group.Items[j].ContainerNo {
				return group.Items[i].Description < group.Items[j].Description
			}
			return group.Items[i].ContainerNo < group.Items[j].ContainerNo
		})
		result = append(result, *group)
	}
	return result
}

type NotificationItem struct {
	Tone    string
	Title   string
	Message string
	URL     string
}

type ProcessModalData struct {
	Type      domain.DispositionType
	Title     string
	Singular  string
	URL       string
	Dashboard domain.ProcessDashboard
}

type PageSizeOption struct {
	Value    int
	URL      string
	Selected bool
}

type PaginationData struct {
	Page        int
	PageSize    int
	TotalItems  int
	TotalPages  int
	StartItem   int
	EndItem     int
	PreviousURL string
	NextURL     string
	HasPrevious bool
	HasNext     bool
	Sizes       []PageSizeOption
}

type ResearchRequestGroup struct {
	RequestNo   string
	RequestDate time.Time
	Items       []domain.InventoryItem
}

type CensusTargetGroup struct {
	TargetID        string
	PhysicalUnitID  string
	LoadType        string
	ContainerNo     string
	ContainerSize   string
	DeterminationNo string
	InventoryType   domain.InventoryType
	StatusCode      string
	StatusLabel     string
	Items           []domain.InventoryItem
}

type RelocationTargetGroup struct {
	TargetKey       string
	PhysicalUnitID  string
	LoadType        string
	ContainerNo     string
	ContainerSize   string
	DeterminationNo string
	InventoryType   domain.InventoryType
	StatusCode      string
	StatusLabel     string
	Items           []domain.InventoryItem
}

type AuctionScheduleGroup struct {
	DocumentNo   string
	DocumentDate time.Time
	Processes    []domain.Disposition
}

type DataCorrectionReportRow struct {
	Record domain.ReconciliationRecord
	Change domain.ReconciliationChange
	Legacy bool
}

type BTDReportRow struct {
	DeterminationNo   string
	DeterminationDate time.Time
	BLNo              string
	BLDate            string
	ManifestNo        string
	ManifestDate      string
	ManifestPosition  string
	LoadType          string
	OriginWarehouse   string
	FacilityName      string
	LocationStatus    string
	ContainerSummary  string
	ContainerCount    int
	GoodsSummary      string
	OwnerName         string
	ItemCount         int
	TotalValue        int64
	StatusLabel       string
	InventoryStatus   string
}

type auctionResultDraft struct {
	ProcessID string `json:"process_id"`
	Outcome   string `json:"outcome"`
	SaleValue string `json:"sale_value"`
}

type pfpdResultDraft struct {
	InventoryID     string `json:"inventory_id"`
	HSCode          string `json:"hs_code"`
	IsRestricted    string `json:"is_restricted"`
	RestrictionRule string `json:"restriction_rule"`
	GoodsValue      string `json:"goods_value"`
}

type PageData struct {
	Title                   string
	Subtitle                string
	Active                  string
	AuthPage                bool
	SignupPage              bool
	OTPPage                 bool
	ForgotPasswordPage      bool
	ResetPasswordPage       bool
	VerifyEmail             string
	CaptchaToken            string
	DemoMode                bool
	User                    auth.Session
	CSRF                    string
	Success                 string
	Error                   string
	Facilities              []domain.Facility
	Stats                   domain.DashboardStats
	DashboardRows           []domain.FacilityBreakdown
	DashboardOccupancy      domain.Occupancy
	DashboardScope          string
	DashboardInventoryScope string
	DashboardInventoryLabel string
	Items                   []domain.InventoryItem
	EligibleItems           []domain.InventoryItem
	Processes               []domain.Disposition
	CandidateProcesses      []domain.Disposition
	InventoryActions        []domain.WorkflowAction
	ProcessActions          []domain.WorkflowAction
	ProcessType             domain.DispositionType
	ProcessTitle            string
	ProcessSingular         string
	ProcessDashboard        domain.ProcessDashboard
	AuctionDashboard        domain.ProcessDashboard
	DestructionDashboard    domain.ProcessDashboard
	GrantDashboard          domain.ProcessDashboard
	ProcessModals           []ProcessModalData
	Query                   string
	FacilityID              string
	InventoryType           domain.InventoryType
	Status                  string
	Sort                    string
	History                 bool
	SearchPerformed         bool
	Search                  SearchOptions
	Now                     time.Time
	ActiveProcesses         int
	ClosedProcesses         int
	ReportTotal             int
	ReportActive            int
	ReportClosed            int
	ReportTotalValue        int64
	ReportAtTPP             int
	ReportTransactionTotal  int
	Report                  ReportOptions
	TPSNames                []string
	BDNCategoryNames        []string
	ItemKindNames           []string
	GoodsConditionNames     []string
	AllocationPurposeNames  []string
	UnitNames               []string
	LoadTypeOptions         []domain.SelectOption
	ContainerSizeOptions    []domain.SelectOption
	ExitOptions             []domain.SelectOption
	TransferTypeOptions     []domain.SelectOption
	Users                   []domain.UserAccount
	Roles                   []domain.RoleProfile
	PermissionDefinitions   []domain.PermissionDefinition
	Parameters              []domain.ParameterOption
	AdminSection            string
	PendingUsers            int
	VerifiedPendingUsers    int
	CanManage               bool
	CanCreateInventory      bool
	CanCreateBTD            bool
	CanCreateBDN            bool
	CanCreateTitipan        bool
	CanRunInventoryActions  bool
	CanEditCapacity         bool
	IdleTimeoutSeconds      int64
	Notifications           []NotificationItem
	NotificationCount       int
	Pagination              PaginationData
	ResearchRequestGroups   []ResearchRequestGroup
	CensusTargetGroups      []CensusTargetGroup
	RelocationTargetGroups  []RelocationTargetGroup
	AuctionScheduleGroups   []AuctionScheduleGroup
	Reconciliations         []domain.ReconciliationRecord
	DataCorrections         []domain.ReconciliationRecord
	DataCorrectionRows      []DataCorrectionReportRow
	ReconciliationTab       string
	EntrustedCategoryNames  []string
	ReportReconciliation    bool
	ReportDataCorrection    bool
	ReportBTD               bool
	BTDReportRows           []BTDReportRow
	ReportPerformance       bool
	Performance             PerformanceReport
	PerformanceOpen         bool
}

var correctionFieldLabels = map[string]string{
	"reference_no":             "Nomor referensi",
	"item_type":                "Jenis inventory",
	"origin_type":              "Jenis inventory asal",
	"bl_no":                    "Nomor BL",
	"bl_date":                  "Tanggal BL",
	"manifest_no":              "Nomor manifest",
	"manifest_date":            "Tanggal manifest",
	"manifest_position":        "Pos manifest",
	"determination_no":         "Nomor penetapan atau dokumen dasar",
	"determination_date":       "Tanggal penetapan atau dokumen dasar",
	"category":                 "Kategori BDN",
	"entrusted_category":       "Kategori barang titipan",
	"source_office":            "Kantor atau unit penitip",
	"description":              "Uraian barang",
	"item_kind":                "Jenis barang",
	"quantity":                 "Jumlah barang",
	"quantity_detail":          "Detail jumlah barang",
	"unit":                     "Satuan barang",
	"goods_value":              "Nilai barang",
	"goods_condition":          "Kondisi barang",
	"location":                 "Lokasi atau blok",
	"location_status":          "Status lokasi",
	"at_tpp":                   "Keberadaan di TPP",
	"owner_name":               "Nama pemilik",
	"owner_address":            "Alamat pemilik",
	"origin_warehouse":         "TPS asal",
	"facility_id":              "ID TPP",
	"facility_name":            "Nama TPP",
	"load_type":                "Jenis muatan",
	"container_no":             "Nomor kontainer",
	"container_size":           "Ukuran kontainer",
	"estimated_volume_m3":      "Perkiraan volume",
	"physical_unit_id":         "Identitas unit fisik",
	"occupancy_primary":        "Unit utama perhitungan kapasitas",
	"pfpd_required":            "Memerlukan penelitian PFPD",
	"research_request_no":      "Nomor request penelitian PFPD",
	"research_request_date":    "Tanggal request penelitian PFPD",
	"hs_code":                  "HS Code",
	"is_restricted":            "Status lartas",
	"restriction_rule":         "Ketentuan lartas",
	"origin_document_type":     "Jenis dokumen asal",
	"origin_document_no":       "Nomor dokumen asal",
	"origin_document_date":     "Tanggal dokumen asal",
	"allocation_purpose":       "Peruntukan BMMN",
	"allocation_proposal_type": "Jenis usulan peruntukan",
	"allocation_proposal_no":   "Nomor usulan peruntukan",
	"allocation_proposal_date": "Tanggal usulan peruntukan",
	"allocation_approval_type": "Jenis persetujuan peruntukan",
	"allocation_approval_no":   "Nomor persetujuan peruntukan",
	"allocation_approval_date": "Tanggal persetujuan peruntukan",
	"exit_document_no":         "Nomor dokumen pengeluaran",
	"exit_document_date":       "Tanggal dokumen pengeluaran",
	"exit_type":                "Jenis pengeluaran",
	"exit_notes":               "Catatan pengeluaran",
	"label":                    "Nama atau label tahapan",
	"document_no":              "Nomor surat, ND, KEP, BA, atau risalah",
	"document_date":            "Tanggal dokumen tahapan",
	"notes":                    "Catatan tahapan",
	"proposal_type":            "Jenis usulan proses",
	"recipient_code":           "Kode penerima",
	"recipient_name":           "Nama penerima",
	"sale_value":               "Nilai hasil lelang",
	"htl_value":                "Nilai HTL",
	"execution_start_date":     "Tanggal mulai pelaksanaan",
	"execution_end_date":       "Tanggal selesai pelaksanaan",
	"schedule_document_no":     "Nomor ND jadwal",
	"schedule_document_date":   "Tanggal ND jadwal",
	"auction_outcome":          "Hasil lelang",
	"allocation_target":        "Tujuan alokasi hasil lelang",
	"destruction_cost":         "Biaya pemusnahan",
	"transfer_type":            "Jenis serah terima",
}

func correctionSectionLabel(section string) string {
	switch strings.TrimSpace(section) {
	case "inventory":
		return "Data utama barang"
	case "timeline":
		return "Dokumen dan timeline"
	case "process":
		return "Data proses penyelesaian"
	default:
		return "Data lainnya"
	}
}

func correctionFieldLabel(field string) string {
	field = strings.TrimSpace(field)
	if label := correctionFieldLabels[field]; label != "" {
		return label
	}
	return strings.Title(strings.ReplaceAll(field, "_", " "))
}

func correctionDisplayValue(field, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return "Kosong"
	}
	switch value {
	case "true":
		return "Ya"
	case "false":
		return "Tidak"
	}
	if strings.HasSuffix(field, "_date") || field == "manifest_date" || field == "determination_date" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed.In(time.Local).Format("02 Jan 2006")
		}
		if len(value) >= 10 {
			if parsed, err := time.Parse("2006-01-02", value[:10]); err == nil {
				return parsed.Format("02 Jan 2006")
			}
		}
	}
	if field == "goods_value" || field == "sale_value" || field == "htl_value" || field == "destruction_cost" {
		if amount, err := strconv.ParseInt(value, 10, 64); err == nil {
			return "Rp " + formatThousands(strconv.FormatInt(amount, 10))
		}
	}
	switch field {
	case "auction_outcome":
		if value == "laku" {
			return "Laku"
		}
		if value == "tidak_laku" {
			return "Tidak laku"
		}
	case "transfer_type":
		return strings.ToUpper(value)
	case "item_type", "origin_type", "load_type", "container_size":
		return strings.ToUpper(value)
	}
	return value
}

func NewServer(cfg config.Config, dataStore store.Store, authManager *auth.Manager, logger *slog.Logger) (*Server, error) {
	funcs := template.FuncMap{
		"date": func(value time.Time) string {
			if value.IsZero() {
				return "—"
			}
			return value.In(time.Local).Format("02 Jan 2006")
		},
		"dateTime": func(value time.Time) string {
			if value.IsZero() {
				return "—"
			}
			return value.In(time.Local).Format("02 Jan 2006, 15:04")
		},
		"isoDate": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Format("2006-01-02")
		},
		"changeSection": correctionSectionLabel,
		"changeField":   correctionFieldLabel,
		"changeValue":   correctionDisplayValue,
		"age":           func(item domain.InventoryItem, now time.Time) int { return item.AgeDays(now) },
		"number": func(value float64) string {
			if value == float64(int64(value)) {
				return formatThousands(strconv.FormatInt(int64(value), 10))
			}
			return strconv.FormatFloat(value, 'f', 2, 64)
		},
		"rupiah": func(value int64) string {
			if value == 0 {
				return "—"
			}
			return "Rp " + formatThousands(strconv.FormatInt(value, 10))
		},
		"percent": func(used, capacity float64) string {
			if capacity <= 0 {
				return "0,0"
			}
			return strings.ReplaceAll(fmt.Sprintf("%.1f", used*100/capacity), ".", ",")
		},
		"initials": initials,
		"lower":    func(value any) string { return strings.ToLower(fmt.Sprint(value)) },
		"statusTone": func(code string) string {
			switch {
			case strings.Contains(code, "selesai"), code == "pengeluaran_selesai", code == "pengeluaran_barang", code == "laku", code == "alokasi_hasil_lelang", code == "ba_serah_terima", code == "ba_musnah":
				return "success"
			case code == "tidak_laku":
				return "danger"
			case strings.Contains(code, "lelang"):
				return "violet"
			case strings.Contains(code, "musnah"):
				return "danger"
			case strings.Contains(code, "hibah"):
				return "teal"
			case code == "ditetapkan", code == "pemindahan":
				return "neutral"
			default:
				return "warning"
			}
		},
		"processLabel":        processLabel,
		"containerSizeLabel":  domain.ContainerSizeLabel,
		"performanceDuration": formatPerformanceDuration,
		"can":                 func(session auth.Session, permission string) bool { return session.Can(permission) },
		"hasPermission": func(values []string, permission string) bool {
			for _, value := range values {
				if value == permission {
					return true
				}
			}
			return false
		},
		"parameterGroupLabel": parameterGroupName,
		"appliesTo": func(value, kind string) bool {
			for _, candidate := range strings.Split(strings.ToUpper(value), ",") {
				if strings.TrimSpace(candidate) == strings.ToUpper(kind) {
					return true
				}
			}
			return false
		},
	}
	pages := []string{"auth", "dashboard", "inventory", "process", "reports", "search", "admin", "reconciliation"}
	templates := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		parsed, err := template.New("layout").Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", page, err)
		}
		templates[page] = parsed
	}
	return &Server{cfg: cfg, store: dataStore, auth: authManager, captcha: newCaptchaManager(cfg.SessionSecret), logger: logger, templates: templates, authLimiter: newRateLimiter()}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(assets, "static")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /captcha.png", s.captchaPNG)
	mux.HandleFunc("GET /captcha/new", s.newCaptcha)
	mux.HandleFunc("GET /signup", s.signupPage)
	mux.HandleFunc("POST /signup", s.signup)
	mux.HandleFunc("GET /signup/verify", s.verifySignupPage)
	mux.HandleFunc("POST /signup/verify", s.verifySignupOTP)
	mux.HandleFunc("POST /signup/resend", s.resendSignupOTP)
	mux.HandleFunc("GET /forgot-password", s.forgotPasswordPage)
	mux.HandleFunc("POST /forgot-password", s.requestPasswordReset)
	mux.HandleFunc("GET /forgot-password/verify", s.resetPasswordPage)
	mux.HandleFunc("POST /forgot-password/verify", s.resetPasswordWithOTP)
	mux.Handle("POST /logout", s.protected(http.HandlerFunc(s.logout)))
	mux.Handle("POST /session/activity", s.protected(http.HandlerFunc(s.sessionActivity)))
	mux.Handle("POST /session/idle-logout", s.protected(http.HandlerFunc(s.idleLogout)))
	mux.Handle("GET /", s.protected(s.requirePermission(domain.PermissionDashboardView, http.HandlerFunc(s.dashboard))))
	mux.Handle("POST /admin/facilities/{id}/capacity", s.protected(s.requirePermission(domain.PermissionDashboardCapacity, http.HandlerFunc(s.updateFacilityCapacity))))
	mux.Handle("GET /inventory", s.protected(s.requirePermission(domain.PermissionInventoryView, http.HandlerFunc(s.inventory))))
	mux.Handle("POST /inventory", s.protected(s.requireAnyPermission(domain.InventoryManagementPermissionCodes(), http.HandlerFunc(s.createInventory))))
	mux.Handle("POST /inventory/import", s.protected(s.requireAnyPermission(domain.InventoryManagementPermissionCodes(), http.HandlerFunc(s.importInventoryWorkbook))))
	mux.Handle("POST /inventory/{id}/event", s.protected(s.requireAnyPermission(domain.InventoryManagementPermissionCodes(), http.HandlerFunc(s.addInventoryEvent))))
	mux.Handle("POST /inventory/bulk-event", s.protected(s.requireAnyPermission(domain.InventoryManagementPermissionCodes(), http.HandlerFunc(s.addBulkInventoryEvent))))
	mux.Handle("POST /admin/inventory/{id}/delete", s.protected(s.requireAdmin(http.HandlerFunc(s.deleteInventory))))
	mux.Handle("GET /proses/{type}", s.protected(http.HandlerFunc(s.processPage)))
	mux.Handle("POST /proses/{type}/bulk-action", s.protected(http.HandlerFunc(s.bulkProcessAction)))
	mux.Handle("GET /rekonsiliasi", s.protected(s.requirePermission(domain.PermissionReconciliationView, http.HandlerFunc(s.reconciliationPage))))
	mux.Handle("POST /rekonsiliasi", s.protected(s.requirePermission(domain.PermissionReconciliationManage, http.HandlerFunc(s.reconcileInventory))))
	mux.Handle("GET /pelaporan", s.protected(s.requirePermission(domain.PermissionReportsView, http.HandlerFunc(s.reports))))
	mux.Handle("GET /pelaporan.csv", s.protected(s.requirePermission(domain.PermissionReportsView, http.HandlerFunc(s.reportCSV))))
	mux.Handle("GET /pelaporan.xlsx", s.protected(s.requirePermission(domain.PermissionReportsView, http.HandlerFunc(s.reportXLSX))))
	mux.Handle("GET /pelaporan.xls", s.protected(s.requirePermission(domain.PermissionReportsView, http.HandlerFunc(s.reportXLS))))
	mux.Handle("GET /pelaporan/performa.xlsx", s.protected(s.requirePermission(domain.PermissionReportsView, http.HandlerFunc(s.performanceXLSX))))
	mux.Handle("GET /pencarian", s.protected(s.requirePermission(domain.PermissionSearchView, http.HandlerFunc(s.searchPage))))
	mux.Handle("GET /api/inventory/search", s.protected(s.requireAnyPermission([]string{domain.PermissionInventoryView, domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView, domain.PermissionSearchView}, http.HandlerFunc(s.searchInventory))))
	mux.Handle("GET /api/inventory/{id}", s.protected(s.requireAnyPermission([]string{domain.PermissionInventoryView, domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView, domain.PermissionSearchView}, http.HandlerFunc(s.inventoryDetail))))
	mux.Handle("GET /api/inventory/{id}/timeline", s.protected(s.requireAnyPermission([]string{domain.PermissionInventoryView, domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView, domain.PermissionSearchView}, http.HandlerFunc(s.inventoryTimeline))))
	mux.Handle("GET /api/proses/{id}/timeline", s.protected(s.requireAnyPermission([]string{domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView}, http.HandlerFunc(s.processTimeline))))
	mux.Handle("GET /documents/{id}/download", s.protected(s.requireAnyPermission([]string{domain.PermissionInventoryView, domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView, domain.PermissionSearchView, domain.PermissionReconciliationView}, http.HandlerFunc(s.downloadDocument))))
	mux.Handle("GET /admin/pendaftaran", s.protected(s.requirePermission(domain.PermissionAdminUsers, http.HandlerFunc(s.adminUsers))))
	mux.Handle("POST /admin/pendaftaran/{id}/approve", s.protected(s.requirePermission(domain.PermissionAdminUsers, http.HandlerFunc(s.approveUser))))
	mux.Handle("POST /admin/pendaftaran/{id}/reject", s.protected(s.requirePermission(domain.PermissionAdminUsers, http.HandlerFunc(s.rejectUser))))
	mux.Handle("POST /admin/pendaftaran/{id}/role", s.protected(s.requirePermission(domain.PermissionAdminUsers, http.HandlerFunc(s.updateUserRole))))
	mux.Handle("POST /admin/pendaftaran/{id}/delete", s.protected(s.requirePermission(domain.PermissionAdminUsers, http.HandlerFunc(s.deleteUser))))
	mux.Handle("GET /admin/roles", s.protected(s.requirePermission(domain.PermissionAdminRoles, http.HandlerFunc(s.adminRoles))))
	mux.Handle("POST /admin/roles", s.protected(s.requirePermission(domain.PermissionAdminRoles, http.HandlerFunc(s.createRole))))
	mux.Handle("POST /admin/roles/{id}/update", s.protected(s.requirePermission(domain.PermissionAdminRoles, http.HandlerFunc(s.updateRole))))
	mux.Handle("POST /admin/roles/{id}/status", s.protected(s.requirePermission(domain.PermissionAdminRoles, http.HandlerFunc(s.setRoleStatus))))
	mux.Handle("POST /admin/roles/{id}/delete", s.protected(s.requirePermission(domain.PermissionAdminRoles, http.HandlerFunc(s.deleteRole))))
	mux.Handle("GET /admin/parameters", s.protected(s.requirePermission(domain.PermissionAdminParameters, http.HandlerFunc(s.adminParameters))))
	mux.Handle("POST /admin/parameters", s.protected(s.requirePermission(domain.PermissionAdminParameters, http.HandlerFunc(s.createParameter))))
	mux.Handle("POST /admin/parameters/{id}/update", s.protected(s.requirePermission(domain.PermissionAdminParameters, http.HandlerFunc(s.updateParameter))))
	mux.Handle("POST /admin/parameters/{id}/status", s.protected(s.requirePermission(domain.PermissionAdminParameters, http.HandlerFunc(s.setParameterStatus))))
	return s.requestID(s.securityHeaders(s.requestLog(s.limitRequestBody(mux))))
}

func (s *Server) baseData(r *http.Request, title, subtitle, active string) PageData {
	session, _ := sessionFromContext(r.Context())
	data := PageData{
		Title: title, Subtitle: subtitle, Active: active, DemoMode: s.cfg.DemoMode,
		User: session, CSRF: s.auth.CSRFToken(session), Success: r.URL.Query().Get("ok"), Error: r.URL.Query().Get("error"), Now: time.Now(),
		TPSNames: domain.CurrentOriginTPS(), BDNCategoryNames: domain.CurrentBDNCategories(), ItemKindNames: domain.CurrentItemKinds(), GoodsConditionNames: domain.CurrentGoodsConditions(),
		AllocationPurposeNames: domain.CurrentAllocationPurposes(), UnitNames: domain.CurrentUnits(), LoadTypeOptions: domain.CurrentLoadTypes(),
		ContainerSizeOptions:   domain.ContainerSizeOptions,
		EntrustedCategoryNames: domain.EntrustedCategoryNames,
		ExitOptions:            domain.CurrentExitOptions(), TransferTypeOptions: domain.CurrentTransferTypes(),
		PermissionDefinitions: domain.PermissionDefinitions,
		IdleTimeoutSeconds:    int64(auth.IdleTimeout / time.Second),
	}
	data.Notifications = s.buildNotifications(r.Context(), session)
	data.NotificationCount = len(data.Notifications)
	return data
}

func (s *Server) buildNotifications(ctx context.Context, session auth.Session) []NotificationItem {
	notifications := make([]NotificationItem, 0, 4)
	appendNotification := func(item NotificationItem) {
		if len(notifications) < 5 {
			notifications = append(notifications, item)
		}
	}

	if session.Can(domain.PermissionAdminUsers) {
		users, err := s.store.ListUsers(ctx)
		if err == nil {
			verifiedPending := 0
			unverifiedPending := 0
			for _, user := range users {
				if user.ApprovalStatus != "pending" {
					continue
				}
				if user.EmailVerified {
					verifiedPending++
				} else {
					unverifiedPending++
				}
			}
			if verifiedPending > 0 {
				appendNotification(NotificationItem{
					Tone: "attention", Title: "Pendaftaran siap disetujui",
					Message: fmt.Sprintf("%d akun sudah mengonfirmasi OTP dan menunggu role.", verifiedPending),
					URL:     "/admin/pendaftaran",
				})
			}
			if unverifiedPending > 0 {
				appendNotification(NotificationItem{
					Tone: "neutral", Title: "Menunggu verifikasi email",
					Message: fmt.Sprintf("%d pendaftar belum menyelesaikan OTP email.", unverifiedPending),
					URL:     "/admin/pendaftaran",
				})
			}
		}
	}

	if session.Can(domain.PermissionInventoryView) || session.CanAny(domain.InventoryManagementPermissionCodes()...) || session.CanAny(domain.PermissionReportsView, domain.PermissionAuctionView, domain.PermissionDestructionView, domain.PermissionGrantView) {
		summary, err := s.store.NotificationSummary(ctx, allowedInventoryTypes(session))
		if err == nil {
			if summary.Overdue60Days > 0 && session.Can(domain.PermissionReportsView) {
				appendNotification(NotificationItem{Tone: "danger", Title: "BTD/BDN melewati 60 hari", Message: fmt.Sprintf("%d barang masih belum ditindaklanjuti sejak penetapan awal.", summary.Overdue60Days), URL: "/pelaporan?preset=overdue_60"})
			}
			if summary.ReadyForExit > 0 && sessionCanPerformInventoryAction(session, "pengeluaran_barang") {
				appendNotification(NotificationItem{Tone: "success", Title: "Barang siap dikeluarkan", Message: fmt.Sprintf("%d barang telah selesai lelang, musnah, hibah, atau PSP dan menunggu pengeluaran.", summary.ReadyForExit), URL: "/inventory?sort=newest"})
			}
			if summary.BMMNWaiting > 0 && session.Can(domain.PermissionReportsView) {
				appendNotification(NotificationItem{Tone: "info", Title: "BMMN menunggu peruntukan", Message: fmt.Sprintf("%d barang BMMN aktif belum memiliki proses penyelesaian.", summary.BMMNWaiting), URL: "/pelaporan?preset=bmmn_allocation"})
			}
		}
	}

	return notifications
}

func (s *Server) render(w http.ResponseWriter, page string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates[page].ExecuteTemplate(w, "layout", data); err != nil {
		s.logger.Error("render template", "page", page, "error", err)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "mode": map[bool]string{true: "demo", false: "supabase"}[s.cfg.DemoMode], "time": time.Now().UTC()})
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.Session(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if _, err := r.Cookie(auth.CookieName); err == nil {
		s.auth.ClearSession(w)
	}
	data := PageData{Title: "Masuk", AuthPage: true, DemoMode: s.cfg.DemoMode, Error: r.URL.Query().Get("error"), Success: r.URL.Query().Get("ok")}
	if token, _, err := s.captcha.newChallenge(); err != nil {
		s.logger.Error("create login captcha", "error", err)
		data.Error = "CAPTCHA belum dapat dibuat. Muat ulang halaman untuk mencoba kembali."
	} else {
		data.CaptchaToken = token
	}
	if r.URL.Query().Get("idle") == "1" {
		data.Error = "Sesi berakhir otomatis karena tidak ada aktivitas selama 30 menit. Silakan masuk kembali."
	}
	if r.URL.Query().Get("confirmed") == "1" {
		data.Success = "Email berhasil dikonfirmasi. Silakan masuk."
	}
	s.render(w, "auth", data)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/login", "error", "Form login tidak valid atau terlalu besar.")
		return
	}
	identity := strings.TrimSpace(r.FormValue("identity"))
	if !s.authAttemptAllowed(r, "login", identity, 8, 15*time.Minute) {
		redirectMessage(w, r, "/login", "error", "Terlalu banyak percobaan masuk. Tunggu sekitar 15 menit lalu coba kembali.")
		return
	}
	if !s.captcha.verify(r.FormValue("captcha_token"), r.FormValue("captcha_answer")) {
		s.writeAudit(r, "auth.login", "account", strings.ToLower(identity), "failed", map[string]any{"reason": "invalid_captcha"})
		redirectMessage(w, r, "/login", "error", "Kode CAPTCHA tidak sesuai atau sudah kedaluwarsa. Masukkan kode baru yang ditampilkan.")
		return
	}
	session, err := s.auth.Login(r.Context(), identity, r.FormValue("password"))
	if err != nil {
		s.writeAudit(r, "auth.login", "account", strings.ToLower(identity), "failed", map[string]any{"reason": "invalid_credentials"})
		redirectMessage(w, r, "/login", "error", "Username/email atau password tidak sesuai, atau email belum dikonfirmasi.")
		return
	}
	if session.Role != "admin" {
		authUserID := strings.TrimPrefix(session.Subject, "user:")
		account, accountErr := s.store.UserByAuthID(r.Context(), authUserID)
		if accountErr != nil {
			redirectMessage(w, r, "/login", "error", "Profil pendaftaran belum tersedia. Hubungi administrator.")
			return
		}
		if !account.EmailVerified {
			redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(account.Email), "error", "Konfirmasi OTP email terlebih dahulu.")
			return
		}
		switch account.ApprovalStatus {
		case "pending":
			redirectMessage(w, r, "/login", "error", "Email sudah terverifikasi. Pendaftaran masih menunggu persetujuan administrator.")
			return
		case "rejected":
			message := "Pendaftaran ditolak oleh administrator."
			if account.RejectionReason != "" {
				message += " Alasan: " + account.RejectionReason
			}
			redirectMessage(w, r, "/login", "error", message)
			return
		case "approved":
		default:
			redirectMessage(w, r, "/login", "error", "Status pendaftaran tidak valid. Hubungi administrator.")
			return
		}
		if account.RoleID == "" || account.RoleName == "" || len(account.Permissions) == 0 {
			redirectMessage(w, r, "/login", "error", "Akun sudah disetujui tetapi role belum aktif. Hubungi administrator.")
			return
		}
		session.DisplayName = account.Name
		session.Email = account.Email
		session.Role = "user"
		session.RoleID = account.RoleID
		session.RoleName = account.RoleName
		session.Permissions = append([]string(nil), account.Permissions...)
		session.SessionVersion = account.SessionVersion
	}
	s.resetAuthAttempts(r, "login", identity)
	if err := s.auth.SetSession(w, session); err != nil {
		redirectMessage(w, r, "/login", "error", "Sesi tidak dapat dibuat.")
		return
	}
	auditRequest := r.WithContext(context.WithValue(r.Context(), sessionKey, session))
	s.writeAudit(auditRequest, "auth.login", "account", session.Subject, "success", map[string]any{"role": session.RoleName})
	http.Redirect(w, r, landingPath(session), http.StatusSeeOther)
}

func (s *Server) signupPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, "auth", PageData{Title: "Daftar", AuthPage: true, SignupPage: true, DemoMode: s.cfg.DemoMode, Error: r.URL.Query().Get("error")})
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/signup", "error", "Form pendaftaran tidak valid.")
		return
	}
	name, email, password := strings.TrimSpace(r.FormValue("name")), strings.ToLower(strings.TrimSpace(r.FormValue("email"))), r.FormValue("password")
	if !s.authAttemptAllowed(r, "signup", email, 5, time.Hour) {
		redirectMessage(w, r, "/signup", "error", "Terlalu banyak permintaan pendaftaran. Tunggu sebelum mencoba kembali.")
		return
	}
	if name == "" || !strings.Contains(email, "@") || len(password) < 8 {
		redirectMessage(w, r, "/signup", "error", "Isi nama, email valid, dan password minimal 8 karakter.")
		return
	}
	result, err := s.auth.Signup(r.Context(), name, email, password)
	if err != nil {
		redirectMessage(w, r, "/signup", "error", err.Error())
		return
	}
	if _, err := s.store.CreateUserApplication(r.Context(), domain.NewUserApplicationInput{AuthUserID: result.UserID, Name: name, Email: result.Email}); err != nil {
		s.logger.Error("create user application", "error", err, "email", result.Email)
		redirectMessage(w, r, "/signup", "error", "Akun Auth berhasil dibuat, tetapi data pendaftaran belum tersimpan. Hubungi administrator.")
		return
	}
	redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(result.Email), "ok", "OTP 6 digit telah dikirim ke email. Masukkan OTP untuk melanjutkan pendaftaran.")
}

func (s *Server) verifySignupPage(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	data := PageData{Title: "Konfirmasi OTP", AuthPage: true, OTPPage: true, VerifyEmail: email, DemoMode: s.cfg.DemoMode, Error: r.URL.Query().Get("error"), Success: r.URL.Query().Get("ok")}
	s.render(w, "auth", data)
}

func (s *Server) verifySignupOTP(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/signup/verify", "error", "Form OTP tidak valid.")
		return
	}
	email, token := strings.ToLower(strings.TrimSpace(r.FormValue("email"))), strings.TrimSpace(r.FormValue("token"))
	if !s.authAttemptAllowed(r, "verify", email, 10, 15*time.Minute) {
		redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "error", "Terlalu banyak percobaan OTP. Tunggu sekitar 15 menit.")
		return
	}
	if !strings.Contains(email, "@") || len(token) != 6 {
		redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "error", "Masukkan email dan OTP 6 digit yang valid.")
		return
	}
	verified, err := s.auth.VerifySignupOTP(r.Context(), email, token)
	if err != nil {
		redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "error", "OTP tidak valid, sudah digunakan, atau kedaluwarsa.")
		return
	}
	s.resetAuthAttempts(r, "verify", email)
	if err := s.store.MarkUserEmailVerified(r.Context(), verified.UserID, verified.Email); err != nil {
		s.logger.Error("mark email verified", "error", err, "email", verified.Email)
	}
	redirectMessage(w, r, "/login", "ok", "Email berhasil dikonfirmasi. Pendaftaran telah dikirim ke administrator dan menunggu persetujuan serta penetapan role.")
}

func (s *Server) resendSignupOTP(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/signup/verify", "error", "Permintaan OTP tidak valid.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if !s.authAttemptAllowed(r, "resend", email, 3, 15*time.Minute) {
		redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "error", "Batas pengiriman ulang OTP tercapai. Tunggu sekitar 15 menit.")
		return
	}
	if !strings.Contains(email, "@") {
		redirectMessage(w, r, "/signup/verify", "error", "Masukkan email yang valid.")
		return
	}
	if err := s.auth.ResendSignupOTP(r.Context(), email); err != nil {
		redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "error", "OTP belum dapat dikirim ulang. Tunggu setidaknya 60 detik lalu coba kembali.")
		return
	}
	redirectMessage(w, r, "/signup/verify?email="+url.QueryEscape(email), "ok", "OTP baru telah dikirim ke email.")
}

func (s *Server) forgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Lupa Password", AuthPage: true, ForgotPasswordPage: true, DemoMode: s.cfg.DemoMode,
		VerifyEmail: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email"))), Error: r.URL.Query().Get("error"), Success: r.URL.Query().Get("ok"),
	}
	s.render(w, "auth", data)
}

func (s *Server) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/forgot-password", "error", "Permintaan reset password tidak valid.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if !strings.Contains(email, "@") {
		redirectMessage(w, r, "/forgot-password", "error", "Masukkan alamat email yang valid.")
		return
	}
	if !s.authAttemptAllowed(r, "password-reset-request", email, 3, 15*time.Minute) {
		redirectMessage(w, r, "/forgot-password?email="+url.QueryEscape(email), "error", "Batas permintaan OTP tercapai. Tunggu sekitar 15 menit lalu coba kembali.")
		return
	}
	if err := s.auth.RequestPasswordReset(r.Context(), email); err != nil {
		s.logger.Warn("request password reset", "error", err, "email", email)
		if s.cfg.DemoMode {
			redirectMessage(w, r, "/forgot-password?email="+url.QueryEscape(email), "error", "Reset password melalui email aktif setelah Supabase dikonfigurasi.")
			return
		}
	}
	s.writeAudit(r, "auth.password_reset.request", "account", email, "success", nil)
	redirectMessage(w, r, "/forgot-password/verify?email="+url.QueryEscape(email), "ok", "Jika email terdaftar, OTP 6 digit telah dikirim. Masukkan OTP dan password baru.")
}

func (s *Server) resetPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Ubah Password", AuthPage: true, ResetPasswordPage: true, DemoMode: s.cfg.DemoMode,
		VerifyEmail: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email"))), Error: r.URL.Query().Get("error"), Success: r.URL.Query().Get("ok"),
	}
	s.render(w, "auth", data)
}

func (s *Server) resetPasswordWithOTP(w http.ResponseWriter, r *http.Request) {
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/forgot-password/verify", "error", "Form reset password tidak valid.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	token := strings.TrimSpace(r.FormValue("token"))
	password, confirmation := r.FormValue("password"), r.FormValue("password_confirmation")
	returnPath := "/forgot-password/verify?email=" + url.QueryEscape(email)
	if !s.authAttemptAllowed(r, "password-reset-verify", email, 8, 15*time.Minute) {
		redirectMessage(w, r, returnPath, "error", "Terlalu banyak percobaan OTP. Tunggu sekitar 15 menit lalu coba kembali.")
		return
	}
	if !strings.Contains(email, "@") || len(token) != 6 {
		redirectMessage(w, r, returnPath, "error", "Masukkan email dan OTP 6 digit yang valid.")
		return
	}
	if len(password) < 8 {
		redirectMessage(w, r, returnPath, "error", "Password baru minimal 8 karakter.")
		return
	}
	if password != confirmation {
		redirectMessage(w, r, returnPath, "error", "Konfirmasi password baru tidak sama.")
		return
	}
	if err := s.auth.ResetPasswordWithOTP(r.Context(), email, token, password); err != nil {
		s.writeAudit(r, "auth.password_reset.complete", "account", email, "failed", map[string]any{"reason": "invalid_or_expired_otp"})
		redirectMessage(w, r, returnPath, "error", "OTP tidak valid, sudah digunakan, atau kedaluwarsa. Minta OTP baru jika diperlukan.")
		return
	}
	s.resetAuthAttempts(r, "password-reset-request", email)
	s.resetAuthAttempts(r, "password-reset-verify", email)
	s.writeAudit(r, "auth.password_reset.complete", "account", email, "success", nil)
	redirectMessage(w, r, "/login", "ok", "Password berhasil diperbarui. Silakan masuk menggunakan password baru.")
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	s.writeAudit(r, "auth.logout", "session", session.Subject, "success", nil)
	s.auth.ClearSession(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) sessionActivity(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "token keamanan tidak valid"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) idleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "token keamanan tidak valid"})
		return
	}
	s.writeAudit(r, "auth.idle_logout", "session", session.Subject, "success", nil)
	s.auth.ClearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	stats, err := s.store.Dashboard(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Dashboard", "Ringkasan inventory dan progres penyelesaian barang", "dashboard")
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	inventoryScope, inventoryScopeLabel := dashboardInventoryScope(r.URL.Query().Get("inventory_scope"), facilities)
	stats, err = s.dashboardStatsForSession(r.Context(), session, stats, facilities, inventoryScope)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data.Stats, data.Facilities = stats, facilities
	data.DashboardInventoryScope = inventoryScope
	data.DashboardInventoryLabel = inventoryScopeLabel
	data.FacilityID = r.URL.Query().Get("tpp")
	data.DashboardOccupancy = stats.Occupancy
	data.DashboardScope = "Gabungan seluruh TPP"
	data.DashboardRows = stats.FacilityBreakdown
	if data.FacilityID != "" {
		data.DashboardRows = nil
		found := false
		for _, row := range stats.FacilityBreakdown {
			if row.FacilityID == data.FacilityID {
				found = true
				data.DashboardRows = append(data.DashboardRows, row)
				data.DashboardOccupancy = domain.Occupancy{YardCapacity: row.YardCapacity, YardUsed: row.YardUsed, ShedCapacity: row.ShedCapacity, ShedUsed: row.ShedUsed}
				data.DashboardScope = row.FacilityName
				break
			}
		}
		if !found {
			data.FacilityID = ""
			data.DashboardRows = stats.FacilityBreakdown
		}
	}
	data.CanEditCapacity = session.Can(domain.PermissionDashboardCapacity)
	performanceFrom, performanceTo := performanceRange(r.URL.Query().Get("performance_from"), r.URL.Query().Get("performance_to"), time.Now())
	data.PerformanceOpen = r.URL.Query().Get("performance") == "1"
	performance, performanceErr := s.performanceReport(r.Context(), session, performanceFrom, performanceTo)
	if performanceErr != nil {
		s.renderStoreError(w, r, performanceErr)
		return
	}
	data.Performance = performance
	for _, processType := range []domain.DispositionType{domain.DispositionAuction, domain.DispositionDestruction, domain.DispositionGrant} {
		viewPermission, _ := processPermissions(processType)
		if !session.Can(viewPermission) {
			continue
		}
		processes, processErr := s.store.ListDispositions(r.Context(), domain.DispositionFilter{
			Type: processType, IncludeInactiveInventory: true, AllowedTypes: allowedInventoryTypes(session), Limit: 5000,
		})
		if processErr != nil {
			s.renderStoreError(w, r, processErr)
			return
		}
		dashboard := buildProcessDashboard(processType, processes, time.Now())
		modal := ProcessModalData{Type: processType, Dashboard: dashboard}
		switch processType {
		case domain.DispositionAuction:
			data.AuctionDashboard = dashboard
			modal.Title, modal.Singular, modal.URL = "Dashboard Lelang", "lelang", "/proses/lelang"
		case domain.DispositionDestruction:
			data.DestructionDashboard = dashboard
			modal.Title, modal.Singular, modal.URL = "Dashboard Pemusnahan", "pemusnahan", "/proses/musnah"
		case domain.DispositionGrant:
			data.GrantDashboard = dashboard
			modal.Title, modal.Singular, modal.URL = "Dashboard Hibah / PSP", "hibah / PSP", "/proses/hibah"
		}
		data.ProcessModals = append(data.ProcessModals, modal)
	}
	s.render(w, "dashboard", data)
}

func (s *Server) updateFacilityCapacity(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectMessage(w, r, "/", "error", "Form kapasitas TPP tidak valid.")
		return
	}
	yardCapacity, yardErr := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("yard_capacity")), ",", "."), 64)
	shedCapacity, shedErr := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("shed_capacity")), ",", "."), 64)
	if yardErr != nil || shedErr != nil || yardCapacity < 0 || shedCapacity < 0 {
		redirectMessage(w, r, "/", "error", "Kapasitas YOR dan SOR harus berupa angka nol atau lebih.")
		return
	}
	facility, err := s.store.UpdateFacilityCapacity(r.Context(), r.PathValue("id"), yardCapacity, shedCapacity)
	if err != nil {
		redirectMessage(w, r, "/", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "dashboard.capacity.update", "facility", facility.ID, "success", map[string]any{"yard_capacity": yardCapacity, "shed_capacity": shedCapacity})
	redirectMessage(w, r, "/?tpp="+url.QueryEscape(facility.ID), "ok", "Kapasitas YOR dan SOR "+facility.Name+" berhasil diperbarui.")
}

func dashboardInventoryScope(raw string, facilities []domain.Facility) (string, string) {
	scope := strings.TrimSpace(raw)
	switch scope {
	case "", "all_office":
		return "all_office", "Seluruh cakupan Kantor Tanjung Priok"
	case "still_tps":
		return "still_tps", "Masih di TPS"
	case "all_tpp":
		return "all_tpp", "Seluruh TPP"
	}
	for _, facility := range facilities {
		if facility.ID == scope {
			return facility.ID, facility.Name
		}
	}
	return "all_office", "Seluruh cakupan Kantor Tanjung Priok"
}

func dashboardInventoryItemInScope(item domain.InventoryItem, scope string) bool {
	if !item.IsActive {
		return false
	}
	switch scope {
	case "", "all_office":
		return true
	case "still_tps":
		return !item.AtTPP
	case "all_tpp":
		return item.AtTPP && strings.TrimSpace(item.FacilityID) != ""
	default:
		return item.AtTPP && item.FacilityID == scope
	}
}

func (s *Server) dashboardStatsForSession(ctx context.Context, session auth.Session, original domain.DashboardStats, facilities []domain.Facility, inventoryScope string) (domain.DashboardStats, error) {
	allowed := allowedInventoryTypes(session)
	items, err := s.store.ListInventory(ctx, domain.InventoryFilter{AllowedTypes: allowed, Limit: 5000})
	if err != nil {
		return domain.DashboardStats{}, err
	}
	stats := domain.DashboardStats{Occupancy: original.Occupancy}
	occupancyByFacility := make(map[string]domain.FacilityBreakdown, len(original.FacilityBreakdown))
	for _, row := range original.FacilityBreakdown {
		occupancyByFacility[row.FacilityID] = row
	}
	byFacility := make(map[string]*domain.FacilityBreakdown, len(facilities))
	for _, facility := range facilities {
		occupancy := occupancyByFacility[facility.ID]
		byFacility[facility.ID] = &domain.FacilityBreakdown{FacilityID: facility.ID, FacilityName: facility.Name, YardCapacity: facility.YardCapacity, YardUsed: occupancy.YardUsed, ShedCapacity: facility.ShedCapacity, ShedUsed: occupancy.ShedUsed}
	}
	now := time.Now()
	btdItems := make([]domain.InventoryItem, 0)
	bdnItems := make([]domain.InventoryItem, 0)
	bmmnItems := make([]domain.InventoryItem, 0)
	titipanItems := make([]domain.InventoryItem, 0)
	scopedItems := make([]domain.InventoryItem, 0, len(items))
	for _, item := range items {
		// Detail per TPP selalu merepresentasikan barang aktif yang benar-benar
		// berada di TPP. Angka ini sengaja tidak mengikuti filter KPI di atas.
		if item.IsActive && item.AtTPP && strings.TrimSpace(item.FacilityID) != "" {
			if row := byFacility[item.FacilityID]; row != nil {
				switch item.Type {
				case domain.InventoryBTD:
					row.BTD++
				case domain.InventoryBDN:
					row.BDN++
				case domain.InventoryBMMN:
					row.BMMN++
				case domain.InventoryTitipan:
					row.Titipan++
				}
				row.Total++
			}
		}

		if !dashboardInventoryItemInScope(item, inventoryScope) {
			continue
		}
		scopedItems = append(scopedItems, item)
		switch item.Type {
		case domain.InventoryBTD:
			stats.BTDTotal++
			btdItems = append(btdItems, item)
		case domain.InventoryBDN:
			stats.BDNTotal++
			bdnItems = append(bdnItems, item)
		case domain.InventoryBMMN:
			stats.BMMNTotal++
			bmmnItems = append(bmmnItems, item)
		case domain.InventoryTitipan:
			stats.TitipanTotal++
			titipanItems = append(titipanItems, item)
		}
		if item.AgeDays(now) >= 45 {
			stats.AttentionItems = append(stats.AttentionItems, item)
		}
	}
	stats.ActiveTotal = stats.BTDTotal + stats.BDNTotal + stats.BMMNTotal + stats.TitipanTotal
	stats.ActiveSummary = domain.SummarizeDashboardInventory(scopedItems)
	stats.BTDSummary = domain.SummarizeDashboardInventory(btdItems)
	stats.BDNSummary = domain.SummarizeDashboardInventory(bdnItems)
	stats.BMMNSummary = domain.SummarizeDashboardInventory(bmmnItems)
	stats.TitipanSummary = domain.SummarizeDashboardInventory(titipanItems)
	for _, facility := range facilities {
		if row := byFacility[facility.ID]; row != nil {
			stats.FacilityBreakdown = append(stats.FacilityBreakdown, *row)
		}
	}
	if len(stats.AttentionItems) > 5 {
		stats.AttentionItems = stats.AttentionItems[:5]
	}
	for _, kind := range []domain.DispositionType{domain.DispositionAuction, domain.DispositionDestruction, domain.DispositionGrant} {
		processes, processErr := s.store.ListDispositions(ctx, domain.DispositionFilter{Type: kind, IncludeInactiveInventory: true, AllowedTypes: allowed, Limit: 5000})
		if processErr != nil {
			return domain.DashboardStats{}, processErr
		}
		for _, process := range processes {
			if process.IsActive {
				switch kind {
				case domain.DispositionAuction:
					stats.AuctionActive++
				case domain.DispositionDestruction:
					stats.DestructionActive++
				case domain.DispositionGrant:
					stats.GrantActive++
				}
			} else if process.UpdatedAt.Year() == now.Year() && process.UpdatedAt.Month() == now.Month() {
				stats.CompletedThisMonth++
			}
		}
	}
	for _, event := range original.RecentEvents {
		item, itemErr := s.store.GetInventory(ctx, event.InventoryID)
		if itemErr == nil && sessionCanAccessItem(session, item) {
			stats.RecentEvents = append(stats.RecentEvents, event)
		}
	}
	return stats, nil
}

func (s *Server) inventory(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	history := r.URL.Query().Get("history") == "1"
	filter := domain.InventoryFilter{Query: r.URL.Query().Get("q"), FacilityID: r.URL.Query().Get("tpp"), Type: domain.InventoryType(strings.ToUpper(r.URL.Query().Get("type"))), Status: r.URL.Query().Get("status"), Sort: r.URL.Query().Get("sort"), OnlyInactive: history, AllowedTypes: allowedInventoryTypes(session)}
	if filter.Type != "" && filter.Type != domain.InventoryBTD && filter.Type != domain.InventoryBDN && filter.Type != domain.InventoryBMMN && filter.Type != domain.InventoryTitipan {
		filter.Type = ""
	}
	total, err := s.store.CountInventory(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	offset, pageSize, pagination := paginationForTotal(total, r)
	filter.Offset, filter.Limit = offset, pageSize
	items, err := s.store.ListInventory(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	allItems, err := s.store.ListInventory(r.Context(), domain.InventoryFilter{Limit: 5000, AllowedTypes: allowedInventoryTypes(session)})
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	title, subtitle := "Inventory", "Semua BTD, BDN, BMMN, dan barang titipan yang masih berstatus aktif"
	if history {
		title, subtitle = "Riwayat Inventory", "Barang yang telah dikeluarkan dan tidak lagi tampil pada inventory aktif"
	}
	data := s.baseData(r, title, subtitle, "inventory")
	data.Items, data.EligibleItems, data.Facilities = items, allItems, facilities
	data.InventoryActions = permittedInventoryActions(session)
	data.Pagination = pagination
	data.ResearchRequestGroups = groupResearchRequests(allItems)
	data.CensusTargetGroups = groupCensusTargets(allItems)
	data.RelocationTargetGroups = groupRelocationTargets(allItems)
	data.Query, data.FacilityID, data.InventoryType, data.Status, data.Sort = filter.Query, filter.FacilityID, filter.Type, filter.Status, filter.Sort
	data.History = history
	data.CanCreateBTD = sessionCanCreateInventory(session, domain.InventoryBTD)
	data.CanCreateBDN = sessionCanCreateInventory(session, domain.InventoryBDN)
	data.CanCreateTitipan = sessionCanCreateInventory(session, domain.InventoryTitipan)
	data.CanCreateInventory = data.CanCreateBTD || data.CanCreateBDN || data.CanCreateTitipan
	data.CanRunInventoryActions = len(data.InventoryActions) > 0
	data.CanManage = data.CanCreateInventory || data.CanRunInventoryActions
	s.render(w, "inventory", data)
}

func (s *Server) createInventory(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/inventory", "error", "Form pencatatan/penetapan tidak valid.")
		return
	}
	kind := domain.InventoryType(strings.ToUpper(strings.TrimSpace(r.FormValue("item_type"))))
	if !sessionCanCreateInventory(session, kind) {
		redirectMessage(w, r, "/inventory", "error", "Role Anda tidak memiliki hak pencatatan/penetapan untuk jenis barang tersebut.")
		return
	}
	atTPP := r.FormValue("at_tpp") == "sudah"
	loadType := strings.ToUpper(strings.TrimSpace(r.FormValue("load_type")))
	baseInput := domain.NewInventoryInput{
		Type: kind, BLNo: strings.TrimSpace(r.FormValue("bl_no")), BLDate: parseDate(r.FormValue("bl_date")), ManifestNo: strings.TrimSpace(r.FormValue("manifest_no")), ManifestDate: parseDate(r.FormValue("manifest_date")), ManifestPosition: strings.TrimSpace(r.FormValue("manifest_position")),
		DeterminationNo: strings.TrimSpace(r.FormValue("determination_no")), DeterminationDate: parseDate(r.FormValue("determination_date")), Category: strings.TrimSpace(r.FormValue("category")),
		EntrustedCategory: strings.TrimSpace(r.FormValue("entrusted_category")), SourceOffice: strings.TrimSpace(r.FormValue("source_office")),
		Location: strings.TrimSpace(r.FormValue("location")), AtTPP: atTPP,
		OwnerName: strings.TrimSpace(r.FormValue("owner_name")), OwnerAddress: strings.TrimSpace(r.FormValue("owner_address")), OriginWarehouse: strings.TrimSpace(r.FormValue("origin_warehouse")),
		FacilityID: r.FormValue("facility_id"), LoadType: loadType, RestrictionRule: strings.TrimSpace(r.FormValue("restriction_rule")), Actor: session.DisplayName,
	}
	if !atTPP {
		baseInput.FacilityID = ""
	}
	validTypeAndCategory := false
	switch kind {
	case domain.InventoryBTD:
		validTypeAndCategory = baseInput.BLNo != "" && !baseInput.BLDate.IsZero() && baseInput.Category == "" && domain.ValidTPS(baseInput.OriginWarehouse)
	case domain.InventoryBDN:
		validTypeAndCategory = domain.ValidBDNCategory(baseInput.Category) && domain.ValidTPS(baseInput.OriginWarehouse)
	case domain.InventoryTitipan:
		validTypeAndCategory = domain.ValidEntrustedCategory(baseInput.EntrustedCategory) && baseInput.SourceOffice != ""
	}
	if !validTypeAndCategory || baseInput.DeterminationNo == "" || baseInput.DeterminationDate.IsZero() || !domain.ValidLoadType(baseInput.LoadType) || atTPP && baseInput.FacilityID == "" {
		redirectMessage(w, r, "/inventory", "error", "Lengkapi dokumen dasar, nomor dan tanggal BL untuk BTD, kategori, jenis muatan, kantor/unit asal, serta TPP jika barang sudah berada di TPP.")
		return
	}

	buildGoodsInput := func(draft goodsLineDraft, physicalUnitID, referenceNo string, occupancyPrimary bool) (domain.NewInventoryInput, bool) {
		input := baseInput
		input.ReferenceNo = referenceNo
		input.PhysicalUnitID = physicalUnitID
		input.OccupancyPrimary = occupancyPrimary
		input.Description = strings.TrimSpace(draft.Description)
		input.ItemKind = strings.TrimSpace(draft.ItemKind)
		input.GoodsValue = parseMoney(draft.GoodsValue)
		input.Quantity = draft.Quantity
		input.QuantityDetail = strings.TrimSpace(draft.QuantityDetail)
		input.Unit = strings.TrimSpace(draft.Unit)
		return input, input.Description != "" && domain.ValidItemKind(input.ItemKind) && input.GoodsValue >= 0 && input.Quantity > 0 && domain.ValidUnit(input.Unit)
	}

	inputs := make([]domain.NewInventoryInput, 0, 8)
	totalGoods := 0
	switch loadType {
	case "FCL":
		var containers []containerDraft
		if err := json.Unmarshal([]byte(r.FormValue("containers_json")), &containers); err != nil || len(containers) == 0 || len(containers) > 100 {
			redirectMessage(w, r, "/inventory", "error", "Untuk FCL, tambahkan minimal satu dan maksimal 100 kontainer beserta identitas barangnya.")
			return
		}
		seen := make(map[string]struct{}, len(containers))
		for containerIndex, container := range containers {
			number, valid := normalizeContainerNumber(container.Number)
			size := strings.TrimSpace(container.Size)
			if !valid || !domain.ValidContainerSize(size) || len(container.Goods) == 0 || len(container.Goods) > 100 {
				redirectMessage(w, r, "/inventory", "error", "Setiap kontainer wajib memiliki nomor, ukuran, dan minimal satu identitas barang yang lengkap.")
				return
			}
			if _, duplicate := seen[number]; duplicate {
				redirectMessage(w, r, "/inventory", "error", "Nomor kontainer "+number+" dimasukkan lebih dari satu kali.")
				return
			}
			seen[number] = struct{}{}
			physicalUnitID := fmt.Sprintf("%s|%s", baseInput.DeterminationNo, strings.ReplaceAll(number, " ", ""))
			for goodsIndex, draft := range container.Goods {
				totalGoods++
				if totalGoods > 500 {
					redirectMessage(w, r, "/inventory", "error", "Maksimal 500 uraian barang dapat disimpan dalam satu dokumen.")
					return
				}
				referenceNo := fmt.Sprintf("%s/C%02d/G%02d", baseInput.DeterminationNo, containerIndex+1, goodsIndex+1)
				input, ok := buildGoodsInput(draft, physicalUnitID, referenceNo, goodsIndex == 0)
				if !ok {
					redirectMessage(w, r, "/inventory", "error", "Lengkapi uraian, jenis barang, nilai awal, jumlah, dan satuan pada seluruh kontainer.")
					return
				}
				input.ContainerNo = number
				input.ContainerSize = size
				inputs = append(inputs, input)
			}
		}
	case "LCL":
		volume, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("estimated_volume_m3")), ",", "."), 64)
		var goods []goodsLineDraft
		if err != nil || volume <= 0 || json.Unmarshal([]byte(r.FormValue("lcl_goods_json")), &goods) != nil || len(goods) == 0 || len(goods) > 100 {
			redirectMessage(w, r, "/inventory", "error", "Untuk LCL, isi volume dan minimal satu identitas barang yang lengkap.")
			return
		}
		physicalUnitID := fmt.Sprintf("%s|LCL", baseInput.DeterminationNo)
		for goodsIndex, draft := range goods {
			referenceNo := fmt.Sprintf("%s/LCL/G%02d", baseInput.DeterminationNo, goodsIndex+1)
			input, ok := buildGoodsInput(draft, physicalUnitID, referenceNo, goodsIndex == 0)
			if !ok {
				redirectMessage(w, r, "/inventory", "error", "Lengkapi uraian, jenis barang, nilai awal, jumlah, dan satuan seluruh barang LCL.")
				return
			}
			input.EstimatedVolumeM3 = volume
			inputs = append(inputs, input)
		}
	default:
		redirectMessage(w, r, "/inventory", "error", "Jenis muatan harus FCL atau LCL.")
		return
	}

	documentID, err := s.optionalDocument(r, session.DisplayName)
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	for index := range inputs {
		inputs[index].DocumentID = documentID
	}
	created, err := s.store.CreateInventories(r.Context(), inputs)
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	message := "Penetapan " + string(kind) + " berhasil ditambahkan ke inventory."
	if kind == domain.InventoryBTD {
		message = "Pencatatan BTD berhasil ditambahkan ke inventory."
	}
	if kind == domain.InventoryTitipan {
		message = "Pemasukan barang titipan kantor/unit lain berhasil ditambahkan ke inventory."
	}
	s.writeAudit(r, "inventory.create", "inventory_batch", baseInput.DeterminationNo, "success", map[string]any{"item_type": string(kind), "rows": len(created), "load_type": loadType, "document_uploaded": documentID != ""})
	if len(created) > 1 {
		if kind == domain.InventoryTitipan {
			message = fmt.Sprintf("Pemasukan barang titipan berhasil dibuat menjadi %d baris inventory berdasarkan identitas barang yang diinput.", len(created))
		} else if kind == domain.InventoryBTD {
			message = fmt.Sprintf("Pencatatan BTD berhasil dibuat menjadi %d baris inventory berdasarkan identitas barang yang diinput.", len(created))
		} else {
			message = fmt.Sprintf("Penetapan %s berhasil dibuat menjadi %d baris inventory berdasarkan identitas barang yang diinput.", kind, len(created))
		}
	}
	redirectMessage(w, r, "/inventory?type="+strings.ToLower(string(kind)), "ok", message)
}

func normalizeContainerNumber(value string) (string, bool) {
	compact := strings.NewReplacer(" ", "", "-", "", ".", "").Replace(strings.ToUpper(strings.TrimSpace(value)))
	if len(compact) != 11 {
		return "", false
	}
	for index := 0; index < 4; index++ {
		if compact[index] < 'A' || compact[index] > 'Z' {
			return "", false
		}
	}
	for index := 4; index < len(compact); index++ {
		if compact[index] < '0' || compact[index] > '9' {
			return "", false
		}
	}
	return compact[:4] + " " + compact[4:10] + "-" + compact[10:], true
}

func (s *Server) deleteInventory(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	item, err := s.store.GetInventory(r.Context(), r.PathValue("id"))
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	if err := s.store.DeleteInventory(r.Context(), item.ID, session.DisplayName); err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "inventory.delete", "inventory", item.ID, "success", map[string]any{"determination_no": item.DeterminationNo, "item_type": string(item.Type)})
	returnTo := strings.TrimSpace(r.FormValue("return_to"))
	if !strings.HasPrefix(returnTo, "/inventory") || strings.HasPrefix(returnTo, "//") {
		returnTo = "/inventory"
	}
	redirectMessage(w, r, returnTo, "ok", "Data barang "+item.DeterminationNo+" berhasil dihapus permanen. Snapshot sebelum penghapusan tersimpan pada audit database.")
}

func (s *Server) addInventoryEvent(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/inventory", "error", "Form tindak lanjut tidak valid.")
		return
	}
	input := inventoryEventInput(r, session.DisplayName)
	if !sessionCanPerformInventoryAction(session, input.Code) {
		redirectMessage(w, r, "/inventory", "error", "Role Anda tidak memiliki hak akses untuk action tersebut.")
		return
	}
	item, err := s.store.GetInventory(r.Context(), r.PathValue("id"))
	if err != nil || !sessionCanAccessItem(session, item) || validateInventoryEvent(item, input) != nil {
		redirectMessage(w, r, "/inventory", "error", "Lengkapi data wajib sesuai tahapan yang dipilih.")
		return
	}
	documentID, err := s.optionalDocument(r, session.DisplayName)
	if err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	input.DocumentID = documentID
	if _, err := s.store.AddInventoryEvent(r.Context(), r.PathValue("id"), input); err != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "inventory.event", "inventory", item.ID, "success", map[string]any{"event_code": input.Code, "document_no": input.DocumentNo, "document_uploaded": documentID != ""})
	redirectMessage(w, r, "/inventory", "ok", "Status inventory dan timeline berhasil diperbarui.")
}

func (s *Server) addBulkInventoryEvent(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/inventory", "error", "Form Action tidak valid.")
		return
	}
	input := inventoryEventInput(r, session.DisplayName)
	if !sessionCanPerformInventoryAction(session, input.Code) {
		redirectMessage(w, r, "/inventory", "error", "Role Anda tidak memiliki hak akses untuk action tersebut.")
		return
	}
	documentID, documentErr := s.optionalDocument(r, session.DisplayName)
	if documentErr != nil {
		redirectMessage(w, r, "/inventory", "error", friendlyError(documentErr))
		return
	}
	input.DocumentID = documentID
	if input.Code == "penelitian_pfpd" {
		s.handleBulkPFPDResearch(w, r, session, input)
		return
	}
	if input.Code == "pencacahan" {
		s.handleBulkInventoryCensus(w, r, session, input)
		return
	}
	if input.Code == "pindah_bongkar_kontainer" {
		s.handleContainerRelocation(w, r, session, input)
		return
	}
	ids := inventoryIDsFromForm(r)
	if len(ids) == 0 {
		redirectMessage(w, r, "/inventory", "error", "Pilih minimal satu barang berdasarkan nomor penetapan, kontainer, atau uraian.")
		return
	}
	if input.Code == "request_penelitian_pfpd" {
		expanded, expandErr := s.expandPhysicalInventoryIDs(r.Context(), session, ids)
		if expandErr != nil {
			redirectMessage(w, r, "/inventory", "error", "Uraian barang dalam kontainer yang dipilih belum dapat dimuat.")
			return
		}
		ids = expanded
	}

	for _, id := range ids {
		item, err := s.store.GetInventory(r.Context(), id)
		if err != nil || !sessionCanAccessItem(session, item) || validateInventoryEvent(item, input) != nil {
			redirectMessage(w, r, "/inventory", "error", "Action dibatalkan karena ada barang yang tidak memenuhi tahapan yang dipilih.")
			return
		}
	}
	for _, id := range ids {
		if _, err := s.store.AddInventoryEvent(r.Context(), id, input); err != nil {
			redirectMessage(w, r, "/inventory", "error", "Sebagian perubahan belum dapat disimpan. Periksa kembali status barang.")
			return
		}
	}
	s.writeAudit(r, "inventory.event.bulk", "inventory_batch", input.Code, "success", map[string]any{"event_code": input.Code, "items": len(ids), "document_no": input.DocumentNo, "document_uploaded": documentID != ""})
	redirectMessage(w, r, "/inventory", "ok", fmt.Sprintf("Action berhasil diterapkan pada %d barang.", len(ids)))
}

func (s *Server) handleContainerRelocation(w http.ResponseWriter, r *http.Request, session auth.Session, base domain.NewEventInput) {
	var draft containerRelocationDraft
	if err := json.Unmarshal([]byte(r.FormValue("container_relocation_json")), &draft); err != nil {
		redirectMessage(w, r, "/inventory", "error", "Lengkapi target bongkar/muat dan minimal satu tujuan untuk setiap target yang dipilih.")
		return
	}
	mode := strings.ToLower(strings.TrimSpace(draft.Mode))
	if mode == "" {
		mode = "bongkar"
	}
	operations := draft.Operations
	if len(operations) == 0 && strings.TrimSpace(draft.InventoryID) != "" && len(draft.Allocations) > 0 {
		operations = []containerRelocationOperationDraft{{InventoryID: draft.InventoryID, Allocations: draft.Allocations}}
	}
	if len(operations) == 0 || len(operations) > 250 {
		redirectMessage(w, r, "/inventory", "error", "Pilih minimal satu target bongkar/muat dan lengkapi tujuan setiap target.")
		return
	}

	type preparedRelocation struct {
		Item        domain.InventoryItem
		Allocations []domain.InventoryLoadAllocation
	}
	prepared := make([]preparedRelocation, 0, len(operations))
	seenIDs := make(map[string]struct{}, len(operations))

	for _, operation := range operations {
		inventoryID := strings.TrimSpace(operation.InventoryID)
		if inventoryID == "" || len(operation.Allocations) == 0 || len(operation.Allocations) > 20 {
			redirectMessage(w, r, "/inventory", "error", "Setiap target bongkar/muat wajib memiliki minimal satu tujuan yang lengkap.")
			return
		}
		if _, exists := seenIDs[inventoryID]; exists {
			redirectMessage(w, r, "/inventory", "error", "Satu barang hanya boleh dipilih satu kali dalam satu penyimpanan bongkar/muat.")
			return
		}
		seenIDs[inventoryID] = struct{}{}

		item, err := s.store.GetInventory(r.Context(), inventoryID)
		if err != nil || !sessionCanAccessItem(session, item) || validateInventoryEvent(item, base) != nil || item.Quantity <= 0 {
			redirectMessage(w, r, "/inventory", "error", "Salah satu target bongkar/muat tidak ditemukan, berada di luar cakupan role, atau tidak lagi dapat diproses.")
			return
		}
		loadType := strings.ToUpper(strings.TrimSpace(item.LoadType))
		if mode == "bongkar" && loadType != "FCL" {
			redirectMessage(w, r, "/inventory", "error", "Mode bongkar hanya dapat digunakan untuk barang FCL dalam kontainer.")
			return
		}
		if mode == "muat" && loadType != "LCL" {
			redirectMessage(w, r, "/inventory", "error", "Mode muat hanya dapat digunakan untuk barang LCL di gudang.")
			return
		}

		allocations := make([]domain.InventoryLoadAllocation, 0, len(operation.Allocations))
		seenFCL := make(map[string]struct{})
		totalQuantity := 0.0
		for _, raw := range operation.Allocations {
			allocation := raw
			allocation.LoadType = strings.ToUpper(strings.TrimSpace(allocation.LoadType))
			allocation.Quantity = math.Round(allocation.Quantity*100) / 100
			if allocation.Quantity <= 0 {
				redirectMessage(w, r, "/inventory", "error", "Kuantitas setiap tujuan harus lebih dari nol.")
				return
			}
			switch allocation.LoadType {
			case "FCL":
				number, valid := normalizeContainerNumber(allocation.ContainerNo)
				allocation.ContainerSize = strings.ToUpper(strings.TrimSpace(allocation.ContainerSize))
				if !valid || !domain.ValidContainerSize(allocation.ContainerSize) {
					redirectMessage(w, r, "/inventory", "error", "Nomor atau ukuran kontainer tujuan tidak valid.")
					return
				}
				allocation.ContainerNo = number
				allocation.EstimatedVolumeM3 = 0
				key := strings.NewReplacer(" ", "", "-", "").Replace(number)
				if _, duplicate := seenFCL[key]; duplicate {
					redirectMessage(w, r, "/inventory", "error", "Satu nomor kontainer tujuan hanya boleh dicantumkan satu kali untuk target yang sama.")
					return
				}
				seenFCL[key] = struct{}{}
			case "LCL":
				allocation.ContainerNo = ""
				allocation.ContainerSize = ""
				allocation.EstimatedVolumeM3 = math.Round(allocation.EstimatedVolumeM3*100) / 100
				if allocation.EstimatedVolumeM3 <= 0 {
					redirectMessage(w, r, "/inventory", "error", "Perkiraan volume untuk tujuan LCL wajib lebih dari nol.")
					return
				}
			default:
				redirectMessage(w, r, "/inventory", "error", "Jenis penempatan tujuan harus FCL atau LCL.")
				return
			}
			totalQuantity += allocation.Quantity
			allocations = append(allocations, allocation)
		}
		if math.Abs(totalQuantity-item.Quantity) > 0.005 {
			redirectMessage(w, r, "/inventory", "error", fmt.Sprintf("Total kuantitas tujuan untuk %s harus sama dengan kuantitas sumber, yaitu %.2f %s.", item.DeterminationNo, item.Quantity, item.Unit))
			return
		}
		if inventoryProcessLocked(item) && len(allocations) > 1 {
			redirectMessage(w, r, "/inventory", "error", "Barang yang sedang atau sudah menjalani proses penyelesaian tetap dapat dibongkar/dimuat, tetapi satu uraian hanya boleh dialokasikan ke satu tujuan agar keterkaitan proses tetap utuh.")
			return
		}
		changed := len(allocations) > 1
		if !changed {
			allocation := allocations[0]
			sourceLoadType := strings.ToUpper(strings.TrimSpace(item.LoadType))
			changed = allocation.LoadType != sourceLoadType
			if !changed && allocation.LoadType == "FCL" {
				sourceContainer, valid := normalizeContainerNumber(item.ContainerNo)
				changed = !valid || sourceContainer != allocation.ContainerNo || strings.ToUpper(strings.TrimSpace(item.ContainerSize)) != allocation.ContainerSize
			}
			if !changed && allocation.LoadType == "LCL" {
				changed = math.Abs(item.EstimatedVolumeM3-allocation.EstimatedVolumeM3) > 0.005
			}
		}
		if !changed {
			redirectMessage(w, r, "/inventory", "error", "Ubah tujuan atau tambahkan pembagian sebelum menyimpan bongkar/muat kontainer.")
			return
		}
		prepared = append(prepared, preparedRelocation{Item: item, Allocations: allocations})
	}

	selectedByTarget := make(map[string]map[string]struct{})
	for _, operation := range prepared {
		targetKey := operation.Item.ID
		if strings.EqualFold(operation.Item.LoadType, "FCL") {
			targetKey = relocationTargetKey(operation.Item)
		}
		if selectedByTarget[targetKey] == nil {
			selectedByTarget[targetKey] = make(map[string]struct{})
		}
		selectedByTarget[targetKey][operation.Item.ID] = struct{}{}
	}
	if mode == "bongkar" {
		allItems, listErr := s.store.ListInventory(r.Context(), domain.InventoryFilter{AllowedTypes: allowedInventoryTypes(session), Limit: 5000})
		if listErr != nil {
			redirectMessage(w, r, "/inventory", "error", "Seluruh uraian dalam kontainer terpilih belum dapat diverifikasi.")
			return
		}
		expectedByTarget := make(map[string]map[string]struct{}, len(selectedByTarget))
		for _, item := range allItems {
			if !strings.EqualFold(item.LoadType, "FCL") {
				continue
			}
			targetKey := relocationTargetKey(item)
			if _, selected := selectedByTarget[targetKey]; !selected {
				continue
			}
			if !item.IsActive || item.Quantity <= 0 || !sessionCanAccessItem(session, item) || validateInventoryEvent(item, base) != nil {
				continue
			}
			if expectedByTarget[targetKey] == nil {
				expectedByTarget[targetKey] = make(map[string]struct{})
			}
			expectedByTarget[targetKey][item.ID] = struct{}{}
		}
		for targetKey, selectedIDs := range selectedByTarget {
			expectedIDs := expectedByTarget[targetKey]
			if len(expectedIDs) == 0 || len(expectedIDs) != len(selectedIDs) {
				redirectMessage(w, r, "/inventory", "error", "Seluruh uraian barang dalam setiap kontainer terpilih wajib dialokasikan sebelum bongkar/pindah disimpan.")
				return
			}
			for inventoryID := range expectedIDs {
				if _, selected := selectedIDs[inventoryID]; !selected {
					redirectMessage(w, r, "/inventory", "error", "Seluruh uraian barang dalam setiap kontainer terpilih wajib dialokasikan sebelum bongkar/pindah disimpan.")
					return
				}
			}
		}
	}

	resultIDs := make([]string, 0)
	sourceIDs := make([]string, 0, len(prepared))
	for _, operation := range prepared {
		updated, err := s.store.RelocateInventoryLoad(r.Context(), domain.InventoryLoadRelocationInput{
			InventoryID:  operation.Item.ID,
			Allocations:  operation.Allocations,
			DocumentNo:   base.DocumentNo,
			DocumentDate: base.DocumentDate,
			Notes:        base.Notes,
			Actor:        base.Actor,
			DocumentID:   base.DocumentID,
		})
		if err != nil {
			redirectMessage(w, r, "/inventory", "error", friendlyError(err))
			return
		}
		sourceIDs = append(sourceIDs, operation.Item.ID)
		for _, current := range updated {
			resultIDs = append(resultIDs, current.ID)
		}
	}
	actionLabel := "Bongkar/Muat Kontainer"
	s.writeAudit(r, "inventory.load.relocate", "inventory", strings.Join(sourceIDs, ","), "success", map[string]any{
		"mode":                 mode,
		"source_inventory_ids": sourceIDs,
		"result_inventory_ids": resultIDs,
		"targets":              len(selectedByTarget),
		"items":                len(prepared),
	})
	redirectMessage(w, r, "/inventory", "ok", fmt.Sprintf("%s berhasil diproses untuk %d target dan %d uraian barang.", actionLabel, len(selectedByTarget), len(prepared)))
}

func (s *Server) handleBulkInventoryCensus(w http.ResponseWriter, r *http.Request, session auth.Session, base domain.NewEventInput) {
	var drafts []censusTargetDraft
	if err := json.Unmarshal([]byte(r.FormValue("census_results_json")), &drafts); err != nil || len(drafts) == 0 || len(drafts) > 100 {
		redirectMessage(w, r, "/inventory", "error", "Pilih minimal satu kontainer FCL atau satu barang LCL, lalu lengkapi hasil pencacahannya.")
		return
	}
	base.PFPDRequired = true
	type preparedCensus struct {
		TargetID string
		Lines    []domain.InventoryGoodsLine
	}
	prepared := make([]preparedCensus, 0, len(drafts))
	seenTargets := make(map[string]struct{}, len(drafts))
	seenUnits := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		targetID := strings.TrimSpace(draft.TargetID)
		if targetID == "" || len(draft.Lines) == 0 || len(draft.Lines) > 100 {
			redirectMessage(w, r, "/inventory", "error", "Data target atau uraian pencacahan tidak valid.")
			return
		}
		if _, duplicate := seenTargets[targetID]; duplicate {
			redirectMessage(w, r, "/inventory", "error", "Target pencacahan yang sama terpilih lebih dari satu kali.")
			return
		}
		seenTargets[targetID] = struct{}{}
		item, err := s.store.GetInventory(r.Context(), targetID)
		if err != nil || !sessionCanAccessItem(session, item) || item.LoadType != strings.TrimSpace(draft.LoadType) {
			redirectMessage(w, r, "/inventory", "error", "Target pencacahan tidak ditemukan atau berada di luar cakupan role Anda.")
			return
		}
		unitKey := domain.InventoryPhysicalUnitKey(item)
		if item.LoadType == "FCL" {
			if _, duplicate := seenUnits[unitKey]; duplicate {
				redirectMessage(w, r, "/inventory", "error", "Satu kontainer FCL hanya boleh dipilih satu kali dalam satu penyimpanan.")
				return
			}
			seenUnits[unitKey] = struct{}{}
		}
		lines := make([]domain.InventoryGoodsLine, 0, len(draft.Lines))
		for _, lineDraft := range draft.Lines {
			line := domain.InventoryGoodsLine{
				InventoryID:    strings.TrimSpace(lineDraft.InventoryID),
				Description:    strings.TrimSpace(lineDraft.Description),
				ItemKind:       strings.TrimSpace(lineDraft.ItemKind),
				GoodsValue:     parseMoney(lineDraft.GoodsValue),
				Quantity:       lineDraft.Quantity,
				QuantityDetail: strings.TrimSpace(lineDraft.QuantityDetail),
				Unit:           strings.TrimSpace(lineDraft.Unit),
				GoodsCondition: strings.TrimSpace(lineDraft.GoodsCondition),
			}
			if line.Description == "" || !domain.ValidItemKind(line.ItemKind) || line.GoodsValue < 0 || line.Quantity <= 0 || !domain.ValidUnit(line.Unit) || !domain.ValidGoodsCondition(line.GoodsCondition) {
				redirectMessage(w, r, "/inventory", "error", "Lengkapi uraian, jenis, nilai awal, jumlah, satuan, dan kondisi untuk setiap barang hasil pencacahan.")
				return
			}
			lines = append(lines, line)
		}
		check := base
		check.Description = lines[0].Description
		check.ItemKind = lines[0].ItemKind
		check.GoodsValue = lines[0].GoodsValue
		check.Quantity = lines[0].Quantity
		check.Unit = lines[0].Unit
		check.GoodsCondition = lines[0].GoodsCondition
		if validateInventoryEvent(item, check) != nil {
			redirectMessage(w, r, "/inventory", "error", "Action pencacahan dibatalkan karena ada barang yang tidak memenuhi tahapan yang dipilih.")
			return
		}
		prepared = append(prepared, preparedCensus{TargetID: targetID, Lines: lines})
	}

	totalRows := 0
	for _, result := range prepared {
		rows, err := s.store.ApplyInventoryCensus(r.Context(), result.TargetID, result.Lines, base)
		if err != nil {
			redirectMessage(w, r, "/inventory", "error", "Hasil pencacahan belum dapat disimpan. Pastikan seluruh uraian lama tetap tercantum dan data baru sudah lengkap.")
			return
		}
		totalRows += len(rows)
	}
	s.writeAudit(r, "inventory.census.bulk", "inventory_batch", base.DocumentNo, "success", map[string]any{"targets": len(prepared), "rows": totalRows, "document_uploaded": base.DocumentID != ""})
	redirectMessage(w, r, "/inventory", "ok", fmt.Sprintf("Hasil pencacahan %d target berhasil disimpan pada %d baris inventory.", len(prepared), totalRows))
}

func (s *Server) expandPhysicalInventoryIDs(ctx context.Context, session auth.Session, ids []string) ([]string, error) {
	selectedUnits := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		item, err := s.store.GetInventory(ctx, id)
		if err != nil || !sessionCanAccessItem(session, item) {
			if err != nil {
				return nil, err
			}
			return nil, store.ErrNotFound
		}
		selectedUnits[domain.InventoryPhysicalUnitKey(item)] = struct{}{}
	}
	items, err := s.store.ListInventory(ctx, domain.InventoryFilter{Limit: 20000, AllowedTypes: allowedInventoryTypes(session)})
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(ids))
	seen := make(map[string]struct{})
	for _, item := range items {
		if _, selected := selectedUnits[domain.InventoryPhysicalUnitKey(item)]; !selected {
			continue
		}
		if _, duplicate := seen[item.ID]; duplicate {
			continue
		}
		seen[item.ID] = struct{}{}
		result = append(result, item.ID)
	}
	if len(result) == 0 {
		return nil, store.ErrNotFound
	}
	return result, nil
}

func censusLinesFromForm(r *http.Request) ([]domain.InventoryGoodsLine, error) {
	mode := strings.TrimSpace(r.FormValue("multiple_goods"))
	if mode != "ya" && mode != "tidak" {
		return nil, store.ErrInvalidTransition
	}
	if mode != "ya" {
		quantity, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("quantity")), ",", "."), 64)
		return []domain.InventoryGoodsLine{{
			Description: strings.TrimSpace(r.FormValue("description")),
			ItemKind:    strings.TrimSpace(r.FormValue("item_kind")),
			Quantity:    quantity,
			Unit:        strings.TrimSpace(r.FormValue("unit")),
		}}, nil
	}
	var lines []domain.InventoryGoodsLine
	if err := json.Unmarshal([]byte(r.FormValue("census_lines_json")), &lines); err != nil || len(lines) < 2 || len(lines) > 100 {
		return nil, store.ErrInvalidTransition
	}
	for index := range lines {
		lines[index].Description = strings.TrimSpace(lines[index].Description)
		lines[index].ItemKind = strings.TrimSpace(lines[index].ItemKind)
		lines[index].Unit = strings.TrimSpace(lines[index].Unit)
	}
	return lines, nil
}

func (s *Server) handleBulkPFPDResearch(w http.ResponseWriter, r *http.Request, session auth.Session, base domain.NewEventInput) {
	var drafts []pfpdResultDraft
	if err := json.Unmarshal([]byte(r.FormValue("pfpd_results_json")), &drafts); err != nil || len(drafts) == 0 || len(drafts) > 200 {
		redirectMessage(w, r, "/inventory", "error", "Buka satu nomor request dan lengkapi hasil penelitian setiap uraian barang.")
		return
	}
	requestNo := ""
	seen := make(map[string]struct{}, len(drafts))
	type preparedResult struct {
		ID    string
		Input domain.NewEventInput
	}
	prepared := make([]preparedResult, 0, len(drafts))
	for _, draft := range drafts {
		id := strings.TrimSpace(draft.InventoryID)
		if id == "" {
			redirectMessage(w, r, "/inventory", "error", "Data uraian penelitian tidak valid.")
			return
		}
		if _, duplicate := seen[id]; duplicate {
			redirectMessage(w, r, "/inventory", "error", "Uraian penelitian yang sama terkirim lebih dari satu kali.")
			return
		}
		seen[id] = struct{}{}
		item, err := s.store.GetInventory(r.Context(), id)
		if err != nil || !sessionCanAccessItem(session, item) || item.StatusCode != "request_penelitian_pfpd" || item.ResearchRequestNo == "" {
			redirectMessage(w, r, "/inventory", "error", "Salah satu uraian tidak lagi berada pada tahap request penelitian PFPD.")
			return
		}
		if requestNo == "" {
			requestNo = item.ResearchRequestNo
		} else if requestNo != item.ResearchRequestNo {
			redirectMessage(w, r, "/inventory", "error", "Satu penyimpanan penelitian hanya boleh memuat satu nomor request PFPD.")
			return
		}
		input := base
		input.HSCode = strings.TrimSpace(draft.HSCode)
		input.RestrictionStatus = strings.TrimSpace(draft.IsRestricted)
		input.IsRestricted = input.RestrictionStatus == "ya"
		input.RestrictionRule = strings.TrimSpace(draft.RestrictionRule)
		input.GoodsValue = parseMoney(draft.GoodsValue)
		if validateInventoryEvent(item, input) != nil {
			redirectMessage(w, r, "/inventory", "error", "Lengkapi HS code, nilai barang, dan status lartas untuk seluruh uraian dalam request.")
			return
		}
		prepared = append(prepared, preparedResult{ID: id, Input: input})
	}
	for _, result := range prepared {
		if _, err := s.store.AddInventoryEvent(r.Context(), result.ID, result.Input); err != nil {
			redirectMessage(w, r, "/inventory", "error", "Sebagian hasil penelitian belum dapat disimpan. Periksa kembali status request.")
			return
		}
	}
	s.writeAudit(r, "inventory.pfpd.bulk", "research_request", requestNo, "success", map[string]any{"items": len(prepared), "document_no": base.DocumentNo, "document_uploaded": base.DocumentID != ""})
	redirectMessage(w, r, "/inventory", "ok", fmt.Sprintf("Penelitian PFPD untuk request %s berhasil disimpan pada %d uraian barang.", requestNo, len(prepared)))
}

func inventoryEventInput(r *http.Request, actor string) domain.NewEventInput {
	quantity, _ := strconv.ParseFloat(strings.ReplaceAll(r.FormValue("quantity"), ",", "."), 64)
	return domain.NewEventInput{
		Code: r.FormValue("event_code"), DocumentNo: strings.TrimSpace(r.FormValue("document_no")), DocumentDate: parseDate(r.FormValue("document_date")), Notes: strings.TrimSpace(r.FormValue("notes")), Actor: actor,
		TargetFacilityID: r.FormValue("target_facility_id"), Description: strings.TrimSpace(r.FormValue("description")), ItemKind: strings.TrimSpace(r.FormValue("item_kind")), Quantity: quantity, Unit: strings.TrimSpace(r.FormValue("unit")), GoodsCondition: strings.TrimSpace(r.FormValue("goods_condition")), PFPDRequired: r.FormValue("pfpd_required") == "ya",
		ResearchRequestNo: strings.TrimSpace(r.FormValue("research_request_no")), ResearchRequestDate: parseDate(r.FormValue("research_request_date")), HSCode: strings.TrimSpace(r.FormValue("hs_code")), RestrictionStatus: r.FormValue("is_restricted"), IsRestricted: r.FormValue("is_restricted") == "ya", RestrictionRule: strings.TrimSpace(r.FormValue("restriction_rule")), GoodsValue: parseMoney(r.FormValue("goods_value")),
		AllocationPurpose: strings.TrimSpace(r.FormValue("allocation_purpose")), ExitType: r.FormValue("exit_type"), ExitNotes: strings.TrimSpace(r.FormValue("exit_notes")),
		AllocationType: strings.TrimSpace(r.FormValue("allocation_type")),
	}
}

func inventoryIDsFromForm(r *http.Request) []string {
	return formIDs(r, "inventory_ids")
}

func formIDs(r *http.Request, field string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, raw := range r.Form[field] {
		for _, id := range strings.FieldsFunc(raw, func(value rune) bool { return value == ',' || value == ' ' }) {
			if id != "" && !seen[id] {
				seen[id] = true
				result = append(result, id)
			}
		}
	}
	return result
}

func validateInventoryEvent(item domain.InventoryItem, input domain.NewEventInput) error {
	action, valid := domain.FindInventoryAction(input.Code)
	processLocked := inventoryProcessLocked(item)
	if !valid || !item.IsActive || processLocked && input.Code != "pengeluaran_barang" && input.Code != "pindah_bongkar_kontainer" || input.DocumentNo == "" || input.DocumentDate.IsZero() {
		return store.ErrInvalidTransition
	}
	if action.BMMNOnly && item.Type != domain.InventoryBMMN || action.NonBMMNOnly && (item.Type == domain.InventoryBMMN || item.Type == domain.InventoryTitipan) {
		return store.ErrInvalidTransition
	}
	switch input.Code {
	case "pemindahan":
		if input.TargetFacilityID == "" {
			return store.ErrInvalidTransition
		}
	case "pencacahan":
		if input.Description == "" || !domain.ValidItemKind(input.ItemKind) || input.Quantity <= 0 || !domain.ValidUnit(input.Unit) || !domain.ValidGoodsCondition(input.GoodsCondition) {
			return store.ErrInvalidTransition
		}
	case "penelitian_pfpd":
		if item.ResearchRequestNo == "" || input.HSCode == "" || input.GoodsValue <= 0 || input.RestrictionStatus != "ya" && input.RestrictionStatus != "tidak" || input.IsRestricted && input.RestrictionRule == "" {
			return store.ErrInvalidTransition
		}
	case "usulan_peruntukan_bmmn", "persetujuan_peruntukan_bmmn":
		if !domain.ValidAllocationPurpose(input.AllocationType) {
			return store.ErrInvalidTransition
		}
		if input.Code == "persetujuan_peruntukan_bmmn" && item.AllocationProposalNo == "" {
			return store.ErrInvalidTransition
		}
	case "pengeluaran_barang":
		if !validInventoryExitRequest(item, input.ExitType) {
			return store.ErrInvalidTransition
		}
	}
	return nil
}

func inventoryProcessLocked(item domain.InventoryItem) bool {
	return strings.TrimSpace(string(item.CurrentDisposition)) != "" || completedInventoryProcessStatus(item.StatusCode)
}

func completedInventoryProcessStatus(code string) bool {
	return code == "laku" || code == "alokasi_hasil_lelang" || code == "ba_musnah" || code == "ba_serah_terima"
}

func validInventoryExitRequest(item domain.InventoryItem, exitType string) bool {
	if !domain.ValidExitType(item.Type, exitType) {
		return false
	}
	if item.Type == domain.InventoryTitipan {
		return exitType == "pengeluaran_barang_titipan" && item.CurrentDisposition == ""
	}
	switch exitType {
	case "lelang":
		return item.StatusCode == "laku" || item.StatusCode == "alokasi_hasil_lelang"
	case "musnah":
		return item.StatusCode == "kep_musnah" || item.StatusCode == "ba_musnah" || item.CurrentDisposition == domain.DispositionDestruction
	case "hibah":
		return item.StatusCode == "ba_serah_terima" && strings.Contains(strings.ToUpper(item.StatusLabel), "HIBAH")
	case "psp":
		return item.StatusCode == "ba_serah_terima" && strings.Contains(strings.ToUpper(item.StatusLabel), "PSP")
	default:
		return item.CurrentDisposition == ""
	}
}

func groupAuctionSchedules(processes []domain.Disposition) []AuctionScheduleGroup {
	groups := make(map[string]*AuctionScheduleGroup)
	order := make([]string, 0)
	for _, process := range processes {
		if process.Type != domain.DispositionAuction || !process.IsActive || process.StatusCode != "jadwal_lelang" {
			continue
		}
		documentNo := strings.TrimSpace(process.ScheduleDocumentNo)
		if documentNo == "" {
			continue
		}
		group := groups[documentNo]
		if group == nil {
			group = &AuctionScheduleGroup{DocumentNo: documentNo, DocumentDate: process.ScheduleDocumentDate}
			groups[documentNo] = group
			order = append(order, documentNo)
		}
		if group.DocumentDate.IsZero() && !process.ScheduleDocumentDate.IsZero() {
			group.DocumentDate = process.ScheduleDocumentDate
		}
		group.Processes = append(group.Processes, process)
	}
	result := make([]AuctionScheduleGroup, 0, len(order))
	for _, documentNo := range order {
		group := groups[documentNo]
		sort.SliceStable(group.Processes, func(i, j int) bool {
			left, right := group.Processes[i].Inventory, group.Processes[j].Inventory
			if left.ContainerNo != right.ContainerNo {
				return left.ContainerNo < right.ContainerNo
			}
			return left.Description < right.Description
		})
		result = append(result, *group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].DocumentDate.Equal(result[j].DocumentDate) {
			return result[i].DocumentNo < result[j].DocumentNo
		}
		return result[i].DocumentDate.After(result[j].DocumentDate)
	})
	return result
}

func (s *Server) processPage(w http.ResponseWriter, r *http.Request) {
	kind, title, singular, actions, ok := processConfig(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	session, _ := sessionFromContext(r.Context())
	viewPermission, _ := processPermissions(kind)
	if !session.Can(viewPermission) {
		s.forbidden(w, r)
		return
	}
	history := r.URL.Query().Get("history") == "1"
	allowedTypes := allowedInventoryTypes(session)
	filter := domain.DispositionFilter{Type: kind, Query: r.URL.Query().Get("q"), FacilityID: r.URL.Query().Get("tpp"), Status: r.URL.Query().Get("status"), Sort: r.URL.Query().Get("sort"), AllowedTypes: allowedTypes}
	if history {
		filter.Status = ""
	}
	switch kind {
	case domain.DispositionAuction:
		filter.IncludeInactiveInventory = history
		if history {
			filter.IncludeStatusCodes = []string{"laku", "alokasi_hasil_lelang", "dialihkan_musnah", "dialihkan_hibah"}
		} else {
			filter.ExcludeStatusCodes = []string{"laku", "alokasi_hasil_lelang", "dialihkan_musnah", "dialihkan_hibah"}
		}
	case domain.DispositionDestruction:
		filter.IncludeInactiveInventory = true
		if history {
			filter.IncludeStatusCodes = []string{"ba_musnah"}
		} else {
			filter.ExcludeStatusCodes = []string{"ba_musnah"}
		}
	default:
		filter.OnlyInactiveInventory = history
	}
	total, err := s.store.CountDispositions(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	offset, pageSize, pagination := paginationForTotal(total, r)
	filter.Offset, filter.Limit = offset, pageSize
	processes, err := s.store.ListDispositions(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	eligibleItems, err := s.store.ListInventory(r.Context(), domain.InventoryFilter{AllowedTypes: allowedTypes, Limit: 20000, Sort: "newest"})
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	eligibleItems = filterItemsForSession(session, eligibleItems)
	filteredEligibleItems := eligibleItems[:0]
	for _, item := range eligibleItems {
		if processSourceEligible(item, kind) {
			filteredEligibleItems = append(filteredEligibleItems, item)
		}
	}
	eligibleItems = filteredEligibleItems
	candidateProcesses, err := s.store.ListDispositions(r.Context(), domain.DispositionFilter{Type: kind, IncludeInactiveInventory: kind == domain.DispositionDestruction, Limit: 1000, AllowedTypes: allowedTypes})
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	processDashboard, err := s.store.ProcessDashboard(r.Context(), kind, time.Now().Year(), allowedTypes)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	pageTitle := title
	subtitle := "Setiap proses dimulai dengan memilih barang aktif dari inventory"
	if history {
		pageTitle = "Riwayat " + title
		switch kind {
		case domain.DispositionAuction:
			subtitle = "Barang laku dan barang tidak laku yang dialihkan ke penyelesaian lain disimpan di sini sebagai jejak proses"
		case domain.DispositionDestruction:
			subtitle = "Pemusnahan yang telah selesai disimpan di sini sebagai jejak penyelesaian"
		default:
			subtitle = "Barang yang telah keluar dari inventory aktif dan tersimpan sebagai jejak penyelesaian"
		}
	}
	data := s.baseData(r, pageTitle, subtitle, string(kind))
	data.Processes, data.Facilities, data.ProcessActions, data.EligibleItems, data.CandidateProcesses = processes, facilities, actions, eligibleItems, candidateProcesses
	data.Pagination = pagination
	data.ProcessType, data.ProcessTitle, data.ProcessSingular = kind, title, singular
	data.Query, data.FacilityID, data.Status, data.Sort = filter.Query, filter.FacilityID, filter.Status, filter.Sort
	data.History = history
	_, managePermission := processPermissions(kind)
	data.CanManage = session.Can(managePermission)
	data.ProcessDashboard = processDashboard
	if kind == domain.DispositionAuction {
		data.AuctionScheduleGroups = groupAuctionSchedules(candidateProcesses)
	}
	s.render(w, "process", data)
}

func buildProcessDashboard(kind domain.DispositionType, processes []domain.Disposition, now time.Time) domain.ProcessDashboard {
	monthLabels := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	dashboard := domain.ProcessDashboard{Year: now.Year(), Chart: make([]domain.ProcessChartPoint, 12)}
	for index, label := range monthLabels {
		dashboard.Chart[index].Label = label
	}
	for _, process := range processes {
		if process.Type != kind {
			continue
		}
		if process.IsActive {
			dashboard.Active++
		}
		if process.CreatedAt.Year() == dashboard.Year {
			dashboard.StartedThisYear++
			dashboard.ThisYear++
		}
		completed := !process.IsActive
		if kind == domain.DispositionAuction && (process.StatusCode == "laku" || process.StatusCode == "tidak_laku" || process.StatusCode == "alokasi_hasil_lelang") {
			// Tahap Selesai Lelang menandai pelaksanaan lelang sudah selesai,
			// walaupun barang laku masih aktif sampai alokasi hasil dicatat.
			completed = true
		}
		if completed && process.UpdatedAt.Year() == dashboard.Year {
			dashboard.CompletedThisYear++
		}
		if process.CreatedAt.Year() != dashboard.Year {
			continue
		}
		month := int(process.CreatedAt.Month()) - 1
		if month < 0 || month >= len(dashboard.Chart) {
			month = int(process.UpdatedAt.Month()) - 1
		}
		if month < 0 || month >= len(dashboard.Chart) {
			continue
		}
		point := &dashboard.Chart[month]
		point.Count++
		point.GoodsValue += process.Inventory.GoodsValue
		point.HTLValue += process.HTLValue
		point.SaleValue += process.SaleValue
		switch kind {
		case domain.DispositionAuction:
			dashboard.TotalGoodsValue += process.Inventory.GoodsValue
			dashboard.TotalHTLValue += process.HTLValue
			dashboard.TotalSaleValue += process.SaleValue
		case domain.DispositionDestruction:
			point.Cost += process.DestructionCost
			dashboard.TotalCost += process.DestructionCost
		case domain.DispositionGrant:
			if process.TransferType == "hibah" {
				point.Grant++
				dashboard.TotalGrant++
			} else if process.TransferType == "psp" {
				point.PSP++
				dashboard.TotalPSP++
			}
		}
		if point.Count > dashboard.MaxCount {
			dashboard.MaxCount = point.Count
		}
		if point.Grant > dashboard.MaxCount {
			dashboard.MaxCount = point.Grant
		}
		if point.PSP > dashboard.MaxCount {
			dashboard.MaxCount = point.PSP
		}
		for _, value := range []int64{point.GoodsValue, point.HTLValue, point.SaleValue, point.Cost} {
			if value > dashboard.MaxMoney {
				dashboard.MaxMoney = value
			}
		}
	}
	if dashboard.MaxCount == 0 {
		dashboard.MaxCount = 1
	}
	if dashboard.MaxMoney == 0 {
		dashboard.MaxMoney = 1
	}
	return dashboard
}

func (s *Server) createProcess(w http.ResponseWriter, r *http.Request) {
	kind, _, _, _, ok := processConfig(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	session, _ := sessionFromContext(r.Context())
	_, managePermission := processPermissions(kind)
	if !session.Can(managePermission) {
		s.forbidden(w, r)
		return
	}
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || r.FormValue("inventory_id") == "" {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Pilih barang inventory terlebih dahulu.")
		return
	}
	item, itemErr := s.store.GetInventory(r.Context(), r.FormValue("inventory_id"))
	if itemErr != nil || !sessionCanAccessItem(session, item) {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Barang tidak ditemukan atau berada di luar cakupan role Anda.")
		return
	}
	createdProcess, err := s.store.CreateDisposition(r.Context(), domain.NewDispositionInput{InventoryID: r.FormValue("inventory_id"), Type: kind, Actor: session.DisplayName, Notes: strings.TrimSpace(r.FormValue("notes"))})
	if err != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "process.create", "disposition", createdProcess.ID, "success", map[string]any{"process_type": string(kind), "inventory_id": item.ID})
	redirectMessage(w, r, "/proses/"+string(kind), "ok", "Proses "+processLabel(kind)+" berhasil dimulai dari inventory.")
}

func (s *Server) addProcessEvent(w http.ResponseWriter, r *http.Request) {
	kind, _, _, _, ok := processConfig(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	session, _ := sessionFromContext(r.Context())
	_, managePermission := processPermissions(kind)
	if !session.Can(managePermission) {
		s.forbidden(w, r)
		return
	}
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Form tindak lanjut tidak valid.")
		return
	}
	input := processEventInput(r, session.DisplayName)
	action, valid := domain.FindDispositionAction(kind, input.Code)
	if !valid {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Pilih tahapan tindak lanjut.")
		return
	}
	input.Label = action.Label
	process, err := s.store.GetDisposition(r.Context(), r.PathValue("id"))
	if err != nil || process.Type != kind || !sessionCanAccessItem(session, process.Inventory) || validateProcessEvent(process, input, action) != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Proses yang dipilih tidak sesuai dengan menu.")
		return
	}
	documentID, documentErr := s.optionalDocument(r, session.DisplayName)
	if documentErr != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", friendlyError(documentErr))
		return
	}
	input.DocumentID = documentID
	if _, err := s.store.AddDispositionEvent(r.Context(), r.PathValue("id"), input); err != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "process.event", "disposition", process.ID, "success", map[string]any{"process_type": string(kind), "event_code": input.Code, "document_no": input.DocumentNo, "document_uploaded": documentID != ""})
	redirectMessage(w, r, "/proses/"+string(kind), "ok", "Tahapan "+processLabel(kind)+" berhasil diperbarui.")
}

func (s *Server) bulkProcessAction(w http.ResponseWriter, r *http.Request) {
	kind, _, _, _, ok := processConfig(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	session, _ := sessionFromContext(r.Context())
	_, managePermission := processPermissions(kind)
	if !session.Can(managePermission) {
		s.forbidden(w, r)
		return
	}
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Form action tidak valid.")
		return
	}
	input := processEventInput(r, session.DisplayName)
	action, valid := domain.FindDispositionAction(kind, input.Code)
	if !valid {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Pilih action terlebih dahulu.")
		return
	}
	documentID, documentErr := s.optionalDocument(r, session.DisplayName)
	if documentErr != nil {
		redirectMessage(w, r, "/proses/"+string(kind), "error", friendlyError(documentErr))
		return
	}
	input.DocumentID = documentID
	if kind == domain.DispositionAuction && input.Code == "kep_htl" {
		s.handleBulkAuctionHTL(w, r, session, input, action)
		return
	}
	if kind == domain.DispositionAuction && input.Code == "selesai_lelang" {
		s.handleBulkAuctionCompletion(w, r, session, input, action)
		return
	}

	if action.CreatesProcess {
		ids := formIDs(r, "inventory_ids")
		if len(ids) == 0 {
			redirectMessage(w, r, "/proses/"+string(kind), "error", "Pilih minimal satu kontainer dari inventory.")
			return
		}
		for _, id := range ids {
			item, err := s.store.GetInventory(r.Context(), id)
			placeholder := domain.Disposition{Type: kind, StatusCode: "proses_" + string(kind), IsActive: true, Inventory: item}
			if err != nil || !sessionCanAccessItem(session, item) || !processSourceEligible(item, kind) || validateProcessEvent(placeholder, input, action) != nil {
				redirectMessage(w, r, "/proses/"+string(kind), "error", "Action dibatalkan karena ada barang yang tidak memenuhi syarat.")
				return
			}
		}
		for _, id := range ids {
			process, err := s.store.CreateDisposition(r.Context(), domain.NewDispositionInput{InventoryID: id, Type: kind, Actor: session.DisplayName, Notes: input.Notes})
			if err != nil {
				redirectMessage(w, r, "/proses/"+string(kind), "error", "Sebagian proses belum dapat dibuat. Periksa kembali status barang.")
				return
			}
			if _, err := s.store.AddDispositionEvent(r.Context(), process.ID, input); err != nil {
				redirectMessage(w, r, "/proses/"+string(kind), "error", friendlyError(err))
				return
			}
		}
		s.writeAudit(r, "process.event.bulk", "disposition_batch", input.Code, "success", map[string]any{"process_type": string(kind), "items": len(ids), "creates_process": true, "document_no": input.DocumentNo, "document_uploaded": documentID != ""})
		s.writeAudit(r, "process.event.bulk", "disposition_batch", input.Code, "success", map[string]any{"process_type": string(kind), "items": len(ids), "creates_process": false, "document_no": input.DocumentNo, "document_uploaded": documentID != ""})
		redirectMessage(w, r, "/proses/"+string(kind), "ok", fmt.Sprintf("%s berhasil diterapkan pada %d barang.", action.Label, len(ids)))
		return
	}

	ids := formIDs(r, "process_ids")
	if len(ids) == 0 {
		redirectMessage(w, r, "/proses/"+string(kind), "error", "Pilih minimal satu barang pada proses ini.")
		return
	}
	for _, id := range ids {
		process, err := s.store.GetDisposition(r.Context(), id)
		if err != nil || process.Type != kind || !sessionCanAccessItem(session, process.Inventory) || validateProcessEvent(process, input, action) != nil {
			redirectMessage(w, r, "/proses/"+string(kind), "error", "Action dibatalkan karena ada barang dengan status yang tidak sesuai.")
			return
		}
	}
	for _, id := range ids {
		if _, err := s.store.AddDispositionEvent(r.Context(), id, input); err != nil {
			redirectMessage(w, r, "/proses/"+string(kind), "error", "Sebagian perubahan belum dapat disimpan. Periksa kembali status barang.")
			return
		}
	}
	redirectMessage(w, r, "/proses/"+string(kind), "ok", fmt.Sprintf("%s berhasil diterapkan pada %d barang.", action.Label, len(ids)))
}

func (s *Server) handleBulkAuctionHTL(w http.ResponseWriter, r *http.Request, session auth.Session, base domain.NewEventInput, action domain.WorkflowAction) {
	var drafts []htlResultDraft
	if err := json.Unmarshal([]byte(r.FormValue("htl_results_json")), &drafts); err != nil || len(drafts) == 0 || len(drafts) > 200 {
		redirectMessage(w, r, "/proses/lelang", "error", "Pilih barang lelang dan isi nilai HTL masing-masing barang.")
		return
	}
	selected := formIDs(r, "process_ids")
	if len(selected) == 0 || len(selected) != len(drafts) {
		redirectMessage(w, r, "/proses/lelang", "error", "Daftar barang dan input nilai HTL tidak konsisten.")
		return
	}
	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
	}
	type preparedHTL struct {
		ProcessID string
		Input     domain.NewEventInput
	}
	prepared := make([]preparedHTL, 0, len(drafts))
	seen := make(map[string]struct{}, len(drafts))
	userNotes := strings.TrimSpace(r.FormValue("notes"))
	for _, draft := range drafts {
		id := strings.TrimSpace(draft.ProcessID)
		if _, ok := selectedSet[id]; !ok || id == "" {
			redirectMessage(w, r, "/proses/lelang", "error", "Ada input HTL yang tidak sesuai dengan barang terpilih.")
			return
		}
		if _, duplicate := seen[id]; duplicate {
			redirectMessage(w, r, "/proses/lelang", "error", "Barang yang sama memiliki lebih dari satu input HTL.")
			return
		}
		seen[id] = struct{}{}
		input := base
		input.HTLValue = parseMoney(draft.HTLValue)
		details := []string{"Nilai HTL: Rp " + formatThousands(strconv.FormatInt(input.HTLValue, 10))}
		if userNotes != "" {
			details = append(details, userNotes)
		}
		input.Notes = strings.Join(details, " · ")
		process, err := s.store.GetDisposition(r.Context(), id)
		if err != nil || process.Type != domain.DispositionAuction || !sessionCanAccessItem(session, process.Inventory) || validateProcessEvent(process, input, action) != nil {
			redirectMessage(w, r, "/proses/lelang", "error", "Lengkapi nilai HTL setiap barang dan pastikan status prosesnya sesuai.")
			return
		}
		prepared = append(prepared, preparedHTL{ProcessID: id, Input: input})
	}
	for _, result := range prepared {
		if _, err := s.store.AddDispositionEvent(r.Context(), result.ProcessID, result.Input); err != nil {
			redirectMessage(w, r, "/proses/lelang", "error", "Sebagian nilai HTL belum dapat disimpan. Periksa kembali status proses.")
			return
		}
	}
	s.writeAudit(r, "process.auction.htl.bulk", "disposition_batch", base.DocumentNo, "success", map[string]any{"items": len(prepared), "document_uploaded": base.DocumentID != ""})
	redirectMessage(w, r, "/proses/lelang", "ok", fmt.Sprintf("KEP Harga Terendah Lelang berhasil diterbitkan untuk %d barang dengan nilai HTL masing-masing.", len(prepared)))
}

func (s *Server) handleBulkAuctionCompletion(w http.ResponseWriter, r *http.Request, session auth.Session, base domain.NewEventInput, action domain.WorkflowAction) {
	scheduleNo := strings.TrimSpace(r.FormValue("auction_schedule_no"))
	var drafts []auctionResultDraft
	if scheduleNo == "" || json.Unmarshal([]byte(r.FormValue("auction_results_json")), &drafts) != nil || len(drafts) == 0 || len(drafts) > 500 {
		redirectMessage(w, r, "/proses/lelang", "error", "Pilih satu ND penjadwalan dan lengkapi hasil setiap kontainer.")
		return
	}
	if base.DocumentNo == "" || base.DocumentDate.IsZero() {
		redirectMessage(w, r, "/proses/lelang", "error", "Lengkapi nomor dan tanggal risalah lelang.")
		return
	}

	all, err := s.store.ListDispositions(r.Context(), domain.DispositionFilter{Type: domain.DispositionAuction, IncludeInactiveInventory: true, Limit: 5000, AllowedTypes: allowedInventoryTypes(session)})
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	expected := make(map[string]domain.Disposition)
	for _, process := range all {
		if process.IsActive && process.StatusCode == "jadwal_lelang" && strings.TrimSpace(process.ScheduleDocumentNo) == scheduleNo && sessionCanAccessItem(session, process.Inventory) {
			expected[process.ID] = process
		}
	}
	if len(expected) == 0 || len(expected) != len(drafts) {
		redirectMessage(w, r, "/proses/lelang", "error", "Seluruh kontainer dalam ND penjadwalan harus ditetapkan hasilnya sekaligus.")
		return
	}

	type preparedAuctionResult struct {
		ID    string
		Input domain.NewEventInput
	}
	prepared := make([]preparedAuctionResult, 0, len(drafts))
	seen := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		id := strings.TrimSpace(draft.ProcessID)
		process, exists := expected[id]
		if !exists {
			redirectMessage(w, r, "/proses/lelang", "error", "Daftar kontainer tidak sesuai dengan ND penjadwalan yang dipilih.")
			return
		}
		if _, duplicate := seen[id]; duplicate {
			redirectMessage(w, r, "/proses/lelang", "error", "Kontainer yang sama terkirim lebih dari satu kali.")
			return
		}
		seen[id] = struct{}{}
		input := base
		input.AuctionOutcome = strings.TrimSpace(draft.Outcome)
		input.SaleValue = parseMoney(draft.SaleValue)
		if input.AuctionOutcome != "laku" && input.AuctionOutcome != "tidak_laku" || input.AuctionOutcome == "laku" && input.SaleValue <= 0 {
			redirectMessage(w, r, "/proses/lelang", "error", "Tetapkan status laku/tidak laku dan harga jual untuk setiap barang yang laku.")
			return
		}
		if input.AuctionOutcome == "tidak_laku" {
			input.SaleValue = 0
		}
		input.Notes = auctionCompletionNotes(input, strings.TrimSpace(r.FormValue("notes")))
		if validateProcessEvent(process, input, action) != nil {
			redirectMessage(w, r, "/proses/lelang", "error", "Salah satu barang tidak lagi berada pada tahap penjadwalan lelang.")
			return
		}
		prepared = append(prepared, preparedAuctionResult{ID: id, Input: input})
	}
	for _, result := range prepared {
		if _, err := s.store.AddDispositionEvent(r.Context(), result.ID, result.Input); err != nil {
			redirectMessage(w, r, "/proses/lelang", "error", "Sebagian hasil lelang belum dapat disimpan. Periksa kembali status barang.")
			return
		}
	}
	s.writeAudit(r, "process.auction.complete.bulk", "auction_schedule", scheduleNo, "success", map[string]any{"items": len(prepared), "document_no": base.DocumentNo, "document_uploaded": base.DocumentID != ""})
	redirectMessage(w, r, "/proses/lelang", "ok", fmt.Sprintf("Hasil lelang berdasarkan %s berhasil disimpan untuk %d barang.", scheduleNo, len(prepared)))
}

func auctionCompletionNotes(input domain.NewEventInput, userNotes string) string {
	details := []string{"Hasil: " + strings.ToUpper(strings.ReplaceAll(input.AuctionOutcome, "_", " "))}
	if input.SaleValue > 0 {
		details = append(details, "Nilai terjual: Rp "+formatThousands(strconv.FormatInt(input.SaleValue, 10)))
	}
	if userNotes != "" {
		details = append(details, userNotes)
	}
	return strings.Join(details, " · ")
}

func processSourceEligible(item domain.InventoryItem, target domain.DispositionType) bool {
	if item.Type == domain.InventoryTitipan || !item.IsActive {
		return false
	}
	if item.CurrentDisposition == domain.DispositionAuction && item.StatusCode == "tidak_laku" {
		return target == domain.DispositionDestruction || target == domain.DispositionGrant
	}
	if item.CurrentDisposition != "" {
		return false
	}
	return item.StatusCode != "alokasi_hasil_lelang" && item.StatusCode != "ba_musnah" && item.StatusCode != "ba_serah_terima"
}

func processEventInput(r *http.Request, actor string) domain.NewEventInput {
	input := domain.NewEventInput{
		Code: r.FormValue("event_code"), DocumentNo: strings.TrimSpace(r.FormValue("document_no")), DocumentDate: parseDate(r.FormValue("document_date")), Notes: strings.TrimSpace(r.FormValue("notes")), Actor: actor,
		SaleValue: parseMoney(r.FormValue("sale_value")), HTLValue: parseMoney(r.FormValue("htl_value")), ExecutionStartDate: parseDate(r.FormValue("execution_start_date")), ExecutionEndDate: parseDate(r.FormValue("execution_end_date")),
		AuctionOutcome: r.FormValue("auction_outcome"), AllocationTarget: strings.TrimSpace(r.FormValue("allocation_target")), DestructionCost: parseMoney(r.FormValue("destruction_cost")), TransferType: r.FormValue("transfer_type"),
		RecipientCode: strings.TrimSpace(r.FormValue("recipient_code")), RecipientName: strings.TrimSpace(r.FormValue("recipient_name")),
	}
	if input.Code == "jadwal_lelang" && !input.ExecutionStartDate.IsZero() && input.ExecutionEndDate.IsZero() {
		input.ExecutionEndDate = input.ExecutionStartDate
	}
	details := make([]string, 0, 3)
	switch input.Code {
	case "kep_htl":
		if input.HTLValue > 0 {
			details = append(details, "Nilai HTL: Rp "+formatThousands(strconv.FormatInt(input.HTLValue, 10)))
		}
	case "jadwal_lelang":
		details = append(details, "Pelaksanaan: "+input.ExecutionStartDate.Format("02-01-2006")+" s.d. "+input.ExecutionEndDate.Format("02-01-2006"))
	case "selesai_lelang":
		details = append(details, "Hasil: "+strings.ToUpper(strings.ReplaceAll(input.AuctionOutcome, "_", " ")))
		if input.SaleValue > 0 {
			details = append(details, "Nilai terjual: Rp "+formatThousands(strconv.FormatInt(input.SaleValue, 10)))
		}
	case "alokasi_hasil_lelang":
		details = append(details, "Alokasi: "+input.AllocationTarget)
	case "kep_musnah", "ba_musnah":
		details = append(details, "Biaya musnah: Rp "+formatThousands(strconv.FormatInt(input.DestructionCost, 10)))
	case "ba_serah_terima":
		details = append(details, "Jenis: "+strings.ToUpper(input.TransferType))
	}
	if input.Notes != "" {
		details = append(details, input.Notes)
	}
	input.Notes = strings.Join(details, " · ")
	return input
}

func validateProcessEvent(process domain.Disposition, input domain.NewEventInput, action domain.WorkflowAction) error {
	if !process.IsActive || input.DocumentNo == "" || input.DocumentDate.IsZero() {
		return store.ErrInvalidTransition
	}
	if !action.CreatesProcess && action.AllowedStatus != "" && !statusAllowed(action.AllowedStatus, process.StatusCode) {
		return store.ErrInvalidTransition
	}
	switch process.Type {
	case domain.DispositionAuction:
		switch input.Code {
		case "kep_lelang":
			if !action.CreatesProcess {
				return store.ErrInvalidTransition
			}
		case "kep_htl":
			if input.HTLValue <= 0 {
				return store.ErrInvalidTransition
			}
		case "jadwal_lelang":
			if input.ExecutionStartDate.IsZero() || !input.ExecutionEndDate.IsZero() && input.ExecutionEndDate.Before(input.ExecutionStartDate) {
				return store.ErrInvalidTransition
			}
		case "selesai_lelang":
			if input.AuctionOutcome != "laku" && input.AuctionOutcome != "tidak_laku" || input.AuctionOutcome == "laku" && input.SaleValue <= 0 {
				return store.ErrInvalidTransition
			}
		case "lelang_penyesuaian":
			if process.Round >= 99 {
				return store.ErrInvalidTransition
			}
		case "alokasi_hasil_lelang":
			if input.AllocationTarget == "" {
				return store.ErrInvalidTransition
			}
		}
	case domain.DispositionDestruction:
		if input.Code != "kep_musnah" && input.Code != "ba_musnah" || input.DestructionCost <= 0 {
			return store.ErrInvalidTransition
		}
	case domain.DispositionGrant:
		if input.Code != "ba_serah_terima" || !domain.ValidTransferType(input.TransferType) {
			return store.ErrInvalidTransition
		}
	}
	return nil
}

func statusAllowed(allowed, status string) bool {
	for _, candidate := range strings.Split(allowed, ",") {
		if strings.TrimSpace(candidate) == status {
			return true
		}
	}
	return false
}

func (s *Server) reconciliationPage(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	items, err := s.store.ListInventory(r.Context(), domain.InventoryFilter{AllowedTypes: allowedInventoryTypes(session), IncludeInactive: true, Limit: 20000})
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	records, err := s.store.ListReconciliations(r.Context(), 500)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	records = filterReconciliationsForSession(session, records)
	reconciliations, dataCorrections := splitReconciliationRecords(records)
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	if tab != "perubahan-data" {
		tab = "rekonsiliasi"
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Rekonsiliasi", "Sesuaikan inventory aplikasi dengan kondisi fisik dan jaga audit setiap perubahan data barang", "rekonsiliasi")
	data.EligibleItems, data.Reconciliations, data.DataCorrections, data.Facilities = items, reconciliations, dataCorrections, facilities
	data.ReconciliationTab = tab
	data.CanManage = session.Can(domain.PermissionReconciliationManage)
	s.render(w, "reconciliation", data)
}

func (s *Server) reconcileInventory(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	if err := parseRequestForm(r); err != nil {
		redirectMessage(w, r, "/rekonsiliasi", "error", "Form rekonsiliasi tidak valid.")
		return
	}
	kind := strings.TrimSpace(r.FormValue("reconciliation_type"))
	if kind == "data_correction" {
		inventoryID := strings.TrimSpace(r.FormValue("inventory_id"))
		current, err := s.store.GetInventory(r.Context(), inventoryID)
		if err != nil || !sessionCanAccessItem(session, current) {
			redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "error", "Inventory tidak ditemukan atau tidak termasuk cakupan role Anda.")
			return
		}
		var correctedItem domain.InventoryItem
		var eventCorrections []domain.EventCorrection
		var processCorrections []domain.DispositionCorrection
		if json.Unmarshal([]byte(r.FormValue("correction_item_json")), &correctedItem) != nil || correctedItem.ID != current.ID ||
			json.Unmarshal([]byte(r.FormValue("correction_events_json")), &eventCorrections) != nil ||
			json.Unmarshal([]byte(r.FormValue("correction_processes_json")), &processCorrections) != nil {
			redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "error", "Data perubahan tidak valid. Muat ulang data barang lalu coba kembali.")
			return
		}
		if !sessionCanAccessItem(session, correctedItem) {
			redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "error", "Role Anda tidak memiliki akses untuk mengubah jenis inventory tersebut.")
			return
		}
		documentID, documentErr := s.optionalDocument(r, session.DisplayName)
		if documentErr != nil {
			redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "error", friendlyError(documentErr))
			return
		}
		correction := domain.InventoryCorrectionInput{
			InventoryID: current.ID, Reason: strings.TrimSpace(r.FormValue("correction_reason")), Actor: session.DisplayName, DocumentID: documentID,
			Item: correctedItem, Events: eventCorrections, Processes: processCorrections,
		}
		record, adjustedItem, correctionErr := s.store.CorrectInventoryData(r.Context(), correction)
		if correctionErr != nil {
			redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "error", friendlyError(correctionErr))
			return
		}
		s.writeAudit(r, "reconciliation.data_correction", "inventory", adjustedItem.ID, "success", map[string]any{"reconciliation_id": record.ID, "reason": correction.Reason, "events": len(eventCorrections), "processes": len(processCorrections), "document_uploaded": documentID != ""})
		redirectMessage(w, r, "/rekonsiliasi?tab=perubahan-data", "ok", "Perubahan data barang berhasil disimpan dan dicatat pada timeline audit.")
		return
	}
	input := domain.NewReconciliationInput{Type: kind, InventoryID: strings.TrimSpace(r.FormValue("inventory_id")), Notes: strings.TrimSpace(r.FormValue("notes")), Actor: session.DisplayName}
	if kind == "recorded_not_found" {
		item, err := s.store.GetInventory(r.Context(), input.InventoryID)
		if err != nil || !sessionCanAccessItem(session, item) {
			redirectMessage(w, r, "/rekonsiliasi", "error", "Inventory tidak ditemukan atau tidak termasuk cakupan role Anda.")
			return
		}
	}
	if kind == "found_not_recorded" {
		itemType := domain.InventoryType(strings.ToUpper(strings.TrimSpace(r.FormValue("item_type"))))
		if !sessionCanAccessItem(session, domain.InventoryItem{Type: itemType}) {
			redirectMessage(w, r, "/rekonsiliasi", "error", "Role Anda tidak memiliki akses untuk jenis barang tersebut.")
			return
		}
		quantity, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("quantity")), ",", "."), 64)
		volume, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("estimated_volume_m3")), ",", "."), 64)
		requestedStatus := strings.TrimSpace(r.FormValue("initial_status_code"))
		statusCode := requestedStatus
		statusLabels := map[string]string{
			"ditetapkan": "Baru ditetapkan", "pencacahan": "Selesai pencacahan", "request_penelitian_pfpd": "Request Penelitian PFPD", "penelitian_pfpd": "Penelitian PFPD", "bmmn_aktif": "BMMN aktif", "barang_titipan_aktif": "Barang titipan aktif",
			"kep_lelang": "KEP Lelang", "kep_htl": "KEP Harga Terendah Lelang", "jadwal_lelang": "Jadwal lelang", "laku": "Laku", "tidak_laku": "Tidak laku", "alokasi_hasil_lelang": "Alokasi hasil lelang",
			"kep_musnah": "KEP Musnah", "ba_musnah": "BA Musnah", "ba_serah_terima_hibah": "BA Serah Terima HIBAH", "ba_serah_terima_psp": "BA Serah Terima PSP",
		}
		dispositionType := domain.DispositionType("")
		transferType := ""
		switch requestedStatus {
		case "kep_lelang", "kep_htl", "jadwal_lelang", "laku", "tidak_laku", "alokasi_hasil_lelang":
			dispositionType = domain.DispositionAuction
		case "kep_musnah", "ba_musnah":
			dispositionType = domain.DispositionDestruction
		case "ba_serah_terima_hibah":
			dispositionType, transferType, statusCode = domain.DispositionGrant, "hibah", "ba_serah_terima"
		case "ba_serah_terima_psp":
			dispositionType, transferType, statusCode = domain.DispositionGrant, "psp", "ba_serah_terima"
		}
		statusLabel := statusLabels[requestedStatus]
		if statusLabel == "" || itemType == domain.InventoryTitipan && dispositionType != "" || requestedStatus == "bmmn_aktif" && itemType != domain.InventoryBMMN || requestedStatus == "barang_titipan_aktif" && itemType != domain.InventoryTitipan {
			redirectMessage(w, r, "/rekonsiliasi", "error", "Status sebenarnya tidak sesuai dengan jenis inventory yang dipilih.")
			return
		}
		input.NewItem = domain.NewInventoryInput{
			ReferenceNo: strings.TrimSpace(r.FormValue("determination_no")), Type: itemType,
			DeterminationNo: strings.TrimSpace(r.FormValue("determination_no")), DeterminationDate: parseDate(r.FormValue("determination_date")),
			ManifestNo: strings.TrimSpace(r.FormValue("manifest_no")), ManifestDate: parseDate(r.FormValue("manifest_date")), ManifestPosition: strings.TrimSpace(r.FormValue("manifest_position")),
			Category: strings.TrimSpace(r.FormValue("category")), EntrustedCategory: strings.TrimSpace(r.FormValue("entrusted_category")), SourceOffice: strings.TrimSpace(r.FormValue("source_office")),
			Description: strings.TrimSpace(r.FormValue("description")), ItemKind: strings.TrimSpace(r.FormValue("item_kind")), Quantity: quantity, Unit: strings.TrimSpace(r.FormValue("unit")), GoodsValue: parseMoney(r.FormValue("goods_value")),
			OriginWarehouse: strings.TrimSpace(r.FormValue("origin_warehouse")), FacilityID: strings.TrimSpace(r.FormValue("facility_id")), Location: strings.TrimSpace(r.FormValue("location")), AtTPP: true,
			LoadType: strings.ToUpper(strings.TrimSpace(r.FormValue("load_type"))), ContainerNo: strings.TrimSpace(r.FormValue("container_no")), ContainerSize: strings.TrimSpace(r.FormValue("container_size")), EstimatedVolumeM3: volume,
			InitialStatusCode: statusCode, InitialStatusLabel: statusLabel, InitialDispositionType: dispositionType, InitialTransferType: transferType, ReconciliationCreated: true, Actor: session.DisplayName,
		}
	}
	documentID, documentErr := s.optionalDocument(r, session.DisplayName)
	if documentErr != nil {
		redirectMessage(w, r, "/rekonsiliasi", "error", friendlyError(documentErr))
		return
	}
	input.DocumentID = documentID
	record, adjustedItem, err := s.store.ReconcileInventory(r.Context(), input)
	if err != nil {
		redirectMessage(w, r, "/rekonsiliasi", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "reconciliation.create", "reconciliation", record.ID, "success", map[string]any{"type": kind, "inventory_id": adjustedItem.ID, "document_uploaded": documentID != ""})
	redirectMessage(w, r, "/rekonsiliasi", "ok", "Rekonsiliasi berhasil disimpan dan inventory telah disesuaikan.")
}

func buildBTDReportRows(items []domain.InventoryItem) []BTDReportRow {
	filtered := make([]domain.InventoryItem, 0, len(items))
	for _, item := range items {
		if item.Type == domain.InventoryBTD {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if !filtered[i].DeterminationDate.Equal(filtered[j].DeterminationDate) {
			return filtered[i].DeterminationDate.After(filtered[j].DeterminationDate)
		}
		if filtered[i].DeterminationNo != filtered[j].DeterminationNo {
			return filtered[i].DeterminationNo < filtered[j].DeterminationNo
		}
		if filtered[i].ContainerNo != filtered[j].ContainerNo {
			return filtered[i].ContainerNo < filtered[j].ContainerNo
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	type goods struct {
		description string
		itemKind    string
		condition   string
		quantity    float64
		unit        string
	}
	type unitGroup struct {
		number  string
		sizes   []string
		volumes []string
		goods   []goods
	}
	type documentGroup struct {
		no                string
		date              time.Time
		blNos             []string
		blDates           []string
		manifestNos       []string
		manifestDates     []string
		manifestPositions []string
		loadTypes         []string
		originWarehouses  []string
		facilities        []string
		locationStatuses  []string
		owners            []string
		statusLabels      []string
		inventoryStatuses []string
		units             []unitGroup
		unitIndex         map[string]int
		fclCount          int
		lclCount          int
		itemCount         int
		total             int64
	}
	documents := make([]documentGroup, 0)
	docIndex := make(map[string]int)
	for _, item := range filtered {
		docKey := strings.ToUpper(strings.TrimSpace(item.DeterminationNo)) + "|" + item.DeterminationDate.Format("2006-01-02")
		docPos, found := docIndex[docKey]
		if !found {
			docPos = len(documents)
			docIndex[docKey] = docPos
			documents = append(documents, documentGroup{no: item.DeterminationNo, date: item.DeterminationDate, unitIndex: make(map[string]int)})
		}
		doc := &documents[docPos]
		doc.itemCount++
		doc.total += item.GoodsValue
		doc.blNos = appendUniqueReportValue(doc.blNos, item.BLNo)
		if !item.BLDate.IsZero() {
			doc.blDates = appendUniqueReportValue(doc.blDates, item.BLDate.Format("02/01/2006"))
		}
		doc.manifestNos = appendUniqueReportValue(doc.manifestNos, item.ManifestNo)
		if !item.ManifestDate.IsZero() {
			doc.manifestDates = appendUniqueReportValue(doc.manifestDates, item.ManifestDate.Format("02/01/2006"))
		}
		doc.manifestPositions = appendUniqueReportValue(doc.manifestPositions, item.ManifestPosition)
		loadType := strings.ToUpper(strings.TrimSpace(item.LoadType))
		doc.loadTypes = appendUniqueReportValue(doc.loadTypes, loadType)
		doc.originWarehouses = appendUniqueReportValue(doc.originWarehouses, item.OriginWarehouse)
		doc.facilities = appendUniqueReportValue(doc.facilities, item.FacilityName)
		locationStatus := strings.TrimSpace(item.LocationStatus)
		if locationStatus == "" {
			if item.AtTPP {
				locationStatus = "Di TPP"
			} else {
				locationStatus = "Di TPS"
			}
		}
		doc.locationStatuses = appendUniqueReportValue(doc.locationStatuses, locationStatus)
		doc.owners = appendUniqueReportValue(doc.owners, item.OwnerName)
		statusLabel := strings.TrimSpace(item.StatusLabel)
		if statusLabel == "" {
			statusLabel = strings.TrimSpace(item.StatusCode)
		}
		doc.statusLabels = appendUniqueReportValue(doc.statusLabels, statusLabel)
		inventoryStatus := "Selesai"
		if item.IsActive {
			inventoryStatus = "Aktif"
		}
		doc.inventoryStatuses = appendUniqueReportValue(doc.inventoryStatuses, inventoryStatus)

		unitKey := strings.TrimSpace(item.PhysicalUnitID)
		number := "LCL"
		if loadType == "FCL" && strings.TrimSpace(item.ContainerNo) != "" {
			number = strings.TrimSpace(item.ContainerNo)
			unitKey = "FCL|" + nonContainerReportKey(number)
		} else if unitKey == "" {
			unitKey = "LCL|" + docKey
		}
		unitPos, unitFound := doc.unitIndex[unitKey]
		if !unitFound {
			unitPos = len(doc.units)
			doc.unitIndex[unitKey] = unitPos
			if loadType == "FCL" {
				doc.fclCount++
			} else {
				doc.lclCount++
			}
			doc.units = append(doc.units, unitGroup{number: number})
		}
		unit := &doc.units[unitPos]
		unit.sizes = appendUniqueReportValue(unit.sizes, item.ContainerSize)
		if item.EstimatedVolumeM3 > 0 {
			unit.volumes = appendUniqueReportValue(unit.volumes, strconv.FormatFloat(item.EstimatedVolumeM3, 'f', -1, 64)+" m³")
		}
		unit.goods = append(unit.goods, goods{
			description: strings.TrimSpace(item.Description), itemKind: strings.TrimSpace(item.ItemKind),
			condition: strings.TrimSpace(item.GoodsCondition), quantity: item.Quantity, unit: strings.TrimSpace(item.Unit),
		})
	}
	rows := make([]BTDReportRow, 0, len(documents))
	for _, doc := range documents {
		containers := make([]string, 0, len(doc.units))
		goodsGroups := make([]string, 0, len(doc.units))
		for _, unit := range doc.units {
			unitNumber := strings.TrimSpace(unit.number)
			if unitNumber == "" {
				unitNumber = "LCL"
			}
			unitSummary := unitNumber
			if strings.EqualFold(unitNumber, "LCL") && len(unit.volumes) > 0 {
				unitSummary += " (" + strings.Join(unit.volumes, ", ") + ")"
			}
			containers = append(containers, unitSummary)
			parts := make([]string, 0, len(unit.goods))
			for _, line := range unit.goods {
				description := line.description
				if description == "" {
					description = "Uraian tidak tersedia"
				}
				metadata := make([]string, 0, 2)
				metadata = appendUniqueReportValue(metadata, line.itemKind)
				metadata = appendUniqueReportValue(metadata, line.condition)
				if len(metadata) > 0 {
					description += " [" + strings.Join(metadata, "; ") + "]"
				}
				quantity := strconv.FormatFloat(line.quantity, 'f', -1, 64)
				if line.unit != "" {
					quantity += " " + line.unit
				}
				parts = append(parts, description+": "+quantity)
			}
			goodsGroups = append(goodsGroups, unitNumber+"("+strings.Join(parts, ", ")+")")
		}
		rows = append(rows, BTDReportRow{
			DeterminationNo: doc.no, DeterminationDate: doc.date,
			BLNo: joinedReportValues(doc.blNos, "-"), BLDate: joinedReportValues(doc.blDates, "-"), ManifestNo: joinedReportValues(doc.manifestNos, "-"),
			ManifestDate: joinedReportValues(doc.manifestDates, "-"), ManifestPosition: joinedReportValues(doc.manifestPositions, "-"),
			LoadType: joinedReportValues(doc.loadTypes, "-"), OriginWarehouse: joinedReportValues(doc.originWarehouses, "-"),
			FacilityName: joinedReportValues(doc.facilities, "Belum di TPP"), LocationStatus: joinedReportValues(doc.locationStatuses, "-"),
			ContainerSummary: strings.Join(containers, "; "), ContainerCount: doc.fclCount, GoodsSummary: strings.Join(goodsGroups, "; "),
			OwnerName: joinedReportValues(doc.owners, "-"), ItemCount: doc.itemCount, TotalValue: doc.total,
			StatusLabel: joinedReportValues(doc.statusLabels, "-"), InventoryStatus: joinedReportValues(doc.inventoryStatuses, "-"),
		})
	}
	return rows
}

func appendUniqueReportValue(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	return append(values, value)
}

func joinedReportValues(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return strings.Join(values, "; ")
}

func nonContainerReportKey(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToUpper(value) {
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func (s *Server) reports(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.Query().Get("preset")) == "performance" {
		session, _ := sessionFromContext(r.Context())
		from, to := performanceRange(r.URL.Query().Get("date_from"), r.URL.Query().Get("date_to"), time.Now())
		performance, err := s.performanceReport(r.Context(), session, from, to)
		if err != nil {
			s.renderStoreError(w, r, err)
			return
		}
		data := s.baseData(r, "Pelaporan", "Susun, saring, dan ekspor laporan inventory sesuai kebutuhan", "pelaporan")
		data.ReportPerformance = true
		data.Performance = performance
		data.Report = ReportOptions{Preset: "performance", Title: "Performa kinerja", Description: "Jumlah penyelesaian dan rata-rata waktu proses berdasarkan rentang tanggal selesai.", ExportURL: performance.ExportURL}
		s.render(w, "reports", data)
		return
	}
	preset := strings.TrimSpace(r.URL.Query().Get("preset"))
	if preset == "reconciliation" || preset == "data_correction" {
		session, _ := sessionFromContext(r.Context())
		if !session.Can(domain.PermissionReconciliationView) {
			s.forbidden(w, r)
			return
		}
		records, err := s.store.ListReconciliations(r.Context(), 20000)
		if err != nil {
			s.renderStoreError(w, r, err)
			return
		}
		records = filterReconciliationsForSession(session, records)
		reconciliations, dataCorrections := splitReconciliationRecords(records)
		data := s.baseData(r, "Pelaporan", "Susun, saring, dan ekspor laporan inventory sesuai kebutuhan", "pelaporan")
		if preset == "data_correction" {
			changeRows := flattenDataCorrectionRows(dataCorrections)
			pageRows, pagination := paginate(changeRows, r)
			data.DataCorrectionRows, data.Pagination = pageRows, pagination
			data.ReportDataCorrection = true
			data.Report = ReportOptions{Preset: "data_correction", Title: "Rekap perubahan data barang", Description: "Audit rinci data yang diubah beserta nilai sebelum, nilai sesudah, alasan, waktu, dan petugas.", ExportURL: "/pelaporan.csv?preset=data_correction", CSVExportURL: "/pelaporan.csv?preset=data_correction", ExcelExportURL: "/pelaporan.xlsx?preset=data_correction"}
			data.ReportTotal = len(changeRows)
			data.ReportTransactionTotal = len(dataCorrections)
		} else {
			pageRecords, pagination := paginate(reconciliations, r)
			data.Reconciliations, data.Pagination = pageRecords, pagination
			data.ReportReconciliation = true
			data.Report = ReportOptions{Preset: "reconciliation", Title: "Rekap rekonsiliasi", Description: "Penambahan atau pengeluaran inventory berdasarkan perbandingan catatan aplikasi dan kondisi fisik di lapangan.", ExportURL: "/pelaporan.csv?preset=reconciliation", CSVExportURL: "/pelaporan.csv?preset=reconciliation", ExcelExportURL: "/pelaporan.xlsx?preset=reconciliation"}
			data.ReportTotal = len(reconciliations)
		}
		s.render(w, "reports", data)
		return
	}

	if preset == "btd" {
		items, filter, report, err := s.reportItems(r, 20000)
		if err != nil {
			s.renderStoreError(w, r, err)
			return
		}
		facilities, err := s.store.Facilities(r.Context())
		if err != nil {
			s.renderStoreError(w, r, err)
			return
		}
		rows := buildBTDReportRows(items)
		pageRows, pagination := paginate(rows, r)
		data := s.baseData(r, "Pelaporan", "Susun, saring, dan ekspor laporan inventory sesuai kebutuhan", "pelaporan")
		data.ReportBTD = true
		data.BTDReportRows = pageRows
		data.Pagination = pagination
		data.Report = report
		data.Facilities = facilities
		data.FacilityID = filter.FacilityID
		data.Status = filter.Status
		data.ReportTotal = len(rows)
		for _, row := range rows {
			data.ReportTotalValue += row.TotalValue
		}
		s.render(w, "reports", data)
		return
	}

	filter, report := s.reportFilter(r)
	total, err := s.store.CountInventory(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	offset, pageSize, pagination := paginationForTotal(total, r)
	filter.Offset, filter.Limit = offset, pageSize
	items, err := s.store.ListInventory(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	summary, err := s.store.InventorySummary(r.Context(), filter)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Pelaporan", "Susun, saring, dan ekspor laporan inventory sesuai kebutuhan", "pelaporan")
	data.Items, data.Facilities = items, facilities
	data.Pagination = pagination
	data.Report = report
	data.Query, data.FacilityID, data.InventoryType, data.Status, data.Sort = filter.Query, filter.FacilityID, filter.Type, filter.Status, filter.Sort
	data.ReportTotal = summary.Total
	data.ReportTotalValue = summary.TotalValue
	data.ReportAtTPP = summary.AtTPP
	data.ReportActive = summary.Active
	data.ReportClosed = summary.Closed
	s.render(w, "reports", data)
}

func (s *Server) searchPage(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	query := r.URL.Query()
	search := SearchOptions{
		Scope:             strings.TrimSpace(query.Get("scope")),
		DateFrom:          strings.TrimSpace(query.Get("date_from")),
		DateTo:            strings.TrimSpace(query.Get("date_to")),
		ItemKind:          strings.TrimSpace(query.Get("item_kind")),
		GoodsCondition:    strings.TrimSpace(query.Get("goods_condition")),
		Category:          strings.TrimSpace(query.Get("category")),
		AllocationPurpose: strings.TrimSpace(query.Get("allocation_purpose")),
		LocationScope:     strings.TrimSpace(query.Get("location")),
		MinValue:          strings.TrimSpace(query.Get("min_value")),
		MaxValue:          strings.TrimSpace(query.Get("max_value")),
	}
	if search.Scope != "active" && search.Scope != "completed" {
		search.Scope = "all"
	}
	if search.ItemKind != "" && !domain.ValidItemKind(search.ItemKind) {
		search.ItemKind = ""
	}
	if search.GoodsCondition != "" && !domain.ValidGoodsCondition(search.GoodsCondition) {
		search.GoodsCondition = ""
	}
	if search.Category != "" && !domain.ValidBDNCategory(search.Category) {
		search.Category = ""
	}
	if search.LocationScope != "tpp" && search.LocationScope != "tps" {
		search.LocationScope = ""
	}
	if search.AllocationPurpose != "" && !domain.ValidAllocationPurpose(search.AllocationPurpose) {
		search.AllocationPurpose = ""
	}
	filter := domain.InventoryFilter{
		Query: strings.TrimSpace(query.Get("q")), FacilityID: strings.TrimSpace(query.Get("tpp")),
		Type: domain.InventoryType(strings.ToUpper(strings.TrimSpace(query.Get("type")))), Status: strings.TrimSpace(query.Get("status")),
		ItemKind: search.ItemKind, GoodsCondition: search.GoodsCondition, Category: search.Category, AllocationPurpose: search.AllocationPurpose, LocationScope: search.LocationScope,
		DateFrom: parseDate(search.DateFrom), DateTo: parseDate(search.DateTo), Sort: strings.TrimSpace(query.Get("sort")),
		AllowedTypes: allowedInventoryTypes(session),
	}
	filter.MinValue = parseMoney(search.MinValue)
	filter.MaxValue = parseMoney(search.MaxValue)
	if filter.MinValue > 0 && filter.MaxValue > 0 && filter.MinValue > filter.MaxValue {
		filter.MinValue, filter.MaxValue = filter.MaxValue, filter.MinValue
		search.MinValue, search.MaxValue = strconv.FormatInt(filter.MinValue, 10), strconv.FormatInt(filter.MaxValue, 10)
	}
	if filter.Type != "" && filter.Type != domain.InventoryBTD && filter.Type != domain.InventoryBDN && filter.Type != domain.InventoryBMMN && filter.Type != domain.InventoryTitipan {
		filter.Type = ""
	}
	if filter.Sort == "" {
		filter.Sort = "determination_newest"
	}
	if !filter.DateFrom.IsZero() && !filter.DateTo.IsZero() && filter.DateFrom.After(filter.DateTo) {
		filter.DateFrom, filter.DateTo = filter.DateTo, filter.DateFrom
		search.DateFrom, search.DateTo = filter.DateFrom.Format("2006-01-02"), filter.DateTo.Format("2006-01-02")
	}
	switch search.Scope {
	case "all":
		filter.IncludeInactive = true
	case "completed":
		filter.OnlyInactive = true
	}
	performed := len(query) > 0
	var items []domain.InventoryItem
	var pagination PaginationData
	var err error
	if performed {
		total, countErr := s.store.CountInventory(r.Context(), filter)
		if countErr != nil {
			s.renderStoreError(w, r, countErr)
			return
		}
		offset, pageSize, pageData := paginationForTotal(total, r)
		filter.Offset, filter.Limit = offset, pageSize
		pagination = pageData
		items, err = s.store.ListInventory(r.Context(), filter)
		if err != nil {
			s.renderStoreError(w, r, err)
			return
		}
	} else {
		_, _, pagination = paginationForTotal(0, r)
	}
	facilities, err := s.store.Facilities(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Pencarian Detail Barang", "Cari barang aktif maupun selesai dan buka detail lengkap beserta timestamp pengerjaannya", "pencarian")
	data.Items, data.Facilities = items, facilities
	data.Pagination = pagination
	data.Query, data.FacilityID, data.InventoryType, data.Status, data.Sort = filter.Query, filter.FacilityID, filter.Type, filter.Status, filter.Sort
	data.Search, data.SearchPerformed = search, performed
	s.render(w, "search", data)
}

func (s *Server) reportCSV(w http.ResponseWriter, r *http.Request) {
	s.exportReport(w, r, reportExportCSV)
}

func (s *Server) reportXLSX(w http.ResponseWriter, r *http.Request) {
	s.exportReport(w, r, reportExportXLSX)
}

func (s *Server) reportXLS(w http.ResponseWriter, r *http.Request) {
	s.exportReport(w, r, reportExportXLS)
}

func (s *Server) reportFilter(r *http.Request) (domain.InventoryFilter, ReportOptions) {
	session, _ := sessionFromContext(r.Context())
	query := r.URL.Query()
	report := ReportOptions{
		Preset: strings.TrimSpace(query.Get("preset")), Scope: strings.TrimSpace(query.Get("scope")),
		DateFrom: strings.TrimSpace(query.Get("date_from")), DateTo: strings.TrimSpace(query.Get("date_to")),
		Location: strings.TrimSpace(query.Get("location")), ItemKind: strings.TrimSpace(query.Get("item_kind")), GoodsCondition: strings.TrimSpace(query.Get("goods_condition")), Category: strings.TrimSpace(query.Get("category")), AllocationPurpose: strings.TrimSpace(query.Get("allocation_purpose")), MinValue: strings.TrimSpace(query.Get("min_value")),
		MaxValue: strings.TrimSpace(query.Get("max_value")), MinAge: strings.TrimSpace(query.Get("min_age")),
	}
	if report.Scope != "active" && report.Scope != "all" && report.Scope != "completed" {
		report.Scope = ""
	}
	if report.Location != "tpp" && report.Location != "tps" {
		report.Location = ""
	}
	if report.ItemKind != "" && !domain.ValidItemKind(report.ItemKind) {
		report.ItemKind = ""
	}
	if report.GoodsCondition != "" && !domain.ValidGoodsCondition(report.GoodsCondition) {
		report.GoodsCondition = ""
	}
	if report.Category != "" && !domain.ValidBDNCategory(report.Category) {
		report.Category = ""
	}
	if report.AllocationPurpose != "" && !domain.ValidAllocationPurpose(report.AllocationPurpose) {
		report.AllocationPurpose = ""
	}
	filter := domain.InventoryFilter{
		Query: strings.TrimSpace(query.Get("q")), FacilityID: strings.TrimSpace(query.Get("tpp")),
		Type: domain.InventoryType(strings.ToUpper(strings.TrimSpace(query.Get("type")))), Status: strings.TrimSpace(query.Get("status")),
		DateFrom: parseDate(report.DateFrom), DateTo: parseDate(report.DateTo),
		AllowedTypes: allowedInventoryTypes(session),
	}
	if filter.Type != "" && filter.Type != domain.InventoryBTD && filter.Type != domain.InventoryBDN && filter.Type != domain.InventoryBMMN && filter.Type != domain.InventoryTitipan {
		filter.Type = ""
	}
	if !filter.DateFrom.IsZero() && !filter.DateTo.IsZero() && filter.DateFrom.After(filter.DateTo) {
		filter.DateFrom, filter.DateTo = filter.DateTo, filter.DateFrom
		report.DateFrom, report.DateTo = filter.DateFrom.Format("2006-01-02"), filter.DateTo.Format("2006-01-02")
	}
	applyReportPresetDefaults(&filter, &report)
	if report.Scope == "" {
		report.Scope = "active"
	}
	minValue, maxValue := parseMoney(report.MinValue), parseMoney(report.MaxValue)
	if minValue < 0 {
		minValue, report.MinValue = 0, ""
	}
	if maxValue < 0 {
		maxValue, report.MaxValue = 0, ""
	}
	if minValue > 0 && maxValue > 0 && minValue > maxValue {
		minValue, maxValue = maxValue, minValue
		report.MinValue, report.MaxValue = strconv.FormatInt(minValue, 10), strconv.FormatInt(maxValue, 10)
	}
	minAge, _ := strconv.Atoi(report.MinAge)
	if minAge < 0 {
		minAge = 0
	}
	if minAge > 36500 {
		minAge = 36500
	}
	if report.MinAge != "" {
		report.MinAge = strconv.Itoa(minAge)
	}
	filter.ItemKind, filter.GoodsCondition, filter.Category, filter.AllocationPurpose, filter.LocationScope = report.ItemKind, report.GoodsCondition, report.Category, report.AllocationPurpose, report.Location
	filter.MinValue, filter.MaxValue, filter.Preset = minValue, maxValue, report.Preset
	if minAge > 0 {
		filter.AgeBefore = parseDate(time.Now().AddDate(0, 0, -minAge).Format("2006-01-02"))
	}
	switch report.Scope {
	case "all":
		filter.IncludeInactive = true
	case "completed":
		filter.OnlyInactive = true
	}
	report.Title, report.Description = reportPresetCopy(report.Preset)
	report.CSVExportURL = reportExportURL("/pelaporan.csv", filter, report)
	report.ExcelExportURL = reportExportURL("/pelaporan.xlsx", filter, report)
	report.ExportURL = report.CSVExportURL
	return filter, report
}

func (s *Server) reportItems(r *http.Request, limit int) ([]domain.InventoryItem, domain.InventoryFilter, ReportOptions, error) {
	filter, report := s.reportFilter(r)
	filter.Limit = limit
	items, err := s.store.ListInventory(r.Context(), filter)
	if err != nil {
		return nil, filter, report, err
	}
	return items, filter, report, nil
}

func applyReportPresetDefaults(filter *domain.InventoryFilter, report *ReportOptions) {
	switch report.Preset {
	case "active_tpp":
		report.Scope, report.Location, filter.Sort = "active", "tpp", "tpp"
	case "overdue_60":
		report.Scope, report.MinAge, filter.Sort = "active", "60", "oldest"
	case "auction_ready":
		report.Scope, filter.Sort = "active", "value_desc"
	case "at_tps":
		report.Scope, report.Location, filter.Sort = "active", "tps", "oldest"
	case "bmmn_allocation":
		report.Scope, filter.Type, filter.Sort = "active", domain.InventoryBMMN, "oldest"
	case "completed":
		report.Scope, filter.Sort = "completed", "newest"
	case "btd":
		if report.Scope == "" {
			report.Scope = "all"
		}
		filter.Type, filter.Sort = domain.InventoryBTD, "determination_newest"
	default:
		report.Preset = ""
		if filter.Sort == "" {
			filter.Sort = "newest"
		}
	}
}

func matchesReportPreset(item domain.InventoryItem, preset string, now time.Time) bool {
	switch preset {
	case "overdue_60":
		initial := item.StatusCode == "masih_di_tps" || item.StatusCode == "ditetapkan"
		return (item.Type == domain.InventoryBTD || item.Type == domain.InventoryBDN) && item.AgeDays(now) >= 60 && initial
	case "auction_ready":
		if item.GoodsValue <= 0 || item.CurrentDisposition != "" {
			return false
		}
		return item.StatusCode == "penelitian_pfpd" || item.Type == domain.InventoryBMMN
	case "bmmn_allocation":
		return item.Type == domain.InventoryBMMN && item.CurrentDisposition == ""
	default:
		return true
	}
}

func reportPresetCopy(preset string) (string, string) {
	switch preset {
	case "active_tpp":
		return "Barang aktif per TPP", "Daftar barang aktif yang saat ini tersebar dan berada di TPP."
	case "overdue_60":
		return "BTD/BDN 60 hari belum ditindaklanjuti", "Barang BTD atau BDN yang telah berumur sekurangnya 60 hari dan masih pada status penetapan awal."
	case "auction_ready":
		return "Potensi barang siap lelang", "Barang bernilai yang sudah diteliti PFPD atau berstatus BMMN, belum masuk proses, diurutkan dari nilai tertinggi."
	case "at_tps":
		return "Barang aktif masih di TPS", "Daftar barang aktif yang belum dipindahkan dari TPS asal ke TPP."
	case "bmmn_allocation":
		return "BMMN menunggu peruntukan", "Daftar BMMN aktif yang belum masuk proses lelang, musnah, atau hibah/PSP."
	case "completed":
		return "Riwayat barang selesai", "Daftar barang yang telah keluar dari inventory aktif."
	case "btd":
		return "Laporan BTD", "Rekap lengkap per dokumen BTD yang memuat BL, manifest, TPS asal, TPP, nomor dan jumlah kontainer/LCL, rincian barang, nilai, dan status."
	default:
		return "Laporan kustom", "Gabungkan rentang tanggal, status inventory, lokasi, nilai, umur, jenis, dan TPP sesuai kebutuhan."
	}
}

func reportExportURL(path string, filter domain.InventoryFilter, report ReportOptions) string {
	values := url.Values{}
	values.Set("scope", report.Scope)
	for key, value := range map[string]string{
		"preset": report.Preset, "q": filter.Query, "tpp": filter.FacilityID, "type": strings.ToLower(string(filter.Type)),
		"status": filter.Status, "date_from": report.DateFrom, "date_to": report.DateTo,
		"location": report.Location, "item_kind": report.ItemKind, "goods_condition": report.GoodsCondition, "category": report.Category, "allocation_purpose": report.AllocationPurpose,
		"min_value": report.MinValue, "max_value": report.MaxValue, "min_age": report.MinAge,
	} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return path + "?" + values.Encode()
}

func reportFilename(report ReportOptions) string {
	suffix := report.Preset
	if suffix == "" {
		suffix = "kustom"
	}
	return "livira-" + suffix + "-" + time.Now().Format("20060102")
}

func (s *Server) searchInventory(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.EligibleInventory(r.Context(), r.URL.Query().Get("q"), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	session, _ := sessionFromContext(r.Context())
	items = filterItemsForSession(session, items)
	if len(items) > 12 {
		items = items[:12]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) inventoryDetail(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetInventory(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": friendlyError(err)})
		return
	}
	session, _ := sessionFromContext(r.Context())
	if !sessionCanAccessItem(session, item) {
		s.forbidden(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) inventoryTimeline(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetInventory(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": friendlyError(err)})
		return
	}
	session, _ := sessionFromContext(r.Context())
	if !sessionCanAccessItem(session, item) {
		s.forbidden(w, r)
		return
	}
	events, err := s.store.Timeline(r.Context(), item.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	processes, err := s.store.ListDispositions(r.Context(), domain.DispositionFilter{InventoryID: item.ID, IncludeInactiveInventory: true, Limit: 200})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	decorateTimelineAttachments(events)
	writeJSON(w, http.StatusOK, map[string]any{"item": item, "events": events, "processes": processes})
}

func (s *Server) processTimeline(w http.ResponseWriter, r *http.Request) {
	process, err := s.store.GetDisposition(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": friendlyError(err)})
		return
	}
	session, _ := sessionFromContext(r.Context())
	viewPermission, _ := processPermissions(process.Type)
	if !session.Can(viewPermission) || !sessionCanAccessItem(session, process.Inventory) {
		s.forbidden(w, r)
		return
	}
	events, err := s.store.Timeline(r.Context(), process.InventoryID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": friendlyError(err)})
		return
	}
	decorateTimelineAttachments(events)
	writeJSON(w, http.StatusOK, map[string]any{"item": process.Inventory, "process": process, "events": events})
}

func decorateTimelineAttachments(events []domain.TimelineEvent) {
	for eventIndex := range events {
		for attachmentIndex := range events[eventIndex].Attachments {
			attachment := &events[eventIndex].Attachments[attachmentIndex]
			attachment.DownloadURL = "/documents/" + url.PathEscape(attachment.ID) + "/download"
		}
	}
}

func (s *Server) downloadDocument(w http.ResponseWriter, r *http.Request) {
	documentID := r.PathValue("id")
	session, _ := sessionFromContext(r.Context())
	allowed, accessErr := s.documentAllowed(r.Context(), session, documentID)
	if accessErr != nil || !allowed {
		s.writeAudit(r, "document.download", "document", documentID, "denied", nil)
		http.NotFound(w, r)
		return
	}
	document, content, err := s.store.GetDocument(r.Context(), documentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fileName := filepath.Base(strings.TrimSpace(document.FileName))
	if fileName == "." || fileName == "" {
		fileName = "dokumen"
	}
	w.Header().Set("Content-Type", document.MIMEType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.Header().Set("Cache-Control", "no-store")
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": fileName})
	if disposition == "" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	s.writeAudit(r, "document.download", "document", documentID, "success", map[string]any{"file_name": fileName, "size_bytes": len(content)})
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.auth.Session(r)
		if err == nil {
			session, err = s.refreshUserSession(r.Context(), session)
		}
		if err != nil {
			s.auth.ClearSession(w)
			message, loginPath := "sesi login diperlukan", "/login"
			if errors.Is(err, auth.ErrSessionIdle) {
				message, loginPath = "sesi berakhir karena tidak ada aktivitas selama 30 menit", "/login?idle=1"
			}
			if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/session/") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": message})
				return
			}
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}
		if r.URL.Path != "/session/idle-logout" {
			if err := s.auth.TouchSession(w, session); err != nil {
				http.Error(w, "sesi belum dapat diperbarui", http.StatusInternalServerError)
				return
			}
			session.LastActivity = time.Now().Unix()
		}
		ctx := context.WithValue(r.Context(), sessionKey, session)
		if err := s.loadRuntimeParameters(ctx, false); err != nil {
			s.logger.Warn("load dynamic parameters", "error", err)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		if session.Role != "admin" {
			s.forbidden(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requirePermission(permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		if !session.Can(permission) {
			s.forbidden(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAnyPermission(permissions []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionFromContext(r.Context())
		if !session.CanAny(permissions...) {
			s.forbidden(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Anda tidak memiliki hak akses untuk data ini"})
		return
	}
	http.Error(w, "403 — Anda tidak memiliki hak akses ke menu ini.", http.StatusForbidden)
}

func landingPath(session auth.Session) string {
	switch {
	case session.Can(domain.PermissionDashboardView):
		return "/"
	case session.Can(domain.PermissionInventoryView):
		return "/inventory"
	case session.Can(domain.PermissionAuctionView):
		return "/proses/lelang"
	case session.Can(domain.PermissionDestructionView):
		return "/proses/musnah"
	case session.Can(domain.PermissionGrantView):
		return "/proses/hibah"
	case session.Can(domain.PermissionReconciliationView):
		return "/rekonsiliasi"
	case session.Can(domain.PermissionReportsView):
		return "/pelaporan"
	case session.Can(domain.PermissionSearchView):
		return "/pencarian"
	case session.Can(domain.PermissionAdminUsers):
		return "/admin/pendaftaran"
	case session.Can(domain.PermissionAdminRoles):
		return "/admin/roles"
	case session.Can(domain.PermissionAdminParameters):
		return "/admin/parameters"
	default:
		return "/login"
	}
}

func allowedInventoryTypes(session auth.Session) []domain.InventoryType {
	if session.Role == "admin" {
		return nil
	}
	result := make([]domain.InventoryType, 0, 4)
	if session.Can(domain.PermissionInventoryBTD) {
		result = append(result, domain.InventoryBTD)
	}
	if session.Can(domain.PermissionInventoryBDN) {
		result = append(result, domain.InventoryBDN)
	}
	if session.Can(domain.PermissionInventoryBMMN) {
		result = append(result, domain.InventoryBMMN)
	}
	if session.Can(domain.PermissionInventoryTitipan) {
		result = append(result, domain.InventoryTitipan)
	}
	if len(result) == 0 {
		return []domain.InventoryType{"__NONE__"}
	}
	if len(result) == 4 {
		return nil
	}
	return result
}

func sessionCanCreateInventory(session auth.Session, kind domain.InventoryType) bool {
	permission := domain.InventoryCreatePermission(kind)
	if permission == "" || !sessionCanAccessItem(session, domain.InventoryItem{Type: kind}) {
		return false
	}
	return session.Can(domain.PermissionInventoryManage) || session.Can(permission)
}

func sessionCanPerformInventoryAction(session auth.Session, code string) bool {
	permission := domain.InventoryActionPermission(code)
	if permission == "" {
		return false
	}
	return session.Can(domain.PermissionInventoryManage) || session.Can(permission)
}

func permittedInventoryActions(session auth.Session) []domain.WorkflowAction {
	result := make([]domain.WorkflowAction, 0, len(domain.InventoryActions))
	for _, action := range domain.InventoryActions {
		if sessionCanPerformInventoryAction(session, action.Code) {
			result = append(result, action)
		}
	}
	return result
}

func filterReconciliationsForSession(session auth.Session, records []domain.ReconciliationRecord) []domain.ReconciliationRecord {
	if session.Role == "admin" {
		return records
	}
	result := make([]domain.ReconciliationRecord, 0, len(records))
	for _, record := range records {
		if sessionCanAccessItem(session, domain.InventoryItem{Type: record.InventoryType}) {
			result = append(result, record)
		}
	}
	return result
}

func splitReconciliationRecords(records []domain.ReconciliationRecord) ([]domain.ReconciliationRecord, []domain.ReconciliationRecord) {
	reconciliations := make([]domain.ReconciliationRecord, 0, len(records))
	corrections := make([]domain.ReconciliationRecord, 0, len(records))
	for _, record := range records {
		if record.Type == "data_correction" {
			corrections = append(corrections, record)
			continue
		}
		reconciliations = append(reconciliations, record)
	}
	return reconciliations, corrections
}

func flattenDataCorrectionRows(records []domain.ReconciliationRecord) []DataCorrectionReportRow {
	rows := make([]DataCorrectionReportRow, 0)
	for _, record := range records {
		if len(record.ChangeDetails) == 0 {
			rows = append(rows, DataCorrectionReportRow{Record: record, Legacy: true})
			continue
		}
		for _, change := range record.ChangeDetails {
			rows = append(rows, DataCorrectionReportRow{Record: record, Change: change})
		}
	}
	return rows
}

func sessionCanAccessItem(session auth.Session, item domain.InventoryItem) bool {
	allowed := allowedInventoryTypes(session)
	if len(allowed) == 0 {
		return true
	}
	for _, kind := range allowed {
		if item.Type == kind {
			return true
		}
	}
	return false
}

func filterItemsForSession(session auth.Session, items []domain.InventoryItem) []domain.InventoryItem {
	if len(allowedInventoryTypes(session)) == 0 {
		return items
	}
	result := make([]domain.InventoryItem, 0, len(items))
	for _, item := range items {
		if sessionCanAccessItem(session, item) {
			result = append(result, item)
		}
	}
	return result
}

func processPermissions(kind domain.DispositionType) (string, string) {
	switch kind {
	case domain.DispositionAuction:
		return domain.PermissionAuctionView, domain.PermissionAuctionManage
	case domain.DispositionDestruction:
		return domain.PermissionDestructionView, domain.PermissionDestructionManage
	case domain.DispositionGrant:
		return domain.PermissionGrantView, domain.PermissionGrantManage
	default:
		return "", ""
	}
}

func (s *Server) validateCSRF(r *http.Request, session auth.Session) bool {
	if token := strings.TrimSpace(r.Header.Get("X-CSRF-Token")); token != "" {
		return s.auth.ValidateCSRF(session, token)
	}
	if err := parseRequestForm(r); err != nil {
		return false
	}
	return s.auth.ValidateCSRF(session, r.FormValue("_csrf"))
}

func parseRequestForm(r *http.Request) error {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return r.ParseMultipartForm(maxDocumentUploadBytes + (1 << 20))
	}
	return r.ParseForm()
}

func (s *Server) optionalDocument(r *http.Request, actor string) (string, error) {
	file, header, err := r.FormFile("document_file")
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("membaca dokumen: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxDocumentUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("membaca dokumen: %w", err)
	}
	if len(content) == 0 {
		return "", errors.New("dokumen yang dipilih kosong")
	}
	if int64(len(content)) > maxDocumentUploadBytes {
		return "", errors.New("ukuran dokumen maksimal 8 MB")
	}
	mimeType := http.DetectContentType(content)
	allowed := mimeType == "application/pdf" || mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" || mimeType == "image/gif"
	if !allowed {
		return "", errors.New("format dokumen harus PDF, JPG, PNG, WEBP, atau GIF")
	}
	fileName := filepath.Base(strings.TrimSpace(header.Filename))
	if fileName == "." || fileName == "" {
		fileName = "dokumen"
	}
	if len(fileName) > 180 {
		fileName = fileName[:180]
	}
	document, err := s.store.CreateDocument(r.Context(), domain.NewDocumentInput{
		FileName: fileName, MIMEType: mimeType, SizeBytes: int64(len(content)), Content: content, UploadedBy: actor,
	})
	if err != nil {
		return "", err
	}
	return document.ID, nil
}

func sessionFromContext(ctx context.Context) (auth.Session, bool) {
	session, ok := ctx.Value(sessionKey).(auth.Session)
	return session, ok
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; form-action 'self'; object-src 'none'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		if s.cfg.Production() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		s.logger.Info("request", "request_id", requestIDFromContext(r.Context()), "method", r.Method, "path", r.URL.Path, "status", status, "bytes", recorder.bytes, "remote_ip", clientIP(r), "duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) renderStoreError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.Error("data store", "error", err)
	http.Error(w, "Data belum dapat dimuat. Periksa konfigurasi database dan coba kembali.", http.StatusInternalServerError)
}

func processConfig(raw string) (domain.DispositionType, string, string, []domain.WorkflowAction, bool) {
	switch raw {
	case "lelang":
		return domain.DispositionAuction, "Lelang", "lelang", domain.AuctionActions, true
	case "musnah":
		return domain.DispositionDestruction, "Pemusnahan", "pemusnahan", domain.DestructionActions, true
	case "hibah":
		return domain.DispositionGrant, "Hibah / PSP", "hibah/PSP", domain.GrantActions, true
	default:
		return "", "", "", nil, false
	}
}

func processLabel(kind domain.DispositionType) string {
	switch kind {
	case domain.DispositionAuction:
		return "Lelang"
	case domain.DispositionDestruction:
		return "Pemusnahan"
	case domain.DispositionGrant:
		return "Hibah/PSP"
	default:
		return "Belum ditentukan"
	}
}

func parseDate(raw string) time.Time {
	value, _ := time.Parse("2006-01-02", strings.TrimSpace(raw))
	return value
}

func parseMoney(raw string) int64 {
	cleaned := strings.NewReplacer("Rp", "", "rp", "", ".", "", ",", "", " ", "").Replace(raw)
	value, _ := strconv.ParseInt(cleaned, 10, 64)
	return value
}

func redirectMessage(w http.ResponseWriter, r *http.Request, path, key, message string) {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	http.Redirect(w, r, path+separator+key+"="+url.QueryEscape(message), http.StatusSeeOther)
}

func friendlyError(err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "Data tidak ditemukan."
	case errors.Is(err, store.ErrConflict):
		return "Barang sudah memiliki proses lelang, pemusnahan, atau hibah yang aktif."
	case errors.Is(err, store.ErrConcurrentUpdate):
		return "Data telah diubah petugas lain. Muat ulang halaman sebelum menyimpan kembali."
	case errors.Is(err, store.ErrInactiveInventory):
		return "Barang sudah selesai dan tidak lagi berada dalam inventory aktif."
	case errors.Is(err, store.ErrInvalidTransition):
		return "Tahapan yang dipilih tidak sesuai dengan status barang saat ini."
	default:
		return "Perubahan belum dapat disimpan. Silakan periksa data dan coba kembali."
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func initials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "TP"
	}
	result := []rune(strings.ToUpper(parts[0]))
	initial := string(result[0])
	if len(parts) > 1 {
		last := []rune(strings.ToUpper(parts[len(parts)-1]))
		initial += string(last[0])
	}
	return initial
}

func formatThousands(value string) string {
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = strings.TrimPrefix(value, "-")
	}
	for index := len(value) - 3; index > 0; index -= 3 {
		value = value[:index] + "." + value[index:]
	}
	if negative {
		return "-" + value
	}
	return value
}

func csvSafeRow(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		trimmed := strings.TrimLeft(value, " \t\r\n")
		if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
			value = "'" + value
		}
		result[index] = value
	}
	return result
}
