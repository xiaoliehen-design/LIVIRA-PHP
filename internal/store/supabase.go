package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

type SupabaseStore struct {
	baseURL       string
	projectURL    string
	serviceKey    string
	storageBucket string
	client        *http.Client
}

func NewSupabaseStore(baseURL, serviceKey, storageBucket string) *SupabaseStore {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	baseURL = strings.TrimSuffix(baseURL, "/rest/v1")
	return &SupabaseStore{
		baseURL:       baseURL + "/rest/v1",
		projectURL:    baseURL,
		serviceKey:    strings.TrimSpace(serviceKey),
		storageBucket: strings.TrimSpace(storageBucket),
		client:        &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *SupabaseStore) doJSON(ctx context.Context, method, resource string, query url.Values, body any, out any) error {
	endpoint := s.baseURL + "/" + strings.TrimLeft(resource, "/")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete && out != nil {
		req.Header.Set("Prefer", "return=representation")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("supabase %s %s: status %d: %s", method, resource, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out != nil && len(payload) > 0 {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("decode supabase response: %w", err)
		}
	}
	return nil
}

func (s *SupabaseStore) CreateDocument(ctx context.Context, input domain.NewDocumentInput) (domain.DocumentAttachment, error) {
	if strings.TrimSpace(input.FileName) == "" || strings.TrimSpace(input.MIMEType) == "" || len(input.Content) == 0 || input.SizeBytes != int64(len(input.Content)) {
		return domain.DocumentAttachment{}, ErrInvalidTransition
	}
	hash := sha256.Sum256(input.Content)
	checksum := hex.EncodeToString(hash[:])
	payload := map[string]any{"file_name": input.FileName, "mime_type": input.MIMEType, "size_bytes": input.SizeBytes, "sha256": checksum, "uploaded_by": input.UploadedBy}
	var objectPath string
	if s.storageBucket != "" {
		objectPath = documentObjectPath(input.MIMEType)
		if err := s.uploadStorageObject(ctx, s.storageBucket, objectPath, input.MIMEType, input.Content); err != nil {
			return domain.DocumentAttachment{}, fmt.Errorf("upload private document: %w", err)
		}
		payload["storage_bucket"], payload["storage_path"] = s.storageBucket, objectPath
	} else {
		payload["content_base64"] = base64.StdEncoding.EncodeToString(input.Content)
	}
	query := url.Values{"select": {"id,file_name,mime_type,size_bytes,uploaded_by,storage_bucket,storage_path,sha256,created_at"}}
	var created []domain.DocumentAttachment
	if err := s.doJSON(ctx, http.MethodPost, "uploaded_documents", query, payload, &created); err != nil {
		if objectPath != "" {
			_ = s.deleteStorageObject(context.Background(), s.storageBucket, objectPath)
		}
		return domain.DocumentAttachment{}, err
	}
	if len(created) == 0 {
		if objectPath != "" {
			_ = s.deleteStorageObject(context.Background(), s.storageBucket, objectPath)
		}
		return domain.DocumentAttachment{}, ErrNotFound
	}
	return created[0], nil
}

func (s *SupabaseStore) GetDocument(ctx context.Context, id string) (domain.DocumentAttachment, []byte, error) {
	var rows []struct {
		domain.DocumentAttachment
		ContentBase64 string `json:"content_base64"`
	}
	query := url.Values{"select": {"id,file_name,mime_type,size_bytes,uploaded_by,storage_bucket,storage_path,sha256,created_at,content_base64"}, "id": {"eq." + id}, "limit": {"1"}}
	if err := s.doJSON(ctx, http.MethodGet, "uploaded_documents", query, nil, &rows); err != nil {
		return domain.DocumentAttachment{}, nil, err
	}
	if len(rows) == 0 {
		return domain.DocumentAttachment{}, nil, ErrNotFound
	}
	var content []byte
	var err error
	if rows[0].StoragePath != "" && rows[0].StorageBucket != "" {
		content, err = s.downloadStorageObject(ctx, rows[0].StorageBucket, rows[0].StoragePath)
	} else {
		content, err = base64.StdEncoding.DecodeString(rows[0].ContentBase64)
	}
	if err != nil {
		return domain.DocumentAttachment{}, nil, err
	}
	if int64(len(content)) != rows[0].SizeBytes {
		return domain.DocumentAttachment{}, nil, errors.New("ukuran dokumen tersimpan tidak konsisten")
	}
	if rows[0].SHA256 != "" {
		hash := sha256.Sum256(content)
		if !strings.EqualFold(hex.EncodeToString(hash[:]), rows[0].SHA256) {
			return domain.DocumentAttachment{}, nil, errors.New("checksum dokumen tidak sesuai")
		}
	}
	return rows[0].DocumentAttachment, content, nil
}

func documentObjectPath(mimeType string) string {
	var random [16]byte
	_, _ = rand.Read(random[:])
	ext := map[string]string{"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}[mimeType]
	return time.Now().UTC().Format("2006/01/02") + "/" + hex.EncodeToString(random[:]) + ext
}

func escapeObjectPath(value string) string {
	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (s *SupabaseStore) storageRequest(ctx context.Context, method, bucket, objectPath, mimeType string, body io.Reader) ([]byte, error) {
	endpoint := s.projectURL + "/storage/v1/object/" + url.PathEscape(bucket) + "/" + escapeObjectPath(objectPath)
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	if mimeType != "" {
		req.Header.Set("Content-Type", mimeType)
	}
	if method == http.MethodPost {
		req.Header.Set("x-upsert", "false")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 9<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("storage status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
func (s *SupabaseStore) uploadStorageObject(ctx context.Context, bucket, path, mime string, content []byte) error {
	_, err := s.storageRequest(ctx, http.MethodPost, bucket, path, mime, bytes.NewReader(content))
	return err
}
func (s *SupabaseStore) downloadStorageObject(ctx context.Context, bucket, path string) ([]byte, error) {
	return s.storageRequest(ctx, http.MethodGet, bucket, path, "", nil)
}
func (s *SupabaseStore) deleteStorageObject(ctx context.Context, bucket, path string) error {
	_, err := s.storageRequest(ctx, http.MethodDelete, bucket, path, "", nil)
	return err
}

func (s *SupabaseStore) DocumentAccess(ctx context.Context, documentID string) ([]domain.DocumentAccess, error) {
	var links []struct {
		InventoryID     string `json:"inventory_id"`
		DispositionType string `json:"disposition_type"`
		Code            string `json:"code"`
	}
	q := url.Values{"select": {"inventory_id,disposition_type,code"}, "document_id": {"eq." + documentID}, "limit": {"100"}}
	if err := s.doJSON(ctx, http.MethodGet, "events", q, nil, &links); err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, ErrNotFound
	}
	ids, seen := []string{}, map[string]bool{}
	for _, l := range links {
		if l.InventoryID != "" && !seen[l.InventoryID] {
			seen[l.InventoryID] = true
			ids = append(ids, l.InventoryID)
		}
	}
	var items []domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodGet, "inventory_items", url.Values{"select": {"*"}, "id": {"in.(" + strings.Join(ids, ",") + ")"}}, nil, &items); err != nil {
		return nil, err
	}
	byID := map[string]domain.InventoryItem{}
	for _, i := range items {
		byID[i.ID] = i
	}
	result := []domain.DocumentAccess{}
	for _, l := range links {
		if i, ok := byID[l.InventoryID]; ok {
			result = append(result, domain.DocumentAccess{Inventory: i, DispositionType: l.DispositionType, EventCode: l.Code})
		}
	}
	if len(result) == 0 {
		return nil, ErrNotFound
	}
	return result, nil
}

func (s *SupabaseStore) NotificationSummary(ctx context.Context, allowed []domain.InventoryType) (domain.NotificationSummary, error) {
	types := []string{}
	for _, t := range allowed {
		types = append(types, string(t))
	}
	var out domain.NotificationSummary
	err := s.doJSON(ctx, http.MethodPost, "rpc/livira_notification_summary", nil, map[string]any{"p_types": types}, &out)
	return out, err
}

func (s *SupabaseStore) WriteAudit(ctx context.Context, e domain.AuditEntry) error {
	metadata := e.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return s.doJSON(ctx, http.MethodPost, "audit_logs", nil, map[string]any{"actor_subject": e.ActorSubject, "actor_name": e.ActorName, "action": e.Action, "entity_type": e.EntityType, "entity_id": e.EntityID, "outcome": e.Outcome, "ip_address": e.IPAddress, "user_agent": e.UserAgent, "request_id": e.RequestID, "metadata": metadata}, nil)
}

func (s *SupabaseStore) Facilities(ctx context.Context) ([]domain.Facility, error) {
	query := url.Values{"select": {"id,name,active,sort_order,yard_capacity,yard_used,shed_capacity,shed_used"}, "active": {"eq.true"}, "order": {"sort_order.asc"}}
	var facilities []domain.Facility
	if err := s.doJSON(ctx, http.MethodGet, "facilities", query, nil, &facilities); err != nil {
		return nil, err
	}
	return facilities, nil
}

func (s *SupabaseStore) UpdateFacilityCapacity(ctx context.Context, id string, yardCapacity, shedCapacity float64) (domain.Facility, error) {
	if strings.TrimSpace(id) == "" || yardCapacity < 0 || shedCapacity < 0 {
		return domain.Facility{}, ErrInvalidTransition
	}
	query := url.Values{"id": {"eq." + id}, "active": {"eq.true"}, "select": {"id,name,active,sort_order,yard_capacity,yard_used,shed_capacity,shed_used"}}
	var updated []domain.Facility
	payload := map[string]any{"yard_capacity": yardCapacity, "shed_capacity": shedCapacity}
	if err := s.doJSON(ctx, http.MethodPatch, "facilities", query, payload, &updated); err != nil {
		return domain.Facility{}, err
	}
	if len(updated) == 0 {
		return domain.Facility{}, ErrNotFound
	}
	return updated[0], nil
}

func (s *SupabaseStore) Dashboard(ctx context.Context) (domain.DashboardStats, error) {
	var stats domain.DashboardStats
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_dashboard_summary", nil, map[string]any{}, &stats); err != nil {
		return domain.DashboardStats{}, err
	}

	// Compatibility guard for databases that still use the dashboard RPC from
	// migration 019. That RPC counted TITIPAN in active_total, but did not expose
	// titipan_total, titipan_summary, or the per-facility TITIPAN breakdown. When
	// the aggregate does not reconcile, recover only the missing TITIPAN rows.
	categoryTotal := stats.BTDTotal + stats.BDNTotal + stats.BMMNTotal + stats.TitipanTotal
	if stats.ActiveTotal != categoryTotal {
		filter := domain.InventoryFilter{Type: domain.InventoryTitipan}
		titipanTotal, err := s.CountInventory(ctx, filter)
		if err != nil {
			return domain.DashboardStats{}, fmt.Errorf("sinkronisasi dashboard barang titipan: %w", err)
		}
		if titipanTotal > 0 {
			filter.Limit = titipanTotal
			titipanItems, err := s.ListInventory(ctx, filter)
			if err != nil {
				return domain.DashboardStats{}, fmt.Errorf("memuat metrik dashboard barang titipan: %w", err)
			}
			stats.TitipanTotal = len(titipanItems)
			stats.TitipanSummary = domain.SummarizeDashboardInventory(titipanItems)

			byFacility := make(map[string]int)
			for _, item := range titipanItems {
				byFacility[item.FacilityID]++
			}
			for index := range stats.FacilityBreakdown {
				stats.FacilityBreakdown[index].Titipan = byFacility[stats.FacilityBreakdown[index].FacilityID]
			}
		}
	}

	// Keep the headline and every facility row mathematically tied to the four
	// inventory cards, so future RPC changes cannot create a visible mismatch.
	stats.ActiveTotal = stats.BTDTotal + stats.BDNTotal + stats.BMMNTotal + stats.TitipanTotal
	for index := range stats.FacilityBreakdown {
		row := &stats.FacilityBreakdown[index]
		row.Total = row.BTD + row.BDN + row.BMMN + row.Titipan
	}
	return stats, nil
}

func inventoryQuery(filter domain.InventoryFilter) (url.Values, bool) {
	query := url.Values{"select": {"*"}}
	if filter.OnlyInactive {
		query.Set("is_active", "eq.false")
	} else if !filter.IncludeInactive {
		query.Set("is_active", "eq.true")
	}
	if !filter.DateFrom.IsZero() {
		query.Add("determination_date", "gte."+filter.DateFrom.Format("2006-01-02"))
	}
	if !filter.DateTo.IsZero() {
		query.Add("determination_date", "lt."+filter.DateTo.AddDate(0, 0, 1).Format("2006-01-02"))
	}
	if !filter.AgeBefore.IsZero() {
		query.Add("determination_date", "lt."+filter.AgeBefore.AddDate(0, 0, 1).Format("2006-01-02"))
	}
	if filter.FacilityID != "" {
		query.Set("facility_id", "eq."+filter.FacilityID)
	}
	if filter.Type != "" {
		if len(filter.AllowedTypes) > 0 && !inventoryTypeAllowed(filter.Type, filter.AllowedTypes) {
			return query, false
		}
		query.Set("item_type", "eq."+string(filter.Type))
	} else if len(filter.AllowedTypes) > 0 {
		values := []string{}
		for _, k := range filter.AllowedTypes {
			values = append(values, string(k))
		}
		query.Set("item_type", "in.("+strings.Join(values, ",")+")")
	}
	if filter.Status != "" {
		query.Set("status_code", "eq."+filter.Status)
	}
	if filter.ItemKind != "" {
		query.Set("item_kind", "eq."+filter.ItemKind)
	}
	if filter.GoodsCondition != "" {
		query.Set("goods_condition", "eq."+filter.GoodsCondition)
	}
	if filter.Category != "" {
		query.Set("category", "eq."+filter.Category)
	}
	if filter.AllocationPurpose != "" {
		query.Set("allocation_purpose", "ilike."+filter.AllocationPurpose)
	}
	if filter.LocationScope == "tpp" {
		query.Set("at_tpp", "eq.true")
	} else if filter.LocationScope == "tps" {
		query.Set("at_tpp", "eq.false")
	}
	if filter.MinValue > 0 {
		query.Add("goods_value", "gte."+strconv.FormatInt(filter.MinValue, 10))
	}
	if filter.MaxValue > 0 {
		query.Add("goods_value", "lte."+strconv.FormatInt(filter.MaxValue, 10))
	}
	switch filter.Preset {
	case "overdue_60":
		if filter.Type != "" && filter.Type != domain.InventoryBTD && filter.Type != domain.InventoryBDN {
			return query, false
		}
		if filter.Type == "" {
			eligible := []string{}
			for _, candidate := range []domain.InventoryType{domain.InventoryBTD, domain.InventoryBDN} {
				if len(filter.AllowedTypes) == 0 || inventoryTypeAllowed(candidate, filter.AllowedTypes) {
					eligible = append(eligible, string(candidate))
				}
			}
			if len(eligible) == 0 {
				return query, false
			}
			query.Set("item_type", "in.("+strings.Join(eligible, ",")+")")
		}
		if filter.Status != "" && filter.Status != "masih_di_tps" && filter.Status != "ditetapkan" {
			return query, false
		}
		if filter.Status == "" {
			query.Set("status_code", "in.(masih_di_tps,ditetapkan)")
		}
	case "auction_ready":
		query.Set("current_disposition", "is.null")
		if filter.MinValue <= 0 {
			query.Set("goods_value", "gt.0")
		}
		query.Set("or", "(status_code.eq.penelitian_pfpd,item_type.eq.BMMN)")
	case "bmmn_allocation":
		if filter.Type != "" && filter.Type != domain.InventoryBMMN {
			return query, false
		}
		query.Set("item_type", "eq.BMMN")
		query.Set("current_disposition", "is.null")
	}
	if strings.TrimSpace(filter.Query) != "" {
		term := postgRESTSearchTerm(filter.Query)
		if term == "" {
			return query, false
		}
		query.Set("search_text", "ilike.*"+term+"*")
	}
	return query, true
}

func (s *SupabaseStore) ListInventory(ctx context.Context, filter domain.InventoryFilter) ([]domain.InventoryItem, error) {
	query, valid := inventoryQuery(filter)
	if !valid {
		return []domain.InventoryItem{}, nil
	}
	switch filter.Sort {
	case "oldest":
		query.Set("order", "determination_date.asc")
	case "determination_newest":
		query.Set("order", "determination_date.desc")
	case "container_asc":
		query.Set("order", "container_no.asc")
	case "container_desc":
		query.Set("order", "container_no.desc")
	case "tpp":
		query.Set("order", "facility_name.asc,updated_at.desc")
	case "value_desc":
		query.Set("order", "goods_value.desc,updated_at.desc")
	case "value_asc":
		query.Set("order", "goods_value.asc,updated_at.desc")
	default:
		query.Set("order", "updated_at.desc")
	}
	if filter.Offset > 0 {
		query.Set("offset", strconv.Itoa(filter.Offset))
	}
	if filter.Limit <= 1000 {
		if filter.Limit > 0 {
			query.Set("limit", strconv.Itoa(filter.Limit))
		}
		var items []domain.InventoryItem
		if err := s.doJSON(ctx, http.MethodGet, "inventory_items", query, nil, &items); err != nil {
			return nil, err
		}
		return items, nil
	}
	items := make([]domain.InventoryItem, 0, filter.Limit)
	baseOffset := filter.Offset
	for offset := 0; offset < filter.Limit; offset += 1000 {
		pageSize := 1000
		if remaining := filter.Limit - offset; remaining < pageSize {
			pageSize = remaining
		}
		query.Set("limit", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(baseOffset+offset))
		var page []domain.InventoryItem
		if err := s.doJSON(ctx, http.MethodGet, "inventory_items", query, nil, &page); err != nil {
			return nil, err
		}
		items = append(items, page...)
		if len(page) < pageSize {
			break
		}
	}
	return items, nil
}

func (s *SupabaseStore) CountInventory(ctx context.Context, filter domain.InventoryFilter) (int, error) {
	query, valid := inventoryQuery(filter)
	if !valid {
		return 0, nil
	}
	query.Set("select", "id")
	query.Set("limit", "1")
	endpoint := s.baseURL + "/inventory_items?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Prefer", "count=exact")
	req.Header.Set("Range", "0-0")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("supabase count inventory: status %d", resp.StatusCode)
	}
	contentRange := resp.Header.Get("Content-Range")
	slash := strings.LastIndex(contentRange, "/")
	if slash < 0 {
		return 0, fmt.Errorf("content-range tidak tersedia")
	}
	total, err := strconv.Atoi(strings.TrimSpace(contentRange[slash+1:]))
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *SupabaseStore) InventorySummary(ctx context.Context, filter domain.InventoryFilter) (domain.InventorySummary, error) {
	queryText := ""
	if strings.TrimSpace(filter.Query) != "" {
		queryText = postgRESTSearchTerm(filter.Query)
		if queryText == "" {
			return domain.InventorySummary{}, nil
		}
	}
	allowedTypes := make([]string, 0, len(filter.AllowedTypes))
	for _, itemType := range filter.AllowedTypes {
		allowedTypes = append(allowedTypes, string(itemType))
	}
	dateString := func(value time.Time) any {
		if value.IsZero() {
			return nil
		}
		return value.Format("2006-01-02")
	}
	payload := map[string]any{
		"p_query":              queryText,
		"p_types":              allowedTypes,
		"p_facility_id":        filter.FacilityID,
		"p_item_type":          string(filter.Type),
		"p_status":             filter.Status,
		"p_item_kind":          filter.ItemKind,
		"p_goods_condition":    filter.GoodsCondition,
		"p_category":           filter.Category,
		"p_allocation_purpose": filter.AllocationPurpose,
		"p_location_scope":     filter.LocationScope,
		"p_include_inactive":   filter.IncludeInactive,
		"p_only_inactive":      filter.OnlyInactive,
		"p_date_from":          dateString(filter.DateFrom),
		"p_date_to":            dateString(filter.DateTo),
		"p_age_before":         dateString(filter.AgeBefore),
		"p_min_value":          filter.MinValue,
		"p_max_value":          filter.MaxValue,
		"p_preset":             filter.Preset,
	}
	var result domain.InventorySummary
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_inventory_summary", nil, payload, &result); err != nil {
		return domain.InventorySummary{}, err
	}
	return result, nil
}

func (s *SupabaseStore) GetInventory(ctx context.Context, id string) (domain.InventoryItem, error) {
	query := url.Values{"select": {"*"}, "id": {"eq." + id}, "limit": {"1"}}
	var items []domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodGet, "inventory_items", query, nil, &items); err != nil {
		return domain.InventoryItem{}, err
	}
	if len(items) == 0 {
		return domain.InventoryItem{}, ErrNotFound
	}
	return items[0], nil
}

