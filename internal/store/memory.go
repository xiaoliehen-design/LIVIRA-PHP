package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

type memoryDocument struct {
	Meta    domain.DocumentAttachment
	Content []byte
}

type MemoryStore struct {
	mu                 sync.RWMutex
	facilities         []domain.Facility
	items              map[string]domain.InventoryItem
	events             map[string][]domain.TimelineEvent
	dispositions       map[string]domain.Disposition
	reconciliations    []domain.ReconciliationRecord
	users              map[string]domain.UserAccount
	roles              map[string]domain.RoleProfile
	parameters         map[string]domain.ParameterOption
	documents          map[string]memoryDocument
	nextUser           int
	nextRole           int
	nextParameter      int
	nextDocument       int
	nextItem           int
	nextEvent          int
	nextProcess        int
	nextReconciliation int
	now                func() time.Time
}

func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		facilities: []domain.Facility{
			{ID: "tpp-transporindo", Name: "TPP Transporindo", Active: true, SortOrder: 10, YardCapacity: 1250, YardUsed: 812, ShedCapacity: 4800, ShedUsed: 3010},
			{ID: "tpp-multi-sejahtera", Name: "TPP Multi Sejahtera", Active: true, SortOrder: 20, YardCapacity: 980, YardUsed: 574, ShedCapacity: 3600, ShedUsed: 2165},
			{ID: "tpp-kbn-marunda", Name: "TPP KBN Marunda", Active: true, SortOrder: 30, YardCapacity: 1450, YardUsed: 963, ShedCapacity: 5200, ShedUsed: 3484},
			{ID: "tpp-graha-segara", Name: "TPP Graha Segara", Active: true, SortOrder: 40, YardCapacity: 1100, YardUsed: 621, ShedCapacity: 4100, ShedUsed: 2398},
		},
		items:              make(map[string]domain.InventoryItem),
		events:             make(map[string][]domain.TimelineEvent),
		dispositions:       make(map[string]domain.Disposition),
		reconciliations:    make([]domain.ReconciliationRecord, 0),
		users:              make(map[string]domain.UserAccount),
		roles:              make(map[string]domain.RoleProfile),
		parameters:         make(map[string]domain.ParameterOption),
		documents:          make(map[string]memoryDocument),
		nextUser:           10,
		nextRole:           10,
		nextParameter:      100,
		nextDocument:       100,
		nextItem:           100,
		nextEvent:          100,
		nextProcess:        100,
		nextReconciliation: 100,
		now:                time.Now,
	}
	m.seed()
	m.seedAccessControl()
	return m
}

func (m *MemoryStore) seed() {
	now := m.now().UTC()
	seeds := []struct {
		id, ref, kind, container, description, owner, facility, statusCode, statusLabel string
		qty                                                                             float64
		unit                                                                            string
		age                                                                             int
	}{
		{"inv-001", "BTD-2026-0041", "BTD", "TCLU 782194-3", "Tekstil dan produk tekstil", "PT Nusantara Tekstil Makmur", "tpp-transporindo", "pencacahan", "Selesai pencacahan", 428, "Bale", 28},
		{"inv-002", "BTD-2026-0047", "BTD", "BMOU 451260-8", "Suku cadang kendaraan bermotor", "PT Artha Mobilindo", "tpp-multi-sejahtera", "pemberitahuan", "Pemberitahuan BTD", 96, "Case", 23},
		{"inv-003", "BMMN-2026-0018", "BMMN", "MSCU 612845-1", "Peralatan elektronik rumah tangga", "PT Elektrindo Jaya", "tpp-kbn-marunda", "proses_lelang", "Proses lelang putaran 1", 615, "Carton", 67},
		{"inv-004", "BDN-2026-0012", "BDN", "CAAU 470193-6", "Mesin pengolah makanan dan komponen", "PT Boga Mesin Indonesia", "tpp-graha-segara", "penelitian_pfpd", "Penelitian PFPD", 8, "Piece", 35},
		{"inv-005", "BTD-2026-0055", "BTD", "TEMU 309827-4", "Bahan kimia industri non-berbahaya", "PT Warna Kimia Persada", "tpp-transporindo", "pemindahan", "Selesai pemindahan", 320, "Drum", 16},
		{"inv-006", "BMMN-2026-0021", "BMMN", "GESU 731904-2", "Produk pangan olahan melewati masa simpan", "PT Sumber Pangan Global", "tpp-multi-sejahtera", "proses_musnah", "Proses pemusnahan", 1240, "Carton", 82},
		{"inv-007", "BDN-2026-0015", "BDN", "FCIU 842650-7", "Komponen pembangkit listrik", "PT Energi Teknik Utama", "tpp-kbn-marunda", "request_penelitian_pfpd", "Request Penelitian PFPD", 21, "Crate", 44},
		{"inv-008", "BMMN-2026-0024", "BMMN", "SEGU 580142-9", "Perangkat jaringan dan server", "PT Data Infrastruktur Asia", "tpp-graha-segara", "tidak_laku", "Tidak laku", 72, "Pallet", 91},
		{"inv-009", "BTD-2026-0062", "BTD", "INKU 294761-0", "Alat kesehatan non-elektromedik", "PT Medika Sarana Nusantara", "", "masih_di_tps", "Masih di TPS", 188, "Carton", 9},
		{"inv-010", "BDN-2026-0019", "BDN", "NLLU 672908-5", "Peralatan konstruksi dan perkakas", "PT Bangun Karya Sentosa", "tpp-multi-sejahtera", "pencacahan", "Selesai pencacahan", 46, "Piece", 31},
		{"inv-011", "BTD-2026-0068", "BTD", "WHLU 538012-7", "Bahan baku alas kaki", "PT Prima Footwear Indonesia", "tpp-kbn-marunda", "pemberitahuan", "Pemberitahuan BTD", 760, "Roll", 14},
		{"inv-012", "BMMN-2026-0027", "BMMN", "TCNU 910245-3", "Komputer dan perangkat pendukung pendidikan", "PT Solusi Digital Cendekia", "tpp-graha-segara", "ba_serah_terima", "BA Serah Terima Hibah", 145, "Piece", 73},
		{"inv-013", "BDN-2026-0023", "BDN", "HLXU 482716-5", "Pompa sentrifugal dan suku cadang", "PT Tirta Rekayasa Mandiri", "tpp-transporindo", "pemindahan", "Selesai pemindahan", 17, "Case", 18},
		{"inv-014", "BTD-2026-0074", "BTD", "MRSU 365190-8", "Mainan anak berbahan plastik", "PT Ceria Niaga Indonesia", "tpp-multi-sejahtera", "penelitian_pfpd", "Penelitian PFPD", 895, "Carton", 37},
		{"inv-015", "BMMN-2026-0030", "BMMN", "OOCU 741528-6", "Furnitur kantor knock-down", "PT Ruang Kerja Sejahtera", "tpp-kbn-marunda", "bmmn_aktif", "BMMN aktif", 110, "Piece", 58},
		{"inv-016", "BDN-2026-0028", "BDN", "TRHU 619340-1", "Katup industri dan fitting baja", "PT Rekatama Industri", "", "masih_di_tps", "Masih di TPS", 64, "Crate", 11},
	}

	for index, seed := range seeds {
		facility := m.facilityByID(seed.facility)
		determined := now.AddDate(0, 0, -seed.age)
		atTPP := seed.facility != ""
		originWarehouse := domain.TPSNames[index%len(domain.TPSNames)]
		location := originWarehouse
		locationStatus := originWarehouse
		if atTPP {
			location = "Blok " + string(rune('A'+index%6))
			locationStatus = facility.Name
		}
		originType := domain.InventoryType(seed.kind)
		if originType == domain.InventoryBMMN {
			if index%2 == 0 {
				originType = domain.InventoryBTD
			} else {
				originType = domain.InventoryBDN
			}
		}
		category := ""
		if originType == domain.InventoryBDN {
			category = domain.BDNCategoryNames[index%len(domain.BDNCategoryNames)]
		}
		item := domain.InventoryItem{
			ID: seed.id, ReferenceNo: seed.ref, Type: domain.InventoryType(seed.kind), OriginType: originType,
			ManifestNo: fmt.Sprintf("BC 1.1-%06d", 11827+index*137), ManifestDate: determined.AddDate(0, 0, -4), ManifestPosition: fmt.Sprintf("%04d", 112+index*9),
			DeterminationNo: fmt.Sprintf("KEP-%03d/KPU.01/2026", 241+index), DeterminationDate: determined,
			Category: category, Description: seed.description, ItemKind: "Barang Umum", Quantity: seed.qty, Unit: seed.unit, GoodsValue: int64(125000000 + index*73500000),
			Location: location, LocationStatus: locationStatus, AtTPP: atTPP, OwnerName: seed.owner, OwnerAddress: "Jakarta, Indonesia",
			OriginWarehouse: originWarehouse, FacilityID: seed.facility, FacilityName: facility.Name,
			LoadType: "FCL", ContainerNo: seed.container, ContainerSize: "20", PhysicalUnitID: seed.id, OccupancyPrimary: true, PFPDRequired: true, StatusCode: seed.statusCode, StatusLabel: seed.statusLabel,
			IsActive: true, CreatedBy: "admin", CreatedAt: determined, UpdatedAt: now.Add(-time.Duration(index) * time.Hour),
		}
		if item.Type == domain.InventoryBMMN {
			item.OriginDocumentType = originDocumentType(originType)
			if originType == domain.InventoryBDN {
				item.OriginDocumentNo = fmt.Sprintf("KEP-BDN-%03d/KPU.01/2026", 180+index)
			} else {
				item.OriginDocumentNo = fmt.Sprintf("BCF 1.5-%03d/KPU.01/2026", 180+index)
			}
			item.OriginDocumentDate = item.DeterminationDate.AddDate(0, 0, -18)
		}
		if seed.statusCode == "request_penelitian_pfpd" || seed.statusCode == "penelitian_pfpd" {
			item.ResearchRequestNo = fmt.Sprintf("ND-REQ-%03d/KPU.01/2026", 80+index)
			item.ResearchRequestDate = determined.AddDate(0, 0, 12)
		}
		if seed.statusCode == "penelitian_pfpd" {
			item.HSCode = fmt.Sprintf("84%02d.10.00", index)
			item.IsRestricted = index%2 == 0
			if item.IsRestricted {
				item.RestrictionRule = "Persetujuan teknis terkait"
			}
		}
		m.items[item.ID] = item
		m.events[item.ID] = []domain.TimelineEvent{
			{ID: fmt.Sprintf("evt-seed-%02d", index+1), InventoryID: item.ID, Code: "ditetapkan", Label: "Ditetapkan sebagai " + seed.kind, DocumentNo: item.DeterminationNo, DocumentDate: determined, Actor: "admin", CreatedAt: determined},
		}
		if seed.statusCode != "ditetapkan" && !strings.HasPrefix(seed.statusCode, "proses_") {
			m.events[item.ID] = append(m.events[item.ID], domain.TimelineEvent{ID: fmt.Sprintf("evt-seed-%02d-b", index+1), InventoryID: item.ID, Code: seed.statusCode, Label: seed.statusLabel, DocumentNo: fmt.Sprintf("ND-%03d/KPU.01/2026", 110+index), Actor: "Petugas PPC IV", CreatedAt: item.UpdatedAt})
		}
	}

	m.seedDisposition("proc-001", "inv-003", domain.DispositionAuction, 1, "jadwal_lelang", "Jadwal lelang ditetapkan", 18)
	m.seedDisposition("proc-002", "inv-006", domain.DispositionDestruction, 1, "kep_musnah", "KEP Musnah diterbitkan", 10)
	m.seedDisposition("proc-003", "inv-008", domain.DispositionAuction, 2, "tidak_laku", "Tidak laku", 7)
	m.seedDisposition("proc-004", "inv-012", domain.DispositionGrant, 1, "ba_serah_terima", "BA Serah Terima Hibah", 12)
}

