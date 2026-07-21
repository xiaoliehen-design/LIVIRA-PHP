package domain

import (
	"strings"
	"time"
)

type InventoryType string

const (
	InventoryBTD     InventoryType = "BTD"
	InventoryBDN     InventoryType = "BDN"
	InventoryBMMN    InventoryType = "BMMN"
	InventoryTitipan InventoryType = "TITIPAN"
)

type DispositionType string

const (
	DispositionAuction     DispositionType = "lelang"
	DispositionDestruction DispositionType = "musnah"
	DispositionGrant       DispositionType = "hibah"
)

type Facility struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Active       bool    `json:"active"`
	SortOrder    int     `json:"sort_order"`
	YardCapacity float64 `json:"yard_capacity"`
	YardUsed     float64 `json:"yard_used"`
	ShedCapacity float64 `json:"shed_capacity"`
	ShedUsed     float64 `json:"shed_used"`
}

type InventoryItem struct {
	ID                     string          `json:"id"`
	ReferenceNo            string          `json:"reference_no"`
	Type                   InventoryType   `json:"item_type"`
	OriginType             InventoryType   `json:"origin_type"`
	BLNo                   string          `json:"bl_no"`
	BLDate                 time.Time       `json:"bl_date"`
	ManifestNo             string          `json:"manifest_no"`
	ManifestDate           time.Time       `json:"manifest_date"`
	ManifestPosition       string          `json:"manifest_position"`
	DeterminationNo        string          `json:"determination_no"`
	DeterminationDate      time.Time       `json:"determination_date"`
	Category               string          `json:"category"`
	EntrustedCategory      string          `json:"entrusted_category"`
	SourceOffice           string          `json:"source_office"`
	Description            string          `json:"description"`
	ItemKind               string          `json:"item_kind"`
	Quantity               float64         `json:"quantity"`
	QuantityDetail         string          `json:"quantity_detail"`
	Unit                   string          `json:"unit"`
	GoodsValue             int64           `json:"goods_value"`
	GoodsCondition         string          `json:"goods_condition"`
	Location               string          `json:"location"`
	LocationStatus         string          `json:"location_status"`
	AtTPP                  bool            `json:"at_tpp"`
	OwnerName              string          `json:"owner_name"`
	OwnerAddress           string          `json:"owner_address"`
	OriginWarehouse        string          `json:"origin_warehouse"`
	FacilityID             string          `json:"facility_id"`
	FacilityName           string          `json:"facility_name"`
	LoadType               string          `json:"load_type"`
	ContainerNo            string          `json:"container_no"`
	ContainerSize          string          `json:"container_size"`
	EstimatedVolumeM3      float64         `json:"estimated_volume_m3"`
	PhysicalUnitID         string          `json:"physical_unit_id"`
	OccupancyPrimary       bool            `json:"occupancy_primary"`
	PFPDRequired           bool            `json:"pfpd_required"`
	ResearchRequestNo      string          `json:"research_request_no"`
	ResearchRequestDate    time.Time       `json:"research_request_date"`
	HSCode                 string          `json:"hs_code"`
	IsRestricted           bool            `json:"is_restricted"`
	RestrictionRule        string          `json:"restriction_rule"`
	OriginDocumentType     string          `json:"origin_document_type"`
	OriginDocumentNo       string          `json:"origin_document_no"`
	OriginDocumentDate     time.Time       `json:"origin_document_date"`
	AllocationPurpose      string          `json:"allocation_purpose"`
	AllocationProposalType string          `json:"allocation_proposal_type"`
	AllocationProposalNo   string          `json:"allocation_proposal_no"`
	AllocationProposalDate time.Time       `json:"allocation_proposal_date"`
	AllocationApprovalType string          `json:"allocation_approval_type"`
	AllocationApprovalNo   string          `json:"allocation_approval_no"`
	AllocationApprovalDate time.Time       `json:"allocation_approval_date"`
	ExitDocumentNo         string          `json:"exit_document_no"`
	ExitDocumentDate       time.Time       `json:"exit_document_date"`
	ExitType               string          `json:"exit_type"`
	ExitNotes              string          `json:"exit_notes"`
	StatusCode             string          `json:"status_code"`
	StatusLabel            string          `json:"status_label"`
	CurrentDisposition     DispositionType `json:"current_disposition"`
	IsActive               bool            `json:"is_active"`
	CreatedBy              string          `json:"created_by"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

func (i InventoryItem) AgeDays(now time.Time) int {
	start := i.DeterminationDate
	if start.IsZero() {
		start = i.CreatedAt
	}
	if start.IsZero() || now.Before(start) {
		return 0
	}
	return int(now.Sub(start).Hours() / 24)
}

type DocumentAttachment struct {
	ID            string    `json:"id"`
	FileName      string    `json:"file_name"`
	MIMEType      string    `json:"mime_type"`
	SizeBytes     int64     `json:"size_bytes"`
	UploadedBy    string    `json:"uploaded_by"`
	StorageBucket string    `json:"storage_bucket,omitempty"`
	StoragePath   string    `json:"storage_path,omitempty"`
	SHA256        string    `json:"sha256,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	DownloadURL   string    `json:"download_url,omitempty"`
}