func (s *SupabaseStore) DeleteInventory(ctx context.Context, id, actor string) error {
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	payload := map[string]any{
		"p_inventory_id": id,
		"p_deleted_by":   strings.TrimSpace(actor),
	}
	if err := s.doJSON(ctx, http.MethodPost, "rpc/admin_delete_inventory", nil, payload, nil); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "inventory_not_found") {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *SupabaseStore) CreateInventory(ctx context.Context, input domain.NewInventoryInput) (domain.InventoryItem, error) {
	items, err := s.CreateInventories(ctx, []domain.NewInventoryInput{input})
	if err != nil {
		return domain.InventoryItem{}, err
	}
	return items[0], nil
}

func (s *SupabaseStore) CreateInventories(ctx context.Context, inputs []domain.NewInventoryInput) ([]domain.InventoryItem, error) {
	if len(inputs) == 0 {
		return nil, ErrInvalidTransition
	}
	facilities, err := s.Facilities(ctx)
	if err != nil {
		return nil, err
	}
	facilityNames := make(map[string]string, len(facilities))
	for _, facility := range facilities {
		facilityNames[facility.ID] = facility.Name
	}

	now := time.Now().UTC()
	payloads := make([]map[string]any, 0, len(inputs))
	normalizedInputs := make([]domain.NewInventoryInput, 0, len(inputs))
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
		if input.DeterminationDate.IsZero() {
			input.DeterminationDate = now
		}
		validType := input.Type == domain.InventoryBTD || input.Type == domain.InventoryBDN || input.Type == domain.InventoryTitipan || input.ReconciliationCreated && input.Type == domain.InventoryBMMN
		validInitialProcess := input.InitialDispositionType == "" || input.ReconciliationCreated && input.Type != domain.InventoryTitipan && (input.InitialDispositionType == domain.DispositionAuction || input.InitialDispositionType == domain.DispositionDestruction || input.InitialDispositionType == domain.DispositionGrant)
		if !validType || !validInitialProcess || input.DeterminationNo == "" || strings.TrimSpace(input.Description) == "" || input.Quantity <= 0 ||
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
		facilityName := ""
		if input.AtTPP {
			facilityName = facilityNames[input.FacilityID]
			if facilityName == "" {
				return nil, ErrInvalidTransition
			}
		} else if input.Type != domain.InventoryTitipan && input.OriginWarehouse == "" && strings.TrimSpace(input.Location) == "" {
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
		if _, duplicate := seenReferences[input.ReferenceNo]; duplicate {
			return nil, ErrConflict
		}
		seenReferences[input.ReferenceNo] = struct{}{}

		location := strings.TrimSpace(input.Location)
		locationStatus := strings.TrimSpace(input.OriginWarehouse)
		statusCode := "masih_di_tps"
		statusLabel := "Masih di TPS"
		var facilityID any
		if input.Type == domain.InventoryTitipan {
			locationStatus = input.SourceOffice
			statusCode = "barang_titipan_aktif"
			statusLabel = "Barang titipan aktif"
			if location == "" {
				location = input.SourceOffice
			}
		}
		if input.AtTPP {
			facilityID = input.FacilityID
			locationStatus = facilityName
			statusCode = "ditetapkan"
			statusLabel = "Ditetapkan sebagai " + domain.InventoryTypeLabel(input.Type)
			if input.Type == domain.InventoryTitipan {
				statusCode = "barang_titipan_aktif"
				statusLabel = "Barang titipan aktif"
			}
			if location == "" {
				location = facilityName
			}
		} else {
			facilityID = nil
			if location == "" {
				if input.Type == domain.InventoryTitipan {
					location = input.SourceOffice
				} else {
					location = input.OriginWarehouse
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
		physicalUnitID := strings.TrimSpace(input.PhysicalUnitID)
		if physicalUnitID == "" {
			physicalUnitID = input.ReferenceNo
			input.OccupancyPrimary = true
		}
		normalizedInputs = append(normalizedInputs, input)
		payloads = append(payloads, map[string]any{
			"reference_no": input.ReferenceNo, "item_type": input.Type, "origin_type": input.Type,
			"bl_no": input.BLNo, "bl_date": nullableTime(input.BLDate), "manifest_no": input.ManifestNo, "manifest_date": nullableTime(input.ManifestDate), "manifest_position": input.ManifestPosition,
			"determination_no": input.DeterminationNo, "determination_date": input.DeterminationDate,
			"category": input.Category, "entrusted_category": input.EntrustedCategory, "source_office": input.SourceOffice,
			"description": input.Description, "item_kind": input.ItemKind, "quantity": input.Quantity, "quantity_detail": input.QuantityDetail, "unit": input.Unit, "goods_value": input.GoodsValue, "goods_condition": input.GoodsCondition,
			"location": location, "location_status": locationStatus, "at_tpp": input.AtTPP, "owner_name": input.OwnerName, "owner_address": input.OwnerAddress,
			"origin_warehouse": input.OriginWarehouse, "facility_id": facilityID, "facility_name": facilityName,
			"load_type": input.LoadType, "container_no": input.ContainerNo, "container_size": input.ContainerSize, "estimated_volume_m3": input.EstimatedVolumeM3,
			"physical_unit_id": physicalUnitID, "occupancy_primary": input.OccupancyPrimary, "pfpd_required": input.PFPDRequired || input.Type != domain.InventoryTitipan,
			"restriction_rule": input.RestrictionRule, "status_code": statusCode, "status_label": statusLabel, "is_active": true, "created_by": input.Actor,
		})
	}

	for index := range payloads {
		input := normalizedInputs[index]
		payloads[index]["_initial_disposition_type"] = string(input.InitialDispositionType)
		payloads[index]["_initial_transfer_type"] = input.InitialTransferType
		payloads[index]["_initial_status_code"] = input.InitialStatusCode
		payloads[index]["_reconciliation_created"] = input.ReconciliationCreated
		payloads[index]["_document_id"] = input.DocumentID
	}
	var created []domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_create_inventories", nil, map[string]any{"p_rows": payloads}, &created); err != nil {
		return nil, mapRPCError(err)
	}
	if len(created) != len(payloads) {
		return nil, ErrNotFound
	}
	return created, nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func mapRPCError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "record changed by another user"), strings.Contains(message, "sqlstate 40001"), strings.Contains(message, `"code":"40001"`):
		return ErrConcurrentUpdate
	case strings.Contains(message, "inventory is inactive"):
		return ErrInactiveInventory
	case strings.Contains(message, "active disposition already exists"), strings.Contains(message, "duplicate key"):
		return ErrConflict
	case strings.Contains(message, "invalid transition"), strings.Contains(message, "invalid disposition type"):
		return ErrInvalidTransition
	case strings.Contains(message, "not found"):
		return ErrNotFound
	default:
		return err
	}
}

