package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/hendra/manajemen-tpp/internal/domain"
	"github.com/hendra/manajemen-tpp/internal/store"
)

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	roles, err := s.store.ListRoles(r.Context(), false)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Setujui Pendaftaran", "Verifikasi email, persetujuan admin, dan penetapan role pengguna", "admin-users")
	data.AdminSection = "users"
	data.Users = users
	data.Roles = roles
	for _, user := range users {
		if user.ApprovalStatus == "pending" {
			data.PendingUsers++
			if user.EmailVerified {
				data.VerifiedPendingUsers++
			}
		}
	}
	s.render(w, "admin", data)
}

func (s *Server) approveUser(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	roleID := strings.TrimSpace(r.FormValue("role_id"))
	if roleID == "" {
		redirectMessage(w, r, "/admin/pendaftaran", "error", "Pilih role sebelum menyetujui pendaftaran.")
		return
	}
	userID := r.PathValue("id")
	if err := s.store.ApproveUser(r.Context(), userID, roleID, session.DisplayName); err != nil {
		s.writeAudit(r, "user.approve", "user", userID, "failed", map[string]any{"role_id": roleID, "error": err.Error()})
		redirectMessage(w, r, "/admin/pendaftaran", "error", "Pendaftaran belum dapat disetujui. Pastikan OTP email sudah dikonfirmasi dan role masih aktif.")
		return
	}
	s.writeAudit(r, "user.approve", "user", userID, "success", map[string]any{"role_id": roleID})
	redirectMessage(w, r, "/admin/pendaftaran", "ok", "Pendaftaran disetujui. Pengguna sekarang dapat masuk sesuai role yang ditetapkan.")
}

func (s *Server) rejectUser(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		redirectMessage(w, r, "/admin/pendaftaran", "error", "Tuliskan alasan penolakan agar dapat ditampilkan kepada pendaftar.")
		return
	}
	userID := r.PathValue("id")
	if err := s.store.RejectUser(r.Context(), userID, reason, session.DisplayName); err != nil {
		s.writeAudit(r, "user.reject", "user", userID, "failed", map[string]any{"error": err.Error()})
		redirectMessage(w, r, "/admin/pendaftaran", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "user.reject", "user", userID, "success", map[string]any{"reason": reason})
	redirectMessage(w, r, "/admin/pendaftaran", "ok", "Pendaftaran telah ditolak.")
}

func (s *Server) updateUserRole(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	roleID := strings.TrimSpace(r.FormValue("role_id"))
	if roleID == "" {
		redirectMessage(w, r, "/admin/pendaftaran", "error", "Pilih role pengguna yang baru.")
		return
	}
	userID := r.PathValue("id")
	if err := s.store.UpdateUserRole(r.Context(), userID, roleID, session.DisplayName); err != nil {
		s.writeAudit(r, "user.role.update", "user", userID, "failed", map[string]any{"role_id": roleID, "error": err.Error()})
		redirectMessage(w, r, "/admin/pendaftaran", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "user.role.update", "user", userID, "success", map[string]any{"role_id": roleID})
	redirectMessage(w, r, "/admin/pendaftaran", "ok", "Role pengguna berhasil diperbarui.")
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	userID := strings.TrimSpace(r.PathValue("id"))
	if userID == "" {
		redirectMessage(w, r, "/admin/pendaftaran", "error", "Pengguna yang akan dihapus tidak valid.")
		return
	}
	deleted, err := s.store.DeleteUser(r.Context(), userID)
	if err != nil {
		s.writeAudit(r, "user.delete", "user", userID, "failed", map[string]any{"error": err.Error()})
		redirectMessage(w, r, "/admin/pendaftaran", "error", "User belum dapat dihapus. Muat ulang halaman dan coba kembali.")
		return
	}
	s.writeAudit(r, "user.delete", "user", userID, "success", map[string]any{"auth_user_id": deleted.AuthUserID, "email": deleted.Email, "approval_status": deleted.ApprovalStatus})
	redirectMessage(w, r, "/admin/pendaftaran", "ok", "User "+deleted.Name+" berhasil dihapus permanen dari pendaftaran dan autentikasi.")
}

func (s *Server) adminRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.store.ListRoles(r.Context(), true)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Role & Hak Akses", "Buat role dengan nama dan kombinasi akses yang dapat disesuaikan", "admin-roles")
	data.AdminSection = "roles"
	data.Roles = roles
	data.PermissionDefinitions = domain.PermissionDefinitions
	s.render(w, "admin", data)
}

