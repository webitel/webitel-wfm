package timeutils

import "time"

type boundaryType string

const (
	IncludeStartExcludeEnd boundaryType = "[)"
	ExcludeStartIncludeEnd boundaryType = "(]"
	ExcludeAll             boundaryType = "()"
	IncludeAll             boundaryType = "[]"
)

type Period struct {
	startDate    time.Time
	endDate      time.Time
	boundaryType boundaryType
}

func NewPeriod(startDate, endDate time.Time, boundaryType boundaryType) Period {
	if startDate.After(endDate) {
		startDate, endDate = endDate, startDate
	}

	return Period{
		startDate:    startDate,
		endDate:      endDate,
		boundaryType: boundaryType,
	}
}

func (p Period) GenerateSeries(years int, months int, days int) []time.Time {
	var series []time.Time
	for d := p.startDate; !d.After(p.endDate); d = d.AddDate(years, months, days) {
		series = append(series, d)
	}

	return series
}

func (p Period) Contains(other Period) bool {
	return p.containsInterval(other)
}

func (p Period) dateInterval() time.Duration {
	return p.endDate.Sub(p.startDate)
}

func (p Period) containsDatePoint(datePoint time.Time, boundaryType boundaryType) bool {
	switch boundaryType {
	case ExcludeAll:
		return datePoint.After(p.startDate) && datePoint.Before(p.endDate)
	case IncludeAll:
		return (datePoint.Equal(p.startDate) || datePoint.After(p.startDate)) && (datePoint.Equal(p.endDate) || datePoint.Before(p.endDate))
	case ExcludeStartIncludeEnd:
		return datePoint.After(p.startDate) && (datePoint.Equal(p.endDate) || datePoint.Before(p.endDate))
	case IncludeStartExcludeEnd:
		fallthrough
	default:
		return (datePoint.Equal(p.startDate) || datePoint.After(p.startDate)) && datePoint.Before(p.endDate)
	}
}

func (p Period) containsInterval(other Period) bool {
	if p.startDate.Before(other.startDate) && p.endDate.After(other.endDate) {
		return true
	}

	if p.startDate.Equal(other.startDate) && p.endDate.Equal(other.endDate) {
		return p.boundaryType == other.boundaryType || p.boundaryType == IncludeAll
	}

	if p.startDate.Equal(other.startDate) {
		return (p.boundaryType[0] == other.boundaryType[0] || "[" == p.boundaryType[0:1]) && p.containsDatePoint(
			p.startDate.Add(other.dateInterval()), p.boundaryType,
		)
	}

	if p.endDate.Equal(other.endDate) {
		return (p.boundaryType[1] == other.boundaryType[1] || "]" == p.boundaryType[1:2]) && p.containsDatePoint(
			p.endDate.Add(-other.dateInterval()), p.boundaryType,
		)
	}

	return false
}