func (s *SupabaseStore) AddInventoryEvent(ctx context.Context, id string, input domain.NewEventInput) (domain.InventoryItem, error) {
	item, err := s.GetInventory(ctx, id)
	if err != nil {
		return domain.InventoryItem{}, err
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
	patch := map[string]any{"status_code": input.Code, "status_label": input.Label}
	switch input.Code {
	case "pemindahan":
		facilities, err := s.Facilities(ctx)
		if err != nil {
			return domain.InventoryItem{}, err
		}
		facilityName := ""
		for _, facility := range facilities {
			if facility.ID == input.TargetFacilityID {
				facilityName = facility.Name
				break
			}
		}
		if facilityName == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["at_tpp"] = true
		patch["facility_id"] = input.TargetFacilityID
		patch["facility_name"] = facilityName
		patch["location"] = facilityName
		patch["location_status"] = facilityName
	case "pencacahan":
		if strings.TrimSpace(input.Description) == "" || !domain.ValidItemKind(strings.TrimSpace(input.ItemKind)) || input.Quantity <= 0 || !domain.ValidUnit(strings.TrimSpace(input.Unit)) || !domain.ValidGoodsCondition(strings.TrimSpace(input.GoodsCondition)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["description"] = strings.TrimSpace(input.Description)
		patch["item_kind"] = strings.TrimSpace(input.ItemKind)
		patch["quantity"] = input.Quantity
		patch["unit"] = strings.TrimSpace(input.Unit)
		patch["goods_condition"] = strings.TrimSpace(input.GoodsCondition)
		patch["pfpd_required"] = true
		patch["research_request_no"] = ""
		patch["research_request_date"] = nil
		patch["hs_code"] = ""
		patch["is_restricted"] = false
		patch["restriction_rule"] = ""
	case "request_penelitian_pfpd":
		patch["research_request_no"] = input.DocumentNo
		patch["research_request_date"] = nullableTime(input.DocumentDate)
	case "penelitian_pfpd":
		if item.ResearchRequestNo == "" || strings.TrimSpace(input.HSCode) == "" || input.GoodsValue <= 0 || input.RestrictionStatus != "ya" && input.RestrictionStatus != "tidak" || input.IsRestricted && strings.TrimSpace(input.RestrictionRule) == "" {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["hs_code"] = strings.TrimSpace(input.HSCode)
		patch["is_restricted"] = input.IsRestricted
		patch["restriction_rule"] = strings.TrimSpace(input.RestrictionRule)
		patch["goods_value"] = input.GoodsValue
	case "penetapan_bmmn":
		if item.Type == domain.InventoryBMMN || item.Type == domain.InventoryTitipan {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["origin_document_type"] = originDocumentType(item.Type)
		patch["origin_document_no"] = item.DeterminationNo
		patch["origin_document_date"] = nullableTime(item.DeterminationDate)
		patch["determination_no"] = input.DocumentNo
		patch["determination_date"] = nullableTime(input.DocumentDate)
		patch["item_type"] = domain.InventoryBMMN
		patch["status_code"] = "bmmn_aktif"
		patch["status_label"] = "Ditetapkan sebagai BMMN"
	case "usulan_peruntukan_bmmn":
		if item.Type != domain.InventoryBMMN || !domain.ValidAllocationPurpose(strings.TrimSpace(input.AllocationType)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["allocation_proposal_type"] = strings.TrimSpace(input.AllocationType)
		patch["allocation_proposal_no"] = input.DocumentNo
		patch["allocation_proposal_date"] = nullableTime(input.DocumentDate)
		patch["allocation_purpose"] = strings.TrimSpace(input.AllocationType)
	case "persetujuan_peruntukan_bmmn":
		if item.Type != domain.InventoryBMMN || item.AllocationProposalNo == "" || !domain.ValidAllocationPurpose(strings.TrimSpace(input.AllocationType)) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["allocation_approval_type"] = strings.TrimSpace(input.AllocationType)
		patch["allocation_approval_no"] = input.DocumentNo
		patch["allocation_approval_date"] = nullableTime(input.DocumentDate)
		patch["allocation_purpose"] = strings.TrimSpace(input.AllocationType)
	case "pengeluaran_barang":
		if !validInventoryExit(item, input.ExitType) {
			return domain.InventoryItem{}, ErrInvalidTransition
		}
		patch["exit_document_no"] = input.DocumentNo
		patch["exit_document_date"] = nullableTime(input.DocumentDate)
		patch["exit_type"] = input.ExitType
		patch["exit_notes"] = strings.TrimSpace(input.ExitNotes)
		patch["is_active"] = false
		keepDestructionOpen := input.ExitType == "musnah" && item.CurrentDisposition == domain.DispositionDestruction
		if !keepDestructionOpen {
			patch["current_disposition"] = nil
		}
		patch["location_status"] = "Barang telah dikeluarkan"
		patch["status_code"] = "pengeluaran_barang"
		patch["status_label"] = "Pengeluaran barang selesai"
	}

	event := map[string]any{
		"code":          input.Code,
		"label":         patch["status_label"],
		"document_no":   input.DocumentNo,
		"document_date": nullableTime(input.DocumentDate),
		"notes":         input.Notes,
		"actor":         input.Actor,
		"document_id":   nullableString(input.DocumentID),
	}
	keepDestructionOpen := input.Code == "pengeluaran_barang" && input.ExitType == "musnah" && item.CurrentDisposition == domain.DispositionDestruction
	payload := map[string]any{
		"p_inventory_id":              id,
		"p_expected_updated_at":       item.UpdatedAt,
		"p_item_patch":                patch,
		"p_close_active_dispositions": input.Code == "pengeluaran_barang",
		"p_keep_destruction_open":     keepDestructionOpen,
		"p_event":                     event,
	}
	var updated domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_apply_inventory_event", nil, payload, &updated); err != nil {
		return domain.InventoryItem{}, mapRPCError(err)
	}
	if updated.ID == "" {
		return domain.InventoryItem{}, ErrNotFound
	}
	return updated, nil
}

func (s *SupabaseStore) ApplyInventoryCensus(ctx context.Context, id string, lines []domain.InventoryGoodsLine, input domain.NewEventInput) ([]domain.InventoryItem, error) {
	item, err := s.GetInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	if !item.IsActive || item.CurrentDisposition != "" || completedProcessStatus(item.StatusCode) || strings.TrimSpace(input.DocumentNo) == "" || input.DocumentDate.IsZero() || len(lines) == 0 || len(lines) > 100 {
		return nil, ErrInvalidTransition
	}

	groupID := strings.TrimSpace(item.PhysicalUnitID)
	if groupID == "" {
		groupID = item.ID
	}

	existing := []domain.InventoryItem{item}
	if item.LoadType == "FCL" {
		query := url.Values{
			"select":           {"*"},
			"physical_unit_id": {"eq." + groupID},
			"is_active":        {"eq.true"},
			"order":            {"created_at.asc"},
		}
		if err := s.doJSON(ctx, http.MethodGet, "inventory_items", query, nil, &existing); err != nil {
			return nil, err
		}
		if len(existing) == 0 {
			existing = []domain.InventoryItem{item}
		}
	}

	existingByID := make(map[string]domain.InventoryItem, len(existing))
	for _, current := range existing {
		existingByID[current.ID] = current
	}
	seenExisting := make(map[string]bool, len(existing))
	for _, line := range lines {
		line.InventoryID = strings.TrimSpace(line.InventoryID)
		line.Description = strings.TrimSpace(line.Description)
		line.ItemKind = strings.TrimSpace(line.ItemKind)
		line.Unit = strings.TrimSpace(line.Unit)
		line.GoodsCondition = strings.TrimSpace(line.GoodsCondition)
		if line.Description == "" || !domain.ValidItemKind(line.ItemKind) || line.GoodsValue < 0 || line.Quantity <= 0 || !domain.ValidUnit(line.Unit) || !domain.ValidGoodsCondition(line.GoodsCondition) {
			return nil, ErrInvalidTransition
		}
		if line.InventoryID == "" {
			if item.LoadType != "FCL" {
				return nil, ErrInvalidTransition
			}
			continue
		}
		if _, ok := existingByID[line.InventoryID]; !ok || seenExisting[line.InventoryID] {
			return nil, ErrInvalidTransition
		}
		seenExisting[line.InventoryID] = true
	}
	if len(seenExisting) != len(existingByID) {
		return nil, ErrInvalidTransition
	}

	now := time.Now().UTC()
	updated := make([]domain.InventoryItem, 0, len(lines))
	for _, line := range lines {
		if line.InventoryID == "" {
			continue
		}
		patch := map[string]any{
			"description": strings.TrimSpace(line.Description), "item_kind": strings.TrimSpace(line.ItemKind), "goods_value": line.GoodsValue,
			"quantity": line.Quantity, "quantity_detail": strings.TrimSpace(line.QuantityDetail), "unit": strings.TrimSpace(line.Unit), "goods_condition": strings.TrimSpace(line.GoodsCondition),
			"physical_unit_id": groupID, "pfpd_required": true,
			"research_request_no": "", "research_request_date": nil, "hs_code": "", "is_restricted": false, "restriction_rule": "",
			"status_code": "pencacahan", "status_label": "Pencacahan", "updated_at": now,
		}
		var rows []domain.InventoryItem
		if err := s.doJSON(ctx, http.MethodPatch, "inventory_items", url.Values{"id": {"eq." + line.InventoryID}, "select": {"*"}}, patch, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, ErrNotFound
		}
		updated = append(updated, rows[0])
	}

	facilityID := any(nil)
	if item.FacilityID != "" {
		facilityID = item.FacilityID
	}
	stamp := now.UnixNano()
	clonePayloads := make([]map[string]any, 0)
	for index, line := range lines {
		if line.InventoryID != "" {
			continue
		}
		clonePayloads = append(clonePayloads, map[string]any{
			"reference_no": fmt.Sprintf("%s/CACAH-G%02d-%d", item.ReferenceNo, index+1, stamp),
			"item_type":    item.Type, "origin_type": item.OriginType,
			"bl_no": item.BLNo, "bl_date": nullableTime(item.BLDate), "manifest_no": item.ManifestNo, "manifest_date": nullableTime(item.ManifestDate), "manifest_position": item.ManifestPosition,
			"determination_no": item.DeterminationNo, "determination_date": nullableTime(item.DeterminationDate), "category": item.Category,
			"description": strings.TrimSpace(line.Description), "item_kind": strings.TrimSpace(line.ItemKind), "goods_value": line.GoodsValue,
			"quantity": line.Quantity, "quantity_detail": strings.TrimSpace(line.QuantityDetail), "unit": strings.TrimSpace(line.Unit), "goods_condition": strings.TrimSpace(line.GoodsCondition),
			"location": item.Location, "location_status": item.LocationStatus, "at_tpp": item.AtTPP,
			"owner_name": item.OwnerName, "owner_address": item.OwnerAddress, "origin_warehouse": item.OriginWarehouse,
			"facility_id": facilityID, "facility_name": item.FacilityName,
			"load_type": item.LoadType, "container_no": item.ContainerNo, "container_size": item.ContainerSize, "estimated_volume_m3": item.EstimatedVolumeM3,
			"physical_unit_id": groupID, "occupancy_primary": false, "pfpd_required": true,
			"research_request_no": "", "research_request_date": nil, "hs_code": "", "is_restricted": false, "restriction_rule": "",
			"origin_document_type": item.OriginDocumentType, "origin_document_no": item.OriginDocumentNo, "origin_document_date": nullableTime(item.OriginDocumentDate),
			"entrusted_category": item.EntrustedCategory, "source_office": item.SourceOffice,
			"allocation_purpose":       item.AllocationPurpose,
			"allocation_proposal_type": item.AllocationProposalType, "allocation_proposal_no": item.AllocationProposalNo, "allocation_proposal_date": nullableTime(item.AllocationProposalDate),
			"allocation_approval_type": item.AllocationApprovalType, "allocation_approval_no": item.AllocationApprovalNo, "allocation_approval_date": nullableTime(item.AllocationApprovalDate),
			"exit_document_no": "", "exit_document_date": nil, "exit_type": "", "exit_notes": "",
			"status_code": "pencacahan", "status_label": "Pencacahan", "current_disposition": nil, "is_active": true,
			"created_by": input.Actor, "created_at": now, "updated_at": now,
		})
	}
	if len(clonePayloads) > 0 {
		var clones []domain.InventoryItem
		if err := s.doJSON(ctx, http.MethodPost, "inventory_items", url.Values{"select": {"*"}}, clonePayloads, &clones); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil, ErrConflict
			}
			return nil, err
		}
		updated = append(updated, clones...)
	}

	events := make([]map[string]any, 0, len(updated))
	for _, current := range updated {
		events = append(events, map[string]any{
			"inventory_id": current.ID, "code": "pencacahan", "label": "Pencacahan",
			"document_no": input.DocumentNo, "document_date": nullableTime(input.DocumentDate), "notes": input.Notes, "actor": input.Actor, "document_id": nullableString(input.DocumentID),
		})
	}
	if err := s.doJSON(ctx, http.MethodPost, "events", nil, events, nil); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *SupabaseStore) RelocateInventoryLoad(ctx context.Context, input domain.InventoryLoadRelocationInput) ([]domain.InventoryItem, error) {
	item, err := s.GetInventory(ctx, strings.TrimSpace(input.InventoryID))
	if err != nil {
		return nil, err
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
			key := strings.NewReplacer(" ", "", "-", "").Replace(number)
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
	if (item.CurrentDisposition != "" || completedProcessStatus(item.StatusCode)) && len(allocations) > 1 {
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

	payload := map[string]any{
		"p_inventory_id":        item.ID,
		"p_expected_updated_at": item.UpdatedAt,
		"p_allocations":         allocations,
		"p_event": map[string]any{
			"code":          "pindah_bongkar_kontainer",
			"label":         "Bongkar/Muat Kontainer",
			"document_no":   strings.TrimSpace(input.DocumentNo),
			"document_date": input.DocumentDate,
			"notes":         strings.TrimSpace(input.Notes),
			"actor":         strings.TrimSpace(input.Actor),
			"document_id":   nullableString(strings.TrimSpace(input.DocumentID)),
		},
	}
	var updated []domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_relocate_inventory_load", nil, payload, &updated); err != nil {
		return nil, mapRPCError(err)
	}
	if len(updated) != len(allocations) {
		return nil, ErrNotFound
	}
	return updated, nil
}

func (s *SupabaseStore) ListEvents(ctx context.Context, limit int) ([]domain.TimelineEvent, error) {
	if limit <= 0 {
		limit = 50000
	}
	query := url.Values{"select": {"*"}, "order": {"created_at.asc"}}
	result := make([]domain.TimelineEvent, 0, minInt(limit, 1000))
	for offset := 0; offset < limit; offset += 1000 {
		pageSize := 1000
		if remaining := limit - offset; remaining < pageSize {
			pageSize = remaining
		}
		query.Set("limit", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(offset))
		var page []domain.TimelineEvent
		if err := s.doJSON(ctx, http.MethodGet, "events", query, nil, &page); err != nil {
			return nil, err
		}
		for index := range page {
			page[index].Attachments = nil
		}
		result = append(result, page...)
		if len(page) < pageSize {
			break
		}
	}
	return result, nil
}

func (s *SupabaseStore) PerformanceSource(ctx context.Context, from, to time.Time, allowed []domain.InventoryType) ([]domain.InventoryItem, []domain.TimelineEvent, error) {
	types := make([]string, 0, len(allowed))
	for _, itemType := range allowed {
		types = append(types, string(itemType))
	}
	payload := map[string]any{
		"p_from":  from.Format("2006-01-02"),
		"p_to":    to.Format("2006-01-02"),
		"p_types": types,
	}
	var source struct {
		Items  []domain.InventoryItem `json:"items"`
		Events []domain.TimelineEvent `json:"events"`
	}
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_performance_source", nil, payload, &source); err != nil {
		return nil, nil, err
	}
	for index := range source.Events {
		source.Events[index].Attachments = nil
	}
	return source.Items, source.Events, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *SupabaseStore) Timeline(ctx context.Context, inventoryID string) ([]domain.TimelineEvent, error) {
	query := url.Values{"select": {"*"}, "inventory_id": {"eq." + inventoryID}, "order": {"created_at.asc"}}
	var events []domain.TimelineEvent
	if err := s.doJSON(ctx, http.MethodGet, "events", query, nil, &events); err != nil {
		return nil, err
	}
	documentIDs := make([]string, 0)
	seen := make(map[string]bool)
	for _, event := range events {
		if event.DocumentID != "" && !seen[event.DocumentID] {
			seen[event.DocumentID] = true
			documentIDs = append(documentIDs, event.DocumentID)
		}
	}
	if len(documentIDs) == 0 {
		return events, nil
	}
	var documents []domain.DocumentAttachment
	docQuery := url.Values{"select": {"id,file_name,mime_type,size_bytes,uploaded_by,created_at"}, "id": {"in.(" + strings.Join(documentIDs, ",") + ")"}}
	if err := s.doJSON(ctx, http.MethodGet, "uploaded_documents", docQuery, nil, &documents); err != nil {
		return nil, err
	}
	byID := make(map[string]domain.DocumentAttachment, len(documents))
	for _, document := range documents {
		byID[document.ID] = document
	}
	for index := range events {
		if document, ok := byID[events[index].DocumentID]; ok {
			events[index].Attachments = []domain.DocumentAttachment{document}
		}
	}
	return events, nil
}

func (s *SupabaseStore) EligibleInventory(ctx context.Context, queryText string, limit int) ([]domain.InventoryItem, error) {
	items, err := s.ListInventory(ctx, domain.InventoryFilter{Query: queryText, Limit: limit * 3})
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

func dispositionQuery(filter domain.DispositionFilter) (url.Values, bool) {
	query := url.Values{"select": {"*"}}
	if filter.Type != "" {
		query.Set("disposition_type", "eq."+string(filter.Type))
	}
	if filter.InventoryID != "" {
		query.Set("inventory_id", "eq."+filter.InventoryID)
	}
	if filter.Status == "active" {
		query.Set("is_active", "eq.true")
	} else if filter.Status == "completed" {
		query.Set("is_active", "eq.false")
	}
	if len(filter.IncludeStatusCodes) > 0 {
		query.Set("status_code", "in.("+strings.Join(filter.IncludeStatusCodes, ",")+")")
	} else if len(filter.ExcludeStatusCodes) > 0 {
		query.Set("status_code", "not.in.("+strings.Join(filter.ExcludeStatusCodes, ",")+")")
	}
	if filter.OnlyInactiveInventory {
		query.Set("inventory_is_active", "eq.false")
	} else if !filter.IncludeInactiveInventory {
		query.Set("inventory_is_active", "eq.true")
	}
	if filter.FacilityID != "" {
		query.Set("inventory_facility_id", "eq."+filter.FacilityID)
	}
	if len(filter.AllowedTypes) > 0 {
		values := make([]string, 0, len(filter.AllowedTypes))
		for _, itemType := range filter.AllowedTypes {
			values = append(values, string(itemType))
		}
		query.Set("inventory_item_type", "in.("+strings.Join(values, ",")+")")
	}
	if strings.TrimSpace(filter.Query) != "" {
		term := postgRESTSearchTerm(filter.Query)
		if term == "" {
			return query, false
		}
		query.Set("inventory_search_text", "ilike.*"+term+"*")
	}
	switch filter.Sort {
	case "oldest":
		query.Set("order", "updated_at.asc")
	case "determination_newest":
		query.Set("order", "inventory_determination_date.desc")
	case "determination_oldest":
		query.Set("order", "inventory_determination_date.asc")
	case "value_desc":
		if filter.Type == domain.DispositionAuction {
			query.Set("order", "htl_value.desc")
		} else {
			query.Set("order", "inventory_goods_value.desc")
		}
	case "value_asc":
		if filter.Type == domain.DispositionAuction {
			query.Set("order", "htl_value.asc")
		} else {
			query.Set("order", "inventory_goods_value.asc")
		}
	default:
		query.Set("order", "updated_at.desc")
	}
	if filter.Limit > 0 {
		query.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		query.Set("offset", strconv.Itoa(filter.Offset))
	}
	return query, true
}

func (s *SupabaseStore) ListDispositions(ctx context.Context, filter domain.DispositionFilter) ([]domain.Disposition, error) {
	query, valid := dispositionQuery(filter)
	if !valid {
		return []domain.Disposition{}, nil
	}
	var rows []domain.Disposition
	if err := s.doJSON(ctx, http.MethodGet, "disposition_details", query, nil, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *SupabaseStore) CountDispositions(ctx context.Context, filter domain.DispositionFilter) (int, error) {
	filter.Limit = 0
	filter.Offset = 0
	query, valid := dispositionQuery(filter)
	if !valid {
		return 0, nil
	}
	query.Set("select", "id")
	query.Set("limit", "1")
	query.Del("order")
	endpoint := s.baseURL + "/disposition_details?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Prefer", "count=exact")
	req.Header.Set("Range", "0-0")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("supabase count dispositions: status %d", resp.StatusCode)
	}
	contentRange := resp.Header.Get("Content-Range")
	slash := strings.LastIndex(contentRange, "/")
	if slash < 0 {
		return 0, fmt.Errorf("content-range disposition tidak tersedia")
	}
	totalText := strings.TrimSpace(contentRange[slash+1:])
	if totalText == "*" {
		return 0, nil
	}
	total, err := strconv.Atoi(totalText)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *SupabaseStore) ProcessDashboard(ctx context.Context, kind domain.DispositionType, year int, allowed []domain.InventoryType) (domain.ProcessDashboard, error) {
	allowedTypes := make([]string, 0, len(allowed))
	for _, itemType := range allowed {
		allowedTypes = append(allowedTypes, string(itemType))
	}
	payload := map[string]any{
		"p_type":  string(kind),
		"p_year":  year,
		"p_types": allowedTypes,
	}
	var result domain.ProcessDashboard
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_process_dashboard", nil, payload, &result); err != nil {
		return domain.ProcessDashboard{}, err
	}
	return result, nil
}

func (s *SupabaseStore) GetDisposition(ctx context.Context, id string) (domain.Disposition, error) {
	var rows []domain.Disposition
	err := s.doJSON(ctx, http.MethodGet, "disposition_details", url.Values{"select": {"*"}, "id": {"eq." + id}, "limit": {"1"}}, nil, &rows)
	if err != nil {
		return domain.Disposition{}, err
	}
	if len(rows) == 0 {
		return domain.Disposition{}, ErrNotFound
	}
	return rows[0], nil
}

func (s *SupabaseStore) CreateDisposition(ctx context.Context, input domain.NewDispositionInput) (domain.Disposition, error) {
	item, err := s.GetInventory(ctx, input.InventoryID)
	if err != nil {
		return domain.Disposition{}, err
	}
	if !item.IsActive {
		return domain.Disposition{}, ErrInactiveInventory
	}
	if completedDispositionStatus(item.StatusCode) {
		return domain.Disposition{}, ErrInvalidTransition
	}
	if item.CurrentDisposition != "" && !canTransferFailedAuction(item, input.Type) {
		return domain.Disposition{}, ErrConflict
	}
	if input.Type != domain.DispositionAuction && input.Type != domain.DispositionDestruction && input.Type != domain.DispositionGrant {
		return domain.Disposition{}, ErrInvalidTransition
	}
	payload := map[string]any{
		"p_inventory_id":        input.InventoryID,
		"p_disposition_type":    input.Type,
		"p_actor":               strings.TrimSpace(input.Actor),
		"p_notes":               strings.TrimSpace(input.Notes),
		"p_expected_updated_at": item.UpdatedAt,
	}
	var created domain.Disposition
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_create_disposition", nil, payload, &created); err != nil {
		return domain.Disposition{}, mapRPCError(err)
	}
	if created.ID == "" {
		return domain.Disposition{}, ErrNotFound
	}
	return created, nil
}