func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	input := roleInputFromRequest(r, session.DisplayName)
	role, err := s.store.CreateRole(r.Context(), input)
	if err != nil {
		s.writeAudit(r, "role.create", "role", "", "failed", map[string]any{"name": input.Name, "error": err.Error()})
		redirectMessage(w, r, "/admin/roles", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "role.create", "role", role.ID, "success", map[string]any{"name": role.Name, "permissions": role.Permissions})
	redirectMessage(w, r, "/admin/roles", "ok", "Role baru berhasil dibuat dan siap ditetapkan kepada pendaftar.")
}

func (s *Server) updateRole(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	input := roleInputFromRequest(r, session.DisplayName)
	roleID := r.PathValue("id")
	role, err := s.store.UpdateRole(r.Context(), roleID, input)
	if err != nil {
		s.writeAudit(r, "role.update", "role", roleID, "failed", map[string]any{"error": err.Error()})
		redirectMessage(w, r, "/admin/roles", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "role.update", "role", roleID, "success", map[string]any{"name": role.Name, "permissions": role.Permissions})
	redirectMessage(w, r, "/admin/roles", "ok", "Role dan hak akses berhasil diperbarui. Sesi aktif dengan hak lama dicabut otomatis.")
}

func roleInputFromRequest(r *http.Request, actor string) domain.NewRoleInput {
	permissions := append([]string(nil), r.Form["permissions"]...)
	return domain.NewRoleInput{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Permissions: permissions,
		Actor:       actor,
	}
}

func (s *Server) setRoleStatus(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	active := r.FormValue("active") == "true"
	roleID := r.PathValue("id")
	if err := s.store.SetRoleActive(r.Context(), roleID, active); err != nil {
		message := friendlyError(err)
		if !active {
			message = "Role belum dapat dinonaktifkan karena masih digunakan oleh akun yang telah disetujui. Pindahkan role pengguna terlebih dahulu."
		}
		s.writeAudit(r, "role.status", "role", roleID, "failed", map[string]any{"active": active, "error": err.Error()})
		redirectMessage(w, r, "/admin/roles", "error", message)
		return
	}
	s.writeAudit(r, "role.status", "role", roleID, "success", map[string]any{"active": active})
	message := "Role dinonaktifkan dan tidak lagi tersedia saat persetujuan pendaftaran."
	if active {
		message = "Role diaktifkan kembali."
	}
	redirectMessage(w, r, "/admin/roles", "ok", message)
}

func (s *Server) deleteRole(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	roleID := strings.TrimSpace(r.PathValue("id"))
	if roleID == "" {
		redirectMessage(w, r, "/admin/roles", "error", "Role yang akan dihapus tidak valid.")
		return
	}
	deleted, err := s.store.DeleteRole(r.Context(), roleID)
	if err != nil {
		message := friendlyError(err)
		if errors.Is(err, store.ErrRoleInUse) {
			message = "Role belum dapat dihapus karena masih ditetapkan kepada pengguna. Pindahkan role pengguna terlebih dahulu."
		}
		s.writeAudit(r, "role.delete", "role", roleID, "failed", map[string]any{"error": err.Error()})
		redirectMessage(w, r, "/admin/roles", "error", message)
		return
	}
	s.writeAudit(r, "role.delete", "role", deleted.ID, "success", map[string]any{"name": deleted.Name, "active": deleted.Active, "system": deleted.System})
	redirectMessage(w, r, "/admin/roles", "ok", "Role "+deleted.Name+" berhasil dihapus permanen.")
}

func (s *Server) adminParameters(w http.ResponseWriter, r *http.Request) {
	parameters, err := s.store.ParameterOptions(r.Context(), "", true)
	if err != nil {
		s.renderStoreError(w, r, err)
		return
	}
	data := s.baseData(r, "Parameter Sistem", "Kelola seluruh master dropdown operasional dari satu menu", "admin-parameters")
	data.AdminSection = "parameters"
	data.Query = strings.TrimSpace(r.URL.Query().Get("q"))
	if data.Query != "" {
		needle := strings.ToLower(data.Query)
		filtered := make([]domain.ParameterOption, 0, len(parameters))
		for _, option := range parameters {
			haystack := strings.ToLower(strings.Join([]string{parameterGroupName(option.GroupCode), option.GroupCode, option.Code, option.Label, option.AppliesTo}, " "))
			if strings.Contains(haystack, needle) {
				filtered = append(filtered, option)
			}
		}
		parameters = filtered
	}
	data.Parameters = parameters
	s.render(w, "admin", data)
}

func (s *Server) createParameter(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	order, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("sort_order")))
	appliesTo := strings.Join(r.Form["applies_to"], ",")
	input := domain.NewParameterInput{
		GroupCode: strings.TrimSpace(r.FormValue("group_code")),
		Code:      strings.TrimSpace(r.FormValue("code")),
		Label:     strings.TrimSpace(r.FormValue("label")),
		AppliesTo: appliesTo,
		SortOrder: order,
		Actor:     session.DisplayName,
	}
	if input.SortOrder <= 0 {
		input.SortOrder = 999
	}
	parameter, err := s.store.CreateParameter(r.Context(), input)
	if err != nil {
		s.writeAudit(r, "parameter.create", "parameter", "", "failed", map[string]any{"group": input.GroupCode, "label": input.Label, "error": err.Error()})
		redirectMessage(w, r, "/admin/parameters", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "parameter.create", "parameter", parameter.ID, "success", map[string]any{"group": parameter.GroupCode, "label": parameter.Label})
	s.refreshRuntimeParameters(r)
	redirectMessage(w, r, "/admin/parameters", "ok", "Parameter berhasil ditambahkan dan langsung tersedia pada dropdown terkait.")
}

