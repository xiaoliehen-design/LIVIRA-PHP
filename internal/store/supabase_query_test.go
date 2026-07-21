package store

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestSupabaseDashboardRecoversTitipanFromLegacyRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/v1/rpc/livira_dashboard_summary":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"active_total":2,
				"btd_total":1,
				"bdn_total":0,
				"bmmn_total":0,
				"facility_breakdown":[{"facility_id":"tpp-1","facility_name":"TPP 1","btd":1,"bdn":0,"bmmn":0,"total":2}]
			}`))
		case "/rest/v1/inventory_items":
			if r.Header.Get("Prefer") == "count=exact" {
				w.Header().Set("Content-Range", "0-0/1")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"id":"tit-1"}]`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{
				"id":"tit-1",
				"reference_no":"TIT-001",
				"determination_no":"DOC-TIT-001",
				"determination_date":"2026-07-16T00:00:00Z",
				"item_type":"TITIPAN",
				"load_type":"FCL",
				"container_no":"ABCD 123456-7",
				"physical_unit_id":"FCL:ABCD1234567",
				"facility_id":"tpp-1",
				"is_active":true
			}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data := NewSupabaseStore(server.URL, "service-key", "")
	data.client = server.Client()
	stats, err := data.Dashboard(context.Background())
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if stats.TitipanTotal != 1 {
		t.Fatalf("TitipanTotal = %d, want 1", stats.TitipanTotal)
	}
	if stats.ActiveTotal != 2 {
		t.Fatalf("ActiveTotal = %d, want 2", stats.ActiveTotal)
	}
	if stats.TitipanSummary.Documents != 1 || stats.TitipanSummary.FCL != 1 || stats.TitipanSummary.LCL != 0 {
		t.Fatalf("TitipanSummary = %+v, want 1 document and 1 FCL", stats.TitipanSummary)
	}
	if len(stats.FacilityBreakdown) != 1 || stats.FacilityBreakdown[0].Titipan != 1 || stats.FacilityBreakdown[0].Total != 2 {
		t.Fatalf("FacilityBreakdown = %+v, want TITIPAN 1 and total 2", stats.FacilityBreakdown)
	}
}

func TestSupabaseDeleteUserRemovesAuthIdentityWithServiceRole(t *testing.T) {
	const (
		appUserID  = "2b3ee47b-7a2f-4634-8557-8634e3e961b7"
		authUserID = "55a403da-5e33-40fb-8ca0-7bd6738361a8"
	)
	authDeleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/v1/app_users":
			if r.Method != http.MethodGet || r.URL.Query().Get("id") != "eq."+appUserID {
				t.Errorf("unexpected app user request: %s %s", r.Method, r.URL.String())
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`[{"id":"` + appUserID + `","auth_user_id":"` + authUserID + `","name":"Hendra","email":"hendra@example.go.id","approval_status":"approved"}]`))
		case "/auth/v1/admin/users/" + authUserID:
			authDeleteCalled = true
			if r.Method != http.MethodDelete || r.Header.Get("Authorization") != "Bearer service-key" || r.Header.Get("apikey") != "service-key" {
				t.Errorf("invalid auth deletion request: method=%s headers=%v", r.Method, r.Header)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data := NewSupabaseStore(server.URL, "service-key", "")
	data.client = server.Client()
	deleted, err := data.DeleteUser(context.Background(), appUserID)
	if err != nil {
		t.Fatal(err)
	}
	if !authDeleteCalled || deleted.ID != appUserID || deleted.AuthUserID != authUserID {
		t.Fatalf("unexpected deletion result: called=%v user=%+v", authDeleteCalled, deleted)
	}
}

