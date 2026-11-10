package nanny

import "time"

type CalendarHoliday struct {
	Date string
	Name string
}

func LookupPaid(holidays []CalendarHoliday, date time.Time) (CalendarHoliday, bool) {
	key := date.Format("2006-01-02")
	for _, h := range holidays {
		if h.Date == key {
			return h, true
		}
	}
	return CalendarHoliday{}, false
}

// DefaultThaiHolidays is the 2026 Thai government / BOT public-holiday calendar,
// plus the original dates when a substitute weekday is also given (nanny may work Saturday).
func DefaultThaiHolidays() []CalendarHoliday {
	return []CalendarHoliday{
		{Date: "2026-01-01", Name: "New Year's Day"},
		{Date: "2026-01-02", Name: "Additional special holiday"},
		{Date: "2026-03-03", Name: "Makha Bucha Day"},
		{Date: "2026-04-06", Name: "Chakri Memorial Day"},
		{Date: "2026-04-13", Name: "Songkran Festival"},
		{Date: "2026-04-14", Name: "Songkran Festival"},
		{Date: "2026-04-15", Name: "Songkran Festival"},
		{Date: "2026-05-01", Name: "National Labor Day"},
		{Date: "2026-05-04", Name: "Coronation Day"},
		{Date: "2026-05-31", Name: "Visakha Bucha Day"},
		{Date: "2026-06-01", Name: "Substitution for Visakha Bucha Day"},
		{Date: "2026-06-03", Name: "H.M. Queen Suthida's Birthday"},
		{Date: "2026-07-28", Name: "H.M. the King's Birthday"},
		{Date: "2026-07-29", Name: "Asalha Bucha Day"},
		{Date: "2026-08-12", Name: "Queen Mother's Birthday / Mother's Day"},
		{Date: "2026-10-13", Name: "King Bhumibol Memorial Day"},
		{Date: "2026-10-23", Name: "Chulalongkorn Memorial Day"},
		{Date: "2026-12-05", Name: "King Bhumibol's Birthday / Father's Day"},
		{Date: "2026-12-07", Name: "Substitution for Father's Day"},
		{Date: "2026-12-10", Name: "Constitution Day"},
		{Date: "2026-12-16", Name: "Royal Cremation holiday"},
		{Date: "2026-12-31", Name: "New Year's Eve"},
	}
}
