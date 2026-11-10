package nanny_test

import (
	"testing"
	"time"

	"github.com/hunterwilliams/familyfinance/internal/domain/nanny"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoursBetween(t *testing.T) {
	start := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.0, nanny.HoursBetween(start, end))
}

func TestCostSatang(t *testing.T) {
	// 5 hours at 200 THB/hour = 1000 THB = 100000 satang
	assert.Equal(t, int64(100000), nanny.CostSatang(5, 20000))
}

func TestExtraEarlyWindow(t *testing.T) {
	start, end, hours, ok := nanny.ExtraEarlyWindow("09:00", "07:00")
	require.True(t, ok)
	assert.Equal(t, "07:00", start)
	assert.Equal(t, "09:00", end)
	assert.Equal(t, 2.0, hours)
}

func bangkok(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Bangkok")
	require.NoError(t, err)
	return loc
}

func TestParseHoursMessage_NannyComeEarlyTomorrow(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, loc)
	cgs := []nanny.Ref{{
		Name:         "Nickname",
		DefaultStart: "09:00",
		DefaultEnd:   "18:00",
	}}

	got, ok := nanny.ParseHoursMessage("nanny come early tomorrow at 7am", cgs, nil, now)
	require.True(t, ok)
	assert.Equal(t, "Nickname", got.CaregiverName)
	assert.Equal(t, "2026-11-11", got.Date)
	assert.Equal(t, "07:00", got.StartTime)
	assert.Equal(t, "09:00", got.EndTime)
	assert.Equal(t, 2.0, got.Hours)
}

func TestParseHoursMessage_NicknameComeEarlyTmrw(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, loc)
	cgs := []nanny.Ref{{
		Name:         "Nickname",
		DefaultStart: "09:00",
		DefaultEnd:   "18:00",
	}}

	got, ok := nanny.ParseHoursMessage("Nickname come early tmrw at 7am", cgs, nil, now)
	require.True(t, ok)
	assert.Equal(t, "Nickname", got.CaregiverName)
	assert.Equal(t, "2026-11-11", got.Date)
	assert.Equal(t, "07:00", got.StartTime)
	assert.Equal(t, "09:00", got.EndTime)
	assert.Equal(t, 2.0, got.Hours)
}

func nicknameRef() nanny.Ref {
	return nanny.Ref{
		Name:         "Nickname",
		DefaultStart: "09:00",
		DefaultEnd:   "18:00",
		DefaultDays:  []int{1, 2, 3, 4, 5, 6}, // Mon–Sat
	}
}

func TestWorksOn_MonSatNotSun(t *testing.T) {
	days := []int{1, 2, 3, 4, 5, 6}
	assert.True(t, nanny.WorksOn(days, time.Monday))
	assert.True(t, nanny.WorksOn(days, time.Saturday))
	assert.False(t, nanny.WorksOn(days, time.Sunday))
}

func TestFormatWeekdays_MonSat(t *testing.T) {
	assert.Equal(t, "Mon–Sat", nanny.FormatWeekdays([]int{1, 2, 3, 4, 5, 6}))
}

func TestParseHoursMessage_NannyHolidayTomorrow(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, loc) // Tuesday; tomorrow is a workday
	got, ok := nanny.ParseHoursMessage("nanny holiday tomorrow", []nanny.Ref{nicknameRef()}, nil, now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindHoliday, got.Kind)
	assert.Equal(t, "Nickname", got.CaregiverName)
	assert.Equal(t, "2026-11-11", got.Date)
	assert.Equal(t, "09:00", got.StartTime)
	assert.Equal(t, "18:00", got.EndTime)
	assert.Equal(t, -9.0, got.Hours)
}

func TestParseHoursMessage_NicknameHolidayTmrw(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, loc)
	got, ok := nanny.ParseHoursMessage("Nickname holiday tmrw", []nanny.Ref{nicknameRef()}, nil, now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindHoliday, got.Kind)
	assert.Equal(t, "Nickname", got.CaregiverName)
	assert.Equal(t, "2026-11-11", got.Date)
	assert.Equal(t, -9.0, got.Hours)
}

func TestParseHoursMessage_HolidayOnSundayIsNotWork(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 11, 10, 22, 0, 0, 0, loc) // Tuesday
	got, ok := nanny.ParseHoursMessage("nanny holiday sunday", []nanny.Ref{nicknameRef()}, nil, now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindHoliday, got.Kind)
	assert.Equal(t, "2026-11-15", got.Date)
	assert.Equal(t, 0.0, got.Hours)
}

func TestNetHours_ExtraCancelsHoliday(t *testing.T) {
	assert.Equal(t, -7.0, nanny.NetHours([]float64{2.0, -9.0}))
}

func TestDefaultThaiHolidays_IncludesSongkranAndFathersDay(t *testing.T) {
	holidays := nanny.DefaultThaiHolidays()
	_, songkran := nanny.LookupPaid(holidays, time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC))
	_, fathers := nanny.LookupPaid(holidays, time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC))
	require.True(t, songkran)
	require.True(t, fathers)
}

func TestParseHoursMessage_PaidHolidayDoesNotOweHours(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 4, 12, 22, 0, 0, 0, loc) // Sunday; tomorrow is Songkran
	got, ok := nanny.ParseHoursMessage("nanny holiday tomorrow", []nanny.Ref{nicknameRef()}, nanny.DefaultThaiHolidays(), now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindPaidHoliday, got.Kind)
	assert.Equal(t, "2026-04-13", got.Date)
	assert.Equal(t, 0.0, got.Hours)
	assert.Contains(t, got.Note, "Songkran")
}

func TestParseHoursMessage_WorkingPaidHolidayIsExtraPay(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 4, 12, 22, 0, 0, 0, loc)
	got, ok := nanny.ParseHoursMessage("nanny working tomorrow", []nanny.Ref{nicknameRef()}, nanny.DefaultThaiHolidays(), now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindWorkedHoliday, got.Kind)
	assert.Equal(t, "Nickname", got.CaregiverName)
	assert.Equal(t, "2026-04-13", got.Date)
	assert.Equal(t, "09:00", got.StartTime)
	assert.Equal(t, "18:00", got.EndTime)
	assert.Equal(t, 9.0, got.Hours)
}

func TestParseHoursMessage_NicknameWorkingPaidHoliday(t *testing.T) {
	loc := bangkok(t)
	now := time.Date(2026, 12, 4, 20, 0, 0, 0, loc) // Friday; Sat 5 Dec is Father's Day and a usual workday
	got, ok := nanny.ParseHoursMessage("Nickname working tmrw", []nanny.Ref{nicknameRef()}, nanny.DefaultThaiHolidays(), now)
	require.True(t, ok)
	assert.Equal(t, nanny.KindWorkedHoliday, got.Kind)
	assert.Equal(t, "2026-12-05", got.Date)
	assert.Equal(t, 9.0, got.Hours)
}
