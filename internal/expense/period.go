package expense

import "time"

type Period string

const (
	PeriodPastWeek        Period = "past_week"
	PeriodPastMonth       Period = "past_month"
	PeriodLastThreeMonths Period = "last_3_months"
	PeriodCustom          Period = "custom"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

func (p *fieldParser) dateRange(period, from, to string) *DateRange {
	if Period(period) != PeriodCustom && (from != "" || to != "") {
		p.errs.Add("period", "from and to require period=custom")
		return nil
	}

	switch Period(period) {
	case "":
		return nil
	case PeriodPastWeek:
		return &DateRange{From: p.today.AddDate(0, 0, -6), To: p.today}
	case PeriodPastMonth:
		return p.trailingMonths(1)
	case PeriodLastThreeMonths:
		return p.trailingMonths(3)
	case PeriodCustom:
		return p.customRange(from, to)
	default:
		p.errs.Add("period", "must be one of past_week, past_month, last_3_months, custom")
		return nil
	}
}

func (p *fieldParser) trailingMonths(months int) *DateRange {
	return &DateRange{From: subtractMonths(p.today, months).AddDate(0, 0, 1), To: p.today}
}

func (p *fieldParser) customRange(from, to string) *DateRange {
	if from == "" || to == "" {
		p.errs.Add("period", "custom period requires both from and to")
		return nil
	}

	errorsBefore := len(p.errs)
	dateRange := &DateRange{From: p.date("from", from), To: p.date("to", to)}

	if len(p.errs) == errorsBefore && dateRange.To.Before(dateRange.From) {
		p.errs.Add("to", "must not be before from")
	}

	return dateRange
}

func subtractMonths(date time.Time, months int) time.Time {
	firstOfTarget := time.Date(date.Year(), date.Month()-time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfTarget := firstOfTarget.AddDate(0, 1, -1).Day()

	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), min(date.Day(), lastDayOfTarget), 0, 0, 0, 0, time.UTC)
}
