package store

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/domain"
)

var editableInventoryCorrectionFields = []string{
	"reference_no", "item_type", "origin_type", "bl_no", "bl_date", "manifest_no", "manifest_date", "manifest_position",
	"determination_no", "determination_date", "category", "entrusted_category", "source_office",
	"description", "item_kind", "quantity", "unit", "goods_value", "goods_condition",
	"location", "location_status", "at_tpp", "owner_name", "owner_address", "origin_warehouse",
	"facility_id", "facility_name", "load_type", "container_no", "container_size", "estimated_volume_m3",
	"physical_unit_id", "occupancy_primary", "pfpd_required", "research_request_no", "research_request_date",
	"hs_code", "is_restricted", "restriction_rule", "origin_document_type", "origin_document_no",
	"origin_document_date", "allocation_purpose", "allocation_proposal_type", "allocation_proposal_no",
	"allocation_proposal_date", "allocation_approval_type", "allocation_approval_no", "allocation_approval_date",
	"exit_document_no", "exit_document_date", "exit_type", "exit_notes",
}

func inventoryCorrectionChanges(before, after domain.InventoryItem) []domain.ReconciliationChange {
	beforeValues := structJSONValues(before)
	afterValues := structJSONValues(after)
	changes := make([]domain.ReconciliationChange, 0)
	for _, field := range editableInventoryCorrectionFields {
		oldValue := beforeValues[field]
		newValue := afterValues[field]
		if oldValue == newValue {
			continue
		}
		changes = append(changes, domain.ReconciliationChange{
			Section: "inventory", Field: field, Before: oldValue, After: newValue,
		})
	}
	return changes
}

func eventCorrectionChanges(before, after domain.TimelineEvent) []domain.ReconciliationChange {
	contextLabel := strings.TrimSpace(before.Label)
	if contextLabel == "" {
		contextLabel = strings.TrimSpace(after.Label)
	}
	changes := make([]domain.ReconciliationChange, 0, 4)
	appendCorrectionChange(&changes, "timeline", before.ID, contextLabel, "label", before.Label, after.Label)
	appendCorrectionChange(&changes, "timeline", before.ID, contextLabel, "document_no", before.DocumentNo, after.DocumentNo)
	appendCorrectionChange(&changes, "timeline", before.ID, contextLabel, "document_date", correctionValue(before.DocumentDate), correctionValue(after.DocumentDate))
	appendCorrectionChange(&changes, "timeline", before.ID, contextLabel, "notes", before.Notes, after.Notes)
	return changes
}

func dispositionCorrectionChanges(before, after domain.Disposition) []domain.ReconciliationChange {
	contextLabel := strings.TrimSpace(before.StatusLabel)
	if contextLabel == "" {
		contextLabel = strings.ToUpper(string(before.Type))
	}
	changes := make([]domain.ReconciliationChange, 0, 13)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "proposal_type", before.ProposalType, after.ProposalType)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "recipient_code", before.RecipientCode, after.RecipientCode)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "recipient_name", before.RecipientName, after.RecipientName)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "sale_value", correctionValue(before.SaleValue), correctionValue(after.SaleValue))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "htl_value", correctionValue(before.HTLValue), correctionValue(after.HTLValue))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "execution_start_date", correctionValue(before.ExecutionStartDate), correctionValue(after.ExecutionStartDate))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "execution_end_date", correctionValue(before.ExecutionEndDate), correctionValue(after.ExecutionEndDate))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "schedule_document_no", before.ScheduleDocumentNo, after.ScheduleDocumentNo)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "schedule_document_date", correctionValue(before.ScheduleDocumentDate), correctionValue(after.ScheduleDocumentDate))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "auction_outcome", before.AuctionOutcome, after.AuctionOutcome)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "allocation_target", before.AllocationTarget, after.AllocationTarget)
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "destruction_cost", correctionValue(before.DestructionCost), correctionValue(after.DestructionCost))
	appendCorrectionChange(&changes, "process", before.ID, contextLabel, "transfer_type", before.TransferType, after.TransferType)
	return changes
}

func appendCorrectionChange(changes *[]domain.ReconciliationChange, section, entityID, contextLabel, field, before, after string) {
	before = strings.TrimSpace(before)
	after = strings.TrimSpace(after)
	if before == after {
		return
	}
	*changes = append(*changes, domain.ReconciliationChange{
		Section: section, EntityID: entityID, Context: contextLabel, Field: field, Before: before, After: after,
	})
}

func structJSONValues(value any) map[string]string {
	result := make(map[string]string)
	reflected := reflect.Indirect(reflect.ValueOf(value))
	if !reflected.IsValid() || reflected.Kind() != reflect.Struct {
		return result
	}
	typeInfo := reflected.Type()
	for index := 0; index < reflected.NumField(); index++ {
		fieldInfo := typeInfo.Field(index)
		jsonName := strings.Split(fieldInfo.Tag.Get("json"), ",")[0]
		if jsonName == "" || jsonName == "-" {
			continue
		}
		result[jsonName] = correctionValue(reflected.Field(index).Interface())
	}
	return result
}

func correctionValue(value any) string {
	switch typed := value.(type) {
	case time.Time:
		if typed.IsZero() {
			return ""
		}
		return typed.UTC().Format("2006-01-02")
	case string:
		return strings.TrimSpace(typed)
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case domain.InventoryType:
		return string(typed)
	case domain.DispositionType:
		return string(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
