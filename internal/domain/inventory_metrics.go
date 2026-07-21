package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var nonContainerIdentifier = regexp.MustCompile(`[^A-Z0-9]+`)

// SummarizeDashboardInventory deduplicates logical documents and physical
// loading units. Multiple goods rows in one container/LCL therefore do not
// inflate the document, FCL, or LCL figures displayed on the dashboard.
func SummarizeDashboardInventory(items []InventoryItem) DashboardInventorySummary {
	documents := make(map[string]struct{})
	fclUnits := make(map[string]struct{})
	lclUnits := make(map[string]struct{})
	for _, item := range items {
		if !item.IsActive {
			continue
		}
		documentNo := strings.ToUpper(strings.Join(strings.Fields(item.DeterminationNo), " "))
		if documentNo == "" {
			documentNo = strings.ToUpper(strings.TrimSpace(item.ReferenceNo))
		}
		documentDate := ""
		if !item.DeterminationDate.IsZero() {
			documentDate = item.DeterminationDate.Format("2006-01-02")
		}
		documents[fmt.Sprintf("%s|%s|%s", item.Type, documentNo, documentDate)] = struct{}{}

		switch strings.ToUpper(strings.TrimSpace(item.LoadType)) {
		case "FCL":
			container := nonContainerIdentifier.ReplaceAllString(strings.ToUpper(item.ContainerNo), "")
			if container == "" {
				container = strings.TrimSpace(item.PhysicalUnitID)
			}
			if container != "" {
				fclUnits[container] = struct{}{}
			}
		case "LCL":
			unit := strings.TrimSpace(item.PhysicalUnitID)
			if unit == "" {
				unit = fmt.Sprintf("%s|%s|%s", item.Type, documentNo, documentDate)
			}
			lclUnits[unit] = struct{}{}
		}
	}
	return DashboardInventorySummary{Documents: len(documents), FCL: len(fclUnits), LCL: len(lclUnits)}
}
