package store

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func (s *SupabaseStore) CreateUserApplication(ctx context.Context, input domain.NewUserApplicationInput) (domain.UserAccount, error) {
	if strings.TrimSpace(input.AuthUserID) == "" || strings.TrimSpace(input.Email) == "" {
		return domain.UserAccount{}, ErrInvalidTransition
	}
	query := url.Values{"select": {"*"}, "auth_user_id": {"eq." + input.AuthUserID}, "limit": {"1"}}
	var existing []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_users", query, nil, &existing); err != nil {
		return domain.UserAccount{}, err
	}
	if len(existing) > 0 {
		patch := map[string]any{"name": strings.TrimSpace(input.Name), "email": strings.ToLower(strings.TrimSpace(input.Email)), "updated_at": time.Now().UTC()}
		patchQuery := url.Values{"id": {"eq." + existing[0].ID}}
		var updated []domain.UserAccount
		if err := s.doJSON(ctx, http.MethodPatch, "app_users", patchQuery, patch, &updated); err != nil {
			return domain.UserAccount{}, err
		}
		if len(updated) > 0 {
			return updated[0], nil
		}
		return existing[0], nil
	}
	payload := map[string]any{"auth_user_id": input.AuthUserID, "name": strings.TrimSpace(input.Name), "email": strings.ToLower(strings.TrimSpace(input.Email)), "approval_status": "pending"}
	var created []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodPost, "app_users", nil, payload, &created); err != nil {
		return domain.UserAccount{}, err
	}
	if len(created) == 0 {
		return domain.UserAccount{}, ErrNotFound
	}
	return created[0], nil
}

func (s *SupabaseStore) MarkUserEmailVerified(ctx context.Context, authUserID, email string) error {
	query := url.Values{}
	if strings.TrimSpace(authUserID) != "" {
		query.Set("auth_user_id", "eq."+authUserID)
	} else {
		query.Set("email", "eq."+strings.ToLower(strings.TrimSpace(email)))
	}
	patch := map[string]any{"email_verified": true, "email_verified_at": time.Now().UTC(), "updated_at": time.Now().UTC()}
	return s.doJSON(ctx, http.MethodPatch, "app_users", query, patch, nil)
}

func (s *SupabaseStore) UserByAuthID(ctx context.Context, authUserID string) (domain.UserAccount, error) {
	query := url.Values{"select": {"*"}, "auth_user_id": {"eq." + authUserID}, "limit": {"1"}}
	var users []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_user_access", query, nil, &users); err != nil {
		return domain.UserAccount{}, err
	}
	if len(users) == 0 {
		return domain.UserAccount{}, ErrNotFound
	}
	users[0].Permissions = domain.NormalizePermissions(users[0].Permissions)
	return users[0], nil
}

func (s *SupabaseStore) enrichUserRole(ctx context.Context, user domain.UserAccount) (domain.UserAccount, error) {
	if user.RoleID == "" {
		return user, nil
	}
	query := url.Values{"select": {"*"}, "id": {"eq." + user.RoleID}, "limit": {"1"}}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodGet, "app_roles", query, nil, &roles); err != nil {
		return domain.UserAccount{}, err
	}
	if len(roles) > 0 && roles[0].Active {
		user.RoleName = roles[0].Name
		user.Permissions = domain.NormalizePermissions(roles[0].Permissions)
	}
	return user, nil
}

func (s *SupabaseStore) ListUsers(ctx context.Context) ([]domain.UserAccount, error) {
	query := url.Values{"select": {"*"}, "order": {"created_at.desc"}, "limit": {"1000"}}
	var users []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_user_access", query, nil, &users); err != nil {
		return nil, err
	}
	for index := range users {
		users[index].Permissions = domain.NormalizePermissions(users[index].Permissions)
	}
	return users, nil
}

