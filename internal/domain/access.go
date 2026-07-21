package domain

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	PermissionDashboardView     = "dashboard.view"
	PermissionDashboardCapacity = "dashboard.capacity.manage"
	PermissionInventoryView     = "inventory.view"

	// PermissionInventoryManage dipertahankan hanya sebagai fallback selama
	// transisi dari role lama. Permission ini tidak lagi ditampilkan pada UI.
	PermissionInventoryManage = "inventory.manage"

	PermissionInventoryCreateBTD     = "inventory.create.btd"
	PermissionInventoryCreateBDN     = "inventory.create.bdn"
	PermissionInventoryCreateTitipan = "inventory.create.titipan"

	PermissionInventoryActionRelocation        = "inventory.action.pemindahan"
	PermissionInventoryActionContainerLoad     = "inventory.action.bongkar_muat"
	PermissionInventoryActionNotification      = "inventory.action.pemberitahuan"
	PermissionInventoryActionCensus            = "inventory.action.pencacahan"
	PermissionInventoryActionResearchRequest   = "inventory.action.request_penelitian_pfpd"
	PermissionInventoryActionResearch          = "inventory.action.penelitian_pfpd"
	PermissionInventoryActionBMMNDetermination = "inventory.action.penetapan_bmmn"
	PermissionInventoryActionBMMNProposal      = "inventory.action.usulan_peruntukan_bmmn"
	PermissionInventoryActionBMMNApproval      = "inventory.action.persetujuan_peruntukan_bmmn"
	PermissionInventoryActionExit              = "inventory.action.pengeluaran_barang"

	PermissionInventoryBTD     = "inventory.type.btd"
	PermissionInventoryBDN     = "inventory.type.bdn"
	PermissionInventoryBMMN    = "inventory.type.bmmn"
	PermissionInventoryTitipan = "inventory.type.titipan"

	PermissionReconciliationView   = "reconciliation.view"
	PermissionReconciliationManage = "reconciliation.manage"
	PermissionAuctionView          = "auction.view"
	PermissionAuctionManage        = "auction.manage"
	PermissionDestructionView      = "destruction.view"
	PermissionDestructionManage    = "destruction.manage"
	PermissionGrantView            = "grant.view"
	PermissionGrantManage          = "grant.manage"
	PermissionReportsView          = "reports.view"
	PermissionSearchView           = "search.view"
	PermissionAdminUsers           = "admin.users"
	PermissionAdminRoles           = "admin.roles"
	PermissionAdminParameters      = "admin.parameters"
)

type PermissionDefinition struct {
	Code        string
	Group       string
	Label       string
	Description string
}