func TestSupabaseListRolesIncludesAssignedUserCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/v1/app_roles":
			if r.Method != http.MethodGet {
				t.Errorf("unexpected roles method %s", r.Method)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`[
				{"id":"role-one","name":"Role One","permissions":["dashboard.view"],"active":true},
				{"id":"role-two","name":"Role Two","permissions":["dashboard.view"],"active":true}
			]`))
		case "/rest/v1/app_users":
			if r.Method != http.MethodGet || r.URL.Query().Get("role_id") != "not.is.null" {
				t.Errorf("unexpected role assignment request: %s %s", r.Method, r.URL.String())
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`[{"role_id":"role-one"},{"role_id":"role-one"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data := NewSupabaseStore(server.URL, "service-key", "")
	data.client = server.Client()
	roles, err := data.ListRoles(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 || roles[0].AssignedUsers != 2 || roles[1].AssignedUsers != 0 {
		t.Fatalf("unexpected roles and usage counts: %+v", roles)
	}
}

func TestSupabaseDeleteRoleUsesDatabaseConstraint(t *testing.T) {
	const roleID = "2b3ee47b-7a2f-4634-8557-8634e3e961b7"
	t.Run("unused role is deleted", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path != "/rest/v1/app_roles" || r.Method != http.MethodDelete || r.URL.Query().Get("id") != "eq."+roleID {
				t.Errorf("unexpected role deletion request: %s %s", r.Method, r.URL.String())
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if r.Header.Get("Prefer") != "return=representation" {
				t.Errorf("role deletion must request the deleted row, got Prefer=%q", r.Header.Get("Prefer"))
			}
			_, _ = w.Write([]byte(`[{"id":"` + roleID + `","name":"Role Kosong","permissions":["dashboard.view"],"active":true}]`))
		}))
		defer server.Close()

		data := NewSupabaseStore(server.URL, "service-key", "")
		data.client = server.Client()
		deleted, err := data.DeleteRole(context.Background(), roleID)
		if err != nil {
			t.Fatal(err)
		}
		if deleted.ID != roleID || deleted.Name != "Role Kosong" {
			t.Fatalf("unexpected deleted role: %+v", deleted)
		}
	})

	t.Run("assigned role is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"23503","message":"update or delete on table app_roles violates foreign key constraint app_users_role_id_fkey"}`))
		}))
		defer server.Close()

		data := NewSupabaseStore(server.URL, "service-key", "")
		data.client = server.Client()
		if _, err := data.DeleteRole(context.Background(), roleID); !errors.Is(err, ErrRoleInUse) {
			t.Fatalf("assigned role must return ErrRoleInUse, got %v", err)
		}
	})
}

func TestInventoryQueryRejectsTypeOutsideRoleScope(t *testing.T) {
	_, valid := inventoryQuery(domain.InventoryFilter{
		Type:         domain.InventoryBMMN,
		AllowedTypes: []domain.InventoryType{domain.InventoryBTD, domain.InventoryBDN},
	})
	if valid {
		t.Fatal("query must reject an explicitly selected inventory type outside the user's role scope")
	}
}

func TestInventoryQueryAuctionReadyUsesIndexedFilters(t *testing.T) {
	query, valid := inventoryQuery(domain.InventoryFilter{
		Preset:       "auction_ready",
		AllowedTypes: []domain.InventoryType{domain.InventoryBTD, domain.InventoryBDN, domain.InventoryBMMN},
	})
	if !valid {
		t.Fatal("auction-ready query should be valid")
	}
	if got := query.Get("current_disposition"); got != "is.null" {
		t.Fatalf("current_disposition filter = %q, want is.null", got)
	}
	if got := query.Get("or"); got != "(status_code.eq.penelitian_pfpd,item_type.eq.BMMN)" {
		t.Fatalf("auction-ready OR filter = %q", got)
	}
}

func TestDispositionQueryAppliesOffset(t *testing.T) {
	query, valid := dispositionQuery(domain.DispositionFilter{Limit: 20, Offset: 40})
	if !valid {
		t.Fatal("disposition query should be valid")
	}
	if got := query.Get("limit"); got != "20" {
		t.Fatalf("limit = %q, want 20", got)
	}
	if got := query.Get("offset"); got != "40" {
		t.Fatalf("offset = %q, want 40", got)
	}
}

func TestDispositionQueryFiltersWorkflowHistoryStatuses(t *testing.T) {
	query, valid := dispositionQuery(domain.DispositionFilter{
		Type:               domain.DispositionAuction,
		IncludeStatusCodes: []string{"laku", "alokasi_hasil_lelang", "dialihkan_musnah", "dialihkan_hibah"},
	})
	if !valid {
		t.Fatal("auction history query should be valid")
	}
	if got := query.Get("status_code"); got != "in.(laku,alokasi_hasil_lelang,dialihkan_musnah,dialihkan_hibah)" {
		t.Fatalf("history status filter = %q", got)
	}

	query, valid = dispositionQuery(domain.DispositionFilter{
		Type:               domain.DispositionDestruction,
		ExcludeStatusCodes: []string{"ba_musnah"},
	})
	if !valid {
		t.Fatal("active destruction query should be valid")
	}
	if got := query.Get("status_code"); got != "not.in.(ba_musnah)" {
		t.Fatalf("active status exclusion = %q", got)
	}
}

func TestDispositionQueryCanTargetOneInventory(t *testing.T) {
	query, valid := dispositionQuery(domain.DispositionFilter{InventoryID: "inventory-123", Limit: 100})
	if !valid {
		t.Fatal("inventory-specific disposition query should be valid")
	}
	if got := query.Get("inventory_id"); got != "eq.inventory-123" {
		t.Fatalf("inventory_id filter = %q, want eq.inventory-123", got)
	}
}
