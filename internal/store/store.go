package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrConflict          = errors.New("barang sudah memiliki proses aktif")
	ErrInvalidTransition = errors.New("perubahan status tidak valid")
	ErrInactiveInventory = errors.New("barang sudah tidak aktif")
	ErrConcurrentUpdate  = errors.New("data telah diubah pengguna lain")
	ErrRoleInUse         = errors.New("role masih digunakan oleh pengguna")
)

type Store interface {
	Facilities(context.Context) ([]domain.Facility, error)
	UpdateFacilityCapacity(context.Context, string, float64, float64) (domain.Facility, error)
	Dashboard(context.Context) (domain.DashboardStats, error)
	ListInventory(context.Context, domain.InventoryFilter) ([]domain.InventoryItem, error)
	CountInventory(context.Context, domain.InventoryFilter) (int, error)
	InventorySummary(context.Context, domain.InventoryFilter) (domain.InventorySummary, error)
	GetInventory(context.Context, string) (domain.InventoryItem, error)
	CreateInventory(context.Context, domain.NewInventoryInput) (domain.InventoryItem, error)
	CreateInventories(context.Context, []domain.NewInventoryInput) ([]domain.InventoryItem, error)
	DeleteInventory(context.Context, string, string) error
	AddInventoryEvent(context.Context, string, domain.NewEventInput) (domain.InventoryItem, error)
	ApplyInventoryCensus(context.Context, string, []domain.InventoryGoodsLine, domain.NewEventInput) ([]domain.InventoryItem, error)
	RelocateInventoryLoad(context.Context, domain.InventoryLoadRelocationInput) ([]domain.InventoryItem, error)
	Timeline(context.Context, string) ([]domain.TimelineEvent, error)
	ListEvents(context.Context, int) ([]domain.TimelineEvent, error)
	PerformanceSource(context.Context, time.Time, time.Time, []domain.InventoryType) ([]domain.InventoryItem, []domain.TimelineEvent, error)
	EligibleInventory(context.Context, string, int) ([]domain.InventoryItem, error)
	ListDispositions(context.Context, domain.DispositionFilter) ([]domain.Disposition, error)
	CountDispositions(context.Context, domain.DispositionFilter) (int, error)
	ProcessDashboard(context.Context, domain.DispositionType, int, []domain.InventoryType) (domain.ProcessDashboard, error)
	GetDisposition(context.Context, string) (domain.Disposition, error)
	CreateDisposition(context.Context, domain.NewDispositionInput) (domain.Disposition, error)
	AddDispositionEvent(context.Context, string, domain.NewEventInput) (domain.Disposition, error)
	ListReconciliations(context.Context, int) ([]domain.ReconciliationRecord, error)
	ReconcileInventory(context.Context, domain.NewReconciliationInput) (domain.ReconciliationRecord, domain.InventoryItem, error)
	CorrectInventoryData(context.Context, domain.InventoryCorrectionInput) (domain.ReconciliationRecord, domain.InventoryItem, error)

	CreateUserApplication(context.Context, domain.NewUserApplicationInput) (domain.UserAccount, error)
	MarkUserEmailVerified(context.Context, string, string) error
	UserByAuthID(context.Context, string) (domain.UserAccount, error)
	ListUsers(context.Context) ([]domain.UserAccount, error)
	ApproveUser(context.Context, string, string, string) error
	RejectUser(context.Context, string, string, string) error
	UpdateUserRole(context.Context, string, string, string) error
	DeleteUser(context.Context, string) (domain.UserAccount, error)

	ListRoles(context.Context, bool) ([]domain.RoleProfile, error)
	CreateRole(context.Context, domain.NewRoleInput) (domain.RoleProfile, error)
	UpdateRole(context.Context, string, domain.NewRoleInput) (domain.RoleProfile, error)
	SetRoleActive(context.Context, string, bool) error
	DeleteRole(context.Context, string) (domain.RoleProfile, error)

	ParameterOptions(context.Context, string, bool) ([]domain.ParameterOption, error)
	CreateParameter(context.Context, domain.NewParameterInput) (domain.ParameterOption, error)
	UpdateParameter(context.Context, string, domain.NewParameterInput) (domain.ParameterOption, error)
	SetParameterActive(context.Context, string, bool) error

	CreateDocument(context.Context, domain.NewDocumentInput) (domain.DocumentAttachment, error)
	GetDocument(context.Context, string) (domain.DocumentAttachment, []byte, error)
	DocumentAccess(context.Context, string) ([]domain.DocumentAccess, error)
	NotificationSummary(context.Context, []domain.InventoryType) (domain.NotificationSummary, error)
	WriteAudit(context.Context, domain.AuditEntry) error
}

func completedProcessStatus(code string) bool {
	return code == "laku" || code == "alokasi_hasil_lelang" || code == "ba_musnah" || code == "ba_serah_terima"
}

func validInventoryExit(item domain.InventoryItem, exitType string) bool {
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
		// Barang yang sudah memiliki KEP musnah dapat keluar secara fisik lebih
		// dahulu. Proses pemusnahan tetap aktif sampai BA Musnah disimpan.
		return item.StatusCode == "kep_musnah" || item.StatusCode == "ba_musnah" || item.CurrentDisposition == domain.DispositionDestruction
	case "hibah":
		return item.StatusCode == "ba_serah_terima" && strings.Contains(strings.ToUpper(item.StatusLabel), "HIBAH")
	case "psp":
		return item.StatusCode == "ba_serah_terima" && strings.Contains(strings.ToUpper(item.StatusLabel), "PSP")
	default:
		return item.CurrentDisposition == ""
	}
}

func inventoryTypeAllowed(kind domain.InventoryType, allowed []domain.InventoryType) bool {
	for _, candidate := range allowed {
		if kind == candidate {
			return true
		}
	}
	return false
}