func (s *SupabaseStore) ApproveUser(ctx context.Context, id, roleID, actor string) error {
	roleQuery := url.Values{"select": {"id,active"}, "id": {"eq." + roleID}, "active": {"eq.true"}, "limit": {"1"}}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodGet, "app_roles", roleQuery, nil, &roles); err != nil {
		return err
	}
	if len(roles) == 0 {
		return ErrInvalidTransition
	}
	userQuery := url.Values{"select": {"id,email_verified"}, "id": {"eq." + id}, "limit": {"1"}}
	var users []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_users", userQuery, nil, &users); err != nil {
		return err
	}
	if len(users) == 0 {
		return ErrNotFound
	}
	if !users[0].EmailVerified {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	patch := map[string]any{"approval_status": "approved", "role_id": roleID, "rejection_reason": "", "approved_by": actor, "approved_at": now, "updated_at": now}
	return s.doJSON(ctx, http.MethodPatch, "app_users", url.Values{"id": {"eq." + id}}, patch, nil)
}

func (s *SupabaseStore) RejectUser(ctx context.Context, id, reason, actor string) error {
	now := time.Now().UTC()
	patch := map[string]any{"approval_status": "rejected", "role_id": nil, "rejection_reason": strings.TrimSpace(reason), "approved_by": actor, "approved_at": now, "updated_at": now}
	return s.doJSON(ctx, http.MethodPatch, "app_users", url.Values{"id": {"eq." + id}}, patch, nil)
}

func (s *SupabaseStore) UpdateUserRole(ctx context.Context, id, roleID, actor string) error {
	roleQuery := url.Values{"select": {"id"}, "id": {"eq." + roleID}, "active": {"eq.true"}, "limit": {"1"}}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodGet, "app_roles", roleQuery, nil, &roles); err != nil {
		return err
	}
	if len(roles) == 0 {
		return ErrInvalidTransition
	}
	userQuery := url.Values{"select": {"id,approval_status"}, "id": {"eq." + id}, "approval_status": {"eq.approved"}, "limit": {"1"}}
	var users []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_users", userQuery, nil, &users); err != nil {
		return err
	}
	if len(users) == 0 {
		return ErrInvalidTransition
	}
	patch := map[string]any{"role_id": roleID, "approved_by": actor, "updated_at": time.Now().UTC()}
	return s.doJSON(ctx, http.MethodPatch, "app_users", url.Values{"id": {"eq." + id}}, patch, nil)
}

func (s *SupabaseStore) DeleteUser(ctx context.Context, id string) (domain.UserAccount, error) {
	query := url.Values{"select": {"*"}, "id": {"eq." + strings.TrimSpace(id)}, "limit": {"1"}}
	var users []domain.UserAccount
	if err := s.doJSON(ctx, http.MethodGet, "app_users", query, nil, &users); err != nil {
		return domain.UserAccount{}, err
	}
	if len(users) == 0 || strings.TrimSpace(users[0].AuthUserID) == "" {
		return domain.UserAccount{}, ErrNotFound
	}
	user := users[0]
	endpoint := s.projectURL + "/auth/v1/admin/users/" + url.PathEscape(user.AuthUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return domain.UserAccount{}, err
	}
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.UserAccount{}, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return domain.UserAccount{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.UserAccount{}, fmt.Errorf("supabase auth delete user: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	// app_users references auth.users ON DELETE CASCADE, so deleting the Auth
	// identity also removes the application profile in the same database.
	return user, nil
}

func (s *SupabaseStore) ListRoles(ctx context.Context, includeInactive bool) ([]domain.RoleProfile, error) {
	query := url.Values{"select": {"*"}, "order": {"name.asc"}, "limit": {"1000"}}
	if !includeInactive {
		query.Set("active", "eq.true")
	}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodGet, "app_roles", query, nil, &roles); err != nil {
		return nil, err
	}
	for index := range roles {
		roles[index].Permissions = domain.NormalizePermissions(roles[index].Permissions)
	}
	assignedUsers := make(map[string]int, len(roles))
	const pageSize = 1000
	for offset := 0; ; offset += pageSize {
		assignmentQuery := url.Values{
			"select":  {"role_id"},
			"role_id": {"not.is.null"},
			"order":   {"id.asc"},
			"limit":   {fmt.Sprintf("%d", pageSize)},
			"offset":  {fmt.Sprintf("%d", offset)},
		}
		var page []struct {
			RoleID string `json:"role_id"`
		}
		if err := s.doJSON(ctx, http.MethodGet, "app_users", assignmentQuery, nil, &page); err != nil {
			return nil, err
		}
		for _, assignment := range page {
			if assignment.RoleID != "" {
				assignedUsers[assignment.RoleID]++
			}
		}
		if len(page) < pageSize {
			break
		}
	}
	for index := range roles {
		roles[index].AssignedUsers = assignedUsers[roles[index].ID]
	}
	return roles, nil
}

func (s *SupabaseStore) CreateRole(ctx context.Context, input domain.NewRoleInput) (domain.RoleProfile, error) {
	name := strings.TrimSpace(input.Name)
	permissions := domain.NormalizePermissions(input.Permissions)
	if name == "" || len(permissions) == 0 {
		return domain.RoleProfile{}, ErrInvalidTransition
	}
	payload := map[string]any{"name": name, "description": strings.TrimSpace(input.Description), "permissions": permissions, "active": true, "created_by": input.Actor}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodPost, "app_roles", nil, payload, &roles); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return domain.RoleProfile{}, ErrConflict
		}
		return domain.RoleProfile{}, err
	}
	if len(roles) == 0 {
		return domain.RoleProfile{}, ErrNotFound
	}
	return roles[0], nil
}

