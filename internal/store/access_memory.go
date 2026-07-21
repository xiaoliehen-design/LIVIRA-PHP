package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

func (m *MemoryStore) seedAccessControl() {
	now := m.now().UTC()
	inventoryPermissions := []string{domain.PermissionDashboardView, domain.PermissionDashboardCapacity, domain.PermissionInventoryView, domain.PermissionInventoryBTD, domain.PermissionInventoryBDN, domain.PermissionInventoryBMMN, domain.PermissionInventoryTitipan, domain.PermissionReconciliationView, domain.PermissionReconciliationManage, domain.PermissionReportsView, domain.PermissionSearchView}
	inventoryPermissions = append(inventoryPermissions, domain.InventoryCreatePermissionCodes()...)
	inventoryPermissions = append(inventoryPermissions, domain.InventoryActionPermissionCodes()...)
	bmmnPermissions := []string{domain.PermissionDashboardView, domain.PermissionInventoryView, domain.PermissionInventoryBMMN, domain.PermissionReportsView, domain.PermissionSearchView}
	bmmnPermissions = append(bmmnPermissions, domain.InventoryActionPermissionCodes()...)
	seedRoles := []domain.RoleProfile{
		{ID: "role-inventory", Name: "Petugas Inventory", Description: "Kelola inventory BTD, BDN, BMMN, barang titipan, dan rekonsiliasi.", Permissions: inventoryPermissions, Active: true, System: true},
		{ID: "role-auction", Name: "Petugas Lelang", Description: "Akses khusus proses dan dashboard lelang.", Permissions: []string{domain.PermissionDashboardView, domain.PermissionAuctionView, domain.PermissionAuctionManage, domain.PermissionInventoryBTD, domain.PermissionInventoryBDN, domain.PermissionInventoryBMMN, domain.PermissionSearchView}, Active: true, System: true},
		{ID: "role-bmmn", Name: "Petugas BMMN", Description: "Akses inventory dan laporan khusus BMMN.", Permissions: bmmnPermissions, Active: true, System: true},
		{ID: "role-grant", Name: "Petugas Hibah / PSP", Description: "Akses khusus penyelesaian hibah dan PSP.", Permissions: []string{domain.PermissionDashboardView, domain.PermissionGrantView, domain.PermissionGrantManage, domain.PermissionInventoryBMMN, domain.PermissionSearchView}, Active: true, System: true},
		{ID: "role-viewer", Name: "Viewer", Description: "Akses baca tanpa perubahan data.", Permissions: []string{domain.PermissionDashboardView, domain.PermissionInventoryView, domain.PermissionInventoryBTD, domain.PermissionInventoryBDN, domain.PermissionInventoryBMMN, domain.PermissionInventoryTitipan, domain.PermissionReconciliationView, domain.PermissionReportsView, domain.PermissionSearchView}, Active: true, System: true},
	}
	for _, role := range seedRoles {
		role.CreatedAt, role.UpdatedAt = now, now
		role.Permissions = domain.NormalizePermissions(role.Permissions)
		m.roles[role.ID] = role
	}
	m.users["user-pending-1"] = domain.UserAccount{ID: "user-pending-1", AuthUserID: "demo-pending-1", Name: "Rina Kartika", Email: "rina.kartika@example.go.id", EmailVerified: true, EmailVerifiedAt: now.Add(-2 * time.Hour), ApprovalStatus: "pending", SessionVersion: 1, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)}
	m.users["user-pending-2"] = domain.UserAccount{ID: "user-pending-2", AuthUserID: "demo-pending-2", Name: "Bagus Prasetyo", Email: "bagus.prasetyo@example.go.id", EmailVerified: false, ApprovalStatus: "pending", SessionVersion: 1, CreatedAt: now.Add(-45 * time.Minute), UpdatedAt: now.Add(-45 * time.Minute)}

	seedParameter := func(id, group, code, label, applies string, order int) {
		m.parameters[id] = domain.ParameterOption{ID: id, GroupCode: group, Code: code, Label: label, AppliesTo: applies, Active: true, System: true, SortOrder: order, CreatedAt: now, UpdatedAt: now}
	}
	for index, label := range domain.BDNCategoryNames {
		seedParameter(fmt.Sprintf("param-bdn-%02d", index+1), domain.ParameterBDNCategory, slugCode(label), label, "BDN", index+1)
	}
	for index, label := range domain.ItemKindNames {
		seedParameter(fmt.Sprintf("param-kind-%02d", index+1), domain.ParameterItemKind, slugCode(label), label, "BTD,BDN,BMMN,TITIPAN", index+1)
	}
	for index, label := range domain.GoodsConditionNames {
		seedParameter(fmt.Sprintf("param-condition-%02d", index+1), domain.ParameterGoodsCondition, slugCode(label), label, "BTD,BDN,BMMN,TITIPAN", index+1)
	}
	for index, label := range domain.UnitNames {
		seedParameter(fmt.Sprintf("param-unit-%02d", index+1), domain.ParameterUnit, slugCode(label), label, "BTD,BDN,BMMN,TITIPAN", index+1)
	}
	for index, label := range domain.AllocationPurposeNames {
		seedParameter(fmt.Sprintf("param-allocation-%02d", index+1), domain.ParameterAllocationPurpose, slugCode(label), label, "BMMN", index+1)
	}
	for index, label := range domain.TPSNames {
		seedParameter(fmt.Sprintf("param-tps-%02d", index+1), domain.ParameterOriginTPS, slugCode(label), label, "BTD,BDN", index+1)
	}
	for index, option := range domain.LoadTypeOptions {
		seedParameter(fmt.Sprintf("param-load-%02d", index+1), domain.ParameterLoadType, option.Code, option.Label, "BTD,BDN,BMMN,TITIPAN", index+1)
	}
	for index, option := range domain.ExitOptions {
		seedParameter(fmt.Sprintf("param-exit-%02d", index+1), domain.ParameterExitType, option.Code, option.Label, option.Types, index+1)
	}
	for index, option := range domain.TransferTypeOptions {
		seedParameter(fmt.Sprintf("param-transfer-%02d", index+1), domain.ParameterTransferType, option.Code, option.Label, option.Types, index+1)
	}
	options, _ := m.ParameterOptions(context.Background(), "", true)
	domain.SetRuntimeParameters(options)
}

func slugCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var result []rune
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			result = append(result, r)
			lastDash = false
		} else if !lastDash && len(result) > 0 {
			result = append(result, '_')
			lastDash = true
		}
	}
	return strings.Trim(string(result), "_")
}

func (m *MemoryStore) CreateUserApplication(_ context.Context, input domain.NewUserApplicationInput) (domain.UserAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, user := range m.users {
		if user.AuthUserID == input.AuthUserID || strings.EqualFold(user.Email, input.Email) {
			user.AuthUserID, user.Name, user.Email = input.AuthUserID, input.Name, strings.ToLower(input.Email)
			user.UpdatedAt = m.now().UTC()
			m.users[id] = user
			return user, nil
		}
	}
	m.nextUser++
	now := m.now().UTC()
	user := domain.UserAccount{ID: fmt.Sprintf("user-%03d", m.nextUser), AuthUserID: input.AuthUserID, Name: strings.TrimSpace(input.Name), Email: strings.ToLower(strings.TrimSpace(input.Email)), ApprovalStatus: "pending", SessionVersion: 1, CreatedAt: now, UpdatedAt: now}
	m.users[user.ID] = user
	return user, nil
}

func (m *MemoryStore) MarkUserEmailVerified(_ context.Context, authUserID, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, user := range m.users {
		if user.AuthUserID == authUserID || strings.EqualFold(user.Email, email) {
			user.EmailVerified = true
			user.EmailVerifiedAt = m.now().UTC()
			user.UpdatedAt = user.EmailVerifiedAt
			m.users[id] = user
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemoryStore) UserByAuthID(_ context.Context, authUserID string) (domain.UserAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.AuthUserID != authUserID {
			continue
		}
		if role, ok := m.roles[user.RoleID]; ok {
			user.RoleName = role.Name
			user.Permissions = append([]string(nil), role.Permissions...)
		}
		return user, nil
	}
	return domain.UserAccount{}, ErrNotFound
}

func (m *MemoryStore) ListUsers(_ context.Context) ([]domain.UserAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.UserAccount, 0, len(m.users))
	for _, user := range m.users {
		if role, ok := m.roles[user.RoleID]; ok {
			user.RoleName = role.Name
			user.Permissions = append([]string(nil), role.Permissions...)
		}
		result = append(result, user)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (m *MemoryStore) ApproveUser(_ context.Context, id, roleID, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	role, ok := m.roles[roleID]
	if !ok || !role.Active || !user.EmailVerified {
		return ErrInvalidTransition
	}
	now := m.now().UTC()
	user.RoleID, user.ApprovalStatus, user.ApprovedBy, user.ApprovedAt, user.UpdatedAt = roleID, "approved", actor, now, now
	user.SessionVersion++
	user.RejectionReason = ""
	m.users[id] = user
	return nil
}

func (m *MemoryStore) RejectUser(_ context.Context, id, reason, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	now := m.now().UTC()
	user.ApprovalStatus, user.RejectionReason, user.ApprovedBy, user.ApprovedAt, user.UpdatedAt = "rejected", strings.TrimSpace(reason), actor, now, now
	user.RoleID = ""
	user.SessionVersion++
	m.users[id] = user
	return nil
}

func (m *MemoryStore) UpdateUserRole(_ context.Context, id, roleID, actor string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	role, ok := m.roles[roleID]
	if !ok || !role.Active || user.ApprovalStatus != "approved" {
		return ErrInvalidTransition
	}
	user.RoleID = roleID
	user.SessionVersion++
	user.ApprovedBy = actor
	user.UpdatedAt = m.now().UTC()
	m.users[id] = user
	return nil
}

func (m *MemoryStore) DeleteUser(_ context.Context, id string) (domain.UserAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return domain.UserAccount{}, ErrNotFound
	}
	delete(m.users, id)
	return user, nil
}

func (m *MemoryStore) ListRoles(_ context.Context, includeInactive bool) ([]domain.RoleProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	assignedUsers := make(map[string]int, len(m.roles))
	for _, user := range m.users {
		if user.RoleID != "" {
			assignedUsers[user.RoleID]++
		}
	}
	result := make([]domain.RoleProfile, 0, len(m.roles))
	for _, role := range m.roles {
		if !includeInactive && !role.Active {
			continue
		}
		role.Permissions = append([]string(nil), role.Permissions...)
		role.AssignedUsers = assignedUsers[role.ID]
		result = append(result, role)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m *MemoryStore) CreateRole(_ context.Context, input domain.NewRoleInput) (domain.RoleProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := strings.TrimSpace(input.Name)
	if name == "" || len(domain.NormalizePermissions(input.Permissions)) == 0 {
		return domain.RoleProfile{}, ErrInvalidTransition
	}
	for _, role := range m.roles {
		if strings.EqualFold(role.Name, name) {
			return domain.RoleProfile{}, ErrConflict
		}
	}
	m.nextRole++
	now := m.now().UTC()
	role := domain.RoleProfile{ID: fmt.Sprintf("role-%03d", m.nextRole), Name: name, Description: strings.TrimSpace(input.Description), Permissions: domain.NormalizePermissions(input.Permissions), Active: true, CreatedAt: now, UpdatedAt: now}
	m.roles[role.ID] = role
	return role, nil
}

func (m *MemoryStore) UpdateRole(_ context.Context, id string, input domain.NewRoleInput) (domain.RoleProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.roles[id]
	if !ok {
		return domain.RoleProfile{}, ErrNotFound
	}
	name, permissions := strings.TrimSpace(input.Name), domain.NormalizePermissions(input.Permissions)
	if name == "" || len(permissions) == 0 {
		return domain.RoleProfile{}, ErrInvalidTransition
	}
	for otherID, other := range m.roles {
		if otherID != id && strings.EqualFold(other.Name, name) {
			return domain.RoleProfile{}, ErrConflict
		}
	}
	role.Name, role.Description, role.Permissions, role.UpdatedAt = name, strings.TrimSpace(input.Description), permissions, m.now().UTC()
	m.roles[id] = role
	for userID, user := range m.users {
		if user.RoleID == id {
			user.SessionVersion++
			user.UpdatedAt = m.now().UTC()
			m.users[userID] = user
		}
	}
	return role, nil
}

func (m *MemoryStore) SetRoleActive(_ context.Context, id string, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	role, ok := m.roles[id]
	if !ok {
		return ErrNotFound
	}
	if !active {
		for _, user := range m.users {
			if user.RoleID == id && user.ApprovalStatus == "approved" {
				return ErrConflict
			}
		}
	}
	role.Active, role.UpdatedAt = active, m.now().UTC()
	m.roles[id] = role
	for userID, user := range m.users {
		if user.RoleID == id {
			user.SessionVersion++
			user.UpdatedAt = m.now().UTC()
			m.users[userID] = user
		}
	}
	return nil
}

func (m *MemoryStore) DeleteRole(_ context.Context, id string) (domain.RoleProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id = strings.TrimSpace(id)
	role, ok := m.roles[id]
	if !ok {
		return domain.RoleProfile{}, ErrNotFound
	}
	for _, user := range m.users {
		if user.RoleID == id {
			return domain.RoleProfile{}, ErrRoleInUse
		}
	}
	delete(m.roles, id)
	role.Permissions = append([]string(nil), role.Permissions...)
	role.AssignedUsers = 0
	return role, nil
}

func (m *MemoryStore) ParameterOptions(_ context.Context, group string, includeInactive bool) ([]domain.ParameterOption, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.ParameterOption, 0, len(m.parameters)+len(m.facilities))
	for _, option := range m.parameters {
		if group != "" && option.GroupCode != group || !includeInactive && !option.Active {
			continue
		}
		result = append(result, option)
	}
	if group == "" || group == domain.ParameterTPP {
		for index, facility := range m.facilities {
			if !includeInactive && !facility.Active {
				continue
			}
			order := facility.SortOrder
			if order <= 0 {
				order = index + 1
			}
			result = append(result, domain.ParameterOption{
				ID: "facility--" + facility.ID, GroupCode: domain.ParameterTPP, Code: facility.ID,
				Label: facility.Name, Active: facility.Active, System: true, SortOrder: order,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].GroupCode == result[j].GroupCode {
			if result[i].SortOrder == result[j].SortOrder {
				return result[i].Label < result[j].Label
			}
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].GroupCode < result[j].GroupCode
	})
	return result, nil
}

func (m *MemoryStore) CreateParameter(_ context.Context, input domain.NewParameterInput) (domain.ParameterOption, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !domain.ValidParameterGroup(input.GroupCode) {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	label, code := strings.TrimSpace(input.Label), slugCode(input.Code)
	if label == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if code == "" {
		code = slugCode(label)
	}
	if input.GroupCode == domain.ParameterTPP {
		for index, facility := range m.facilities {
			if strings.EqualFold(facility.ID, code) || strings.EqualFold(facility.Name, label) {
				if facility.Active {
					return domain.ParameterOption{}, ErrConflict
				}
				facility.Active, facility.Name = true, label
				if input.SortOrder > 0 {
					facility.SortOrder = input.SortOrder
				}
				m.facilities[index] = facility
				return domain.ParameterOption{ID: "facility--" + facility.ID, GroupCode: domain.ParameterTPP, Code: facility.ID, Label: facility.Name, Active: true, SortOrder: facility.SortOrder}, nil
			}
		}
		facility := domain.Facility{ID: code, Name: label, Active: true, SortOrder: input.SortOrder}
		if facility.SortOrder <= 0 {
			facility.SortOrder = (len(m.facilities) + 1) * 10
		}
		m.facilities = append(m.facilities, facility)
		return domain.ParameterOption{ID: "facility--" + facility.ID, GroupCode: domain.ParameterTPP, Code: facility.ID, Label: facility.Name, Active: true, SortOrder: facility.SortOrder}, nil
	}
	for id, option := range m.parameters {
		if option.GroupCode == input.GroupCode && (strings.EqualFold(option.Code, code) || strings.EqualFold(option.Label, label)) {
			if !option.Active {
				option.Active, option.Label, option.AppliesTo, option.UpdatedAt = true, label, input.AppliesTo, m.now().UTC()
				m.parameters[id] = option
				return option, nil
			}
			return domain.ParameterOption{}, ErrConflict
		}
	}
	m.nextParameter++
	now := m.now().UTC()
	option := domain.ParameterOption{ID: fmt.Sprintf("param-%03d", m.nextParameter), GroupCode: input.GroupCode, Code: code, Label: label, AppliesTo: normalizeAppliesTo(input.GroupCode, input.AppliesTo), Active: true, SortOrder: input.SortOrder, CreatedAt: now, UpdatedAt: now}
	m.parameters[option.ID] = option
	return option, nil
}

func (m *MemoryStore) UpdateParameter(_ context.Context, id string, input domain.NewParameterInput) (domain.ParameterOption, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	label := strings.TrimSpace(input.Label)
	if label == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if strings.HasPrefix(id, "facility--") {
		facilityID := strings.TrimPrefix(id, "facility--")
		for index, facility := range m.facilities {
			if facility.ID != facilityID {
				continue
			}
			for otherIndex, other := range m.facilities {
				if otherIndex != index && strings.EqualFold(other.Name, label) {
					return domain.ParameterOption{}, ErrConflict
				}
			}
			facility.Name = label
			if input.SortOrder > 0 {
				facility.SortOrder = input.SortOrder
			}
			m.facilities[index] = facility
			for itemID, item := range m.items {
				if item.FacilityID == facility.ID {
					oldName := item.FacilityName
					item.FacilityName = facility.Name
					if item.AtTPP && (item.LocationStatus == oldName || item.LocationStatus == "") {
						item.LocationStatus = facility.Name
					}
					item.UpdatedAt = m.now().UTC()
					m.items[itemID] = item
				}
			}
			return domain.ParameterOption{ID: id, GroupCode: domain.ParameterTPP, Code: facility.ID, Label: facility.Name, Active: facility.Active, System: true, SortOrder: facility.SortOrder}, nil
		}
		return domain.ParameterOption{}, ErrNotFound
	}
	option, ok := m.parameters[id]
	if !ok {
		return domain.ParameterOption{}, ErrNotFound
	}
	for otherID, other := range m.parameters {
		if otherID != id && other.GroupCode == option.GroupCode && strings.EqualFold(other.Label, label) {
			return domain.ParameterOption{}, ErrConflict
		}
	}
	option.Label = label
	option.AppliesTo = normalizeAppliesTo(option.GroupCode, input.AppliesTo)
	if option.GroupCode == domain.ParameterExitType && option.AppliesTo == "" {
		return domain.ParameterOption{}, ErrInvalidTransition
	}
	if input.SortOrder > 0 {
		option.SortOrder = input.SortOrder
	}
	option.UpdatedAt = m.now().UTC()
	m.parameters[id] = option
	return option, nil
}

func (m *MemoryStore) SetParameterActive(_ context.Context, id string, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.HasPrefix(id, "facility--") {
		facilityID := strings.TrimPrefix(id, "facility--")
		if !active {
			for _, item := range m.items {
				if item.IsActive && item.FacilityID == facilityID {
					return ErrConflict
				}
			}
		}
		for index, facility := range m.facilities {
			if facility.ID == facilityID {
				facility.Active = active
				m.facilities[index] = facility
				return nil
			}
		}
		return ErrNotFound
	}
	option, ok := m.parameters[id]
	if !ok {
		return ErrNotFound
	}
	option.Active, option.UpdatedAt = active, m.now().UTC()
	m.parameters[id] = option
	return nil
}

func normalizeAppliesTo(group, value string) string {
	switch group {
	case domain.ParameterBDNCategory:
		return "BDN"
	case domain.ParameterAllocationPurpose, domain.ParameterTransferType:
		return "BMMN"
	case domain.ParameterOriginTPS:
		return "BTD,BDN"
	case domain.ParameterItemKind, domain.ParameterGoodsCondition, domain.ParameterUnit, domain.ParameterLoadType:
		return "BTD,BDN,BMMN,TITIPAN"
	}
	allowed := map[string]bool{"BTD": true, "BDN": true, "BMMN": true, "TITIPAN": true}
	seen := map[string]bool{}
	result := make([]string, 0, 4)
	for _, raw := range strings.Split(strings.ToUpper(value), ",") {
		raw = strings.TrimSpace(raw)
		if allowed[raw] && !seen[raw] {
			seen[raw] = true
			result = append(result, raw)
		}
	}
	return strings.Join(result, ",")
}