var PermissionDefinitions = []PermissionDefinition{
	{PermissionDashboardView, "Umum", "Lihat dashboard", "Melihat ringkasan inventory dan progres penyelesaian."},
	{PermissionDashboardCapacity, "Umum", "Edit kapasitas YOR/SOR", "Mengubah kapasitas penampungan YOR dan SOR setiap TPP dari dashboard."},
	{PermissionInventoryView, "Inventory", "Lihat inventory", "Melihat daftar dan detail barang."},

	{PermissionInventoryCreateBTD, "Inventory — Input awal", "Pencatatan BTD", "Mencatat BTD baru secara manual maupun melalui upload Excel."},
	{PermissionInventoryCreateBDN, "Inventory — Input awal", "Penetapan BDN", "Mencatat penetapan BDN baru secara manual maupun melalui upload Excel."},
	{PermissionInventoryCreateTitipan, "Inventory — Input awal", "Pemasukan barang titipan", "Mencatat barang titipan kantor atau unit lain secara manual maupun melalui upload Excel."},

	{PermissionInventoryActionRelocation, "Inventory — Action", "Action pemindahan", "Memindahkan barang dari TPS atau TPP asal ke TPP tujuan."},
	{PermissionInventoryActionContainerLoad, "Inventory — Action", "Action bongkar/muat kontainer", "Membongkar, memuat, atau memindahkan penempatan fisik barang FCL/LCL."},
	{PermissionInventoryActionNotification, "Inventory — Action", "Action pemberitahuan", "Mencatat surat pemberitahuan BTD atau BDN."},
	{PermissionInventoryActionCensus, "Inventory — Action", "Action pencacahan", "Mencatat hasil pencacahan per kontainer atau uraian barang."},
	{PermissionInventoryActionResearchRequest, "Inventory — Action", "Action request penelitian PFPD", "Membuat dokumen permintaan penelitian PFPD."},
	{PermissionInventoryActionResearch, "Inventory — Action", "Action penelitian PFPD", "Mencatat HS, lartas, dan nilai barang hasil penelitian PFPD."},
	{PermissionInventoryActionBMMNDetermination, "Inventory — Action", "Action penetapan BMMN", "Mengubah BTD atau BDN menjadi BMMN."},
	{PermissionInventoryActionBMMNProposal, "Inventory — Action", "Action usulan peruntukan BMMN", "Mencatat dokumen usulan peruntukan BMMN."},
	{PermissionInventoryActionBMMNApproval, "Inventory — Action", "Action persetujuan peruntukan BMMN", "Mencatat dokumen persetujuan peruntukan BMMN."},
	{PermissionInventoryActionExit, "Inventory — Action", "Action pengeluaran barang", "Mencatat dokumen pengeluaran dan menutup inventory aktif."},

	{PermissionInventoryBTD, "Cakupan barang", "Akses BTD", "Mengakses data Barang Tidak Dikuasai."},
	{PermissionInventoryBDN, "Cakupan barang", "Akses BDN", "Mengakses data Barang Dikuasai Negara."},
	{PermissionInventoryBMMN, "Cakupan barang", "Akses BMMN", "Mengakses data Barang Milik Negara."},
	{PermissionInventoryTitipan, "Cakupan barang", "Akses barang titipan", "Mengakses inventory barang titipan kantor atau unit lain."},
	{PermissionReconciliationView, "Rekonsiliasi", "Lihat rekonsiliasi", "Melihat daftar dan laporan hasil rekonsiliasi fisik dengan aplikasi."},
	{PermissionReconciliationManage, "Rekonsiliasi", "Kelola rekonsiliasi", "Menambah atau mengeluarkan inventory berdasarkan hasil rekonsiliasi."},
	{PermissionAuctionView, "Lelang", "Lihat lelang", "Melihat dashboard, daftar, dan riwayat lelang."},
	{PermissionAuctionManage, "Lelang", "Kelola lelang", "Memulai dan memperbarui tahapan lelang."},
	{PermissionDestructionView, "Pemusnahan", "Lihat pemusnahan", "Melihat dashboard, daftar, dan riwayat pemusnahan."},
	{PermissionDestructionManage, "Pemusnahan", "Kelola pemusnahan", "Memulai dan memperbarui tahapan pemusnahan."},
	{PermissionGrantView, "Hibah / PSP", "Lihat hibah / PSP", "Melihat dashboard, daftar, dan riwayat hibah atau PSP."},
	{PermissionGrantManage, "Hibah / PSP", "Kelola hibah / PSP", "Memulai dan memperbarui tahapan hibah atau PSP."},
	{PermissionReportsView, "Analitik", "Lihat dan ekspor pelaporan", "Menyusun filter dan mengunduh laporan CSV maupun Excel."},
	{PermissionSearchView, "Analitik", "Pencarian detail barang", "Mencari detail barang dan membuka timestamp pengerjaan."},
}

var inventoryCreatePermissions = map[InventoryType]string{
	InventoryBTD:     PermissionInventoryCreateBTD,
	InventoryBDN:     PermissionInventoryCreateBDN,
	InventoryTitipan: PermissionInventoryCreateTitipan,
}

var inventoryActionPermissions = map[string]string{
	"pemindahan":                  PermissionInventoryActionRelocation,
	"pindah_bongkar_kontainer":    PermissionInventoryActionContainerLoad,
	"pemberitahuan":               PermissionInventoryActionNotification,
	"pencacahan":                  PermissionInventoryActionCensus,
	"request_penelitian_pfpd":     PermissionInventoryActionResearchRequest,
	"penelitian_pfpd":             PermissionInventoryActionResearch,
	"penetapan_bmmn":              PermissionInventoryActionBMMNDetermination,
	"usulan_peruntukan_bmmn":      PermissionInventoryActionBMMNProposal,
	"persetujuan_peruntukan_bmmn": PermissionInventoryActionBMMNApproval,
	"pengeluaran_barang":          PermissionInventoryActionExit,
}

func InventoryCreatePermission(kind InventoryType) string {
	return inventoryCreatePermissions[kind]
}

func InventoryActionPermission(code string) string {
	return inventoryActionPermissions[strings.TrimSpace(code)]
}

func InventoryCreatePermissionCodes() []string {
	return []string{
		PermissionInventoryCreateBTD,
		PermissionInventoryCreateBDN,
		PermissionInventoryCreateTitipan,
	}
}

func InventoryActionPermissionCodes() []string {
	return []string{
		PermissionInventoryActionRelocation,
		PermissionInventoryActionContainerLoad,
		PermissionInventoryActionNotification,
		PermissionInventoryActionCensus,
		PermissionInventoryActionResearchRequest,
		PermissionInventoryActionResearch,
		PermissionInventoryActionBMMNDetermination,
		PermissionInventoryActionBMMNProposal,
		PermissionInventoryActionBMMNApproval,
		PermissionInventoryActionExit,
	}
}