func (s *SupabaseStore) UpdateRole(ctx context.Context, id string, input domain.NewRoleInput) (domain.RoleProfile, error) {
	name := strings.TrimSpace(input.Name)
	permissions := domain.NormalizePermissions(input.Permissions)
	if name == "" || len(permissions) == 0 {
		return domain.RoleProfile{}, ErrInvalidTransition
	}
	patch := map[string]any{"name": name, "description": strings.TrimSpace(input.Description), "permissions": permissions, "updated_at": time.Now().UTC()}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodPatch, "app_roles", url.Values{"id": {"eq." + id}}, patch, &roles); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return domain.RoleProfile{}, ErrConflict
		}
		return domain.RoleProfile{}, err
	}
	if len(roles) == 0 {
		return domain.RoleProfile{}, ErrNotFound
	}
	return roles[0], nil
}

func (s *SupabaseStore) SetRoleActive(ctx context.Context, id string, active bool) error {
	if !active {
		query := url.Values{"select": {"id"}, "role_id": {"eq." + id}, "approval_status": {"eq.approved"}, "limit": {"1"}}
		var users []domain.UserAccount
		if err := s.doJSON(ctx, http.MethodGet, "app_users", query, nil, &users); err != nil {
			return err
		}
		if len(users) > 0 {
			return ErrConflict
		}
	}
	return s.doJSON(ctx, http.MethodPatch, "app_roles", url.Values{"id": {"eq." + id}}, map[string]any{"active": active, "updated_at": time.Now().UTC()}, nil)
}

func (s *SupabaseStore) DeleteRole(ctx context.Context, id string) (domain.RoleProfile, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.RoleProfile{}, ErrNotFound
	}
	query := url.Values{"select": {"*"}, "id": {"eq." + id}}
	var roles []domain.RoleProfile
	if err := s.doJSON(ctx, http.MethodDelete, "app_roles", query, nil, &roles); err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, `"code":"23503"`) || strings.Contains(message, "app_users_role_id_fkey") || strings.Contains(message, "foreign key constraint") {
			return domain.RoleProfile{}, ErrRoleInUse
		}
		return domain.RoleProfile{}, err
	}
	if len(roles) == 0 {
		return domain.RoleProfile{}, ErrNotFound
	}
	roles[0].Permissions = domain.NormalizePermissions(roles[0].Permissions)
	roles[0].AssignedUsers = 0
	return roles[0], nil
}

