package web

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hendra/manajemen-tpp/internal/auth"
	"github.com/hendra/manajemen-tpp/internal/domain"
)

const (
	performanceAuction     = "auction"
	performanceDestruction = "destruction"
	performanceGrant       = "grant"
	performanceCensus      = "census"
	performancePFPD        = "pfpd"
	performanceBMMN        = "bmmn_conversion"
)

type PerformanceMetric struct {
	Code            string
	Label           string
	Description     string
	Count           int
	DurationSamples int
	AverageHours    float64
	AverageDays     float64
}

type PerformanceDetail struct {
	MetricCode         string
	MetricLabel        string
	CompletionDocument string
	CompletionDate     time.Time
	StartDocument      string
	StartDate          time.Time
	DurationHours      float64
	DurationDays       float64
	DurationValid      bool
	InventoryCount     int
}

type PerformanceReport struct {
	DateFrom       time.Time
	DateTo         time.Time
	DateFromInput  string
	DateToInput    string
	PeriodLabel    string
	TotalCompleted int
	Metrics        []PerformanceMetric
	Details        []PerformanceDetail
	ExportURL      string
}

type performanceDefinition struct {
	Code        string
	Label       string
	Description string
	EventCode   string
}

var performanceDefinitions = []performanceDefinition{
	{Code: performanceAuction, Label: "Performa lelang", Description: "Selesai lelang, dihitung sejak penetapan awal BTD/BDN.", EventCode: "selesai_lelang"},
	{Code: performanceDestruction, Label: "Performa musnah", Description: "BA Musnah, dihitung sejak penetapan awal BTD/BDN.", EventCode: "ba_musnah"},
	{Code: performanceGrant, Label: "Performa hibah/PSP", Description: "BA Serah Terima Hibah/PSP, dihitung sejak penetapan awal BTD/BDN.", EventCode: "ba_serah_terima"},
	{Code: performanceCensus, Label: "Performa cacah", Description: "Pencacahan selesai, dihitung sejak penetapan sampai BA Cacah.", EventCode: "pencacahan"},
	{Code: performancePFPD, Label: "Performa penilaian PFPD", Description: "Penilaian selesai, dihitung sejak request penelitian PFPD.", EventCode: "penelitian_pfpd"},
	{Code: performanceBMMN, Label: "Konversi BMMN", Description: "Penetapan BMMN dari BTD/BDN, dihitung sejak penetapan awal.", EventCode: "penetapan_bmmn"},
}

type performanceGroup struct {
	definition   performanceDefinition
	documentNo   string
	completion   time.Time
	start        time.Time
	startDoc     string
	inventoryIDs map[string]struct{}
}