func (s *SupabaseStore) AddDispositionEvent(ctx context.Context, id string, input domain.NewEventInput) (domain.Disposition, error) {
	process, err := s.GetDisposition(ctx, id)
	if err != nil {
		return domain.Disposition{}, err
	}
	if !process.IsActive {
		return domain.Disposition{}, ErrInvalidTransition
	}
	action, valid := domain.FindDispositionAction(process.Type, input.Code)
	if !valid || validateDispositionTransition(process, input, action) != nil {
		return domain.Disposition{}, ErrInvalidTransition
	}
	input.Label = dispositionStatusLabel(input.Code, action.Label)
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
	processPatch := map[string]any{"round": process.Round, "status_code": statusCode, "status_label": input.Label}
	if input.Code == "selesai_lelang" {
		processPatch["sale_value"] = input.SaleValue
		processPatch["auction_outcome"] = input.AuctionOutcome
	}
	if input.Code == "kep_htl" {
		processPatch["htl_value"] = input.HTLValue
	}
	if input.Code == "jadwal_lelang" {
		processPatch["execution_start_date"] = nullableTime(input.ExecutionStartDate)
		processPatch["execution_end_date"] = nullableTime(input.ExecutionEndDate)
		processPatch["schedule_document_no"] = input.DocumentNo
		processPatch["schedule_document_date"] = nullableTime(input.DocumentDate)
	}
	if input.Code == "alokasi_hasil_lelang" {
		processPatch["allocation_target"] = input.AllocationTarget
	}
	if input.Code == "kep_musnah" || input.Code == "ba_musnah" {
		processPatch["destruction_cost"] = input.DestructionCost
	}
	if input.Code == "ba_serah_terima" {
		processPatch["transfer_type"] = input.TransferType
	}
	if input.RecipientCode != "" {
		processPatch["recipient_code"] = input.RecipientCode
	}
	if input.RecipientName != "" {
		processPatch["recipient_name"] = input.RecipientName
	}
	itemPatch := map[string]any{"status_code": statusCode, "status_label": input.Label}
	if input.Code == "alokasi_hasil_lelang" || input.Code == "ba_musnah" || input.Code == "ba_serah_terima" {
		processPatch["is_active"] = false
		itemPatch["current_disposition"] = nil
	}
	event := map[string]any{
		"code":          input.Code,
		"label":         input.Label,
		"document_no":   input.DocumentNo,
		"document_date": nullableTime(input.DocumentDate),
		"notes":         input.Notes,
		"actor":         input.Actor,
		"document_id":   nullableString(input.DocumentID),
	}
	payload := map[string]any{
		"p_disposition_id":      id,
		"p_expected_updated_at": process.UpdatedAt,
		"p_process_patch":       processPatch,
		"p_item_patch":          itemPatch,
		"p_event":               event,
	}
	var updated domain.Disposition
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_apply_disposition_event", nil, payload, &updated); err != nil {
		return domain.Disposition{}, mapRPCError(err)
	}
	if updated.ID == "" {
		return domain.Disposition{}, ErrNotFound
	}
	return updated, nil
}