func (m *MemoryStore) seedDisposition(id, inventoryID string, kind domain.DispositionType, round int, code, label string, age int) {
	now := m.now().UTC()
	item := m.items[inventoryID]
	active := code != "ba_serah_terima" && code != "ba_musnah" && code != "alokasi_hasil_lelang"
	if active {
		item.CurrentDisposition = kind
	} else {
		item.CurrentDisposition = ""
	}
	item.StatusCode = code
	item.StatusLabel = label
	item.UpdatedAt = now.AddDate(0, 0, -age)
	m.items[inventoryID] = item
	process := domain.Disposition{ID: id, InventoryID: inventoryID, Type: kind, Round: round, StatusCode: code, StatusLabel: label, IsActive: active, CreatedBy: "admin", CreatedAt: now.AddDate(0, 0, -(age + 4)), UpdatedAt: now.AddDate(0, 0, -age), Inventory: item}
	if code == "jadwal_lelang" {
		process.HTLValue = item.GoodsValue * 75 / 100
		process.ScheduleDocumentNo = "ND-JADWAL-" + strings.ToUpper(id)
		process.ScheduleDocumentDate = process.UpdatedAt
		process.ExecutionStartDate = process.UpdatedAt.AddDate(0, 0, 7)
	}
	if code == "tidak_laku" {
		process.AuctionOutcome = "tidak_laku"
		process.HTLValue = item.GoodsValue * 75 / 100
	}
	if code == "kep_musnah" {
		process.DestructionCost = 18500000
	}
	if code == "ba_serah_terima" {
		process.TransferType = "hibah"
	}
	m.dispositions[id] = process
	m.events[inventoryID] = append(m.events[inventoryID], domain.TimelineEvent{ID: "evt-" + id, InventoryID: inventoryID, DispositionID: id, DispositionType: string(kind), Code: code, Label: label, Actor: "Petugas PPC IV", CreatedAt: process.UpdatedAt})
}

func (m *MemoryStore) facilityByID(id string) domain.Facility {
	if id == "" {
		return domain.Facility{}
	}
	for _, f := range m.facilities {
		if f.ID == id {
			return f
		}
	}
	return domain.Facility{}
}

func (m *MemoryStore) Facilities(context.Context) ([]domain.Facility, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Facility, 0, len(m.facilities))
	for _, facility := range m.facilities {
		if facility.Active {
			result = append(result, facility)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder == result[j].SortOrder {
			return result[i].Name < result[j].Name
		}
		return result[i].SortOrder < result[j].SortOrder
	})
	return result, nil
}

func (m *MemoryStore) UpdateFacilityCapacity(_ context.Context, id string, yardCapacity, shedCapacity float64) (domain.Facility, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if yardCapacity < 0 || shedCapacity < 0 {
		return domain.Facility{}, ErrInvalidTransition
	}
	for index, facility := range m.facilities {
		if facility.ID != id || !facility.Active {
			continue
		}
		facility.YardCapacity = yardCapacity
		facility.ShedCapacity = shedCapacity
		m.facilities[index] = facility
		return facility, nil
	}
	return domain.Facility{}, ErrNotFound
}

func (m *MemoryStore) Dashboard(ctx context.Context) (domain.DashboardStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := domain.DashboardStats{}
	byFacility := make(map[string]*domain.FacilityBreakdown)
	for _, f := range m.facilities {
		if !f.Active {
			continue
		}
		byFacility[f.ID] = &domain.FacilityBreakdown{FacilityID: f.ID, FacilityName: f.Name, YardCapacity: f.YardCapacity, ShedCapacity: f.ShedCapacity}
		stats.Occupancy.YardCapacity += f.YardCapacity
		stats.Occupancy.ShedCapacity += f.ShedCapacity
	}
	now := m.now()
	activeItems := make([]domain.InventoryItem, 0, len(m.items))
	btdItems := make([]domain.InventoryItem, 0)
	bdnItems := make([]domain.InventoryItem, 0)
	bmmnItems := make([]domain.InventoryItem, 0)
	titipanItems := make([]domain.InventoryItem, 0)
	occupiedUnits := make(map[string]struct{})
	for _, item := range m.items {
		if !item.IsActive {
			continue
		}
		stats.ActiveTotal++
		activeItems = append(activeItems, item)
		breakdown := byFacility[item.FacilityID]
		switch item.Type {
		case domain.InventoryBTD:
			stats.BTDTotal++
			btdItems = append(btdItems, item)
			if breakdown != nil {
				breakdown.BTD++
			}
		case domain.InventoryBDN:
			stats.BDNTotal++
			bdnItems = append(bdnItems, item)
			if breakdown != nil {
				breakdown.BDN++
			}
		case domain.InventoryBMMN:
			stats.BMMNTotal++
			bmmnItems = append(bmmnItems, item)
			if breakdown != nil {
				breakdown.BMMN++
			}
		case domain.InventoryTitipan:
			stats.TitipanTotal++
			titipanItems = append(titipanItems, item)
			if breakdown != nil {
				breakdown.Titipan++
			}
		}
		if breakdown != nil {
			breakdown.Total++
			yardUsed, shedUsed := domain.InventoryOccupancy(item)
			if yardUsed > 0 || shedUsed > 0 {
				unitKey := domain.InventoryPhysicalUnitKey(item)
				if _, counted := occupiedUnits[unitKey]; !counted {
					occupiedUnits[unitKey] = struct{}{}
					breakdown.YardUsed += yardUsed
					breakdown.ShedUsed += shedUsed
					stats.Occupancy.YardUsed += yardUsed
					stats.Occupancy.ShedUsed += shedUsed
				}
			}
		}
		if item.AgeDays(now) >= 45 {
			stats.AttentionItems = append(stats.AttentionItems, item)
		}
	}
	stats.ActiveSummary = domain.SummarizeDashboardInventory(activeItems)
	stats.BTDSummary = domain.SummarizeDashboardInventory(btdItems)
	stats.BDNSummary = domain.SummarizeDashboardInventory(bdnItems)
	stats.BMMNSummary = domain.SummarizeDashboardInventory(bmmnItems)
	stats.TitipanSummary = domain.SummarizeDashboardInventory(titipanItems)
	for _, process := range m.dispositions {
		if process.IsActive {
			switch process.Type {
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
	for _, f := range m.facilities {
		if row := byFacility[f.ID]; row != nil {
			stats.FacilityBreakdown = append(stats.FacilityBreakdown, *row)
		}
	}
	for _, list := range m.events {
		stats.RecentEvents = append(stats.RecentEvents, list...)
	}
	sort.Slice(stats.RecentEvents, func(i, j int) bool { return stats.RecentEvents[i].CreatedAt.After(stats.RecentEvents[j].CreatedAt) })
	if len(stats.RecentEvents) > 6 {
		stats.RecentEvents = stats.RecentEvents[:6]
	}
	sort.Slice(stats.AttentionItems, func(i, j int) bool {
		return stats.AttentionItems[i].DeterminationDate.Before(stats.AttentionItems[j].DeterminationDate)
	})
	if len(stats.AttentionItems) > 5 {
		stats.AttentionItems = stats.AttentionItems[:5]
	}
	return stats, nil
}

func (m *MemoryStore) ListInventory(ctx context.Context, filter domain.InventoryFilter) ([]domain.InventoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]domain.InventoryItem, 0, len(m.items))
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, item := range m.items {
		if filter.OnlyInactive && item.IsActive {
			continue
		}
		if !filter.OnlyInactive && !filter.IncludeInactive && !item.IsActive {
			continue
		}
		if !filter.DateFrom.IsZero() && item.DeterminationDate.Before(filter.DateFrom) {
			continue
		}
		if !filter.DateTo.IsZero() && !item.DeterminationDate.Before(filter.DateTo.AddDate(0, 0, 1)) {
			continue
		}
		if !filter.AgeBefore.IsZero() && !item.DeterminationDate.Before(filter.AgeBefore.AddDate(0, 0, 1)) {
			continue
		}
		if filter.FacilityID != "" && item.FacilityID != filter.FacilityID {
			continue
		}
		if filter.Type != "" && item.Type != filter.Type {
			continue
		}
		if len(filter.AllowedTypes) > 0 && !inventoryTypeAllowed(item.Type, filter.AllowedTypes) {
			continue
		}
		if filter.Status != "" && item.StatusCode != filter.Status {
			continue
		}
		if filter.ItemKind != "" && item.ItemKind != filter.ItemKind || filter.GoodsCondition != "" && item.GoodsCondition != filter.GoodsCondition || filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if filter.AllocationPurpose != "" && !strings.EqualFold(item.AllocationPurpose, filter.AllocationPurpose) {
			continue
		}
		if filter.LocationScope == "tpp" && !item.AtTPP || filter.LocationScope == "tps" && item.AtTPP {
			continue
		}
		if filter.MinValue > 0 && item.GoodsValue < filter.MinValue || filter.MaxValue > 0 && item.GoodsValue > filter.MaxValue {
			continue
		}
		if !inventoryPresetMatch(item, filter.Preset, m.now()) {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(fmt.Sprintf("%+v", item))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		items = append(items, item)
	}
	sortInventory(items, filter.Sort)
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start > len(items) {
		start = len(items)
	}
	end := len(items)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return items[start:end], nil
}

func (m *MemoryStore) CountInventory(ctx context.Context, filter domain.InventoryFilter) (int, error) {
	filter.Offset, filter.Limit = 0, 0
	items, err := m.ListInventory(ctx, filter)
	return len(items), err
}

func (m *MemoryStore) InventorySummary(ctx context.Context, filter domain.InventoryFilter) (domain.InventorySummary, error) {
	filter.Offset, filter.Limit = 0, 0
	items, err := m.ListInventory(ctx, filter)
	if err != nil {
		return domain.InventorySummary{}, err
	}
	var summary domain.InventorySummary
	for _, item := range items {
		summary.Total++
		summary.TotalValue += item.GoodsValue
		if item.AtTPP {
			summary.AtTPP++
		}
		if item.IsActive {
			summary.Active++
		} else {
			summary.Closed++
		}
	}
	return summary, nil
}

func inventoryPresetMatch(item domain.InventoryItem, preset string, now time.Time) bool {
	switch preset {
	case "overdue_60":
		initial := item.StatusCode == "masih_di_tps" || item.StatusCode == "ditetapkan"
		return (item.Type == domain.InventoryBTD || item.Type == domain.InventoryBDN) && item.AgeDays(now) >= 60 && initial
	case "auction_ready":
		return item.GoodsValue > 0 && item.CurrentDisposition == "" && (item.StatusCode == "penelitian_pfpd" || item.Type == domain.InventoryBMMN)
	case "bmmn_allocation":
		return item.Type == domain.InventoryBMMN && item.CurrentDisposition == ""
	default:
		return true
	}
}

func sortInventory(items []domain.InventoryItem, order string) {
	switch order {
	case "oldest":
		sort.Slice(items, func(i, j int) bool { return items[i].DeterminationDate.Before(items[j].DeterminationDate) })
	case "determination_newest":
		sort.Slice(items, func(i, j int) bool { return items[i].DeterminationDate.After(items[j].DeterminationDate) })
	case "container_asc":
		sort.Slice(items, func(i, j int) bool { return items[i].ContainerNo < items[j].ContainerNo })
	case "container_desc":
		sort.Slice(items, func(i, j int) bool { return items[i].ContainerNo > items[j].ContainerNo })
	case "tpp":
		sort.Slice(items, func(i, j int) bool { return items[i].FacilityName < items[j].FacilityName })
	case "value_desc":
		sort.Slice(items, func(i, j int) bool { return items[i].GoodsValue > items[j].GoodsValue })
	case "value_asc":
		sort.Slice(items, func(i, j int) bool { return items[i].GoodsValue < items[j].GoodsValue })
	default:
		sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	}
}

func (m *MemoryStore) GetInventory(ctx context.Context, id string) (domain.InventoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return domain.InventoryItem{}, ErrNotFound
	}
	return item, nil
}

func (m *MemoryStore) DeleteInventory(ctx context.Context, id, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return ErrNotFound
	}
	for processID, process := range m.dispositions {
		if process.InventoryID == id {
			delete(m.dispositions, processID)
		}
	}
	delete(m.events, id)
	delete(m.items, id)
	return nil
}