func statusCodeIncluded(code string, allowed []string) bool {
	for _, candidate := range allowed {
		if code == candidate {
			return true
		}
	}
	return false
}

func canTransferFailedAuction(item domain.InventoryItem, target domain.DispositionType) bool {
	return item.IsActive && item.CurrentDisposition == domain.DispositionAuction && item.StatusCode == "tidak_laku" && (target == domain.DispositionDestruction || target == domain.DispositionGrant)
}

func validCorrectionReason(reason string) bool {
	return reason == "Kesalahan input" || reason == "Error pada saat pengisian awal"
}

func normalizedContainerNumber(value string) (string, bool) {
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

func correctedInventoryItem(current, proposed domain.InventoryItem) (domain.InventoryItem, error) {
	proposed.ReferenceNo = strings.TrimSpace(proposed.ReferenceNo)
	proposed.BLNo = strings.TrimSpace(proposed.BLNo)
	proposed.ManifestNo = strings.TrimSpace(proposed.ManifestNo)
	proposed.ManifestPosition = strings.TrimSpace(proposed.ManifestPosition)
	proposed.DeterminationNo = strings.TrimSpace(proposed.DeterminationNo)
	proposed.Category = strings.TrimSpace(proposed.Category)
	proposed.EntrustedCategory = strings.TrimSpace(proposed.EntrustedCategory)
	proposed.SourceOffice = strings.TrimSpace(proposed.SourceOffice)
	proposed.Description = strings.TrimSpace(proposed.Description)
	proposed.ItemKind = strings.TrimSpace(proposed.ItemKind)
	proposed.Unit = strings.TrimSpace(proposed.Unit)
	proposed.GoodsCondition = strings.TrimSpace(proposed.GoodsCondition)
	proposed.Location = strings.TrimSpace(proposed.Location)
	proposed.LocationStatus = strings.TrimSpace(proposed.LocationStatus)
	proposed.OwnerName = strings.TrimSpace(proposed.OwnerName)
	proposed.OwnerAddress = strings.TrimSpace(proposed.OwnerAddress)
	proposed.OriginWarehouse = strings.TrimSpace(proposed.OriginWarehouse)
	proposed.FacilityID = strings.TrimSpace(proposed.FacilityID)
	proposed.FacilityName = strings.TrimSpace(proposed.FacilityName)
	proposed.LoadType = strings.ToUpper(strings.TrimSpace(proposed.LoadType))
	proposed.ContainerNo = strings.TrimSpace(proposed.ContainerNo)
	proposed.ContainerSize = strings.ToUpper(strings.TrimSpace(proposed.ContainerSize))
	if proposed.ContainerSize == "45" {
		proposed.ContainerSize = "45HC"
	}
	proposed.PhysicalUnitID = strings.TrimSpace(proposed.PhysicalUnitID)
	proposed.ResearchRequestNo = strings.TrimSpace(proposed.ResearchRequestNo)
	proposed.HSCode = strings.TrimSpace(proposed.HSCode)
	proposed.RestrictionRule = strings.TrimSpace(proposed.RestrictionRule)
	proposed.OriginDocumentType = strings.TrimSpace(proposed.OriginDocumentType)
	proposed.OriginDocumentNo = strings.TrimSpace(proposed.OriginDocumentNo)
	proposed.AllocationPurpose = strings.TrimSpace(proposed.AllocationPurpose)
	proposed.AllocationProposalType = strings.TrimSpace(proposed.AllocationProposalType)
	proposed.AllocationProposalNo = strings.TrimSpace(proposed.AllocationProposalNo)
	proposed.AllocationApprovalType = strings.TrimSpace(proposed.AllocationApprovalType)
	proposed.AllocationApprovalNo = strings.TrimSpace(proposed.AllocationApprovalNo)
	proposed.ExitDocumentNo = strings.TrimSpace(proposed.ExitDocumentNo)
	proposed.ExitType = strings.TrimSpace(proposed.ExitType)
	proposed.ExitNotes = strings.TrimSpace(proposed.ExitNotes)

	if proposed.ReferenceNo == "" || proposed.DeterminationNo == "" || proposed.DeterminationDate.IsZero() || proposed.Description == "" || proposed.ItemKind == "" || proposed.Quantity < 0 || proposed.GoodsValue < 0 || proposed.EstimatedVolumeM3 < 0 {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.Type != domain.InventoryBTD && proposed.Type != domain.InventoryBDN && proposed.Type != domain.InventoryBMMN && proposed.Type != domain.InventoryTitipan {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.OriginType != domain.InventoryBTD && proposed.OriginType != domain.InventoryBDN && proposed.OriginType != domain.InventoryBMMN && proposed.OriginType != domain.InventoryTitipan {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.LoadType != "FCL" && proposed.LoadType != "LCL" {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.LoadType == "FCL" && (proposed.ContainerNo == "" || !domain.ValidContainerSize(proposed.ContainerSize)) {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.LoadType == "LCL" && proposed.EstimatedVolumeM3 <= 0 {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if proposed.Type == domain.InventoryTitipan {
		if proposed.SourceOffice == "" || proposed.EntrustedCategory == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
	} else {
		proposed.EntrustedCategory = ""
		proposed.SourceOffice = ""
	}

	proposed.ID = current.ID
	proposed.StatusCode = current.StatusCode
	proposed.StatusLabel = current.StatusLabel
	proposed.CurrentDisposition = current.CurrentDisposition
	proposed.IsActive = current.IsActive
	proposed.CreatedBy = current.CreatedBy
	proposed.CreatedAt = current.CreatedAt
	proposed.UpdatedAt = current.UpdatedAt
	return proposed, nil
}