func (s *SupabaseStore) ParameterOptions(ctx context.Context, group string, includeInactive bool) ([]domain.ParameterOption, error) {
	options := make([]domain.ParameterOption, 0, 128)
	if group == "" || group != domain.ParameterTPP {
		query := url.Values{"select": {"*"}, "order": {"group_code.asc,sort_order.asc,label.asc"}, "limit": {"2000"}}
		if group != "" {
			query.Set("group_code", "eq."+group)
		}
		if !includeInactive {
			query.Set("active", "eq.true")
		}
		if err := s.doJSON(ctx, http.MethodGet, "app_parameters", query, nil, &options); err != nil {
			return nil, err
		}
	}
	if group == "" || group == domain.ParameterTPP {
		query := url.Values{"select": {"id,name,active,sort_order,created_at"}, "order": {"sort_order.asc,name.asc"}, "limit": {"1000"}}
		if !includeInactive {
			query.Set("active", "eq.true")
		}
		var facilities []struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Active    bool      `json:"active"`
			SortOrder int       `json:"sort_order"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := s.doJSON(ctx, http.MethodGet, "facilities", query, nil, &facilities); err != nil {
			return nil, err
		}
		for _, facility := range facilities {
			options = append(options, domain.ParameterOption{
				ID: "facility--" + facility.ID, GroupCode: domain.ParameterTPP, Code: facility.ID,
				Label: facility.Name, Active: facility.Active, System: true, SortOrder: facility.SortOrder,
				CreatedAt: facility.CreatedAt,
			})
		}
	}
	return options, nil
}

func (s *SupabaseStore) CreateParameter(ctx context.Context, input domain.NewParameterInput) (domain.ParameterOption, error) {
	if !domain.ValidParameterGroup(input.GroupCode) {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	label := strings.TrimSpace(input.Label)
	code := slugCode(input.Code)
	if code == "" {
		code = slugCode(label)
	}
	if label == "" || code == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if input.GroupCode == domain.ParameterTPP {
		var existing []struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Active    bool      `json:"active"`
			SortOrder int       `json:"sort_order"`
			CreatedAt time.Time `json:"created_at"`
		}
		// First try by ID, then rely on the table's unique name constraint for duplicate labels.
		query := url.Values{"select": {"id,name,active,sort_order,created_at"}, "id": {"eq." + code}, "limit": {"1"}}
		if err := s.doJSON(ctx, http.MethodGet, "facilities", query, nil, &existing); err != nil {
			return domain.ParameterOption{}, err
		}
		if len(existing) > 0 {
			if existing[0].Active {
				return domain.ParameterOption{}, ErrConflict
			}
			patch := map[string]any{"name": label, "active": true, "sort_order": input.SortOrder}
			if err := s.doJSON(ctx, http.MethodPatch, "facilities", url.Values{"id": {"eq." + code}}, patch, nil); err != nil {
				return domain.ParameterOption{}, err
			}
			return domain.ParameterOption{ID: "facility--" + code, GroupCode: domain.ParameterTPP, Code: code, Label: label, Active: true, SortOrder: input.SortOrder}, nil
		}
		payload := map[string]any{"id": code, "name": label, "active": true, "sort_order": input.SortOrder, "yard_capacity": 0, "yard_used": 0, "shed_capacity": 0, "shed_used": 0}
		var created []struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Active    bool      `json:"active"`
			SortOrder int       `json:"sort_order"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := s.doJSON(ctx, http.MethodPost, "facilities", nil, payload, &created); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ParameterOption{}, ErrConflict
			}
			return domain.ParameterOption{}, err
		}
		if len(created) == 0 {
			return domain.ParameterOption{}, ErrNotFound
		}
		return domain.ParameterOption{ID: "facility--" + created[0].ID, GroupCode: domain.ParameterTPP, Code: created[0].ID, Label: created[0].Name, Active: created[0].Active, SortOrder: created[0].SortOrder, CreatedAt: created[0].CreatedAt}, nil
	}
	existingQuery := url.Values{"select": {"*"}, "group_code": {"eq." + input.GroupCode}, "code": {"eq." + code}, "limit": {"1"}}
	var existing []domain.ParameterOption
	if err := s.doJSON(ctx, http.MethodGet, "app_parameters", existingQuery, nil, &existing); err != nil {
		return domain.ParameterOption{}, err
	}
	applies := normalizeAppliesTo(input.GroupCode, input.AppliesTo)
	if input.GroupCode == domain.ParameterExitType && applies == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if len(existing) > 0 {
		if existing[0].Active {
			return domain.ParameterOption{}, ErrConflict
		}
		patch := map[string]any{"label": label, "applies_to": applies, "sort_order": input.SortOrder, "active": true, "updated_at": time.Now().UTC()}
		var updated []domain.ParameterOption
		if err := s.doJSON(ctx, http.MethodPatch, "app_parameters", url.Values{"id": {"eq." + existing[0].ID}}, patch, &updated); err != nil {
			return domain.ParameterOption{}, err
		}
		if len(updated) == 0 {
			return domain.ParameterOption{}, ErrNotFound
		}
		return updated[0], nil
	}
	payload := map[string]any{"group_code": input.GroupCode, "code": code, "label": label, "applies_to": applies, "sort_order": input.SortOrder, "active": true, "created_by": input.Actor}
	var created []domain.ParameterOption
	if err := s.doJSON(ctx, http.MethodPost, "app_parameters", nil, payload, &created); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return domain.ParameterOption{}, ErrConflict
		}
		return domain.ParameterOption{}, err
	}
	if len(created) == 0 {
		return domain.ParameterOption{}, ErrNotFound
	}
	return created[0], nil
}