func performanceRange(queryFrom, queryTo string, now time.Time) (time.Time, time.Time) {
	from := parseDate(queryFrom)
	to := parseDate(queryTo)
	if from.IsZero() && to.IsZero() {
		from = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		to = time.Date(now.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
	} else {
		if from.IsZero() {
			from = time.Date(to.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		if to.IsZero() {
			to = time.Date(from.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
		}
	}
	if from.After(to) {
		from, to = to, from
	}
	return from, to
}

func (s *Server) performanceReport(ctx context.Context, session auth.Session, from, to time.Time) (PerformanceReport, error) {
	items, events, err := s.store.PerformanceSource(ctx, from, to, allowedInventoryTypes(session))
	if err != nil {
		return PerformanceReport{}, err
	}
	return buildPerformanceReport(items, events, from, to), nil
}

func buildPerformanceReport(items []domain.InventoryItem, events []domain.TimelineEvent, from, to time.Time) PerformanceReport {
	report := PerformanceReport{
		DateFrom:      from,
		DateTo:        to,
		DateFromInput: from.Format("2006-01-02"),
		DateToInput:   to.Format("2006-01-02"),
		PeriodLabel:   performancePeriodLabel(from, to),
	}
	itemByID := make(map[string]domain.InventoryItem, len(items))
	for _, item := range items {
		if item.Type == domain.InventoryTitipan {
			continue
		}
		itemByID[item.ID] = item
	}
	eventsByInventory := make(map[string][]domain.TimelineEvent)
	for _, event := range events {
		if _, ok := itemByID[event.InventoryID]; !ok {
			continue
		}
		eventsByInventory[event.InventoryID] = append(eventsByInventory[event.InventoryID], event)
	}
	groups := make(map[string]*performanceGroup)
	for inventoryID, item := range itemByID {
		itemEvents := eventsByInventory[inventoryID]
		sort.SliceStable(itemEvents, func(i, j int) bool {
			if itemEvents[i].CreatedAt.Equal(itemEvents[j].CreatedAt) {
				return itemEvents[i].ID < itemEvents[j].ID
			}
			return itemEvents[i].CreatedAt.Before(itemEvents[j].CreatedAt)
		})
		preferences := performanceEventPreferences(itemEvents)
		initialDate, initialDocument := initialDetermination(item, itemEvents)
		requestDate := time.Time{}
		requestDocument := ""
		for _, event := range itemEvents {
			completion := performanceEventDate(event)
			if event.Code == "request_penelitian_pfpd" || event.Code == "siap_peruntukan" {
				requestDate = completion
				requestDocument = strings.TrimSpace(event.DocumentNo)
				continue
			}
			definition, matched := performanceDefinitionForEvent(event.Code)
			if !matched || completion.IsZero() || !dateWithin(completion, from, to) {
				continue
			}
			if !preferences.include(event, definition) {
				continue
			}
			start, startDocument := initialDate, initialDocument
			if definition.Code == performancePFPD {
				start, startDocument = requestDate, requestDocument
				if start.IsZero() && !item.ResearchRequestDate.IsZero() && !item.ResearchRequestDate.After(completion) {
					start, startDocument = item.ResearchRequestDate, item.ResearchRequestNo
				}
			}
			key := performanceGroupKey(definition.Code, event, completion)
			group := groups[key]
			if group == nil {
				group = &performanceGroup{
					definition:   definition,
					documentNo:   strings.TrimSpace(event.DocumentNo),
					completion:   completion,
					start:        start,
					startDoc:     startDocument,
					inventoryIDs: make(map[string]struct{}),
				}
				groups[key] = group
			}
			if completion.After(group.completion) {
				group.completion = completion
			}
			if !start.IsZero() && (group.start.IsZero() || start.Before(group.start)) {
				group.start = start
				group.startDoc = startDocument
			}
			group.inventoryIDs[inventoryID] = struct{}{}
		}
	}

	metricIndexByCode := make(map[string]int, len(performanceDefinitions))
	for _, definition := range performanceDefinitions {
		metricIndexByCode[definition.Code] = len(report.Metrics)
		report.Metrics = append(report.Metrics, PerformanceMetric{Code: definition.Code, Label: definition.Label, Description: definition.Description})
	}
	var durationSums = make(map[string]float64, len(performanceDefinitions))
	for _, group := range groups {
		metricIndex, ok := metricIndexByCode[group.definition.Code]
		if !ok {
			continue
		}
		metric := &report.Metrics[metricIndex]
		metric.Count++
		report.TotalCompleted++
		detail := PerformanceDetail{
			MetricCode:         group.definition.Code,
			MetricLabel:        group.definition.Label,
			CompletionDocument: group.documentNo,
			CompletionDate:     group.completion,
			StartDocument:      group.startDoc,
			StartDate:          group.start,
			InventoryCount:     len(group.inventoryIDs),
		}
		if !group.start.IsZero() && !group.completion.Before(group.start) {
			detail.DurationValid = true
			detail.DurationHours = group.completion.Sub(group.start).Hours()
			detail.DurationDays = detail.DurationHours / 24
			metric.DurationSamples++
			durationSums[group.definition.Code] += detail.DurationHours
		}
		report.Details = append(report.Details, detail)
	}
	for index := range report.Metrics {
		metric := &report.Metrics[index]
		if metric.DurationSamples > 0 {
			metric.AverageHours = durationSums[metric.Code] / float64(metric.DurationSamples)
			metric.AverageDays = metric.AverageHours / 24
		}
	}
	sort.Slice(report.Details, func(i, j int) bool {
		if report.Details[i].CompletionDate.Equal(report.Details[j].CompletionDate) {
			return report.Details[i].MetricLabel < report.Details[j].MetricLabel
		}
		return report.Details[i].CompletionDate.After(report.Details[j].CompletionDate)
	})
	report.ExportURL = fmt.Sprintf("/pelaporan/performa.xlsx?date_from=%s&date_to=%s", report.DateFromInput, report.DateToInput)
	return report
}

type performanceEventPreference struct {
	hasCanonical            bool
	hasCanonicalDisposition bool
	hasAnyDisposition       bool
}

type performanceEventPreferenceMap map[string]performanceEventPreference

func performanceEventPreferences(events []domain.TimelineEvent) performanceEventPreferenceMap {
	result := make(performanceEventPreferenceMap)
	for _, event := range events {
		definition, matched := performanceDefinitionForEvent(event.Code)
		if !matched {
			continue
		}
		preference := result[definition.Code]
		hasDisposition := strings.TrimSpace(event.DispositionID) != ""
		if hasDisposition {
			preference.hasAnyDisposition = true
		}
		if isCanonicalPerformanceEvent(event.Code, definition) {
			preference.hasCanonical = true
			if hasDisposition {
				preference.hasCanonicalDisposition = true
			}
		}
		result[definition.Code] = preference
	}
	return result
}

func (preferences performanceEventPreferenceMap) include(event domain.TimelineEvent, definition performanceDefinition) bool {
	preference := preferences[definition.Code]
	canonical := isCanonicalPerformanceEvent(event.Code, definition)
	if !canonical && preference.hasCanonical {
		// Data lama dapat menyimpan status akhir seperti "laku" atau
		// "tidak_laku" di samping event resmi selesai_lelang. Event resmi
		// diprioritaskan agar satu penyelesaian tidak dihitung dua kali.
		return false
	}
	if canonical && isDispositionPerformanceMetric(definition.Code) && preference.hasCanonicalDisposition && strings.TrimSpace(event.DispositionID) == "" {
		// Beberapa seed/rekaman lama memiliki salinan status akhir tanpa
		// referensi proses. Jika event resmi proses tersedia, abaikan salinan.
		return false
	}
	if !preference.hasCanonical && isDispositionPerformanceMetric(definition.Code) && preference.hasAnyDisposition && strings.TrimSpace(event.DispositionID) == "" {
		// Jika seluruh data masih memakai kode status lama, event yang memiliki
		// referensi proses tetap lebih kuat daripada salinan status inventory.
		return false
	}
	return true
}

func isCanonicalPerformanceEvent(code string, definition performanceDefinition) bool {
	return strings.EqualFold(strings.TrimSpace(code), definition.EventCode)
}

func isDispositionPerformanceMetric(code string) bool {
	return code == performanceAuction || code == performanceDestruction || code == performanceGrant
}

func performanceDefinitionForEvent(code string) (performanceDefinition, bool) {
	normalized := strings.TrimSpace(strings.ToLower(code))
	for _, definition := range performanceDefinitions {
		if normalized == definition.EventCode {
			return definition, true
		}
	}
	switch normalized {
	case "laku", "tidak_laku":
		return performanceDefinitions[0], true
	case "penelitian_hs_lartas":
		return performanceDefinitions[4], true
	default:
		return performanceDefinition{}, false
	}
}

func initialDetermination(item domain.InventoryItem, events []domain.TimelineEvent) (time.Time, string) {
	if !item.OriginDocumentDate.IsZero() {
		return item.OriginDocumentDate, item.OriginDocumentNo
	}
	var earliest time.Time
	document := ""
	for _, event := range events {
		if event.Code != "ditetapkan" && event.Code != "masih_di_tps" {
			continue
		}
		candidate := performanceEventDate(event)
		if candidate.IsZero() || !earliest.IsZero() && !candidate.Before(earliest) {
			continue
		}
		earliest, document = candidate, event.DocumentNo
	}
	if !earliest.IsZero() {
		return earliest, document
	}
	return item.DeterminationDate, item.DeterminationNo
}

func performanceEventDate(event domain.TimelineEvent) time.Time {
	if !event.DocumentDate.IsZero() {
		return event.DocumentDate
	}
	return event.CreatedAt
}

func performanceGroupKey(metricCode string, event domain.TimelineEvent, completion time.Time) string {
	document := strings.ToUpper(strings.TrimSpace(event.DocumentNo))
	key := metricCode + "|" + document + "|" + completion.Format("2006-01-02")
	if document == "" {
		if strings.TrimSpace(event.DispositionID) != "" {
			key += "|" + event.DispositionID
		} else {
			key += "|" + event.ID
		}
	}
	return key
}

func dateWithin(value, from, to time.Time) bool {
	valueDate := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)
	return !valueDate.Before(fromDate) && !valueDate.After(toDate)
}

func performancePeriodLabel(from, to time.Time) string {
	if from.Year() == to.Year() && from.Month() == time.January && from.Day() == 1 && to.Month() == time.December && to.Day() == 31 {
		return fmt.Sprintf("Tahun %d", from.Year())
	}
	return fmt.Sprintf("%s – %s", from.Format("02 Jan 2006"), to.Format("02 Jan 2006"))
}

func formatPerformanceDuration(hours float64, samples int) string {
	if samples <= 0 {
		return "—"
	}
	if hours >= 24 {
		return strings.ReplaceAll(fmt.Sprintf("%.1f hari", hours/24), ".", ",")
	}
	if hours >= 1 {
		return strings.ReplaceAll(fmt.Sprintf("%.1f jam", hours), ".", ",")
	}
	return strings.ReplaceAll(fmt.Sprintf("%.0f menit", hours*60), ".", ",")
}