func (m *MemoryStore) CreateInventory(ctx context.Context, input domain.NewInventoryInput) (domain.InventoryItem, error) {
	items, err := m.CreateInventories(ctx, []domain.NewInventoryInput{input})
	if err != nil {
		return domain.InventoryItem{}, err
	}
	return items[0], nil
}

func (m *MemoryStore) CreateInventories(_ context.Context, inputs []domain.NewInventoryInput) ([]domain.InventoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(inputs) == 0 {
		return nil, ErrInvalidTransition
	}

	normalized := make([]domain.NewInventoryInput, len(inputs))
	seenReferences := make(map[string]struct{}, len(inputs))
	seenContainers := make(map[string]string, len(inputs))
	for index, input := range inputs {
		if strings.TrimSpace(input.LoadType) == "" {
			if strings.TrimSpace(input.ContainerNo) == "" {
				input.LoadType = "LCL"
				if input.EstimatedVolumeM3 <= 0 {
					input.EstimatedVolumeM3 = input.Quantity
				}
			} else {
				input.LoadType = "FCL"
			}
		}
		input.LoadType = strings.ToUpper(strings.TrimSpace(input.LoadType))
		input.ContainerNo = strings.ToUpper(strings.TrimSpace(input.ContainerNo))
		input.ContainerSize = strings.TrimSpace(input.ContainerSize)
		input.ReferenceNo = strings.TrimSpace(input.ReferenceNo)
		input.DeterminationNo = strings.TrimSpace(input.DeterminationNo)
		input.Category = strings.TrimSpace(input.Category)
		input.EntrustedCategory = strings.TrimSpace(input.EntrustedCategory)
		input.SourceOffice = strings.TrimSpace(input.SourceOffice)
		input.OriginWarehouse = strings.TrimSpace(input.OriginWarehouse)
		if input.ReferenceNo == "" {
			input.ReferenceNo = input.DeterminationNo
			if len(inputs) > 1 {
				input.ReferenceNo = fmt.Sprintf("%s/%02d", input.DeterminationNo, index+1)
			}
		}

		validType := input.Type == domain.InventoryBTD || input.Type == domain.InventoryBDN || input.Type == domain.InventoryTitipan || input.ReconciliationCreated && input.Type == domain.InventoryBMMN
		validInitialProcess := input.InitialDispositionType == "" || input.ReconciliationCreated && input.Type != domain.InventoryTitipan && (input.InitialDispositionType == domain.DispositionAuction || input.InitialDispositionType == domain.DispositionDestruction || input.InitialDispositionType == domain.DispositionGrant)
		if !validType || !validInitialProcess || input.DeterminationNo == "" || input.DeterminationDate.IsZero() ||
			strings.TrimSpace(input.Description) == "" || input.Quantity <= 0 ||
			!domain.ValidItemKind(input.ItemKind) || !domain.ValidUnit(input.Unit) || !domain.ValidLoadType(input.LoadType) {
			return nil, ErrInvalidTransition
		}
		switch input.Type {
		case domain.InventoryBTD:
			if input.Category != "" || !domain.ValidTPS(input.OriginWarehouse) {
				return nil, ErrInvalidTransition
			}
		case domain.InventoryBDN:
			if !domain.ValidBDNCategory(input.Category) || !domain.ValidTPS(input.OriginWarehouse) {
				return nil, ErrInvalidTransition
			}
		case domain.InventoryTitipan:
			if !domain.ValidEntrustedCategory(input.EntrustedCategory) || input.SourceOffice == "" {
				return nil, ErrInvalidTransition
			}
		case domain.InventoryBMMN:
			if !input.ReconciliationCreated {
				return nil, ErrInvalidTransition
			}
		}
		if input.AtTPP {
			facility := m.facilityByID(input.FacilityID)
			if facility.ID == "" || !facility.Active {
				return nil, ErrInvalidTransition
			}
		} else if input.Type != domain.InventoryTitipan && strings.TrimSpace(input.OriginWarehouse) == "" && strings.TrimSpace(input.Location) == "" {
			return nil, ErrInvalidTransition
		}
		if input.LoadType == "FCL" {
			if input.ContainerSize == "" && input.ContainerNo != "" {
				input.ContainerSize = "20"
			}
			if input.ContainerNo == "" || !domain.ValidContainerSize(input.ContainerSize) {
				return nil, ErrInvalidTransition
			}
			unitID := strings.TrimSpace(input.PhysicalUnitID)
			if priorUnit, duplicate := seenContainers[input.ContainerNo]; duplicate && (unitID == "" || priorUnit != unitID) {
				return nil, ErrConflict
			}
			seenContainers[input.ContainerNo] = unitID
			input.EstimatedVolumeM3 = 0
		} else if input.LoadType == "LCL" {
			if input.EstimatedVolumeM3 <= 0 {
				return nil, ErrInvalidTransition
			}
			input.ContainerNo = ""
			input.ContainerSize = ""
		}
		if strings.TrimSpace(input.PhysicalUnitID) == "" {
			input.OccupancyPrimary = true
		}
		if _, duplicate := seenReferences[input.ReferenceNo]; duplicate {
			return nil, ErrConflict
		}
		for _, item := range m.items {
			if item.ReferenceNo == input.ReferenceNo || input.ContainerNo != "" && item.IsActive && strings.EqualFold(item.ContainerNo, input.ContainerNo) {
				return nil, ErrConflict
			}
		}
		seenReferences[input.ReferenceNo] = struct{}{}
		normalized[index] = input
	}

	created := make([]domain.InventoryItem, 0, len(normalized))
	for _, input := range normalized {
		m.nextItem++
		id := fmt.Sprintf("inv-%03d", m.nextItem)
		now := m.now().UTC()
		facility := m.facilityByID(input.FacilityID)
		location := strings.TrimSpace(input.Location)
		locationStatus := strings.TrimSpace(input.OriginWarehouse)
		statusCode := "masih_di_tps"
		statusLabel := "Masih di TPS"
		if input.Type == domain.InventoryTitipan {
			locationStatus = strings.TrimSpace(input.SourceOffice)
			statusCode = "barang_titipan_aktif"
			statusLabel = "Barang titipan aktif"
			if location == "" {
				location = input.SourceOffice
			}
		}
		if input.AtTPP {
			locationStatus = facility.Name
			statusCode = "ditetapkan"
			statusLabel = "Ditetapkan sebagai " + domain.InventoryTypeLabel(input.Type)
			if input.Type == domain.InventoryTitipan {
				statusCode = "barang_titipan_aktif"
				statusLabel = "Barang titipan aktif"
			}
			if location == "" {
				location = facility.Name
			}
		} else {
			input.FacilityID = ""
			facility = domain.Facility{}
			if location == "" {
				if input.Type == domain.InventoryTitipan {
					location = input.SourceOffice
				} else {
					location = strings.TrimSpace(input.OriginWarehouse)
				}
			}
		}
		if input.InitialStatusCode != "" {
			statusCode = strings.TrimSpace(input.InitialStatusCode)
			statusLabel = strings.TrimSpace(input.InitialStatusLabel)
			if statusLabel == "" {
				statusLabel = statusCode
			}
		}
		item := domain.InventoryItem{
			ID: id, ReferenceNo: input.ReferenceNo, Type: input.Type, OriginType: input.Type,
			BLNo: input.BLNo, BLDate: input.BLDate, ManifestNo: input.ManifestNo, ManifestDate: input.ManifestDate, ManifestPosition: input.ManifestPosition,
			DeterminationNo: input.DeterminationNo, DeterminationDate: input.DeterminationDate, Category: input.Category,
			EntrustedCategory: input.EntrustedCategory, SourceOffice: input.SourceOffice,
			Description: input.Description, ItemKind: input.ItemKind, Quantity: input.Quantity, QuantityDetail: input.QuantityDetail, Unit: input.Unit, GoodsValue: input.GoodsValue, GoodsCondition: input.GoodsCondition,
			Location: location, LocationStatus: locationStatus, AtTPP: input.AtTPP, OwnerName: input.OwnerName, OwnerAddress: input.OwnerAddress,
			OriginWarehouse: input.OriginWarehouse, FacilityID: input.FacilityID, FacilityName: facility.Name,
			LoadType: input.LoadType, ContainerNo: input.ContainerNo, ContainerSize: input.ContainerSize, EstimatedVolumeM3: input.EstimatedVolumeM3,
			PhysicalUnitID: input.PhysicalUnitID, OccupancyPrimary: input.OccupancyPrimary, PFPDRequired: input.PFPDRequired,
			RestrictionRule: input.RestrictionRule, StatusCode: statusCode, StatusLabel: statusLabel,
			IsActive: true, CreatedBy: input.Actor, CreatedAt: now, UpdatedAt: now,
		}
		if item.PhysicalUnitID == "" {
			item.PhysicalUnitID = id
			item.OccupancyPrimary = true
		}
		if !input.PFPDRequired && input.Type != domain.InventoryTitipan {
			item.PFPDRequired = true
		}
		dispositionID := ""
		dispositionType := ""
		if input.InitialDispositionType != "" {
			m.nextProcess++
			dispositionID = fmt.Sprintf("proc-%03d", m.nextProcess)
			active := input.InitialStatusCode != "alokasi_hasil_lelang" && input.InitialStatusCode != "ba_musnah" && input.InitialStatusCode != "ba_serah_terima"
			process := domain.Disposition{
				ID: dispositionID, InventoryID: id, Type: input.InitialDispositionType, Round: 1,
				StatusCode: item.StatusCode, StatusLabel: item.StatusLabel, TransferType: input.InitialTransferType,
				IsActive: active, CreatedBy: input.Actor, CreatedAt: now, UpdatedAt: now,
			}
			if input.InitialStatusCode == "jadwal_lelang" {
				process.ScheduleDocumentNo = input.DeterminationNo
				process.ScheduleDocumentDate = input.DeterminationDate
			}
			if active {
				item.CurrentDisposition = input.InitialDispositionType
			}
			process.Inventory = item
			m.dispositions[dispositionID] = process
			dispositionType = string(input.InitialDispositionType)
		}
		m.items[id] = item
		eventNotes := ""
		if input.ReconciliationCreated {
			eventNotes = "Inventory ditambahkan melalui rekonsiliasi kondisi fisik."
		}
		m.nextEvent++
		m.events[id] = []domain.TimelineEvent{{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: id, DispositionID: dispositionID, DispositionType: dispositionType, Code: item.StatusCode, Label: item.StatusLabel, DocumentNo: item.DeterminationNo, DocumentDate: item.DeterminationDate, Notes: eventNotes, Actor: input.Actor, CreatedAt: now, Attachments: m.documentAttachments(input.DocumentID)}}
		created = append(created, item)
	}
	return created, nil
}

