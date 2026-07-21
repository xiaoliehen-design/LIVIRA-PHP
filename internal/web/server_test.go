package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hendra/manajemen-tpp/internal/auth"
	"github.com/hendra/manajemen-tpp/internal/config"
	"github.com/hendra/manajemen-tpp/internal/domain"
	"github.com/hendra/manajemen-tpp/internal/store"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Config{AppName: "LIVIRA", AppEnv: "test", SessionSecret: "test-secret-long-enough", AdminUsername: "test-admin", AdminPassword: "local-test-password-2026", DemoMode: true}
	authManager := auth.NewManager(cfg.SessionSecret, false, cfg.AdminUsername, cfg.AdminPassword, "", "", "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server, err := NewServer(cfg, store.NewMemoryStore(), authManager, logger)
	if err != nil {
		t.Fatal(err)
	}
	return server.Routes()
}

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestProtectedPageRedirectsToLogin(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/inventory", nil))
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/login" {
		t.Fatalf("expected login redirect, got status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestLoginDoesNotExposeDemoCredentials(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d", recorder.Code)
	}
	if strings.Contains(body, "AKUN DEMO") || strings.Contains(body, "local-test-password-2026") || strings.Contains(body, ">hendraganteng<") {
		t.Fatal("login page exposed demo credentials")
	}
	if !strings.Contains(body, "LIVIRA") {
		t.Fatal("new LIVIRA brand not rendered")
	}
	for _, expected := range []string{"Lupa password?", "Verifikasi CAPTCHA", "captcha_token", "/captcha.png"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("login page did not render %q", expected)
		}
	}
}

func TestForgotPasswordPagesExplainEmailOTPFlow(t *testing.T) {
	for path, expectedValues := range map[string][]string{
		"/forgot-password":        {"Lupa password", "Email terdaftar", "Kirim OTP reset password"},
		"/forgot-password/verify": {"Buat password baru", "Kode OTP", "Konfirmasi password baru", "Verifikasi OTP dan ubah password"},
	} {
		recorder := httptest.NewRecorder()
		testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, recorder.Code)
		}
		for _, expected := range expectedValues {
			if !strings.Contains(recorder.Body.String(), expected) {
				t.Fatalf("%s did not render %q", path, expected)
			}
		}
	}
}

func TestSignupPageExplainsOTPAndAdminApproval(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/signup", nil))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected signup 200, got %d", recorder.Code)
	}
	for _, expected := range []string{"OTP 6 digit", "Daftar dan kirim OTP", "email aktif"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("signup page did not render %q", expected)
		}
	}
}

