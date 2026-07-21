package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func TestBTDReportFilterPreservesUserSelections(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=btd&scope=completed&date_from=2026-01-31&date_to=2026-01-01&status=pencacahan&location=tpp&tpp=facility-1", nil)
	filter, report := (&Server{}).reportFilter(request)

	if report.Scope != "completed" || !filter.OnlyInactive || filter.IncludeInactive {
		t.Fatalf("BTD inventory scope was not preserved: report=%q includeInactive=%v onlyInactive=%v", report.Scope, filter.IncludeInactive, filter.OnlyInactive)
	}
	if report.DateFrom != "2026-01-01" || report.DateTo != "2026-01-31" {
		t.Fatalf("BTD date range was not normalized: %s to %s", report.DateFrom, report.DateTo)
	}
	if filter.Type != domain.InventoryBTD || filter.Status != "pencacahan" {
		t.Fatalf("BTD type/status filters were not applied: type=%q status=%q", filter.Type, filter.Status)
	}
	if report.Location != "tpp" || filter.LocationScope != "tpp" || filter.FacilityID != "facility-1" {
		t.Fatalf("BTD location filters were not applied: location=%q scope=%q facility=%q", report.Location, filter.LocationScope, filter.FacilityID)
	}
	for _, expected := range []string{"scope=completed", "date_from=2026-01-01", "date_to=2026-01-31", "status=pencacahan", "location=tpp", "tpp=facility-1"} {
		if !strings.Contains(report.ExcelExportURL, expected) {
			t.Fatalf("Excel export URL did not preserve %q: %s", expected, report.ExcelExportURL)
		}
	}
}

func TestBTDReportFilterDefaultsToAllInventoryStates(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=btd", nil)
	filter, report := (&Server{}).reportFilter(request)
	if report.Scope != "all" || !filter.IncludeInactive || filter.OnlyInactive {
		t.Fatalf("default BTD scope should include active and completed inventory: report=%q includeInactive=%v onlyInactive=%v", report.Scope, filter.IncludeInactive, filter.OnlyInactive)
	}
}