func InventoryManagementPermissionCodes() []string {
	result := []string{PermissionInventoryManage}
	result = append(result, InventoryCreatePermissionCodes()...)
	result = append(result, InventoryActionPermissionCodes()...)
	return result
}

func AllPermissionCodes() []string {
	result := make([]string, 0, len(PermissionDefinitions))
	for _, permission := range PermissionDefinitions {
		result = append(result, permission.Code)
	}
	return result
}

func ValidPermission(code string) bool {
	for _, permission := range PermissionDefinitions {
		if permission.Code == code {
			return true
		}
	}
	return false
}

func NormalizePermissions(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] || !ValidPermission(value) {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	dependencies := map[string][]string{
		PermissionDashboardCapacity:                {PermissionDashboardView},
		PermissionInventoryCreateBTD:               {PermissionInventoryView, PermissionInventoryBTD},
		PermissionInventoryCreateBDN:               {PermissionInventoryView, PermissionInventoryBDN},
		PermissionInventoryCreateTitipan:           {PermissionInventoryView, PermissionInventoryTitipan},
		PermissionInventoryActionRelocation:        {PermissionInventoryView},
		PermissionInventoryActionContainerLoad:     {PermissionInventoryView},
		PermissionInventoryActionNotification:      {PermissionInventoryView},
		PermissionInventoryActionCensus:            {PermissionInventoryView},
		PermissionInventoryActionResearchRequest:   {PermissionInventoryView},
		PermissionInventoryActionResearch:          {PermissionInventoryView},
		PermissionInventoryActionBMMNDetermination: {PermissionInventoryView},
		PermissionInventoryActionBMMNProposal:      {PermissionInventoryView},
		PermissionInventoryActionBMMNApproval:      {PermissionInventoryView},
		PermissionInventoryActionExit:              {PermissionInventoryView},
		PermissionAuctionManage:                    {PermissionAuctionView},
		PermissionDestructionManage:                {PermissionDestructionView},
		PermissionGrantManage:                      {PermissionGrantView},
		PermissionReconciliationManage:             {PermissionReconciliationView},
	}
	for manage, required := range dependencies {
		if !seen[manage] {
			continue
		}
		for _, dependency := range required {
			if seen[dependency] {
				continue
			}
			seen[dependency] = true
			result = append(result, dependency)
		}
	}
	sort.Strings(result)
	return result
}

type RoleProfile struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Permissions   []string  `json:"permissions"`
	Active        bool      `json:"active"`
	System        bool      `json:"system"`
	AssignedUsers int       `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type NewRoleInput struct {
	Name        string
	Description string
	Permissions []string
	Actor       string
}

type UserAccount struct {
	ID              string    `json:"id"`
	AuthUserID      string    `json:"auth_user_id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	EmailVerified   bool      `json:"email_verified"`
	EmailVerifiedAt time.Time `json:"email_verified_at"`
	ApprovalStatus  string    `json:"approval_status"`
	RoleID          string    `json:"role_id"`
	RoleName        string    `json:"role_name"`
	Permissions     []string  `json:"permissions"`
	SessionVersion  int64     `json:"session_version"`
	RejectionReason string    `json:"rejection_reason"`
	ApprovedBy      string    `json:"approved_by"`
	ApprovedAt      time.Time `json:"approved_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type NewUserApplicationInput struct {
	AuthUserID string
	Name       string
	Email      string
}

type ParameterOption struct {
	ID        string    `json:"id"`
	GroupCode string    `json:"group_code"`
	Code      string    `json:"code"`
	Label     string    `json:"label"`
	AppliesTo string    `json:"applies_to"`
	Active    bool      `json:"active"`
	System    bool      `json:"system"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NewParameterInput struct {
	GroupCode string
	Code      string
	Label     string
	AppliesTo string
	SortOrder int
	Actor     string
}

const (
	ParameterBDNCategory       = "bdn_category"
	ParameterItemKind          = "item_kind"
	ParameterGoodsCondition    = "goods_condition"
	ParameterUnit              = "unit"
	ParameterAllocationPurpose = "allocation_purpose"
	ParameterOriginTPS         = "origin_tps"
	ParameterTPP               = "tpp"
	ParameterLoadType          = "load_type"
	ParameterExitType          = "exit_type"
	ParameterTransferType      = "transfer_type"
)

var ParameterGroupCodes = []string{
	ParameterBDNCategory,
	ParameterItemKind,
	ParameterGoodsCondition,
	ParameterUnit,
	ParameterAllocationPurpose,
	ParameterOriginTPS,
	ParameterTPP,
	ParameterLoadType,
	ParameterExitType,
	ParameterTransferType,
}