func postgRESTSearchTerm(value string) string {
	var builder strings.Builder
	for _, char := range strings.TrimSpace(value) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || unicode.IsSpace(char) || strings.ContainsRune("-_/.'()", char) {
			builder.WriteRune(char)
		}
	}
	return strings.TrimSpace(builder.String())
}

func (s *SupabaseStore) ListReconciliations(ctx context.Context, limit int) ([]domain.ReconciliationRecord, error) {
	query := url.Values{"select": {"*"}, "order": {"created_at.desc"}}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var records []domain.ReconciliationRecord
	if err := s.doJSON(ctx, http.MethodGet, "reconciliations", query, nil, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *SupabaseStore) ReconcileInventory(ctx context.Context, input domain.NewReconciliationInput) (domain.ReconciliationRecord, domain.InventoryItem, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Notes == "" || strings.TrimSpace(input.Actor) == "" {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}
	if input.Type == "found_not_recorded" {
		input.NewItem.ReconciliationCreated = true
		input.NewItem.Actor = input.Actor
		input.NewItem.DocumentID = input.DocumentID
		item, err := s.CreateInventory(ctx, input.NewItem)
		if err != nil {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
		}
		payload := map[string]any{
			"reconciliation_type": input.Type, "action": "added", "inventory_id": item.ID,
			"inventory_reference": item.ReferenceNo, "inventory_type": item.Type,
			"result_status_code": item.StatusCode, "result_status_label": item.StatusLabel,
			"notes": input.Notes, "actor": input.Actor,
		}
		var rows []domain.ReconciliationRecord
		if err := s.doJSON(ctx, http.MethodPost, "reconciliations", url.Values{"select": {"*"}}, payload, &rows); err != nil || len(rows) == 0 {
			if err != nil {
				return domain.ReconciliationRecord{}, item, err
			}
			return domain.ReconciliationRecord{}, item, ErrNotFound
		}
		return rows[0], item, nil
	}
	if input.Type != "recorded_not_found" || strings.TrimSpace(input.InventoryID) == "" {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}
	item, err := s.GetInventory(ctx, input.InventoryID)
	if err != nil {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
	}
	if !item.IsActive {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInactiveInventory
	}
	now := time.Now().UTC()
	patch := map[string]any{
		"is_active": false, "current_disposition": nil,
		"status_code": "rekonsiliasi_tidak_ditemukan", "status_label": "Tidak ditemukan di lapangan",
		"location_status": "Tidak ditemukan saat rekonsiliasi", "updated_at": now,
	}
	var updated []domain.InventoryItem
	if err := s.doJSON(ctx, http.MethodPatch, "inventory_items", url.Values{"id": {"eq." + item.ID}, "select": {"*"}}, patch, &updated); err != nil || len(updated) == 0 {
		if err != nil {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
		}
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrNotFound
	}
	_ = s.doJSON(ctx, http.MethodPatch, "dispositions", url.Values{"inventory_id": {"eq." + item.ID}, "is_active": {"eq.true"}}, map[string]any{"is_active": false, "updated_at": now}, nil)
	_ = s.doJSON(ctx, http.MethodPost, "events", nil, map[string]any{
		"inventory_id": item.ID, "code": updated[0].StatusCode, "label": updated[0].StatusLabel,
		"notes": input.Notes, "actor": input.Actor, "created_at": now, "document_id": nullableString(input.DocumentID),
	}, nil)
	payload := map[string]any{
		"reconciliation_type": input.Type, "action": "removed", "inventory_id": item.ID,
		"inventory_reference": item.ReferenceNo, "inventory_type": item.Type,
		"previous_status_code": item.StatusCode, "previous_status_label": item.StatusLabel,
		"result_status_code": updated[0].StatusCode, "result_status_label": updated[0].StatusLabel,
		"notes": input.Notes, "actor": input.Actor,
	}
	var rows []domain.ReconciliationRecord
	if err := s.doJSON(ctx, http.MethodPost, "reconciliations", url.Values{"select": {"*"}}, payload, &rows); err != nil || len(rows) == 0 {
		if err != nil {
			return domain.ReconciliationRecord{}, updated[0], err
		}
		return domain.ReconciliationRecord{}, updated[0], ErrNotFound
	}
	return rows[0], updated[0], nil
}

func (s *SupabaseStore) CorrectInventoryData(ctx context.Context, input domain.InventoryCorrectionInput) (domain.ReconciliationRecord, domain.InventoryItem, error) {
	input.InventoryID = strings.TrimSpace(input.InventoryID)
	input.Actor = strings.TrimSpace(input.Actor)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.InventoryID == "" || input.Actor == "" || !validCorrectionReason(input.Reason) {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
	}
	current, err := s.GetInventory(ctx, input.InventoryID)
	if err != nil {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
	}
	corrected, err := correctedInventoryItem(current, input.Item)
	if err != nil {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, err
	}
	if corrected.FacilityID != "" {
		facilities, facilityErr := s.Facilities(ctx)
		if facilityErr != nil {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, facilityErr
		}
		found := false
		for _, facility := range facilities {
			if facility.ID == corrected.FacilityID {
				corrected.FacilityName = facility.Name
				found = true
				break
			}
		}
		if !found {
			return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrInvalidTransition
		}
	}
	itemPatch := map[string]any{
		"reference_no": corrected.ReferenceNo, "item_type": corrected.Type, "origin_type": corrected.OriginType,
		"bl_no": corrected.BLNo, "bl_date": nullableTime(corrected.BLDate),
		"manifest_no": corrected.ManifestNo, "manifest_date": nullableTime(corrected.ManifestDate), "manifest_position": corrected.ManifestPosition,
		"determination_no": corrected.DeterminationNo, "determination_date": nullableTime(corrected.DeterminationDate),
		"category": corrected.Category, "entrusted_category": corrected.EntrustedCategory, "source_office": corrected.SourceOffice,
		"description": corrected.Description, "item_kind": corrected.ItemKind, "quantity": corrected.Quantity, "unit": corrected.Unit,
		"goods_value": corrected.GoodsValue, "goods_condition": corrected.GoodsCondition,
		"location": corrected.Location, "location_status": corrected.LocationStatus, "at_tpp": corrected.AtTPP,
		"owner_name": corrected.OwnerName, "owner_address": corrected.OwnerAddress, "origin_warehouse": corrected.OriginWarehouse,
		"facility_id": nullableString(corrected.FacilityID), "facility_name": corrected.FacilityName,
		"load_type": corrected.LoadType, "container_no": corrected.ContainerNo, "container_size": corrected.ContainerSize,
		"estimated_volume_m3": corrected.EstimatedVolumeM3, "physical_unit_id": corrected.PhysicalUnitID, "occupancy_primary": corrected.OccupancyPrimary,
		"pfpd_required": corrected.PFPDRequired, "research_request_no": corrected.ResearchRequestNo, "research_request_date": nullableTime(corrected.ResearchRequestDate),
		"hs_code": corrected.HSCode, "is_restricted": corrected.IsRestricted, "restriction_rule": corrected.RestrictionRule,
		"origin_document_type": corrected.OriginDocumentType, "origin_document_no": corrected.OriginDocumentNo, "origin_document_date": nullableTime(corrected.OriginDocumentDate),
		"allocation_purpose": corrected.AllocationPurpose, "allocation_proposal_type": corrected.AllocationProposalType,
		"allocation_proposal_no": corrected.AllocationProposalNo, "allocation_proposal_date": nullableTime(corrected.AllocationProposalDate),
		"allocation_approval_type": corrected.AllocationApprovalType, "allocation_approval_no": corrected.AllocationApprovalNo,
		"allocation_approval_date": nullableTime(corrected.AllocationApprovalDate), "exit_document_no": corrected.ExitDocumentNo,
		"exit_document_date": nullableTime(corrected.ExitDocumentDate), "exit_type": corrected.ExitType, "exit_notes": corrected.ExitNotes,
	}
	eventPatches := make([]map[string]any, 0, len(input.Events))
	for _, correction := range input.Events {
		if strings.TrimSpace(correction.ID) == "" {
			continue
		}
		eventPatches = append(eventPatches, map[string]any{
			"id": correction.ID, "label": strings.TrimSpace(correction.Label), "document_no": strings.TrimSpace(correction.DocumentNo),
			"document_date": nullableTime(correction.DocumentDate), "notes": strings.TrimSpace(correction.Notes),
		})
	}
	processPatches := make([]map[string]any, 0, len(input.Processes))
	for _, correction := range input.Processes {
		if strings.TrimSpace(correction.ID) == "" {
			continue
		}
		processPatches = append(processPatches, map[string]any{
			"id": correction.ID, "proposal_type": strings.TrimSpace(correction.ProposalType), "recipient_code": strings.TrimSpace(correction.RecipientCode),
			"recipient_name": strings.TrimSpace(correction.RecipientName), "sale_value": correction.SaleValue, "htl_value": correction.HTLValue,
			"execution_start_date": nullableTime(correction.ExecutionStartDate), "execution_end_date": nullableTime(correction.ExecutionEndDate),
			"schedule_document_no": strings.TrimSpace(correction.ScheduleDocumentNo), "schedule_document_date": nullableTime(correction.ScheduleDocumentDate),
			"auction_outcome": strings.TrimSpace(correction.AuctionOutcome), "allocation_target": strings.TrimSpace(correction.AllocationTarget),
			"destruction_cost": correction.DestructionCost, "transfer_type": strings.TrimSpace(correction.TransferType),
		})
	}
	payload := map[string]any{
		"p_inventory_id": input.InventoryID, "p_actor": input.Actor, "p_reason": input.Reason,
		"p_item_patch": itemPatch, "p_event_patches": eventPatches, "p_process_patches": processPatches,
		"p_document_id": nullableString(input.DocumentID), "p_expected_updated_at": current.UpdatedAt,
	}
	var result struct {
		Record domain.ReconciliationRecord `json:"record"`
		Item   domain.InventoryItem        `json:"item"`
	}
	if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_correct_inventory_data", nil, payload, &result); err != nil {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, mapRPCError(err)
	}
	if result.Record.ID == "" || result.Item.ID == "" {
		return domain.ReconciliationRecord{}, domain.InventoryItem{}, ErrNotFound
	}
	return result.Record, result.Item, nil
}