func (s *SupabaseStore) UpdateParameter(ctx context.Context, id string, input domain.NewParameterInput) (domain.ParameterOption, error) {
	label := strings.TrimSpace(input.Label)
	if label == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if strings.HasPrefix(id, "facility--") {
		facilityID := strings.TrimPrefix(id, "facility--")
		if facilityID == "" {
			return domain.ParameterOption{}, ErrNotFound
		}
		payload := map[string]any{"p_facility_id": facilityID, "p_name": label, "p_sort_order": input.SortOrder}
		var row struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Active    bool      `json:"active"`
			SortOrder int       `json:"sort_order"`
			CreatedAt time.Time `json:"created_at"`
		}
		if err := s.doJSON(ctx, http.MethodPost, "rpc/livira_update_facility_parameter", nil, payload, &row); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ParameterOption{}, ErrConflict
			}
			return domain.ParameterOption{}, mapRPCError(err)
		}
		if row.ID == "" {
			return domain.ParameterOption{}, ErrNotFound
		}
		return domain.ParameterOption{ID: id, GroupCode: domain.ParameterTPP, Code: row.ID, Label: row.Name, Active: row.Active, System: true, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt}, nil
	}
	query := url.Values{"select": {"*"}, "id": {"eq." + id}, "limit": {"1"}}
	var existing []domain.ParameterOption
	if err := s.doJSON(ctx, http.MethodGet, "app_parameters", query, nil, &existing); err != nil {
		return domain.ParameterOption{}, err
	}
	if len(existing) == 0 {
		return domain.ParameterOption{}, ErrNotFound
	}
	applies := normalizeAppliesTo(existing[0].GroupCode, input.AppliesTo)
	if existing[0].GroupCode == domain.ParameterExitType && applies == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	patch := map[string]any{"label": label, "applies_to": applies, "updated_at": time.Now().UTC()}
	if input.SortOrder > 0 {
		patch["sort_order"] = input.SortOrder
	}
	var updated []domain.ParameterOption
	if err := s.doJSON(ctx, http.MethodPatch, "app_parameters", url.Values{"id": {"eq." + id}, "select": {"*"}}, patch, &updated); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return domain.ParameterOption{}, ErrConflict
		}
		return domain.ParameterOption{}, err
	}
	if len(updated) == 0 {
		return domain.ParameterOption{}, ErrNotFound
	}
	return updated[0], nil
}

func (s *SupabaseStore) SetParameterActive(ctx context.Context, id string, active bool) error {
	if strings.HasPrefix(id, "facility--") {
		facilityID := strings.TrimPrefix(id, "facility--")
		if facilityID == "" {
			return ErrNotFound
		}
		if !active {
			query := url.Values{"select": {"id"}, "facility_id": {"eq." + facilityID}, "is_active": {"eq.true"}, "limit": {"1"}}
			var items []domain.InventoryItem
			if err := s.doJSON(ctx, http.MethodGet, "inventory_items", query, nil, &items); err != nil {
				return err
			}
			if len(items) > 0 {
				return ErrConflict
			}
		}
		return s.doJSON(ctx, http.MethodPatch, "facilities", url.Values{"id": {"eq." + facilityID}}, map[string]any{"active": active}, nil)
	}
	return s.doJSON(ctx, http.MethodPatch, "app_parameters", url.Values{"id": {"eq." + id}}, map[string]any{"active": active, "updated_at": time.Now().UTC()}, nil)
}