type NewDocumentInput struct {
	FileName   string
	MIMEType   string
	SizeBytes  int64
	Content    []byte
	UploadedBy string
}

// DocumentAccess describes the inventory/process owning a document and is used
// by the backend to authorize downloads before reading private file content.
type DocumentAccess struct {
	Inventory       InventoryItem `json:"inventory"`
	DispositionType string        `json:"disposition_type"`
	EventCode       string        `json:"event_code"`
}

type NotificationSummary struct {
	Overdue60Days int `json:"overdue_60_days"`
	ReadyForExit  int `json:"ready_for_exit"`
	BMMNWaiting   int `json:"bmmn_waiting"`
}

type AuditEntry struct {
	ActorSubject string
	ActorName    string
	Action       string
	EntityType   string
	EntityID     string
	Outcome      string
	IPAddress    string
	UserAgent    string
	RequestID    string
	Metadata     map[string]any
}

type TimelineEvent struct {
	ID              string               `json:"id"`
	InventoryID     string               `json:"inventory_id"`
	DispositionID   string               `json:"disposition_id,omitempty"`
	Code            string               `json:"code"`
	Label           string               `json:"label"`
	DocumentNo      string               `json:"document_no,omitempty"`
	DocumentDate    time.Time            `json:"document_date,omitempty"`
	Notes           string               `json:"notes,omitempty"`
	Actor           string               `json:"actor"`
	CreatedAt       time.Time            `json:"created_at"`
	DispositionType string               `json:"disposition_type,omitempty"`
	DocumentID      string               `json:"document_id,omitempty"`
	Attachments     []DocumentAttachment `json:"attachments,omitempty"`
}