func TestAdminCanLoginAndOpenDashboard(t *testing.T) {
	handler := testHandler(t)
	captcha := newCaptchaManager("test-secret-long-enough")
	captchaToken, captchaAnswer, err := captcha.newChallenge()
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"identity": {"test-admin"}, "password": {"local-test-password-2026"}, "captcha_token": {captchaToken}, "captcha_answer": {captchaAnswer}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected login redirect, got %d", recorder.Code)
	}
	response := recorder.Result()
	var sessionCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == auth.CookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}
	dashboardRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	dashboardRequest.AddCookie(sessionCookie)
	dashboardRecorder := httptest.NewRecorder()
	handler.ServeHTTP(dashboardRecorder, dashboardRequest)
	if dashboardRecorder.Code != http.StatusOK {
		t.Fatalf("expected dashboard 200, got %d", dashboardRecorder.Code)
	}
	dashboardBody := dashboardRecorder.Body.String()
	if !strings.Contains(dashboardBody, "Total inventory aktif") || !strings.Contains(dashboardBody, "Barang Titipan") || !strings.Contains(dashboardBody, `class="kpi-card primary-card kpi-card-link" href="/inventory"`) {
		t.Fatal("dashboard KPI was not rendered as a clickable inventory link")
	}
	if !strings.Contains(dashboardBody, `name="idle-timeout-seconds" content="1800"`) || !strings.Contains(dashboardBody, `name="csrf-token"`) {
		t.Fatal("dashboard did not expose the signed idle-session configuration")
	}
	for _, expected := range []string{`data-popover-toggle="notifications"`, "Pendaftaran siap disetujui", `data-popover-toggle="account-menu"`, `data-open-profile-modal`, `id="profile-modal"`, "Detail Profil"} {
		if !strings.Contains(dashboardBody, expected) {
			t.Fatalf("dashboard did not render topbar interaction %q", expected)
		}
	}
	if !strings.Contains(dashboardRecorder.Body.String(), "Yard Occupancy Ratio") || !strings.Contains(dashboardRecorder.Body.String(), "Shed Occupancy Ratio") {
		t.Fatal("dashboard YOR/SOR content not rendered")
	}
	for _, expected := range []string{"Performa kinerja", `data-open-performance-dashboard`, `id="performance-dashboard-modal"`, `name="performance_from"`, `name="performance_to"`, "Performa penilaian PFPD", "Konversi BMMN", "/pelaporan/performa.xlsx"} {
		if !strings.Contains(dashboardBody, expected) {
			t.Fatalf("dashboard performance popup did not render %q", expected)
		}
	}
	if strings.Contains(dashboardBody, "Selesai bulan ini") {
		t.Fatal("legacy completed-this-month card is still rendered")
	}

	for _, path := range []string{"/inventory", "/proses/lelang", "/proses/musnah", "/proses/hibah", "/rekonsiliasi", "/pelaporan", "/pencarian"} {
		pageRequest := httptest.NewRequest(http.MethodGet, path, nil)
		pageRequest.AddCookie(sessionCookie)
		pageRecorder := httptest.NewRecorder()
		handler.ServeHTTP(pageRecorder, pageRequest)
		if pageRecorder.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, pageRecorder.Code)
		}
		if strings.Contains(pageRecorder.Body.String(), "TPP L4") {
			t.Fatalf("%s rendered excluded TPP L4", path)
		}
		body := pageRecorder.Body.String()
		if path == "/inventory" {
			for _, expected := range []string{"data-open-inventory-action", "PT Agung Raya", "Barang Lartas Ps. 53 (4)", "Barang Berbahaya (B3)", "Hewan atau Tumbuhan Hidup", "Usulan Peruntukan", "PENGHAPUSAN", "PENGELUARAN BARANG TITIPAN", "Pilih semua hasil yang tampil", "History", "Pemasukan barang titipan", "Barang Titipan", "data-determination-no-label", "Kantor/unit penitip", "Penetapan terbaru", "Nilai tertinggi", "Nilai terendah", "data-delete-inventory-form", "/admin/inventory/inv-001/delete", "data-fcl-container-section", "containers_json", "Perkiraan volume barang", "40&#39; HC", "45&#39; HC", "Kontainer dan identitas barang", "Pilih target pencacahan", "Kondisi barang", "Pilih nomor request penelitian PFPD", "data-pfpd-request-search", "Cari nomor request penelitian PFPD", "data-table-scroll-top", "page_size"} {
				if !strings.Contains(body, expected) {
					t.Fatalf("inventory did not render %q", expected)
				}
			}
			if strings.Contains(body, "Action inventory massal") {
				t.Fatal("legacy bulk action bar is still rendered")
			}
			if strings.Contains(body, "Apakah penelitian PFPD diperlukan?") {
				t.Fatal("PFPD requirement question must not be rendered in census action")
			}
			if strings.Contains(body, "Berada di TPP") || strings.Contains(body, "Berada di TPS") {
				t.Fatal("inventory location status still uses the legacy 'Berada di' prefix")
			}
		}
		if path == "/" {
			for _, expected := range []string{"data-open-capacity-editor", "process-dashboard-lelang", "process-dashboard-musnah", "process-dashboard-hibah", "Proses dimulai tahun ini", "Selesai tahun ini", "Total nilai jual", "Total biaya musnah", "Barang dihibahkan"} {
				if !strings.Contains(body, expected) {
					t.Fatalf("dashboard did not render %q", expected)
				}
			}
		}
		if path == "/proses/lelang" {
			for _, expected := range []string{"Penerbitan KEP Lelang", "Penerbitan KEP Harga Terendah Lelang", "Lelang Penyesuaian", "Alokasi Hasil Lelang", "Pilih ND penjadwalan lelang", "Cari nomor ND penjadwalan lelang", "auction_results_json", "HTL / hasil lelang", "Harga Terendah Lelang per barang", "htl_results_json", "HTL tertinggi", "HTL terendah", "History", "name=\"document_file\""} {
				if !strings.Contains(body, expected) {
					t.Fatalf("auction page did not render %q", expected)
				}
			}
			if strings.Contains(body, "Ringkasan dan tren") {
				t.Fatal("auction dashboard must only appear as a popup on the main dashboard")
			}
			if strings.Contains(body, "REFERENSI PROSES") || strings.Contains(strings.ToLower(body), "biaya lelang") {
				t.Fatal("auction page still renders the hidden process reference or an auction-cost component")
			}
		}
		if path == "/proses/musnah" && (!strings.Contains(body, "Penerbitan KEP Musnah") || !strings.Contains(body, "Berita Acara Musnah") || !strings.Contains(body, "barang lelang berstatus Tidak Laku") || strings.Contains(body, "Ringkasan dan tren")) {
			t.Fatal("destruction page did not render the workflow-only view")
		}
		if path == "/proses/hibah" && (!strings.Contains(body, "Berita Acara Serah Terima") || strings.Contains(body, "Ringkasan dan tren")) {
			t.Fatal("grant page did not render the workflow-only view")
		}
		if strings.HasPrefix(path, "/proses/") && strings.Contains(body, "REFERENSI PROSES") {
			t.Fatal("process reference block must be hidden on workflow pages")
		}
		if path == "/rekonsiliasi" {
			for _, expected := range []string{"data-reconciliation-mode=\"reconciliation\"", "data-reconciliation-mode=\"data_correction\"", "Tercatat di aplikasi tetapi tidak ada di lapangan", "Tidak ada di aplikasi tetapi ditemukan di lapangan", "Perubahan data barang", "Alasan perubahan", "Kesalahan input", "Error pada saat pengisian awal", "Catatan rekonsiliasi", "Barang Titipan", "Selisih catatan dan kondisi fisik", "multipart/form-data", "name=\"document_file\""} {
				if !strings.Contains(body, expected) {
					t.Fatalf("reconciliation page did not render %q", expected)
				}
			}
		}
		if path == "/pelaporan" {
			for _, expected := range []string{"Laporan yang sering dipakai", "BTD/BDN ≥60 hari", "Potensi siap lelang", "Rekap rekonsiliasi", "Rekap perubahan data barang", "name=\"date_from\"", "name=\"scope\"", "Ekspor CSV sesuai filter", "Ekspor Excel", "Tampilkan", "Berikutnya"} {
				if !strings.Contains(body, expected) {
					t.Fatalf("report page did not render %q", expected)
				}
			}
		}
	}

	performancePageRequest := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=performance&date_from=2026-01-01&date_to=2026-12-31", nil)
	performancePageRequest.AddCookie(sessionCookie)
	performancePageRecorder := httptest.NewRecorder()
	handler.ServeHTTP(performancePageRecorder, performancePageRequest)
	performancePageBody := performancePageRecorder.Body.String()
	if performancePageRecorder.Code != http.StatusOK {
		t.Fatalf("expected performance report 200, got %d", performancePageRecorder.Code)
	}
	for _, expected := range []string{"Laporan performa", "Performa kinerja Tahun 2026", "Performa lelang", "Performa musnah", "Performa hibah/PSP", "Performa cacah", "Performa penilaian PFPD", "Konversi BMMN", "Unduh laporan Excel", "Rincian dasar pengukuran"} {
		if !strings.Contains(performancePageBody, expected) {
			t.Fatalf("performance report did not render %q", expected)
		}
	}

	performanceExcelRequest := httptest.NewRequest(http.MethodGet, "/pelaporan/performa.xlsx?date_from=2026-01-01&date_to=2026-12-31", nil)
	performanceExcelRequest.AddCookie(sessionCookie)
	performanceExcelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(performanceExcelRecorder, performanceExcelRequest)
	if performanceExcelRecorder.Code != http.StatusOK {
		t.Fatalf("expected performance xlsx 200, got %d", performanceExcelRecorder.Code)
	}
	if contentType := performanceExcelRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "spreadsheetml.sheet") {
		t.Fatalf("unexpected performance xlsx content type %q", contentType)
	}
	if !strings.HasPrefix(performanceExcelRecorder.Body.String(), "PK") {
		t.Fatal("performance xlsx response is not a ZIP-based workbook")
	}

	pagedInventoryRequest := httptest.NewRequest(http.MethodGet, "/inventory?page_size=10&page=2", nil)
	pagedInventoryRequest.AddCookie(sessionCookie)
	pagedInventoryRecorder := httptest.NewRecorder()
	handler.ServeHTTP(pagedInventoryRecorder, pagedInventoryRequest)
	pagedInventoryBody := pagedInventoryRecorder.Body.String()
	if pagedInventoryRecorder.Code != http.StatusOK || !strings.Contains(pagedInventoryBody, "Halaman 2 dari 2") || !strings.Contains(pagedInventoryBody, ">10</a>") || !strings.Contains(pagedInventoryBody, "Sebelumnya") {
		t.Fatal("inventory pagination with 10 rows per page was not rendered")
	}

	adminPages := map[string][]string{
		"/admin/pendaftaran": {"Daftar pendaftaran pengguna", "OTP terverifikasi", "Pilih role", "Hapus user", "data-delete-user-form"},
		"/admin/roles":       {"Buat role baru", "Kelola lelang", "Akses BMMN", "Akses barang titipan", "Lihat rekonsiliasi", "Kelola rekonsiliasi", "0 pengguna", "Hapus role", "data-delete-role-form"},
		"/admin/parameters":  {"Tambah parameter dropdown", "Kategori BDN", "Kondisi barang", "Satuan barang", "Jenis peruntukan BMMN", "TPS asal", "Nama TPP", "Jenis muatan (FCL/LCL)", "Jenis serah terima (Hibah/PSP)", "Barang Titipan", "Hapus dari dropdown"},
	}
	for path, expectedValues := range adminPages {
		pageRequest := httptest.NewRequest(http.MethodGet, path, nil)
		pageRequest.AddCookie(sessionCookie)
		pageRecorder := httptest.NewRecorder()
		handler.ServeHTTP(pageRecorder, pageRequest)
		if pageRecorder.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, pageRecorder.Code)
		}
		pageBody := pageRecorder.Body.String()
		for _, expected := range expectedValues {
			if !strings.Contains(pageBody, expected) {
				t.Fatalf("%s did not render %q", path, expected)
			}
		}
		if path == "/admin/parameters" {
			if strings.Contains(pageBody, "<th>Kode</th>") || strings.Contains(pageBody, "<td><code>") {
				t.Fatal("system parameter table still exposes the internal code column")
			}
			if !strings.Contains(pageBody, "Cari kelompok, label, atau cakupan") {
				t.Fatal("system parameter search placeholder still suggests searching by a visible code column")
			}
		}
	}

	reportRequest := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=auction_ready", nil)
	reportRequest.AddCookie(sessionCookie)
	reportRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reportRecorder, reportRequest)
	if reportRecorder.Code != http.StatusOK || !strings.Contains(reportRecorder.Body.String(), "Potensi barang siap lelang") || strings.Contains(reportRecorder.Body.String(), "name=\"sort\"") {
		t.Fatal("auction-ready report preset was not applied")
	}

	reconciliationReportRequest := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=reconciliation", nil)
	reconciliationReportRequest.AddCookie(sessionCookie)
	reconciliationReportRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reconciliationReportRecorder, reconciliationReportRequest)
	if reconciliationReportRecorder.Code != http.StatusOK || !strings.Contains(reconciliationReportRecorder.Body.String(), "Rekap rekonsiliasi") || !strings.Contains(reconciliationReportRecorder.Body.String(), "Jenis rekonsiliasi") {
		t.Fatal("reconciliation report preset was not rendered")
	}

	dataCorrectionReportRequest := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=data_correction", nil)
	dataCorrectionReportRequest.AddCookie(sessionCookie)
	dataCorrectionReportRecorder := httptest.NewRecorder()
	handler.ServeHTTP(dataCorrectionReportRecorder, dataCorrectionReportRequest)
	if dataCorrectionReportRecorder.Code != http.StatusOK || !strings.Contains(dataCorrectionReportRecorder.Body.String(), "Rekap perubahan data barang") || !strings.Contains(dataCorrectionReportRecorder.Body.String(), "Nilai sebelum") {
		t.Fatal("data correction report preset was not rendered")
	}

	csvRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.csv?scope=active&date_from=2026-01-01&date_to=2026-12-31", nil)
	csvRequest.AddCookie(sessionCookie)
	csvRecorder := httptest.NewRecorder()
	handler.ServeHTTP(csvRecorder, csvRequest)
	if csvRecorder.Code != http.StatusOK || !strings.Contains(csvRecorder.Header().Get("Content-Disposition"), "livira-kustom") || !strings.Contains(csvRecorder.Body.String(), "Umur (Hari)") || !strings.Contains(csvRecorder.Body.String(), "Peruntukan BMMN") {
		t.Fatal("filtered CSV report was not generated")
	}

	excelRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.xlsx?scope=active&date_from=2026-01-01&date_to=2026-12-31", nil)
	excelRequest.AddCookie(sessionCookie)
	excelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(excelRecorder, excelRequest)
	if excelRecorder.Code != http.StatusOK || !strings.Contains(excelRecorder.Header().Get("Content-Disposition"), "livira-kustom") || !strings.Contains(excelRecorder.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatal("filtered Excel report was not generated")
	}
	if contentType := excelRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "spreadsheetml.sheet") {
		t.Fatalf("unexpected report Excel content type %q", contentType)
	}
	if !strings.HasPrefix(excelRecorder.Body.String(), "PK") {
		t.Fatal("report Excel response is not a ZIP-based workbook")
	}

	reconciliationCSVRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.csv?preset=reconciliation", nil)
	reconciliationCSVRequest.AddCookie(sessionCookie)
	reconciliationCSVRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reconciliationCSVRecorder, reconciliationCSVRequest)
	if reconciliationCSVRecorder.Code != http.StatusOK || !strings.Contains(reconciliationCSVRecorder.Header().Get("Content-Disposition"), "livira-rekonsiliasi") || !strings.Contains(reconciliationCSVRecorder.Body.String(), "Jenis Rekonsiliasi") {
		t.Fatal("reconciliation CSV report was not generated")
	}

	reconciliationExcelRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.xlsx?preset=reconciliation", nil)
	reconciliationExcelRequest.AddCookie(sessionCookie)
	reconciliationExcelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reconciliationExcelRecorder, reconciliationExcelRequest)
	if reconciliationExcelRecorder.Code != http.StatusOK || !strings.Contains(reconciliationExcelRecorder.Header().Get("Content-Disposition"), "livira-rekonsiliasi") || !strings.HasPrefix(reconciliationExcelRecorder.Body.String(), "PK") {
		t.Fatal("reconciliation Excel report was not generated")
	}

	dataCorrectionCSVRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.csv?preset=data_correction", nil)
	dataCorrectionCSVRequest.AddCookie(sessionCookie)
	dataCorrectionCSVRecorder := httptest.NewRecorder()
	handler.ServeHTTP(dataCorrectionCSVRecorder, dataCorrectionCSVRequest)
	if dataCorrectionCSVRecorder.Code != http.StatusOK || !strings.Contains(dataCorrectionCSVRecorder.Header().Get("Content-Disposition"), "livira-perubahan-data-barang") || !strings.Contains(dataCorrectionCSVRecorder.Body.String(), "Nilai Sebelum") || !strings.Contains(dataCorrectionCSVRecorder.Body.String(), "Nilai Sesudah") {
		t.Fatal("data correction CSV report was not generated")
	}

	dataCorrectionExcelRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.xlsx?preset=data_correction", nil)
	dataCorrectionExcelRequest.AddCookie(sessionCookie)
	dataCorrectionExcelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(dataCorrectionExcelRecorder, dataCorrectionExcelRequest)
	if dataCorrectionExcelRecorder.Code != http.StatusOK || !strings.Contains(dataCorrectionExcelRecorder.Header().Get("Content-Disposition"), "livira-perubahan-data-barang") || !strings.HasPrefix(dataCorrectionExcelRecorder.Body.String(), "PK") {
		t.Fatal("data correction Excel report was not generated")
	}

	btdExcelRequest := httptest.NewRequest(http.MethodGet, "/pelaporan.xlsx?preset=btd", nil)
	btdExcelRequest.AddCookie(sessionCookie)
	btdExcelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(btdExcelRecorder, btdExcelRequest)
	if btdExcelRecorder.Code != http.StatusOK || !strings.Contains(btdExcelRecorder.Header().Get("Content-Disposition"), "livira-btd") || !strings.HasPrefix(btdExcelRecorder.Body.String(), "PK") {
		t.Fatal("BTD Excel report was not generated")
	}

	btdReportRequest := httptest.NewRequest(http.MethodGet, "/pelaporan?preset=btd&scope=completed&date_from=2026-01-01&date_to=2026-12-31&status=pencacahan&location=tpp", nil)
	btdReportRequest.AddCookie(sessionCookie)
	btdReportRecorder := httptest.NewRecorder()
	handler.ServeHTTP(btdReportRecorder, btdReportRequest)
	if btdReportRecorder.Code != http.StatusOK {
		t.Fatalf("BTD filtered report returned status %d", btdReportRecorder.Code)
	}
	for _, expected := range []string{"Filter Laporan BTD", "Tanggal BTD dari", "Tanggal BTD sampai", "Status inventory", "Status barang", "Lokasi barang", "Reset filter BTD"} {
		if !strings.Contains(btdReportRecorder.Body.String(), expected) {
			t.Fatalf("BTD filtered report did not contain %q", expected)
		}
	}

	searchRequest := httptest.NewRequest(http.MethodGet, "/pencarian?scope=all&sort=value_desc", nil)
	searchRequest.AddCookie(sessionCookie)
	searchRecorder := httptest.NewRecorder()
	handler.ServeHTTP(searchRecorder, searchRequest)
	if searchRecorder.Code != http.StatusOK || !strings.Contains(searchRecorder.Body.String(), "barang yang relevan") || !strings.Contains(searchRecorder.Body.String(), "data-detail-url") || !strings.Contains(searchRecorder.Body.String(), "data-timeline-url") {
		t.Fatal("parameterized detail search did not render clickable detail and timeline results")
	}

	for _, path := range []string{"/inventory?history=1", "/proses/lelang?history=1", "/proses/musnah?history=1", "/proses/hibah?history=1"} {
		historyRequest := httptest.NewRequest(http.MethodGet, path, nil)
		historyRequest.AddCookie(sessionCookie)
		historyRecorder := httptest.NewRecorder()
		handler.ServeHTTP(historyRecorder, historyRequest)
		if historyRecorder.Code != http.StatusOK || !strings.Contains(historyRecorder.Body.String(), "Kembali ke") {
			t.Fatalf("history page %s was not rendered correctly", path)
		}
	}

	csrfMarker := `name="csrf-token" content="`
	csrfStart := strings.Index(dashboardBody, csrfMarker)
	if csrfStart < 0 {
		t.Fatal("csrf meta token not found")
	}
	csrfStart += len(csrfMarker)
	csrfEnd := strings.Index(dashboardBody[csrfStart:], `"`)
	if csrfEnd < 0 {
		t.Fatal("csrf meta token was malformed")
	}
	csrf := dashboardBody[csrfStart : csrfStart+csrfEnd]
	userDeleteForm := url.Values{"_csrf": {csrf}}
	userDeleteRequest := httptest.NewRequest(http.MethodPost, "/admin/pendaftaran/user-pending-2/delete", strings.NewReader(userDeleteForm.Encode()))
	userDeleteRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userDeleteRequest.AddCookie(sessionCookie)
	userDeleteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(userDeleteRecorder, userDeleteRequest)
	if userDeleteRecorder.Code != http.StatusSeeOther {
		t.Fatalf("expected admin user delete redirect, got %d", userDeleteRecorder.Code)
	}
	usersAfterDeleteRequest := httptest.NewRequest(http.MethodGet, "/admin/pendaftaran", nil)
	usersAfterDeleteRequest.AddCookie(sessionCookie)
	usersAfterDeleteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(usersAfterDeleteRecorder, usersAfterDeleteRequest)
	if usersAfterDeleteRecorder.Code != http.StatusOK || strings.Contains(usersAfterDeleteRecorder.Body.String(), "bagus.prasetyo@example.go.id") {
		t.Fatal("deleted user is still rendered in the registration list")
	}
	deleteForm := url.Values{"_csrf": {csrf}, "return_to": {"/inventory"}}
	deleteRequest := httptest.NewRequest(http.MethodPost, "/admin/inventory/inv-001/delete", strings.NewReader(deleteForm.Encode()))
	deleteRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteRequest.AddCookie(sessionCookie)
	deleteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusSeeOther {
		t.Fatalf("expected admin delete redirect, got %d", deleteRecorder.Code)
	}
	deletedRequest := httptest.NewRequest(http.MethodGet, "/api/inventory/inv-001", nil)
	deletedRequest.AddCookie(sessionCookie)
	deletedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deletedRecorder, deletedRequest)
	if deletedRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected deleted inventory to return 404, got %d", deletedRecorder.Code)
	}
}