func ValidParameterGroup(code string) bool {
	for _, candidate := range ParameterGroupCodes {
		if candidate == code {
			return true
		}
	}
	return false
}

var runtimeParameters = struct {
	sync.RWMutex
	loaded             bool
	bdnCategories      []string
	itemKinds          []string
	goodsConditions    []string
	units              []string
	allocationPurposes []string
	originTPS          []string
	loadTypes          []SelectOption
	exitOptions        []SelectOption
	transferTypes      []SelectOption
}{
	bdnCategories:      append([]string(nil), BDNCategoryNames...),
	itemKinds:          append([]string(nil), ItemKindNames...),
	goodsConditions:    append([]string(nil), GoodsConditionNames...),
	units:              append([]string(nil), UnitNames...),
	allocationPurposes: append([]string(nil), AllocationPurposeNames...),
	originTPS:          append([]string(nil), TPSNames...),
	loadTypes:          append([]SelectOption(nil), LoadTypeOptions...),
	exitOptions:        append([]SelectOption(nil), ExitOptions...),
	transferTypes:      append([]SelectOption(nil), TransferTypeOptions...),
}

// SetRuntimeParameters replaces configurable dropdown values used by validation.
// The caller should pass active and inactive rows together. A group that exists
// but has no active row becomes empty; a group that does not yet exist keeps its
// built-in defaults so an older database can still start before migration 009.
func SetRuntimeParameters(options []ParameterOption) {
	present := make(map[string]bool)
	bdn := make([]string, 0)
	itemKinds := make([]string, 0)
	goodsConditions := make([]string, 0)
	units := make([]string, 0)
	allocationPurposes := make([]string, 0)
	originTPS := make([]string, 0)
	loadTypes := make([]SelectOption, 0)
	exits := make([]SelectOption, 0)
	transferTypes := make([]SelectOption, 0)
	for _, option := range options {
		present[option.GroupCode] = true
		if !option.Active {
			continue
		}
		switch option.GroupCode {
		case ParameterBDNCategory:
			bdn = append(bdn, option.Label)
		case ParameterItemKind:
			itemKinds = append(itemKinds, option.Label)
		case ParameterGoodsCondition:
			goodsConditions = append(goodsConditions, option.Label)
		case ParameterUnit:
			units = append(units, option.Label)
		case ParameterAllocationPurpose:
			allocationPurposes = append(allocationPurposes, option.Label)
		case ParameterOriginTPS:
			originTPS = append(originTPS, option.Label)
		case ParameterLoadType:
			loadTypes = append(loadTypes, SelectOption{Code: option.Code, Label: option.Label, Types: option.AppliesTo})
		case ParameterExitType:
			exits = append(exits, SelectOption{Code: option.Code, Label: option.Label, Types: option.AppliesTo})
		case ParameterTransferType:
			transferTypes = append(transferTypes, SelectOption{Code: option.Code, Label: option.Label, Types: option.AppliesTo})
		}
	}
	runtimeParameters.Lock()
	defer runtimeParameters.Unlock()
	if present[ParameterBDNCategory] {
		runtimeParameters.bdnCategories = bdn
	}
	if present[ParameterItemKind] {
		runtimeParameters.itemKinds = itemKinds
	}
	if present[ParameterGoodsCondition] {
		runtimeParameters.goodsConditions = goodsConditions
	}
	if present[ParameterUnit] {
		runtimeParameters.units = units
	}
	if present[ParameterAllocationPurpose] {
		runtimeParameters.allocationPurposes = allocationPurposes
	}
	if present[ParameterOriginTPS] {
		runtimeParameters.originTPS = originTPS
	}
	if present[ParameterLoadType] {
		runtimeParameters.loadTypes = loadTypes
	}
	if present[ParameterExitType] {
		runtimeParameters.exitOptions = exits
	}
	if present[ParameterTransferType] {
		runtimeParameters.transferTypes = transferTypes
	}
	runtimeParameters.loaded = true
}

func CurrentBDNCategories() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.bdnCategories...)
}

func CurrentItemKinds() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.itemKinds...)
}

func CurrentGoodsConditions() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.goodsConditions...)
}

func CurrentUnits() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.units...)
}

func CurrentAllocationPurposes() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.allocationPurposes...)
}

func CurrentOriginTPS() []string {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]string(nil), runtimeParameters.originTPS...)
}

func CurrentLoadTypes() []SelectOption {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]SelectOption(nil), runtimeParameters.loadTypes...)
}

func CurrentExitOptions() []SelectOption {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]SelectOption(nil), runtimeParameters.exitOptions...)
}

func CurrentTransferTypes() []SelectOption {
	runtimeParameters.RLock()
	defer runtimeParameters.RUnlock()
	return append([]SelectOption(nil), runtimeParameters.transferTypes...)
}