func (m *MemoryStore) AddInventoryEvent(ctx context.Context, id string, input domain.NewEventInput) (domain.InventoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[id]
	if !ok {
		return domain.InventoryItem{}, ErrNotFound
	}
	if !item.IsActive {
		return domain.InventoryItem{}, ErrInactiveInventory
	}
	action, valid := domain.FindInventoryAction(input.Code)
	if !valid || item.CurrentDisposition != "" && input.Code != "pengeluaran_barang" || completedProcessStatus(item.StatusCode) && input.Code != "pengeluaran_barang" {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	if strings.TrimSpace(input.DocumentNo) == "" || input.DocumentDate.IsZero() {
		return domain.InventoryItem{}, ErrInvalidTransition
	}
	input.Label = action.Label
	switch input.Code {
	case "pemindahan":
		if strings.TrimSpace(input.TargetFacilityID) == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		facility := m.facilityByID(input.TargetFacilityID)
		if facility.ID == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.AtTPP = true
		item.FacilityID = facility.ID
		item.FacilityName = facility.Name
		item.Location = facility.Name
		item.LocationStatus = facility.Name
	case "pencacahan":
		if strings.TrimSpace(input.Description) == "" || !domain.ValidItemKind(strings.TrimSpace(input.ItemKind)) || input.Quantity <= 0 || !domain.ValidUnit(strings.TrimSpace(input.Unit)) || !domain.ValidGoodsCondition(strings.TrimSpace(input.GoodsCondition)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.Description = strings.TrimSpace(input.Description)
		item.ItemKind = strings.TrimSpace(input.ItemKind)
		item.Quantity = input.Quantity
		item.Unit = strings.TrimSpace(input.Unit)
		item.GoodsCondition = strings.TrimSpace(input.GoodsCondition)
		item.PFPDRequired = true
		item.ResearchRequestNo = ""
		item.ResearchRequestDate = time.Time{}
		item.HSCode = ""
		item.IsRestricted = false
		item.RestrictionRule = ""
	case "request_penelitian_pfpd":
		item.ResearchRequestNo = input.DocumentNo
		item.ResearchRequestDate = input.DocumentDate
	case "penelitian_pfpd":
		if item.ResearchRequestNo == "" || strings.TrimSpace(input.HSCode) == "" || input.GoodsValue <= 0 || input.RestrictionStatus != "ya" && input.RestrictionStatus != "tidak" || input.IsRestricted && strings.TrimSpace(input.RestrictionRule) == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.HSCode = strings.TrimSpace(input.HSCode)
		item.IsRestricted = input.IsRestricted
		item.RestrictionRule = strings.TrimSpace(input.RestrictionRule)
		item.GoodsValue = input.GoodsValue
	case "penetapan_bmmn":
		if item.Type == domain.InventoryBMMN || item.Type == domain.InventoryTitipan {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.OriginDocumentType = originDocumentType(item.Type)
		item.OriginDocumentNo = item.DeterminationNo
		item.OriginDocumentDate = item.DeterminationDate
		item.DeterminationNo = input.DocumentNo
		item.DeterminationDate = input.DocumentDate
		item.Type = domain.InventoryBMMN
		item.StatusCode = "bmmn_aktif"
		item.StatusLabel = "Ditetapkan sebagai BMMN"
	case "usulan_peruntukan_bmmn":
		if item.Type != domain.InventoryBMMN || !domain.ValidAllocationPurpose(strings.TrimSpace(input.AllocationType)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.AllocationProposalType = strings.TrimSpace(input.AllocationType)
		item.AllocationProposalNo = input.DocumentNo
		item.AllocationProposalDate = input.DocumentDate
		item.AllocationPurpose = item.AllocationProposalType
	case "persetujuan_peruntukan_bmmn":
		if item.Type != domain.InventoryBMMN || item.AllocationProposalNo == "" || !domain.ValidAllocationPurpose(strings.TrimSpace(input.AllocationType)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.AllocationApprovalType = strings.TrimSpace(input.AllocationType)
		item.AllocationApprovalNo = input.DocumentNo
		item.AllocationApprovalDate = input.DocumentDate
		item.AllocationPurpose = item.AllocationApprovalType
	case "pengeluaran_barang":
		if !validInventoryExit(item, input.ExitType) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		item.ExitDocumentNo = input.DocumentNo
		item.ExitDocumentDate = input.DocumentDate
		item.ExitType = input.ExitType
		item.ExitNotes = strings.TrimSpace(input.ExitNotes)
		item.IsActive = false
		keepDestructionOpen := input.ExitType == "musnah" && item.CurrentDisposition == domain.DispositionDestruction
		if !keepDestructionOpen {
			item.CurrentDisposition = ""
		}
		item.LocationStatus = "Barang telah dikeluarkan"
		item.StatusCode = "pengeluaran_barang"
		item.StatusLabel = "Pengeluaran barang selesai"
		for processID, process := range m.dispositions {
			if process.InventoryID == item.ID && process.IsActive {
				if keepDestructionOpen && process.Type == domain.DispositionDestruction {
					process.Inventory = item
					m.dispositions[processID] = process
					continue
				}
				process.IsActive = false
				process.UpdatedAt = m.now().UTC()
				m.dispositions[processID] = process
			}
		}
	}
	if input.Code != "penetapan_bmmn" && input.Code != "pengeluaran_barang" {
		item.StatusCode = input.Code
		item.StatusLabel = input.Label
	}
	item.UpdatedAt = m.now().UTC()
	m.items[id] = item
	m.nextEvent++
	m.events[id] = append(m.events[id], domain.TimelineEvent{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: id, Code: input.Code, Label: item.StatusLabel, DocumentNo: input.DocumentNo, DocumentDate: input.DocumentDate, Notes: input.Notes, Actor: input.Actor, CreatedAt: item.UpdatedAt, Attachments: m.documentAttachments(input.DocumentID)})
	return item, nil
}

func (m *MemoryStore) ApplyInventoryCensus(_ context.Context, id string, lines []domain.InventoryGoodsLine, input domain.NewEventInput) ([]domain.InventoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !item.IsActive || item.CurrentDisposition != "" || completedProcessStatus(item.StatusCode) || strings.TrimSpace(input.DocumentNo) == "" || input.DocumentDate.IsZero() || len(lines) == 0 || len(lines) > 100 {
		return nil, ErrInvalidTransition
	}
	for _, line := range lines {
		if strings.TrimSpace(line.Description) == "" || !domain.ValidItemKind(strings.TrimSpace(line.ItemKind)) || line.GoodsValue < 0 || line.Quantity <= 0 || !domain.ValidUnit(strings.TrimSpace(line.Unit)) || !domain.ValidGoodsCondition(strings.TrimSpace(line.GoodsCondition)) {
			return nil, ErrInvalidTransition
		}
	}

	groupID := strings.TrimSpace(item.PhysicalUnitID)
	if groupID == "" {
		groupID = item.ID
	}
	existing := make(map[string]domain.InventoryItem)
	if strings.EqualFold(item.LoadType, "FCL") {
		for _, candidate := range m.items {
			candidateGroup := strings.TrimSpace(candidate.PhysicalUnitID)
			if candidateGroup == "" {
				candidateGroup = candidate.ID
			}
			if candidate.IsActive && candidateGroup == groupID {
				existing[candidate.ID] = candidate
			}
		}
	} else {
		existing[item.ID] = item
	}

	seenExisting := make(map[string]struct{}, len(existing))
	newCount := 0
	for _, line := range lines {
		lineID := strings.TrimSpace(line.InventoryID)
		if lineID == "" {
			newCount++
			continue
		}
		candidate, found := existing[lineID]
		if !found || candidate.CurrentDisposition != "" {
			return nil, ErrInvalidTransition
		}
		if _, duplicate := seenExisting[lineID]; duplicate {
			return nil, ErrInvalidTransition
		}
		seenExisting[lineID] = struct{}{}
	}
	if len(seenExisting) != len(existing) || (!strings.EqualFold(item.LoadType, "FCL") && newCount > 0) {
		return nil, ErrInvalidTransition
	}

	now := m.now().UTC()
	applyLine := func(target domain.InventoryItem, line domain.InventoryGoodsLine) domain.InventoryItem {
		target.Description = strings.TrimSpace(line.Description)
		target.ItemKind = strings.TrimSpace(line.ItemKind)
		target.GoodsValue = line.GoodsValue
		target.Quantity = line.Quantity
		target.QuantityDetail = strings.TrimSpace(line.QuantityDetail)
		target.Unit = strings.TrimSpace(line.Unit)
		target.GoodsCondition = strings.TrimSpace(line.GoodsCondition)
		target.PFPDRequired = true
		target.PhysicalUnitID = groupID
		target.ResearchRequestNo = ""
		target.ResearchRequestDate = time.Time{}
		target.HSCode = ""
		target.IsRestricted = false
		target.RestrictionRule = ""
		target.StatusCode = "pencacahan"
		target.StatusLabel = "Pencacahan"
		target.UpdatedAt = now
		return target
	}

	result := make([]domain.InventoryItem, 0, len(lines))
	for index, line := range lines {
		lineID := strings.TrimSpace(line.InventoryID)
		if lineID != "" {
			updated := applyLine(existing[lineID], line)
			m.items[lineID] = updated
			m.nextEvent++
			m.events[lineID] = append(m.events[lineID], domain.TimelineEvent{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: lineID, Code: "pencacahan", Label: "Pencacahan", DocumentNo: input.DocumentNo, DocumentDate: input.DocumentDate, Notes: input.Notes, Actor: input.Actor, CreatedAt: now, Attachments: m.documentAttachments(input.DocumentID)})
			result = append(result, updated)
			continue
		}
		m.nextItem++
		cloneID := fmt.Sprintf("inv-%03d", m.nextItem)
		clone := item
		clone.ID = cloneID
		clone.ReferenceNo = fmt.Sprintf("%s/CACAH-G%02d-%03d", item.ReferenceNo, index+1, m.nextItem)
		clone.OccupancyPrimary = false
		clone.CreatedBy = input.Actor
		clone.CreatedAt = now
		clone = applyLine(clone, line)
		m.items[cloneID] = clone
		m.nextEvent++
		m.events[cloneID] = []domain.TimelineEvent{{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: cloneID, Code: "pencacahan", Label: "Pencacahan", DocumentNo: input.DocumentNo, DocumentDate: input.DocumentDate, Notes: input.Notes, Actor: input.Actor, CreatedAt: now, Attachments: m.documentAttachments(input.DocumentID)}}
		result = append(result, clone)
	}
	return result, nil
}

func (m *MemoryStore) RelocateInventoryLoad(_ context.Context, input domain.InventoryLoadRelocationInput) ([]domain.InventoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[strings.TrimSpace(input.InventoryID)]
	if !ok {
		return nil, ErrNotFound
	}
	if !item.IsActive {
		return nil, ErrInactiveInventory
	}
	if strings.TrimSpace(input.DocumentNo) == "" || input.DocumentDate.IsZero() || strings.TrimSpace(input.Actor) == "" || len(input.Allocations) == 0 || len(input.Allocations) > 20 || item.Quantity <= 0 {
		return nil, ErrInvalidTransition
	}

	allocations := make([]domain.InventoryLoadAllocation, 0, len(input.Allocations))
	totalQuantity := 0.0
	seenFCL := make(map[string]struct{})
	for _, raw := range input.Allocations {
		allocation := raw
		allocation.LoadType = strings.ToUpper(strings.TrimSpace(allocation.LoadType))
		allocation.Quantity = math.Round(allocation.Quantity*100) / 100
		if allocation.Quantity <= 0 {
			return nil, ErrInvalidTransition
		}
		switch allocation.LoadType {
		case "FCL":
			number, valid := normalizedContainerNumber(allocation.ContainerNo)
			allocation.ContainerSize = strings.ToUpper(strings.TrimSpace(allocation.ContainerSize))
			if !valid || !domain.ValidContainerSize(allocation.ContainerSize) {
				return nil, ErrInvalidTransition
			}
			allocation.ContainerNo = number
			allocation.EstimatedVolumeM3 = 0
			key := strings.ReplaceAll(strings.ReplaceAll(number, " ", ""), "-", "")
			if _, duplicate := seenFCL[key]; duplicate {
				return nil, ErrInvalidTransition
			}
			seenFCL[key] = struct{}{}
		case "LCL":
			allocation.ContainerNo = ""
			allocation.ContainerSize = ""
			allocation.EstimatedVolumeM3 = math.Round(allocation.EstimatedVolumeM3*100) / 100
			if allocation.EstimatedVolumeM3 <= 0 {
				return nil, ErrInvalidTransition
			}
		default:
			return nil, ErrInvalidTransition
		}
		totalQuantity += allocation.Quantity
		allocations = append(allocations, allocation)
	}
	if math.Abs(totalQuantity-item.Quantity) > 0.005 {
		return nil, ErrInvalidTransition
	}
	processLocked := item.CurrentDisposition != "" || completedProcessStatus(item.StatusCode)
	if processLocked && len(allocations) > 1 {
		return nil, ErrInvalidTransition
	}
	changed := len(allocations) > 1
	if !changed {
		allocation := allocations[0]
		sourceLoadType := strings.ToUpper(strings.TrimSpace(item.LoadType))
		changed = allocation.LoadType != sourceLoadType
		if !changed && allocation.LoadType == "FCL" {
			sourceContainer, valid := normalizedContainerNumber(item.ContainerNo)
			changed = !valid || sourceContainer != allocation.ContainerNo || strings.ToUpper(strings.TrimSpace(item.ContainerSize)) != allocation.ContainerSize
		}
		if !changed && allocation.LoadType == "LCL" {
			changed = math.Abs(item.EstimatedVolumeM3-allocation.EstimatedVolumeM3) > 0.005
		}
	}
	if !changed {
		return nil, ErrInvalidTransition
	}

	now := m.now().UTC()
	sourceHistory := append([]domain.TimelineEvent(nil), m.events[item.ID]...)
	originalQuantity := item.Quantity
	originalValue := item.GoodsValue
	originalUnitID := strings.TrimSpace(item.PhysicalUnitID)
	if originalUnitID == "" {
		originalUnitID = item.ID
	}
	affectedUnits := map[string]struct{}{originalUnitID: {}}
	resultIDs := make([]string, 0, len(allocations))
	allocatedValue := int64(0)

	physicalUnitFor := func(allocation domain.InventoryLoadAllocation, index int) string {
		if allocation.LoadType == "LCL" {
			return fmt.Sprintf("LCL:%s:%d:%d", item.ID, now.UnixNano(), index+1)
		}
		if current, valid := normalizedContainerNumber(item.ContainerNo); valid && current == allocation.ContainerNo && strings.TrimSpace(item.PhysicalUnitID) != "" {
			return strings.TrimSpace(item.PhysicalUnitID)
		}
		for _, candidate := range m.items {
			if !candidate.IsActive || !strings.EqualFold(candidate.LoadType, "FCL") || candidate.FacilityID != item.FacilityID || candidate.AtTPP != item.AtTPP {
				continue
			}
			if number, valid := normalizedContainerNumber(candidate.ContainerNo); valid && number == allocation.ContainerNo {
				if unitID := strings.TrimSpace(candidate.PhysicalUnitID); unitID != "" {
					return unitID
				}
			}
		}
		compact := strings.NewReplacer(" ", "", "-", "").Replace(allocation.ContainerNo)
		return "FCL:" + compact
	}

	applyAllocation := func(target domain.InventoryItem, allocation domain.InventoryLoadAllocation, physicalUnitID string, goodsValue int64) domain.InventoryItem {
		target.LoadType = allocation.LoadType
		target.ContainerNo = allocation.ContainerNo
		target.ContainerSize = allocation.ContainerSize
		target.EstimatedVolumeM3 = allocation.EstimatedVolumeM3
		target.PhysicalUnitID = physicalUnitID
		target.OccupancyPrimary = false
		target.Quantity = allocation.Quantity
		target.GoodsValue = goodsValue
		// Bongkar/muat hanya mengubah penempatan fisik. Status proses barang
		// harus tetap sama seperti sebelum action dijalankan.
		target.UpdatedAt = now
		return target
	}

	for index, allocation := range allocations {
		goodsValue := originalValue - allocatedValue
		if index < len(allocations)-1 {
			goodsValue = int64(math.Round(float64(originalValue) * allocation.Quantity / originalQuantity))
			remainingValue := originalValue - allocatedValue
			if goodsValue < 0 {
				goodsValue = 0
			}
			if goodsValue > remainingValue {
				goodsValue = remainingValue
			}
			allocatedValue += goodsValue
		}
		physicalUnitID := physicalUnitFor(allocation, index)
		affectedUnits[physicalUnitID] = struct{}{}

		var target domain.InventoryItem
		if index == 0 {
			target = applyAllocation(item, allocation, physicalUnitID, goodsValue)
			m.items[target.ID] = target
		} else {
			m.nextItem++
			target = item
			target.ID = fmt.Sprintf("inv-%03d", m.nextItem)
			target.ReferenceNo = fmt.Sprintf("%s/MOVE-%02d-%03d", item.ReferenceNo, index+1, m.nextItem)
			target.CreatedBy = input.Actor
			target.CreatedAt = now
			target = applyAllocation(target, allocation, physicalUnitID, goodsValue)
			m.items[target.ID] = target
			m.events[target.ID] = append([]domain.TimelineEvent(nil), sourceHistory...)
			for eventIndex := range m.events[target.ID] {
				m.nextEvent++
				m.events[target.ID][eventIndex].ID = fmt.Sprintf("evt-%03d", m.nextEvent)
				m.events[target.ID][eventIndex].InventoryID = target.ID
				m.events[target.ID][eventIndex].DispositionID = ""
				m.events[target.ID][eventIndex].DispositionType = ""
			}
		}
		m.nextEvent++
		m.events[target.ID] = append(m.events[target.ID], domain.TimelineEvent{
			ID:           fmt.Sprintf("evt-%03d", m.nextEvent),
			InventoryID:  target.ID,
			Code:         "pindah_bongkar_kontainer",
			Label:        "Bongkar/Muat Kontainer",
			DocumentNo:   input.DocumentNo,
			DocumentDate: input.DocumentDate,
			Notes:        input.Notes,
			Actor:        input.Actor,
			CreatedAt:    now,
			Attachments:  m.documentAttachments(input.DocumentID),
		})
		resultIDs = append(resultIDs, target.ID)
	}

	// Rebalance every affected physical unit so YOR/SOR counts each container
	// or LCL lot exactly once, including a source container that still contains
	// other goods rows after this row is moved away.
	for unitID := range affectedUnits {
		primaryID := ""
		var primaryCreated time.Time
		for id, candidate := range m.items {
			candidateUnit := strings.TrimSpace(candidate.PhysicalUnitID)
			if candidateUnit == "" {
				candidateUnit = candidate.ID
			}
			if !candidate.IsActive || candidateUnit != unitID {
				continue
			}
			candidate.OccupancyPrimary = false
			m.items[id] = candidate
			if primaryID == "" || candidate.CreatedAt.Before(primaryCreated) || candidate.CreatedAt.Equal(primaryCreated) && id < primaryID {
				primaryID = id
				primaryCreated = candidate.CreatedAt
			}
		}
		if primaryID != "" {
			candidate := m.items[primaryID]
			candidate.OccupancyPrimary = true
			m.items[primaryID] = candidate
		}
	}

	result := make([]domain.InventoryItem, 0, len(resultIDs))
	for _, id := range resultIDs {
		result = append(result, m.items[id])
	}
	return result, nil
}

func originDocumentType(kind domain.InventoryType) string {
	if kind == domain.InventoryBDN {
		return "KEP BDN"
	}
	return "BCF 1.5"
}

func (m *MemoryStore) CreateDocument(_ context.Context, input domain.NewDocumentInput) (domain.DocumentAttachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(input.FileName) == "" || strings.TrimSpace(input.MIMEType) == "" || len(input.Content) == 0 || input.SizeBytes != int64(len(input.Content)) {
		return domain.DocumentAttachment{}, ErrInvalidTransition
	}
	m.nextDocument++
	id := fmt.Sprintf("doc-%03d", m.nextDocument)
	hash := sha256.Sum256(input.Content)
	meta := domain.DocumentAttachment{ID: id, FileName: input.FileName, MIMEType: input.MIMEType, SizeBytes: input.SizeBytes, UploadedBy: input.UploadedBy, SHA256: hex.EncodeToString(hash[:]), CreatedAt: m.now().UTC()}
	m.documents[id] = memoryDocument{Meta: meta, Content: append([]byte(nil), input.Content...)}
	return meta, nil
}

func (m *MemoryStore) GetDocument(_ context.Context, id string) (domain.DocumentAttachment, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	document, ok := m.documents[id]
	if !ok {
		return domain.DocumentAttachment{}, nil, ErrNotFound
	}
	return document.Meta, append([]byte(nil), document.Content...), nil
}

func (m *MemoryStore) DocumentAccess(_ context.Context, id string) ([]domain.DocumentAccess, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.DocumentAccess
	for inventoryID, events := range m.events {
		for _, event := range events {
			for _, attachment := range event.Attachments {
				if attachment.ID == id {
					if item, ok := m.items[inventoryID]; ok {
						result = append(result, domain.DocumentAccess{Inventory: item, DispositionType: string(event.DispositionType), EventCode: event.Code})
					}
				}
			}
		}
	}
	if len(result) == 0 {
		return nil, ErrNotFound
	}
	return result, nil
}

func (m *MemoryStore) NotificationSummary(_ context.Context, allowed []domain.InventoryType) (domain.NotificationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := m.now()
	var out domain.NotificationSummary
	for _, item := range m.items {
		if !item.IsActive || len(allowed) > 0 && !inventoryTypeAllowed(item.Type, allowed) {
			continue
		}
		if (item.Type == domain.InventoryBTD || item.Type == domain.InventoryBDN) && item.AgeDays(now) >= 60 && (item.StatusCode == "masih_di_tps" || item.StatusCode == "ditetapkan") {
			out.Overdue60Days++
		}
		if item.StatusCode == "laku" || item.StatusCode == "alokasi_hasil_lelang" || item.StatusCode == "ba_musnah" || item.StatusCode == "ba_serah_terima" {
			out.ReadyForExit++
		}
		if item.Type == domain.InventoryBMMN && item.CurrentDisposition == "" {
			out.BMMNWaiting++
		}
	}
	return out, nil
}
func (m *MemoryStore) WriteAudit(context.Context, domain.AuditEntry) error { return nil }

func (m *MemoryStore) documentAttachments(documentID string) []domain.DocumentAttachment {
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return nil
	}
	document, ok := m.documents[documentID]
	if !ok {
		return nil
	}
	return []domain.DocumentAttachment{document.Meta}
}

func (m *MemoryStore) ListEvents(_ context.Context, limit int) ([]domain.TimelineEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.TimelineEvent, 0)
	for _, events := range m.events {
		for _, event := range events {
			copyEvent := event
			copyEvent.Attachments = nil
			result = append(result, copyEvent)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *MemoryStore) PerformanceSource(_ context.Context, from, to time.Time, allowed []domain.InventoryType) ([]domain.InventoryItem, []domain.TimelineEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	completionCodes := map[string]bool{
		"selesai_lelang": true, "laku": true, "tidak_laku": true,
		"ba_musnah": true, "ba_serah_terima": true, "pencacahan": true,
		"penelitian_pfpd": true, "penelitian_hs_lartas": true, "penetapan_bmmn": true,
	}
	sourceCodes := map[string]bool{
		"ditetapkan": true, "masih_di_tps": true,
		"request_penelitian_pfpd": true, "siap_peruntukan": true,
	}
	for code := range completionCodes {
		sourceCodes[code] = true
	}
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toExclusive := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	selected := make(map[string]bool)
	for inventoryID, events := range m.events {
		item, ok := m.items[inventoryID]
		if !ok || item.Type == domain.InventoryTitipan || len(allowed) > 0 && !inventoryTypeAllowed(item.Type, allowed) {
			continue
		}
		for _, event := range events {
			if !completionCodes[strings.ToLower(strings.TrimSpace(event.Code))] {
				continue
			}
			completed := event.DocumentDate
			if completed.IsZero() {
				completed = event.CreatedAt
			}
			if !completed.Before(fromDay) && completed.Before(toExclusive) {
				selected[inventoryID] = true
				break
			}
		}
	}
	items := make([]domain.InventoryItem, 0, len(selected))
	events := make([]domain.TimelineEvent, 0)
	for inventoryID := range selected {
		items = append(items, m.items[inventoryID])
		for _, event := range m.events[inventoryID] {
			if !sourceCodes[strings.ToLower(strings.TrimSpace(event.Code))] {
				continue
			}
			copyEvent := event
			copyEvent.Attachments = nil
			events = append(events, copyEvent)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
	return items, events, nil
}

func (m *MemoryStore) Timeline(ctx context.Context, inventoryID string) ([]domain.TimelineEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.items[inventoryID]; !ok {
		return nil, ErrNotFound
	}
	events := append([]domain.TimelineEvent(nil), m.events[inventoryID]...)
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.Before(events[j].CreatedAt) })
	return events, nil
}

func (m *MemoryStore) EligibleInventory(ctx context.Context, query string, limit int) ([]domain.InventoryItem, error) {
	items, err := m.ListInventory(ctx, domain.InventoryFilter{Query: query, Sort: "newest"})
	if err != nil {
		return nil, err
	}
	result := items[:0]
	for _, item := range items {
		if item.CurrentDisposition == "" && !completedDispositionStatus(item.StatusCode) {
			result = append(result, item)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func completedDispositionStatus(code string) bool {
	return code == "alokasi_hasil_lelang" || code == "ba_musnah" || code == "ba_serah_terima"
}

func (m *MemoryStore) ListDispositions(ctx context.Context, filter domain.DispositionFilter) ([]domain.Disposition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	result := make([]domain.Disposition, 0)
	for _, process := range m.dispositions {
		item := m.items[process.InventoryID]
		process.Inventory = item
		if filter.InventoryID != "" && process.InventoryID != filter.InventoryID {
			continue
		}
		if filter.OnlyInactiveInventory && item.IsActive {
			continue
		}
		if !filter.OnlyInactiveInventory && !filter.IncludeInactiveInventory && !item.IsActive {
			continue
		}
		if filter.Type != "" && process.Type != filter.Type {
			continue
		}
		if len(filter.IncludeStatusCodes) > 0 && !statusCodeIncluded(process.StatusCode, filter.IncludeStatusCodes) {
			continue
		}
		if len(filter.ExcludeStatusCodes) > 0 && statusCodeIncluded(process.StatusCode, filter.ExcludeStatusCodes) {
			continue
		}
		if filter.FacilityID != "" && item.FacilityID != filter.FacilityID {
			continue
		}
		if len(filter.AllowedTypes) > 0 && !inventoryTypeAllowed(item.Type, filter.AllowedTypes) {
			continue
		}
		if filter.Status == "active" && !process.IsActive || filter.Status == "completed" && process.IsActive {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(strings.Join([]string{item.ContainerNo, item.ReferenceNo, item.Description, item.OwnerName, process.StatusLabel}, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		result = append(result, process)
	}
	sort.Slice(result, func(i, j int) bool {
		switch filter.Sort {
		case "oldest":
			return result[i].UpdatedAt.Before(result[j].UpdatedAt)
		case "determination_newest":
			return result[i].Inventory.DeterminationDate.After(result[j].Inventory.DeterminationDate)
		case "determination_oldest":
			return result[i].Inventory.DeterminationDate.Before(result[j].Inventory.DeterminationDate)
		case "value_desc":
			if filter.Type == domain.DispositionAuction {
				return result[i].HTLValue > result[j].HTLValue
			}
			return result[i].Inventory.GoodsValue > result[j].Inventory.GoodsValue
		case "value_asc":
			if filter.Type == domain.DispositionAuction {
				return result[i].HTLValue < result[j].HTLValue
			}
			return result[i].Inventory.GoodsValue < result[j].Inventory.GoodsValue
		default:
			return result[i].UpdatedAt.After(result[j].UpdatedAt)
		}
	})
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start > len(result) {
		start = len(result)
	}
	end := len(result)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return result[start:end], nil
}

func (m *MemoryStore) CountDispositions(ctx context.Context, filter domain.DispositionFilter) (int, error) {
	filter.Limit = 0
	filter.Offset = 0
	rows, err := m.ListDispositions(ctx, filter)
	return len(rows), err
}

func (m *MemoryStore) ProcessDashboard(ctx context.Context, kind domain.DispositionType, year int, allowed []domain.InventoryType) (domain.ProcessDashboard, error) {
	rows, err := m.ListDispositions(ctx, domain.DispositionFilter{Type: kind, IncludeInactiveInventory: true, AllowedTypes: allowed})
	if err != nil {
		return domain.ProcessDashboard{}, err
	}
	labels := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	out := domain.ProcessDashboard{Year: year, Chart: make([]domain.ProcessChartPoint, 12)}
	for i := range labels {
		out.Chart[i].Label = labels[i]
	}
	for _, process := range rows {
		if process.IsActive {
			out.Active++
		}
		if process.CreatedAt.Year() == year {
			out.StartedThisYear++
			out.ThisYear++
		}
		completed := !process.IsActive || kind == domain.DispositionAuction && (process.StatusCode == "laku" || process.StatusCode == "tidak_laku" || process.StatusCode == "alokasi_hasil_lelang")
		if completed && process.UpdatedAt.Year() == year {
			out.CompletedThisYear++
		}
		if process.CreatedAt.Year() != year {
			continue
		}
		month := int(process.CreatedAt.Month()) - 1
		if month < 0 || month >= len(out.Chart) {
			continue
		}
		point := &out.Chart[month]
		point.Count++
		point.GoodsValue += process.Inventory.GoodsValue
		point.HTLValue += process.HTLValue
		point.SaleValue += process.SaleValue
		switch kind {
		case domain.DispositionAuction:
			out.TotalGoodsValue += process.Inventory.GoodsValue
			out.TotalHTLValue += process.HTLValue
			out.TotalSaleValue += process.SaleValue
		case domain.DispositionDestruction:
			point.Cost += process.DestructionCost
			out.TotalCost += process.DestructionCost
		case domain.DispositionGrant:
			if process.TransferType == "hibah" {
				point.Grant++
				out.TotalGrant++
			} else if process.TransferType == "psp" {
				point.PSP++
				out.TotalPSP++
			}
		}
		if point.Count > out.MaxCount {
			out.MaxCount = point.Count
		}
		if point.Grant > out.MaxCount {
			out.MaxCount = point.Grant
		}
		if point.PSP > out.MaxCount {
			out.MaxCount = point.PSP
		}
		for _, value := range []int64{point.GoodsValue, point.HTLValue, point.SaleValue, point.Cost} {
			if value > out.MaxMoney {
				out.MaxMoney = value
			}
		}
	}
	if out.MaxCount == 0 {
		out.MaxCount = 1
	}
	if out.MaxMoney == 0 {
		out.MaxMoney = 1
	}
	return out, nil
}

func (m *MemoryStore) GetDisposition(ctx context.Context, id string) (domain.Disposition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	process, ok := m.dispositions[id]
	if !ok {
		return domain.Disposition{}, ErrNotFound
	}
	process.Inventory = m.items[process.InventoryID]
	return process, nil
}

func (m *MemoryStore) CreateDisposition(ctx context.Context, input domain.NewDispositionInput) (domain.Disposition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[input.InventoryID]
	if !ok {
		return domain.Disposition{}, ErrNotFound
	}
	if !item.IsActive {
		return domain.Disposition{}, ErrInactiveInventory
	}
	if completedDispositionStatus(item.StatusCode) {
		return domain.Disposition{}, ErrInvalidTransition
	}
	transferFailedAuction := canTransferFailedAuction(item, input.Type)
	if item.CurrentDisposition != "" && !transferFailedAuction {
		return domain.Disposition{}, ErrConflict
	}
	if input.Type != domain.DispositionAuction && input.Type != domain.DispositionDestruction && input.Type != domain.DispositionGrant {
		return domain.Disposition{}, ErrInvalidTransition
	}
	m.nextProcess++
	id := fmt.Sprintf("proc-%03d", m.nextProcess)
	now := m.now().UTC()
	if transferFailedAuction {
		transferCode, transferLabel := "dialihkan_musnah", "Dialihkan ke pemusnahan"
		if input.Type == domain.DispositionGrant {
			transferCode, transferLabel = "dialihkan_hibah", "Dialihkan ke hibah/PSP"
		}
		found := false
		for processID, existing := range m.dispositions {
			if existing.InventoryID == item.ID && existing.Type == domain.DispositionAuction && existing.IsActive && existing.StatusCode == "tidak_laku" {
				existing.IsActive = false
				existing.StatusCode = transferCode
				existing.StatusLabel = transferLabel
				existing.UpdatedAt = now
				m.dispositions[processID] = existing
				m.nextEvent++
				m.events[item.ID] = append(m.events[item.ID], domain.TimelineEvent{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: item.ID, DispositionID: existing.ID, DispositionType: string(domain.DispositionAuction), Code: transferCode, Label: transferLabel, Notes: "Barang lelang tidak laku dialihkan ke proses " + dispositionLabel(input.Type) + ".", Actor: input.Actor, CreatedAt: now})
				found = true
				break
			}
		}
		if !found {
			return domain.Disposition{}, ErrConflict
		}
	}
	label := map[domain.DispositionType]string{domain.DispositionAuction: "Proses lelang dimulai", domain.DispositionDestruction: "Proses pemusnahan dimulai", domain.DispositionGrant: "Proses hibah/PSP dimulai"}[input.Type]
	code := "proses_" + string(input.Type)
	process := domain.Disposition{ID: id, InventoryID: item.ID, Type: input.Type, Round: 1, StatusCode: code, StatusLabel: label, IsActive: true, CreatedBy: input.Actor, CreatedAt: now, UpdatedAt: now}
	item.CurrentDisposition = input.Type
	item.StatusCode = code
	item.StatusLabel = label
	item.UpdatedAt = now
	m.items[item.ID] = item
	process.Inventory = item
	m.dispositions[id] = process
	m.nextEvent++
	m.events[item.ID] = append(m.events[item.ID], domain.TimelineEvent{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: item.ID, DispositionID: id, DispositionType: string(input.Type), Code: code, Label: label, Notes: input.Notes, Actor: input.Actor, CreatedAt: now})
	return process, nil
}

func (m *MemoryStore) AddDispositionEvent(ctx context.Context, id string, input domain.NewEventInput) (domain.Disposition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	process, ok := m.dispositions[id]
	if !ok {
		return domain.Disposition{}, ErrNotFound
	}
	if !process.IsActive {
		return domain.Disposition{}, ErrInvalidTransition
	}
	action, valid := domain.FindDispositionAction(process.Type, input.Code)
	if !valid || validateDispositionTransition(process, input, action) != nil {
		return domain.Disposition{}, ErrInvalidTransition
	}
	input.Label = dispositionStatusLabel(input.Code, action.Label)
	item := m.items[process.InventoryID]
	if input.Code == "lelang_penyesuaian" {
		process.Round++
		input.Label = fmt.Sprintf("Lelang penyesuaian putaran %d", process.Round)
	}
	statusCode := input.Code
	if input.Code == "selesai_lelang" {
		statusCode = input.AuctionOutcome
		if input.AuctionOutcome == "laku" {
			input.Label = "Laku"
		} else {
			input.Label = "Tidak laku"
		}
	}
	if input.Code == "ba_serah_terima" {
		input.Label = "BA Serah Terima " + strings.ToUpper(input.TransferType)
	}
	process.StatusCode = statusCode
	process.StatusLabel = input.Label
	if input.Code == "kep_htl" {
		process.HTLValue = input.HTLValue
	}
	if input.Code == "jadwal_lelang" {
		process.ExecutionStartDate = input.ExecutionStartDate
		process.ExecutionEndDate = input.ExecutionEndDate
		process.ScheduleDocumentNo = input.DocumentNo
		process.ScheduleDocumentDate = input.DocumentDate
	}
	if input.Code == "selesai_lelang" {
		process.AuctionOutcome = input.AuctionOutcome
		process.SaleValue = input.SaleValue
	}
	if input.Code == "alokasi_hasil_lelang" {
		process.AllocationTarget = input.AllocationTarget
	}
	if input.Code == "kep_musnah" || input.Code == "ba_musnah" {
		process.DestructionCost = input.DestructionCost
	}
	if input.Code == "ba_serah_terima" {
		process.TransferType = input.TransferType
	}
	if input.RecipientCode != "" {
		process.RecipientCode = input.RecipientCode
	}
	if input.RecipientName != "" {
		process.RecipientName = input.RecipientName
	}
	process.UpdatedAt = m.now().UTC()
	item.StatusCode = statusCode
	item.StatusLabel = input.Label
	item.UpdatedAt = process.UpdatedAt
	if input.Code == "alokasi_hasil_lelang" || input.Code == "ba_musnah" || input.Code == "ba_serah_terima" {
		process.IsActive = false
		item.CurrentDisposition = ""
	}
	m.items[item.ID] = item
	process.Inventory = item
	m.dispositions[id] = process
	m.nextEvent++
	m.events[item.ID] = append(m.events[item.ID], domain.TimelineEvent{ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: item.ID, DispositionID: id, DispositionType: string(process.Type), Code: input.Code, Label: item.StatusLabel, DocumentNo: input.DocumentNo, DocumentDate: input.DocumentDate, Notes: input.Notes, Actor: input.Actor, CreatedAt: process.UpdatedAt, Attachments: m.documentAttachments(input.DocumentID)})
	return process, nil
}

func validateDispositionTransition(process domain.Disposition, input domain.NewEventInput, action domain.WorkflowAction) error {
	if !process.IsActive || strings.TrimSpace(input.DocumentNo) == "" || input.DocumentDate.IsZero() {
		return ErrInvalidTransition
	}
	if action.CreatesProcess {
		if process.StatusCode != "proses_"+string(process.Type) {
			return ErrInvalidTransition
		}
	} else if action.AllowedStatus != "" && !allowedDispositionStatus(action.AllowedStatus, process.StatusCode) {
		return ErrInvalidTransition
	}
	switch process.Type {
	case domain.DispositionAuction:
		switch input.Code {
		case "kep_htl":
			if input.HTLValue <= 0 {
				return ErrInvalidTransition
			}
		case "jadwal_lelang":
			if input.ExecutionStartDate.IsZero() || !input.ExecutionEndDate.IsZero() && input.ExecutionEndDate.Before(input.ExecutionStartDate) {
				return ErrInvalidTransition
			}
		case "selesai_lelang":
			if input.AuctionOutcome != "laku" && input.AuctionOutcome != "tidak_laku" || input.AuctionOutcome == "laku" && input.SaleValue <= 0 {
				return ErrInvalidTransition
			}
		case "alokasi_hasil_lelang":
			if strings.TrimSpace(input.AllocationTarget) == "" {
				return ErrInvalidTransition
			}
		}
	case domain.DispositionDestruction:
		if input.DestructionCost <= 0 {
			return ErrInvalidTransition
		}
	case domain.DispositionGrant:
		if !domain.ValidTransferType(input.TransferType) {
			return ErrInvalidTransition
		}
	}
	return nil
}

func allowedDispositionStatus(allowed, status string) bool {
	for _, candidate := range strings.Split(allowed, ",") {
		if strings.TrimSpace(candidate) == status {
			return true
		}
	}
	return false
}

func dispositionStatusLabel(code, fallback string) string {
	switch code {
	case "kep_lelang":
		return "Mulai lelang"
	case "kep_htl":
		return "HTL ditetapkan"
	case "jadwal_lelang":
		return "Jadwal lelang ditetapkan"
	case "alokasi_hasil_lelang":
		return "Alokasi hasil lelang"
	case "kep_musnah":
		return "KEP Musnah diterbitkan"
	case "ba_musnah":
		return "Pemusnahan selesai"
	default:
		return fallback
	}
}

func dispositionLabel(kind domain.DispositionType) string {
	switch kind {
	case domain.DispositionAuction:
		return "lelang dan pengeluaran barang"
	case domain.DispositionDestruction:
		return "pemusnahan"
	default:
		return "hibah/PSP dan pengeluaran barang"
	}
}

func (m *MemoryStore) ListReconciliations(_ context.Context, limit int) ([]domain.ReconciliationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := append([]domain.ReconciliationRecord(nil), m.reconciliations...)
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *MemoryStore) ReconcileInventory(ctx context.Context, input domain.NewReconciliationInput) (domain.ReconciliationRecord, domain.InventoryItem, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Notes == "" || input.Actor == "" {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}

	if input.Type == "found_not_recorded" {
		input.NewItem.ReconciliationCreated = true
		input.NewItem.Actor = input.Actor
		input.NewItem.DocumentID = input.DocumentID
		item, err := m.CreateInventory(ctx, input.NewItem)
		if err != nil {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.nextReconciliation++
		record := domain.ReconciliationRecord{
			ID: fmt.Sprintf("rec-%03d", m.nextReconciliation), Type: input.Type, Action: "added",
			InventoryID: item.ID, InventoryReference: item.ReferenceNo, InventoryType: item.Type,
			ResultStatusCode: item.StatusCode, ResultStatusLabel: item.StatusLabel,
			Notes: input.Notes, Actor: input.Actor, CreatedAt: m.now().UTC(),
		}
		m.reconciliations = append(m.reconciliations, record)
		return record, item, nil
	}

	if input.Type != "recorded_not_found" || strings.TrimSpace(input.InventoryID) == "" {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[input.InventoryID]
	if !ok {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrNotFound
	}
	if !item.IsActive {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInactiveInventory
	}
	previousCode, previousLabel := item.StatusCode, item.StatusLabel
	item.IsActive = false
	item.CurrentDisposition = ""
	item.StatusCode = "rekonsiliasi_tidak_ditemukan"
	item.StatusLabel = "Tidak ditemukan di lapangan"
	item.LocationStatus = "Tidak ditemukan saat rekonsiliasi"
	item.UpdatedAt = m.now().UTC()
	m.items[item.ID] = item
	for processID, process := range m.dispositions {
		if process.InventoryID == item.ID && process.IsActive {
			process.IsActive = false
			process.UpdatedAt = item.UpdatedAt
			m.dispositions[processID] = process
		}
	}
	m.nextEvent++
	m.events[item.ID] = append(m.events[item.ID], domain.TimelineEvent{
		ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: item.ID, Code: item.StatusCode,
		Label: item.StatusLabel, Notes: input.Notes, Actor: input.Actor, CreatedAt: item.UpdatedAt,
		Attachments: m.documentAttachments(input.DocumentID),
	})
	m.nextReconciliation++
	record := domain.ReconciliationRecord{
		ID: fmt.Sprintf("rec-%03d", m.nextReconciliation), Type: input.Type, Action: "removed",
		InventoryID: item.ID, InventoryReference: item.ReferenceNo, InventoryType: item.Type,
		PreviousStatusCode: previousCode, PreviousStatusLabel: previousLabel,
		ResultStatusCode: item.StatusCode, ResultStatusLabel: item.StatusLabel,
		Notes: input.Notes, Actor: input.Actor, CreatedAt: item.UpdatedAt,
	}
	m.reconciliations = append(m.reconciliations, record)
	return record, item, nil
}

func (m *MemoryStore) CorrectInventoryData(_ context.Context, input domain.InventoryCorrectionInput) (domain.ReconciliationRecord, domain.InventoryItem, error) {
	input.InventoryID = strings.TrimSpace(input.InventoryID)
	input.Actor = strings.TrimSpace(input.Actor)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.InventoryID == "" || input.Actor == "" || !validCorrectionReason(input.Reason) {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.items[input.InventoryID]
	if !ok {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrNotFound
	}
	corrected, err := correctedInventoryItem(current, input.Item)
	if err != nil {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
	}
	changeDetails := inventoryCorrectionChanges(current, corrected)
	for id, item := range m.items {
		if id != current.ID && strings.EqualFold(strings.TrimSpace(item.ReferenceNo), corrected.ReferenceNo) {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrConflict
		}
	}
	if corrected.FacilityID != "" {
		facility := m.facilityByID(corrected.FacilityID)
		if facility.ID == "" {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
		corrected.FacilityName = facility.Name
	}

	// Prepare all event changes on a copy first. Nothing is committed until every
	// requested correction has passed validation, so the in-memory store mirrors
	// the atomic behavior of the Supabase RPC.
	eventByID := make(map[string]domain.EventCorrection, len(input.Events))
	for _, correction := range input.Events {
		id := strings.TrimSpace(correction.ID)
		if id != "" {
			correction.ID = id
			eventByID[id] = correction
		}
	}
	correctedEvents := append([]domain.TimelineEvent(nil), m.events[current.ID]...)
	seenEvents := make(map[string]bool, len(eventByID))
	for index, event := range correctedEvents {
		correction, exists := eventByID[event.ID]
		if !exists {
			continue
		}
		event.Label = strings.TrimSpace(correction.Label)
		if event.Label == "" {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
		event.DocumentNo = strings.TrimSpace(correction.DocumentNo)
		event.DocumentDate = correction.DocumentDate
		beforeEvent := event
		event.Notes = strings.TrimSpace(correction.Notes)
		correctedEvents[index] = event
		changeDetails = append(changeDetails, eventCorrectionChanges(beforeEvent, event)...)
		seenEvents[event.ID] = true
	}
	for id := range eventByID {
		if !seenEvents[id] {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
	}

	// Prepare process changes on copies and reject IDs that do not belong to the
	// selected item. This prevents partial updates when one process is invalid.
	processByID := make(map[string]domain.DispositionCorrection, len(input.Processes))
	for _, correction := range input.Processes {
		id := strings.TrimSpace(correction.ID)
		if id != "" {
			correction.ID = id
			processByID[id] = correction
		}
	}
	correctedProcesses := make(map[string]domain.Disposition, len(processByID))
	seenProcesses := make(map[string]bool, len(processByID))
	for processID, process := range m.dispositions {
		if process.InventoryID != current.ID {
			continue
		}
		correction, exists := processByID[process.ID]
		if !exists {
			continue
		}
		if correction.SaleValue < 0 || correction.HTLValue < 0 || correction.DestructionCost < 0 || (!correction.ExecutionEndDate.IsZero() && correction.ExecutionEndDate.Before(correction.ExecutionStartDate)) {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
		process.ProposalType = strings.TrimSpace(correction.ProposalType)
		process.RecipientCode = strings.TrimSpace(correction.RecipientCode)
		process.RecipientName = strings.TrimSpace(correction.RecipientName)
		process.SaleValue = correction.SaleValue
		process.HTLValue = correction.HTLValue
		process.ExecutionStartDate = correction.ExecutionStartDate
		process.ExecutionEndDate = correction.ExecutionEndDate
		process.ScheduleDocumentNo = strings.TrimSpace(correction.ScheduleDocumentNo)
		process.ScheduleDocumentDate = correction.ScheduleDocumentDate
		process.AuctionOutcome = strings.TrimSpace(correction.AuctionOutcome)
		process.AllocationTarget = strings.TrimSpace(correction.AllocationTarget)
		process.DestructionCost = correction.DestructionCost
		beforeProcess := process
		process.TransferType = strings.TrimSpace(correction.TransferType)
		correctedProcesses[processID] = process
		changeDetails = append(changeDetails, dispositionCorrectionChanges(beforeProcess, process)...)
		seenProcesses[process.ID] = true
	}
	for id := range processByID {
		if !seenProcesses[id] {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
	}
	if len(changeDetails) == 0 {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}

	now := m.now().UTC()
	corrected.UpdatedAt = now
	m.items[current.ID] = corrected
	m.events[current.ID] = correctedEvents
	for processID, process := range correctedProcesses {
		process.UpdatedAt = now
		m.dispositions[processID] = process
	}

	m.nextEvent++
	notes := "Data barang diperbarui melalui rekonsiliasi. Alasan perubahan: " + input.Reason + "."
	m.events[current.ID] = append(m.events[current.ID], domain.TimelineEvent{
		ID: fmt.Sprintf("evt-%03d", m.nextEvent), InventoryID: current.ID, Code: "perubahan_data_barang",
		Label: "Perubahan data barang", Notes: notes, Actor: input.Actor, CreatedAt: now,
		Attachments: m.documentAttachments(input.DocumentID),
	})
	m.nextReconciliation++
	record := domain.ReconciliationRecord{
		ID: fmt.Sprintf("rec-%03d", m.nextReconciliation), Type: "data_correction", Action: "updated",
		InventoryID: current.ID, InventoryReference: corrected.ReferenceNo, InventoryType: corrected.Type,
		PreviousStatusCode: current.StatusCode, PreviousStatusLabel: current.StatusLabel,
		ResultStatusCode: corrected.StatusCode, ResultStatusLabel: corrected.StatusLabel,
		CorrectionReason: input.Reason, ChangeDetails: changeDetails,
		Notes: notes, Actor: input.Actor, CreatedAt: now,
	}
	m.reconciliations = append(m.reconciliations, record)
	return record, corrected, nil
}