func TestExpandPhysicalInventoryIDsIncludesEveryGoodsLineInOneContainer(t *testing.T) {
	data := store.NewMemoryStore()
	ctx := context.Background()
	now := time.Now()
	created, err := data.CreateInventory(ctx, domain.NewInventoryInput{
		Type: domain.InventoryBTD, DeterminationNo: "KEP-PFPD-GROUP", DeterminationDate: now,
		Description: "Uraian awal", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece",
		AtTPP: true, FacilityID: "tpp-graha-segara", OriginWarehouse: "PT Agung Raya",
		LoadType: "FCL", ContainerNo: "PFPD 123456-7", ContainerSize: "20", Actor: "Tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.ApplyInventoryCensus(ctx, created.ID, []domain.InventoryGoodsLine{
		{InventoryID: created.ID, Description: "Uraian satu", ItemKind: "Barang Umum", Quantity: 1, Unit: "Piece", GoodsCondition: "Bekas"},
		{Description: "Uraian dua", ItemKind: "Barang Berharga", Quantity: 2, Unit: "Piece", GoodsCondition: "Baru"},
	}, domain.NewEventInput{DocumentNo: "BA-CACAH", DocumentDate: now, PFPDRequired: true, Actor: "Tester"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{store: data}
	ids, err := server.expandPhysicalInventoryIDs(ctx, auth.Session{Role: "admin"}, []string{rows[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected request selection to expand to 2 goods lines, got %d", len(ids))
	}
}

func TestCensusLinesFromFormDoesNotRequirePFPDChoice(t *testing.T) {
	form := url.Values{
		"multiple_goods": {"tidak"},
		"description":    {"Mesin dan komponen"},
		"item_kind":      {"Barang Umum"},
		"quantity":       {"12"},
		"unit":           {"Piece"},
	}
	req := httptest.NewRequest(http.MethodPost, "/inventory/bulk-event", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	lines, err := censusLinesFromForm(req)
	if err != nil {
		t.Fatalf("census form should not require pfpd_required: %v", err)
	}
	if len(lines) != 1 || lines[0].Description != "Mesin dan komponen" || lines[0].Quantity != 12 {
		t.Fatalf("unexpected census lines: %#v", lines)
	}
}

func TestGroupAuctionSchedulesUsesSchedulingDocumentAsBundle(t *testing.T) {
	now := time.Date(2026, time.July, 15, 10, 0, 0, 0, time.UTC)
	processes := []domain.Disposition{
		{ID: "proc-2", Type: domain.DispositionAuction, IsActive: true, StatusCode: "jadwal_lelang", ScheduleDocumentNo: "ND-02/2026", ScheduleDocumentDate: now.Add(-24 * time.Hour), Inventory: domain.InventoryItem{ContainerNo: "ZZZZ 000002-2", Description: "Barang kedua"}},
		{ID: "proc-1", Type: domain.DispositionAuction, IsActive: true, StatusCode: "jadwal_lelang", ScheduleDocumentNo: "ND-01/2026", ScheduleDocumentDate: now, Inventory: domain.InventoryItem{ContainerNo: "BBBB 000001-1", Description: "Barang B"}},
		{ID: "proc-3", Type: domain.DispositionAuction, IsActive: true, StatusCode: "jadwal_lelang", ScheduleDocumentNo: "ND-01/2026", ScheduleDocumentDate: now, Inventory: domain.InventoryItem{ContainerNo: "AAAA 000001-0", Description: "Barang A"}},
		{ID: "proc-ignored-status", Type: domain.DispositionAuction, IsActive: true, StatusCode: "kep_htl", ScheduleDocumentNo: "ND-IGNORED"},
		{ID: "proc-ignored-inactive", Type: domain.DispositionAuction, IsActive: false, StatusCode: "jadwal_lelang", ScheduleDocumentNo: "ND-IGNORED"},
	}

	groups := groupAuctionSchedules(processes)
	if len(groups) != 2 {
		t.Fatalf("expected 2 scheduling bundles, got %d", len(groups))
	}
	if groups[0].DocumentNo != "ND-01/2026" || len(groups[0].Processes) != 2 {
		t.Fatalf("latest ND bundle was not grouped correctly: %#v", groups[0])
	}
	if groups[0].Processes[0].ID != "proc-3" || groups[0].Processes[1].ID != "proc-1" {
		t.Fatalf("items inside an ND bundle were not ordered by container: %#v", groups[0].Processes)
	}
	if groups[1].DocumentNo != "ND-02/2026" || len(groups[1].Processes) != 1 {
		t.Fatalf("second ND bundle was not grouped correctly: %#v", groups[1])
	}
}

func TestBuildProcessDashboardSeparatesStartedAndCompletedThisYear(t *testing.T) {
	now := time.Date(2026, time.July, 15, 10, 0, 0, 0, time.UTC)
	processes := []domain.Disposition{
		{ID: "active-new", Type: domain.DispositionAuction, IsActive: true, StatusCode: "jadwal_lelang", CreatedAt: now.AddDate(0, -1, 0), UpdatedAt: now.AddDate(0, -1, 0)},
		{ID: "closed-old", Type: domain.DispositionAuction, IsActive: false, StatusCode: "alokasi_hasil_lelang", CreatedAt: now.AddDate(-1, 0, 0), UpdatedAt: now.Add(-24 * time.Hour)},
		{ID: "auction-finished-but-allocation-pending", Type: domain.DispositionAuction, IsActive: true, StatusCode: "laku", CreatedAt: now.AddDate(-1, 0, 0), UpdatedAt: now.Add(-48 * time.Hour)},
		{ID: "closed-last-year", Type: domain.DispositionAuction, IsActive: false, StatusCode: "tidak_laku", CreatedAt: now.AddDate(-1, -1, 0), UpdatedAt: now.AddDate(-1, 0, 0)},
	}

	dashboard := buildProcessDashboard(domain.DispositionAuction, processes, now)
	if dashboard.StartedThisYear != 1 {
		t.Fatalf("expected 1 process started this year, got %d", dashboard.StartedThisYear)
	}
	if dashboard.CompletedThisYear != 2 {
		t.Fatalf("expected 2 auctions completed this year, got %d", dashboard.CompletedThisYear)
	}
	if dashboard.Active != 2 {
		t.Fatalf("expected 2 active workflow records, got %d", dashboard.Active)
	}
}

func TestLandingPathSupportsReconciliationOnlyRole(t *testing.T) {
	session := auth.Session{Permissions: []string{domain.PermissionReconciliationView}}
	if got := landingPath(session); got != "/rekonsiliasi" {
		t.Fatalf("expected reconciliation landing page, got %q", got)
	}
}

func TestFilterReconciliationsRespectsInventoryTypePermissions(t *testing.T) {
	records := []domain.ReconciliationRecord{
		{ID: "rec-btd", InventoryType: domain.InventoryBTD},
		{ID: "rec-titipan", InventoryType: domain.InventoryTitipan},
	}
	session := auth.Session{Permissions: []string{domain.PermissionInventoryBTD}}
	filtered := filterReconciliationsForSession(session, records)
	if len(filtered) != 1 || filtered[0].ID != "rec-btd" {
		t.Fatalf("unexpected filtered records: %#v", filtered)
	}
	adminRecords := filterReconciliationsForSession(auth.Session{Role: "admin"}, records)
	if len(adminRecords) != len(records) {
		t.Fatalf("admin should see all reconciliation records: %#v", adminRecords)
	}
}

func TestFlattenDataCorrectionRowsMatchesExportGranularity(t *testing.T) {
	records := []domain.ReconciliationRecord{
		{
			ID: "correction-current",
			ChangeDetails: []domain.ReconciliationChange{
				{Section: "inventory", Field: "description", Before: "Barang lama", After: "Barang baru"},
				{Section: "inventory", Field: "quantity", Before: "1", After: "2"},
			},
		},
		{ID: "correction-legacy"},
	}

	rows := flattenDataCorrectionRows(records)
	if len(rows) != 3 {
		t.Fatalf("expected one row per changed field plus one legacy row, got %d", len(rows))
	}
	if rows[0].Legacy || rows[0].Change.Field != "description" {
		t.Fatalf("unexpected first change row: %#v", rows[0])
	}
	if rows[1].Legacy || rows[1].Change.Field != "quantity" {
		t.Fatalf("unexpected second change row: %#v", rows[1])
	}
	if !rows[2].Legacy || rows[2].Record.ID != "correction-legacy" {
		t.Fatalf("legacy correction should remain visible as one row: %#v", rows[2])
	}
}

func TestDashboardInventoryScopeFiltering(t *testing.T) {
	facilities := []domain.Facility{{ID: "tpp-a", Name: "TPP A"}, {ID: "tpp-b", Name: "TPP B"}}
	tests := []struct {
		raw       string
		wantScope string
		wantLabel string
	}{
		{"", "all_office", "Seluruh cakupan Kantor Tanjung Priok"},
		{"still_tps", "still_tps", "Masih di TPS"},
		{"all_tpp", "all_tpp", "Seluruh TPP"},
		{"tpp-a", "tpp-a", "TPP A"},
		{"unknown", "all_office", "Seluruh cakupan Kantor Tanjung Priok"},
	}
	for _, test := range tests {
		scope, label := dashboardInventoryScope(test.raw, facilities)
		if scope != test.wantScope || label != test.wantLabel {
			t.Fatalf("scope %q returned (%q, %q), want (%q, %q)", test.raw, scope, label, test.wantScope, test.wantLabel)
		}
	}

	atTPS := domain.InventoryItem{IsActive: true, AtTPP: false}
	atTPPA := domain.InventoryItem{IsActive: true, AtTPP: true, FacilityID: "tpp-a"}
	atTPPB := domain.InventoryItem{IsActive: true, AtTPP: true, FacilityID: "tpp-b"}
	if !dashboardInventoryItemInScope(atTPS, "all_office") || !dashboardInventoryItemInScope(atTPPA, "all_office") {
		t.Fatal("all_office scope must include active goods at TPS and TPP")
	}
	if dashboardInventoryItemInScope(atTPS, "all_tpp") || !dashboardInventoryItemInScope(atTPPA, "all_tpp") {
		t.Fatal("all_tpp scope must include only goods already at a TPP")
	}
	if !dashboardInventoryItemInScope(atTPS, "still_tps") || dashboardInventoryItemInScope(atTPPA, "still_tps") {
		t.Fatal("still_tps scope must include only goods that have not arrived at a TPP")
	}
	if !dashboardInventoryItemInScope(atTPPA, "tpp-a") || dashboardInventoryItemInScope(atTPPB, "tpp-a") {
		t.Fatal("specific TPP scope returned the wrong facility")
	}
}

func TestGroupRelocationTargetsDeduplicatesFCLContainerAndKeepsAllGoods(t *testing.T) {
	items := []domain.InventoryItem{
		{ID: "goods-1", IsActive: true, Type: domain.InventoryBTD, LoadType: "FCL", PhysicalUnitID: "unit-1", ContainerNo: "ABCD 123456-7", ContainerSize: "20", DeterminationNo: "BTD-001", StatusCode: "pencacahan", StatusLabel: "Pencacahan", Quantity: 2, Description: "Alat mandi", OccupancyPrimary: true},
		{ID: "goods-2", IsActive: true, Type: domain.InventoryBTD, LoadType: "FCL", PhysicalUnitID: "legacy-unit-different", ContainerNo: "ABCD 123456-7", ContainerSize: "20", DeterminationNo: "BTD-001", StatusCode: "pencacahan", StatusLabel: "Pencacahan", Quantity: 8, Description: "Suku cadang mesin"},
		{ID: "goods-3", IsActive: true, Type: domain.InventoryBDN, LoadType: "LCL", PhysicalUnitID: "unit-2", DeterminationNo: "BDN-001", StatusCode: "jadwal_lelang", StatusLabel: "Penjadwalan Lelang", CurrentDisposition: domain.DispositionAuction, Quantity: 1, Description: "Dokumen"},
	}

	groups := groupRelocationTargets(items)
	if len(groups) != 2 {
		t.Fatalf("expected one FCL container target and one LCL target, got %d: %#v", len(groups), groups)
	}
	var fcl RelocationTargetGroup
	for _, group := range groups {
		if group.LoadType == "FCL" {
			fcl = group
		}
	}
	if fcl.TargetKey == "" || len(fcl.Items) != 2 {
		t.Fatalf("FCL target must appear once and contain both goods descriptions: %#v", fcl)
	}
	if fcl.Items[0].ID != "goods-1" || fcl.Items[1].ID != "goods-2" {
		t.Fatalf("unexpected goods order in FCL target: %#v", fcl.Items)
	}
}

func TestDashboardFacilityBreakdownDoesNotFollowKPIScope(t *testing.T) {
	ctx := context.Background()
	data := store.NewMemoryStore()
	server := &Server{store: data}
	original, err := data.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	facilities, err := data.Facilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	session := auth.Session{Role: "admin"}
	allOffice, err := server.dashboardStatsForSession(ctx, session, original, facilities, "all_office")
	if err != nil {
		t.Fatal(err)
	}
	stillTPS, err := server.dashboardStatsForSession(ctx, session, original, facilities, "still_tps")
	if err != nil {
		t.Fatal(err)
	}
	if len(allOffice.FacilityBreakdown) != len(stillTPS.FacilityBreakdown) {
		t.Fatalf("facility row count changed with KPI scope: %d vs %d", len(allOffice.FacilityBreakdown), len(stillTPS.FacilityBreakdown))
	}
	for index := range allOffice.FacilityBreakdown {
		left, right := allOffice.FacilityBreakdown[index], stillTPS.FacilityBreakdown[index]
		if left.FacilityID != right.FacilityID || left.BTD != right.BTD || left.BDN != right.BDN || left.BMMN != right.BMMN || left.Titipan != right.Titipan || left.Total != right.Total {
			t.Fatalf("detail per TPP changed with KPI scope: all=%+v tps=%+v", left, right)
		}
	}
	if allOffice.ActiveTotal < stillTPS.ActiveTotal {
		t.Fatalf("all-office KPI must include at least all TPS goods: all=%d tps=%d", allOffice.ActiveTotal, stillTPS.ActiveTotal)
	}
}