type Disposition struct {
	ID                   string          `json:"id"`
	InventoryID          string          `json:"inventory_id"`
	Type                 DispositionType `json:"disposition_type"`
	Round                int             `json:"round"`
	StatusCode           string          `json:"status_code"`
	StatusLabel          string          `json:"status_label"`
	ProposalType         string          `json:"proposal_type"`
	RecipientCode        string          `json:"recipient_code"`
	RecipientName        string          `json:"recipient_name"`
	SaleValue            int64           `json:"sale_value"`
	HTLValue             int64           `json:"htl_value"`
	ExecutionStartDate   time.Time       `json:"execution_start_date"`
	ExecutionEndDate     time.Time       `json:"execution_end_date"`
	ScheduleDocumentNo   string          `json:"schedule_document_no"`
	ScheduleDocumentDate time.Time       `json:"schedule_document_date"`
	AuctionOutcome       string          `json:"auction_outcome"`
	AllocationTarget     string          `json:"allocation_target"`
	DestructionCost      int64           `json:"destruction_cost"`
	TransferType         string          `json:"transfer_type"`
	IsActive             bool            `json:"is_active"`
	CreatedBy            string          `json:"created_by"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	Inventory            InventoryItem   `json:"inventory,omitempty"`
}

type InventoryFilter struct {
	Query             string
	AllowedTypes      []InventoryType
	FacilityID        string
	Type              InventoryType
	Status            string
	ItemKind          string
	GoodsCondition    string
	Category          string
	AllocationPurpose string
	LocationScope     string
	Sort              string
	IncludeInactive   bool
	OnlyInactive      bool
	DateFrom          time.Time
	DateTo            time.Time
	AgeBefore         time.Time
	MinValue          int64
	MaxValue          int64
	Preset            string
	Limit             int
	Offset            int
}

type InventorySummary struct {
	Total      int   `json:"total"`
	TotalValue int64 `json:"total_value"`
	AtTPP      int   `json:"at_tpp"`
	Active     int   `json:"active"`
	Closed     int   `json:"closed"`
}

type DispositionFilter struct {
	InventoryID              string
	Query                    string
	AllowedTypes             []InventoryType
	FacilityID               string
	Type                     DispositionType
	Status                   string
	IncludeStatusCodes       []string
	ExcludeStatusCodes       []string
	Sort                     string
	IncludeInactiveInventory bool
	OnlyInactiveInventory    bool
	Limit                    int
	Offset                   int
}

type NewInventoryInput struct {
	ReferenceNo            string
	Type                   InventoryType
	BLNo                   string
	BLDate                 time.Time
	ManifestNo             string
	ManifestDate           time.Time
	ManifestPosition       string
	DeterminationNo        string
	DeterminationDate      time.Time
	Category               string
	EntrustedCategory      string
	SourceOffice           string
	Description            string
	ItemKind               string
	Quantity               float64
	QuantityDetail         string
	Unit                   string
	GoodsValue             int64
	GoodsCondition         string
	Location               string
	AtTPP                  bool
	OwnerName              string
	OwnerAddress           string
	OriginWarehouse        string
	FacilityID             string
	LoadType               string
	ContainerNo            string
	ContainerSize          string
	EstimatedVolumeM3      float64
	PhysicalUnitID         string
	OccupancyPrimary       bool
	PFPDRequired           bool
	RestrictionRule        string
	InitialStatusCode      string
	InitialStatusLabel     string
	InitialDispositionType DispositionType
	InitialTransferType    string
	ReconciliationCreated  bool
	Actor                  string
	DocumentID             string
}

type InventoryGoodsLine struct {
	InventoryID    string  `json:"inventory_id,omitempty"`
	Description    string  `json:"description"`
	ItemKind       string  `json:"item_kind"`
	GoodsValue     int64   `json:"goods_value,omitempty"`
	Quantity       float64 `json:"quantity"`
	QuantityDetail string  `json:"quantity_detail,omitempty"`
	Unit           string  `json:"unit"`
	GoodsCondition string  `json:"goods_condition"`
}

// InventoryLoadAllocation represents one physical destination created when a
// goods row is moved or unpacked. A single source row may be split into
// multiple FCL containers and/or LCL warehouse lots, while its total quantity
// and value remain conserved.
type InventoryLoadAllocation struct {
	LoadType          string  `json:"load_type"`
	ContainerNo       string  `json:"container_no,omitempty"`
	ContainerSize     string  `json:"container_size,omitempty"`
	EstimatedVolumeM3 float64 `json:"estimated_volume_m3,omitempty"`
	Quantity          float64 `json:"quantity"`
}

type InventoryLoadRelocationInput struct {
	InventoryID  string                    `json:"inventory_id"`
	Allocations  []InventoryLoadAllocation `json:"allocations"`
	DocumentNo   string                    `json:"document_no"`
	DocumentDate time.Time                 `json:"document_date"`
	Notes        string                    `json:"notes"`
	Actor        string                    `json:"actor"`
	DocumentID   string                    `json:"document_id,omitempty"`
}

type NewEventInput struct {
	Code                string
	Label               string
	DocumentNo          string
	DocumentDate        time.Time
	Notes               string
	Actor               string
	SaleValue           int64
	HTLValue            int64
	ExecutionStartDate  time.Time
	ExecutionEndDate    time.Time
	AuctionOutcome      string
	AllocationTarget    string
	DestructionCost     int64
	TransferType        string
	RecipientCode       string
	RecipientName       string
	TargetFacilityID    string
	Description         string
	ItemKind            string
	Quantity            float64
	Unit                string
	PFPDRequired        bool
	ResearchRequestNo   string
	ResearchRequestDate time.Time
	HSCode              string
	RestrictionStatus   string
	IsRestricted        bool
	RestrictionRule     string
	GoodsValue          int64
	GoodsCondition      string
	AllocationPurpose   string
	AllocationType      string
	ExitType            string
	ExitNotes           string
	DocumentID          string
}

type NewDispositionInput struct {
	InventoryID string
	Type        DispositionType
	Actor       string
	Notes       string
}

type Occupancy struct {
	YardCapacity float64 `json:"yard_capacity"`
	YardUsed     float64 `json:"yard_used"`
	ShedCapacity float64 `json:"shed_capacity"`
	ShedUsed     float64 `json:"shed_used"`
}

type FacilityBreakdown struct {
	FacilityID   string  `json:"facility_id"`
	FacilityName string  `json:"facility_name"`
	BTD          int     `json:"btd"`
	BDN          int     `json:"bdn"`
	BMMN         int     `json:"bmmn"`
	Titipan      int     `json:"titipan"`
	Total        int     `json:"total"`
	YardCapacity float64 `json:"yard_capacity"`
	YardUsed     float64 `json:"yard_used"`
	ShedCapacity float64 `json:"shed_capacity"`
	ShedUsed     float64 `json:"shed_used"`
}

type DashboardInventorySummary struct {
	Documents int `json:"documents"`
	FCL       int `json:"fcl"`
	LCL       int `json:"lcl"`
}

type DashboardStats struct {
	ActiveTotal        int                       `json:"active_total"`
	BTDTotal           int                       `json:"btd_total"`
	BDNTotal           int                       `json:"bdn_total"`
	BMMNTotal          int                       `json:"bmmn_total"`
	TitipanTotal       int                       `json:"titipan_total"`
	ActiveSummary      DashboardInventorySummary `json:"active_summary"`
	BTDSummary         DashboardInventorySummary `json:"btd_summary"`
	BDNSummary         DashboardInventorySummary `json:"bdn_summary"`
	BMMNSummary        DashboardInventorySummary `json:"bmmn_summary"`
	TitipanSummary     DashboardInventorySummary `json:"titipan_summary"`
	AuctionActive      int                       `json:"auction_active"`
	DestructionActive  int                       `json:"destruction_active"`
	GrantActive        int                       `json:"grant_active"`
	CompletedThisMonth int                       `json:"completed_this_month"`
	Occupancy          Occupancy                 `json:"occupancy"`
	FacilityBreakdown  []FacilityBreakdown       `json:"facility_breakdown"`
	RecentEvents       []TimelineEvent           `json:"recent_events"`
	AttentionItems     []InventoryItem           `json:"attention_items"`
}

type ProcessChartPoint struct {
	Label      string `json:"label"`
	Count      int    `json:"count"`
	GoodsValue int64  `json:"goods_value"`
	HTLValue   int64  `json:"htl_value"`
	SaleValue  int64  `json:"sale_value"`
	Cost       int64  `json:"cost"`
	Grant      int    `json:"grant"`
	PSP        int    `json:"psp"`
}

type ProcessDashboard struct {
	Year              int                 `json:"year"`
	Active            int                 `json:"active"`
	ThisYear          int                 `json:"this_year"`
	StartedThisYear   int                 `json:"started_this_year"`
	CompletedThisYear int                 `json:"completed_this_year"`
	TotalGoodsValue   int64               `json:"total_goods_value"`
	TotalHTLValue     int64               `json:"total_htl_value"`
	TotalSaleValue    int64               `json:"total_sale_value"`
	TotalCost         int64               `json:"total_cost"`
	TotalGrant        int                 `json:"total_grant"`
	TotalPSP          int                 `json:"total_psp"`
	MaxCount          int                 `json:"max_count"`
	MaxMoney          int64               `json:"max_money"`
	Chart             []ProcessChartPoint `json:"chart"`
}

type ReconciliationChange struct {
	Section  string `json:"section"`
	EntityID string `json:"entity_id,omitempty"`
	Context  string `json:"context,omitempty"`
	Field    string `json:"field"`
	Before   string `json:"before"`
	After    string `json:"after"`
}

type ReconciliationRecord struct {
	ID                  string                 `json:"id"`
	Type                string                 `json:"reconciliation_type"`
	Action              string                 `json:"action"`
	InventoryID         string                 `json:"inventory_id"`
	InventoryReference  string                 `json:"inventory_reference"`
	InventoryType       InventoryType          `json:"inventory_type"`
	PreviousStatusCode  string                 `json:"previous_status_code"`
	PreviousStatusLabel string                 `json:"previous_status_label"`
	ResultStatusCode    string                 `json:"result_status_code"`
	ResultStatusLabel   string                 `json:"result_status_label"`
	CorrectionReason    string                 `json:"correction_reason"`
	ChangeDetails       []ReconciliationChange `json:"change_details"`
	Notes               string                 `json:"notes"`
	Actor               string                 `json:"actor"`
	CreatedAt           time.Time              `json:"created_at"`
}

type NewReconciliationInput struct {
	Type        string
	InventoryID string
	Notes       string
	Actor       string
	DocumentID  string
	NewItem     NewInventoryInput
	Correction  InventoryCorrectionInput
}

type EventCorrection struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	DocumentNo   string    `json:"document_no"`
	DocumentDate time.Time `json:"document_date"`
	Notes        string    `json:"notes"`
}

type DispositionCorrection struct {
	ID                   string    `json:"id"`
	ProposalType         string    `json:"proposal_type"`
	RecipientCode        string    `json:"recipient_code"`
	RecipientName        string    `json:"recipient_name"`
	SaleValue            int64     `json:"sale_value"`
	HTLValue             int64     `json:"htl_value"`
	ExecutionStartDate   time.Time `json:"execution_start_date"`
	ExecutionEndDate     time.Time `json:"execution_end_date"`
	ScheduleDocumentNo   string    `json:"schedule_document_no"`
	ScheduleDocumentDate time.Time `json:"schedule_document_date"`
	AuctionOutcome       string    `json:"auction_outcome"`
	AllocationTarget     string    `json:"allocation_target"`
	DestructionCost      int64     `json:"destruction_cost"`
	TransferType         string    `json:"transfer_type"`
}

type InventoryCorrectionInput struct {
	InventoryID string
	Reason      string
	Actor       string
	DocumentID  string
	Item        InventoryItem
	Events      []EventCorrection
	Processes   []DispositionCorrection
}

type WorkflowAction struct {
	Code           string
	Label          string
	Description    string
	Document       string
	BMMNOnly       bool
	NonBMMNOnly    bool
	CreatesProcess bool
	AllowedStatus  string
}

var InventoryActions = []WorkflowAction{
	{Code: "pemindahan", Label: "Pemindahan", Description: "Pindahkan barang dari TPS atau TPP asal ke TPP tujuan dan perbarui lokasi.", Document: "No. ST/SPRIN/BA"},
	{Code: "pindah_bongkar_kontainer", Label: "Bongkar/Muat Kontainer", Description: "Bongkar barang FCL ke kontainer lain atau ke gudang (LCL), serta muat barang LCL di gudang ke dalam kontainer.", Document: "No. BA/ST pemindahan"},
	{Code: "pemberitahuan", Label: "Pemberitahuan", Description: "Catat surat pemberitahuan BTD atau BDN.", Document: "Nomor surat"},
	{Code: "pencacahan", Label: "Pencacahan", Description: "Periksa dan perbarui uraian, jumlah, satuan, serta jenis barang.", Document: "No. BA Cacah"},
	{Code: "request_penelitian_pfpd", Label: "Request Penelitian PFPD", Description: "Buat dokumen permintaan penelitian kepada PFPD.", Document: "Nomor dokumen request"},
	{Code: "penelitian_pfpd", Label: "Penelitian PFPD", Description: "Catat HS, lartas, keterangan lartas, dan nilai barang berdasarkan request.", Document: "Nomor dokumen penelitian"},
	{Code: "penetapan_bmmn", Label: "Penetapan BMMN", Description: "Ubah BTD atau BDN menjadi BMMN dan pertahankan jenis asalnya.", Document: "No. SKEP BMMN", NonBMMNOnly: true},
	{Code: "usulan_peruntukan_bmmn", Label: "Usulan Peruntukan", Description: "Catat jenis peruntukan serta nomor dan tanggal dokumen usulan.", Document: "No. Dok. Usulan", BMMNOnly: true},
	{Code: "persetujuan_peruntukan_bmmn", Label: "Persetujuan Peruntukan", Description: "Catat jenis peruntukan serta nomor dan tanggal dokumen persetujuan.", Document: "No. Dok. Persetujuan", BMMNOnly: true},
	{Code: "pengeluaran_barang", Label: "Pengeluaran Barang", Description: "Catat dokumen, tanggal, jenis pengeluaran, dan keluarkan barang dari inventory aktif.", Document: "Nomor dokumen pengeluaran"},
}

var AuctionActions = []WorkflowAction{
	{Code: "kep_lelang", Label: "Penerbitan KEP Lelang", Description: "Tetapkan barang inventory yang masuk ke proses lelang.", Document: "No. KEP Lelang", CreatesProcess: true},
	{Code: "kep_htl", Label: "Penerbitan KEP Harga Terendah Lelang", Description: "Tetapkan nilai HTL tanpa mengganti nilai penelitian PFPD.", Document: "No. KEP HTL", AllowedStatus: "kep_lelang"},
	{Code: "jadwal_lelang", Label: "Penjadwalan Lelang", Description: "Catat ND jadwal dan tanggal pelaksanaan tunggal atau rentang tanggal.", Document: "No. ND Jadwal Lelang", AllowedStatus: "kep_htl,lelang_penyesuaian"},
	{Code: "selesai_lelang", Label: "Selesai Lelang", Description: "Catat risalah serta hasil laku atau tidak laku.", Document: "No. Risalah Lelang", AllowedStatus: "jadwal_lelang"},
	{Code: "lelang_penyesuaian", Label: "Lelang Penyesuaian", Description: "Mulai putaran penyesuaian khusus barang berstatus tidak laku, tanpa penetapan HTL baru.", Document: "No. KEP Lelang Penyesuaian", AllowedStatus: "tidak_laku"},
	{Code: "alokasi_hasil_lelang", Label: "Alokasi Hasil Lelang", Description: "Catat KEP dan tujuan alokasi hasil untuk barang yang laku.", Document: "No. KEP Alokasi Hasil Lelang", AllowedStatus: "laku"},
}

var DestructionActions = []WorkflowAction{
	{Code: "kep_musnah", Label: "Penerbitan KEP Musnah", Description: "Catat KEP Musnah, tanggal, biaya, dan barang yang dimusnahkan.", Document: "No. KEP Musnah", CreatesProcess: true},
	{Code: "ba_musnah", Label: "Berita Acara Musnah", Description: "Catat BA Musnah, tanggal, dan biaya pelaksanaan pemusnahan.", Document: "No. BA Musnah", AllowedStatus: "kep_musnah"},
}

var GrantActions = []WorkflowAction{
	{Code: "ba_serah_terima", Label: "Berita Acara Serah Terima", Description: "Catat jenis Hibah/PSP, nomor BA, tanggal, dan barang yang diserahterimakan.", Document: "No. BA Serah Terima", CreatesProcess: true},
}

type SelectOption struct {
	Code  string
	Label string
	Types string
}

var TPSNames = []string{
	"PT Agung Raya",
	"PT Indonesian Air & Marine Supply (Utara)",
	"PT Indonesian Air & Marine Supply (Barat)",
	"PT Pelabuhan Indonesia II (Persero)",
	"PT Indofood Sukses Makmur Tbk",
	"PT Lautan Tirta Transportama",
	"PT Multi Terminal Indonesia (CDC Banda)",
	"PT Pelabuhan Tanjung Priok (Ambon)",
	"PT Pelabuhan Tanjung Priok (101-101U)",
	"PT IPC Terminal Petikemas (Terminal 3)",
	"PT. Pelabuhan Indonesia (Persero) Regional 2 Tanjung Priok",
	"PT Primanata Jasa Persada",
	"PT Wira Mitra Prima",
	"PT Pelabuhan Indonesia II (Persero) (NPCT1)",
	"PT Pesaka Loka Kirana",
	"PT Dharma Kartika Bhakti",
	"PT Inti Mandiri Utama Trans",
	"PT Agung Raya (Barat)",
}

var BDNCategoryNames = []string{
	"Barang Lartas Ps. 53 (4)",
	"Barang/Sarkut yg Ditegah Pejabat BC",
	"Barang/Sarkut yg Ditinggalkan di KP",
	"BKC & Barang Lain yg berasal dari Pelanggar Tidak Dikenal",
	"BKC yg berasal dari pemilik tidak diketahui",
}

var ItemKindNames = []string{
	"Barang Umum",
	"Barang Berbahaya (B3)",
	"Hewan atau Tumbuhan Hidup",
	"Barang Peka Waktu",
	"Barang Berharga",
}

var GoodsConditionNames = []string{
	"Baru",
	"Bekas",
	"Rusak",
	"Segar",
	"Busuk",
}

var AllocationPurposeNames = []string{
	"Lelang",
	"Musnah",
	"Hibah",
	"PSP",
}

var UnitNames = []string{
	"Ampoule", "Bobbin", "Bundle", "Bag", "Bale", "Barrel (petroleum) (458,987 dm3)",
	"Bottle", "Box", "Can", "Coil", "Centimetre", "Crate", "Case", "Carton", "Drum",
	"Dozen", "Gram", "Kilogram", "Litre ( 1 dm3 )", "Milligram", "Millilitre", "Millimetre",
	"Square metre", "Cubic metre", "Metre", "Unpacked or unpackaged", "Number of international units",
	"number of pairs", "Piece", "Pail", "Tray / Tray Pack", "Pallet", "Roll", "Reel", "Sack",
	"Set", "Sheet", "Stick, cigarette", "Metric ton (1000 kg)", "Bulk, liquid", "Yard (0.9144 m)",
}

var LoadTypeOptions = []SelectOption{
	{Code: "FCL", Label: "FCL"},
	{Code: "LCL", Label: "LCL"},
}

var TransferTypeOptions = []SelectOption{
	{Code: "hibah", Label: "Hibah", Types: "BMMN"},
	{Code: "psp", Label: "PSP", Types: "BMMN"},
}

var ExitOptions = []SelectOption{
	{Code: "impor_untuk_dipakai", Label: "IMPOR UTK DIPAKAI", Types: "BTD"},
	{Code: "reekspor", Label: "REEKSPOR", Types: "BTD,BDN"},
	{Code: "batal_ekspor", Label: "BATAL EKSPOR", Types: "BTD"},
	{Code: "ekspor", Label: "EKSPOR", Types: "BTD"},
	{Code: "keluarkan_ke_tpb", Label: "KELUARKAN KE TPB", Types: "BTD"},
	{Code: "lelang", Label: "LELANG", Types: "BTD,BDN,BMMN"},
	{Code: "musnah", Label: "MUSNAH", Types: "BTD,BDN,BMMN"},
	{Code: "psp", Label: "PSP", Types: "BTD,BDN,BMMN"},
	{Code: "hibah", Label: "HIBAH", Types: "BTD,BDN,BMMN"},
	{Code: "diserahkan_ke_aph_lain", Label: "DISERAHKAN KE APH LAIN", Types: "BTD,BDN"},
	{Code: "pembatalan_bdn", Label: "PEMBATALAN BDN", Types: "BDN"},
	{Code: "diserahkan_ke_ppns", Label: "DISERAHKAN KE PPNS", Types: "BDN"},
	{Code: "penghapusan", Label: "PENGHAPUSAN", Types: "BMMN"},
	{Code: "pengeluaran_barang_titipan", Label: "PENGELUARAN BARANG TITIPAN", Types: "TITIPAN"},
}

var EntrustedCategoryNames = []string{"BTD", "BDN", "BMMN", "Tidak Teridentifikasi"}

func ValidEntrustedCategory(value string) bool {
	return contains(EntrustedCategoryNames, value)
}

func InventoryTypeLabel(kind InventoryType) string {
	if kind == InventoryTitipan {
		return "Barang Titipan"
	}
	return string(kind)
}

func ValidBDNCategory(value string) bool {
	return contains(CurrentBDNCategories(), value)
}

func ValidTPS(value string) bool {
	return contains(CurrentOriginTPS(), value)
}

func ValidUnit(value string) bool {
	return contains(CurrentUnits(), value)
}

func ValidItemKind(value string) bool {
	return contains(CurrentItemKinds(), value)
}

func ValidGoodsCondition(value string) bool {
	return contains(CurrentGoodsConditions(), value)
}

func ValidAllocationPurpose(value string) bool {
	return contains(CurrentAllocationPurposes(), value)
}

var ContainerSizeOptions = []SelectOption{
	{Code: "20", Label: "20'"},
	{Code: "40", Label: "40'"},
	{Code: "40HC", Label: "40' HC"},
	{Code: "45HC", Label: "45' HC"},
}

func ValidContainerSize(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "45" { // kompatibilitas data lama sebelum opsi High Cube dibedakan
		return true
	}
	for _, option := range ContainerSizeOptions {
		if option.Code == value {
			return true
		}
	}
	return false
}

func ContainerSizeLabel(size string) string {
	switch strings.ToUpper(strings.TrimSpace(size)) {
	case "20":
		return "20'"
	case "40":
		return "40'"
	case "40HC":
		return "40' HC"
	case "45", "45HC":
		return "45' HC"
	default:
		return "—"
	}
}

func ContainerTEU(size string) float64 {
	switch strings.ToUpper(strings.TrimSpace(size)) {
	case "40", "40HC":
		return 2
	case "45", "45HC":
		return 2.25
	default:
		return 1
	}
}

func InventoryOccupancy(item InventoryItem) (yardTEU, shedM3 float64) {
	if !item.IsActive || !item.AtTPP || item.FacilityID == "" {
		return 0, 0
	}
	if item.PhysicalUnitID != "" && !item.OccupancyPrimary {
		return 0, 0
	}
	if strings.EqualFold(item.LoadType, "FCL") {
		return ContainerTEU(item.ContainerSize), 0
	}
	if strings.EqualFold(item.LoadType, "LCL") && item.EstimatedVolumeM3 > 0 {
		return 0, item.EstimatedVolumeM3
	}
	return 0, 0
}

func InventoryPhysicalUnitKey(item InventoryItem) string {
	if strings.TrimSpace(item.PhysicalUnitID) != "" {
		return strings.TrimSpace(item.PhysicalUnitID)
	}
	if strings.EqualFold(item.LoadType, "FCL") && strings.TrimSpace(item.ContainerNo) != "" {
		return "FCL:" + strings.ToUpper(strings.NewReplacer(" ", "", "-", "", ".", "").Replace(item.ContainerNo))
	}
	if item.ID != "" {
		return "ITEM:" + item.ID
	}
	return "REF:" + item.ReferenceNo
}

func ValidLoadType(code string) bool {
	for _, option := range CurrentLoadTypes() {
		if option.Code == code {
			return true
		}
	}
	return false
}

func ValidTransferType(code string) bool {
	for _, option := range CurrentTransferTypes() {
		if option.Code == code {
			return true
		}
	}
	return false
}

func ValidExitType(kind InventoryType, code string) bool {
	for _, option := range CurrentExitOptions() {
		if option.Code == code {
			for _, allowed := range []InventoryType{InventoryBTD, InventoryBDN, InventoryBMMN, InventoryTitipan} {
				if kind == allowed && containsType(option.Types, allowed) {
					return true
				}
			}
		}
	}
	return false
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsType(types string, kind InventoryType) bool {
	for _, value := range strings.Split(types, ",") {
		if strings.TrimSpace(value) == string(kind) {
			return true
		}
	}
	return false
}

func FindInventoryAction(code string) (WorkflowAction, bool) {
	return findAction(InventoryActions, code)
}

func FindDispositionAction(kind DispositionType, code string) (WorkflowAction, bool) {
	var actions []WorkflowAction
	switch kind {
	case DispositionAuction:
		actions = AuctionActions
	case DispositionDestruction:
		actions = DestructionActions
	case DispositionGrant:
		actions = GrantActions
	default:
		return WorkflowAction{}, false
	}
	return findAction(actions, code)
}

func findAction(actions []WorkflowAction, code string) (WorkflowAction, bool) {
	for _, action := range actions {
		if action.Code == code {
			return action, true
		}
	}
	return WorkflowAction{}, false
}