func (s *Server) setParameterStatus(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	active := r.FormValue("active") == "true"
	parameterID := r.PathValue("id")
	if err := s.store.SetParameterActive(r.Context(), parameterID, active); err != nil {
		message := friendlyError(err)
		if !active && strings.HasPrefix(parameterID, "facility--") && errors.Is(err, store.ErrConflict) {
			message = "TPP belum dapat dinonaktifkan karena masih digunakan oleh inventory aktif. Pindahkan atau selesaikan barangnya terlebih dahulu."
		}
		s.writeAudit(r, "parameter.status", "parameter", parameterID, "failed", map[string]any{"active": active, "error": err.Error()})
		redirectMessage(w, r, "/admin/parameters", "error", message)
		return
	}
	s.writeAudit(r, "parameter.status", "parameter", parameterID, "success", map[string]any{"active": active})
	s.refreshRuntimeParameters(r)
	message := "Parameter dinonaktifkan dan tidak lagi muncul pada dropdown. Data lama tetap tersimpan."
	if active {
		message = "Parameter diaktifkan kembali."
	}
	redirectMessage(w, r, "/admin/parameters", "ok", message)
}

func (s *Server) updateParameter(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionFromContext(r.Context())
	if !s.validateCSRF(r, session) {
		http.Error(w, "token keamanan tidak valid", http.StatusForbidden)
		return
	}
	order, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("sort_order")))
	input := domain.NewParameterInput{
		Label: strings.TrimSpace(r.FormValue("label")), AppliesTo: strings.Join(r.Form["applies_to"], ","), SortOrder: order, Actor: session.DisplayName,
	}
	if input.SortOrder <= 0 {
		input.SortOrder = 999
	}
	parameterID := r.PathValue("id")
	parameter, err := s.store.UpdateParameter(r.Context(), parameterID, input)
	if err != nil {
		s.writeAudit(r, "parameter.update", "parameter", parameterID, "failed", map[string]any{"error": err.Error()})
		redirectMessage(w, r, "/admin/parameters", "error", friendlyError(err))
		return
	}
	s.writeAudit(r, "parameter.update", "parameter", parameterID, "success", map[string]any{"label": parameter.Label, "sort_order": parameter.SortOrder})
	s.refreshRuntimeParameters(r)
	redirectMessage(w, r, "/admin/parameters", "ok", "Parameter berhasil diperbarui dan langsung berlaku pada dropdown terkait.")
}

func parameterGroupName(group string) string {
	switch group {
	case domain.ParameterBDNCategory:
		return "Kategori BDN"
	case domain.ParameterItemKind:
		return "Jenis barang"
	case domain.ParameterGoodsCondition:
		return "Kondisi barang"
	case domain.ParameterUnit:
		return "Satuan barang"
	case domain.ParameterAllocationPurpose:
		return "Jenis peruntukan BMMN"
	case domain.ParameterOriginTPS:
		return "TPS asal"
	case domain.ParameterTPP:
		return "Nama TPP"
	case domain.ParameterLoadType:
		return "Jenis muatan"
	case domain.ParameterExitType:
		return "Jenis pengeluaran"
	case domain.ParameterTransferType:
		return "Jenis serah terima"
	default:
		return group
	}
}

func (s *Server) refreshRuntimeParameters(r *http.Request) {
	s.invalidateParameterCache()
	if err := s.loadRuntimeParameters(r.Context(), true); err != nil {
		s.logger.Warn("refresh parameters", "error", err)
	}
}
