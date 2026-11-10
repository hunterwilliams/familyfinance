package nanny

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

func HoursBetween(start, end time.Time) float64 {
	d := end.Sub(start)
	if d <= 0 {
		return 0
	}
	return d.Hours()
}

func CostSatang(hours float64, hourlyRateSatang int64) int64 {
	return int64(hours * float64(hourlyRateSatang))
}

func ParseClock(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "Z")
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errInvalidClock
}

var errInvalidClock = errString("invalid clock")

type errString string

func (e errString) Error() string { return string(e) }

func ClockHHMM(s string) string {
	t, err := ParseClock(s)
	if err != nil {
		return strings.TrimSpace(s)
	}
	return t.Format("15:04")
}

func HoursBetweenClocks(start, end string) float64 {
	s, err1 := ParseClock(start)
	e, err2 := ParseClock(end)
	if err1 != nil || err2 != nil {
		return 0
	}
	return HoursBetween(s, e)
}

// ExtraEarlyWindow is the extra paid slice when someone arrives before the usual start.
func ExtraEarlyWindow(defaultStart, arrival string) (start, end string, hours float64, ok bool) {
	arrival = ClockHHMM(arrival)
	defaultStart = ClockHHMM(defaultStart)
	h := HoursBetweenClocks(arrival, defaultStart)
	if h <= 0 {
		return "", "", 0, false
	}
	return arrival, defaultStart, h, true
}

func ApplyShiftDefaults(start, end string, hours float64, ref Ref) (string, string, float64, bool) {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start != "" && end != "" {
		if hours <= 0 {
			hours = HoursBetweenClocks(start, end)
		}
		return ClockHHMM(start), ClockHHMM(end), hours, hours > 0
	}
	if start != "" && end == "" && ref.DefaultStart != "" {
		return ExtraEarlyWindow(ref.DefaultStart, start)
	}
	if end != "" && start == "" && ref.DefaultEnd != "" {
		return ExtraLateWindow(ref.DefaultEnd, end)
	}
	return start, end, hours, hours > 0
}

// ExtraLateWindow is the extra paid slice when someone stays after the usual end.
func ExtraLateWindow(defaultEnd, leave string) (start, end string, hours float64, ok bool) {
	leave = ClockHHMM(leave)
	defaultEnd = ClockHHMM(defaultEnd)
	h := HoursBetweenClocks(defaultEnd, leave)
	if h <= 0 {
		return "", "", 0, false
	}
	return defaultEnd, leave, h, true
}

const (
	KindExtra         = "extra"
	KindHoliday       = "holiday"
	KindPaidHoliday   = "paid_holiday"
	KindWorkedHoliday = "worked_holiday"
)

var DefaultWorkdays = []int{1, 2, 3, 4, 5, 6} // Mon–Sat

type Ref struct {
	Name         string
	DefaultStart string
	DefaultEnd   string
	DefaultDays  []int
}

type HoursIntent struct {
	CaregiverName string
	Date          string
	StartTime     string
	EndTime       string
	Hours         float64
	Note          string
	Kind          string
}

var (
	clockRe   = regexp.MustCompile(`(?i)\b(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(a\.?m\.?|p\.?m\.?)?\b`)
	tmrwRe    = regexp.MustCompile(`(?i)\b(tmrw|tmr)\b`)
	weekdayRe = regexp.MustCompile(`(?i)\b(sunday|monday|tuesday|wednesday|thursday|friday|saturday|sun|mon|tue|tues|wed|thu|thur|thurs|fri|sat)\b`)
	offWordRe  = regexp.MustCompile(`(?i)\boff\b`)
	workWordRe = regexp.MustCompile(`(?i)\b(working|works|worked|coming in)\b|\bwork(ing)? (tomorrow|today|on)\b`)
)

func NormalizeWeekdays(days []int) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, d := range days {
		if d < 0 || d > 6 {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Ints(out)
	if len(out) == 0 {
		return append([]int(nil), DefaultWorkdays...)
	}
	return out
}

func WorksOn(days []int, wd time.Weekday) bool {
	for _, d := range NormalizeWeekdays(days) {
		if d == int(wd) {
			return true
		}
	}
	return false
}

func FormatWeekdays(days []int) string {
	days = NormalizeWeekdays(days)
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	if len(days) == 0 {
		return ""
	}
	consecutive := true
	for i := 1; i < len(days); i++ {
		if days[i] != days[i-1]+1 {
			consecutive = false
			break
		}
	}
	if consecutive && len(days) > 1 {
		return names[days[0]] + "–" + names[days[len(days)-1]]
	}
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = names[d]
	}
	return strings.Join(parts, ", ")
}

func UsualShiftHours(start, end string) float64 {
	return HoursBetweenClocks(start, end)
}

func NetHours(hours []float64) float64 {
	var s float64
	for _, h := range hours {
		s += h
	}
	return s
}

func HolidayFor(ref Ref, date time.Time, paid []CalendarHoliday) HoursIntent {
	out := HoursIntent{
		Kind:          KindHoliday,
		CaregiverName: ref.Name,
		Date:          date.Format("2006-01-02"),
		Note:          "holiday",
	}
	if ph, ok := LookupPaid(paid, date); ok {
		out.Kind = KindPaidHoliday
		out.Note = ph.Name
		return out
	}
	if !WorksOn(ref.DefaultDays, date.Weekday()) {
		out.Note = "not a usual work day"
		return out
	}
	h := UsualShiftHours(ref.DefaultStart, ref.DefaultEnd)
	if h <= 0 {
		return out
	}
	out.StartTime = ClockHHMM(ref.DefaultStart)
	out.EndTime = ClockHHMM(ref.DefaultEnd)
	out.Hours = -h
	return out
}

func WorkedHolidayFor(ref Ref, date time.Time, paid []CalendarHoliday) HoursIntent {
	out := HoursIntent{
		Kind:          KindWorkedHoliday,
		CaregiverName: ref.Name,
		Date:          date.Format("2006-01-02"),
		Note:          "worked paid holiday",
	}
	ph, ok := LookupPaid(paid, date)
	if !ok {
		out.Note = "not a paid holiday"
		return out
	}
	h := UsualShiftHours(ref.DefaultStart, ref.DefaultEnd)
	if h <= 0 {
		return out
	}
	out.StartTime = ClockHHMM(ref.DefaultStart)
	out.EndTime = ClockHHMM(ref.DefaultEnd)
	out.Hours = h
	out.Note = "worked paid holiday: " + ph.Name
	return out
}

func ParseHoursMessage(text string, caregivers []Ref, paid []CalendarHoliday, now time.Time) (HoursIntent, bool) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return HoursIntent{}, false
	}
	lower := strings.ToLower(raw)
	lower = tmrwRe.ReplaceAllString(lower, "tomorrow")

	early := strings.Contains(lower, "early") || strings.Contains(lower, "earlier")
	late := strings.Contains(lower, "late") || strings.Contains(lower, "overtime")
	holiday := isHolidayMessage(lower)
	worked := isWorkedHolidayMessage(lower)

	cg, ok := resolveCaregiver(lower, caregivers, early || late || holiday || worked)
	if !ok {
		return HoursIntent{}, false
	}

	date := parseWorkDate(lower, now)
	if worked {
		return WorkedHolidayFor(cg, date, paid), true
	}
	if holiday {
		return HolidayFor(cg, date, paid), true
	}

	clocks := parseClocks(lower, early, late)
	if len(clocks) == 0 {
		return HoursIntent{}, false
	}
	if !early && !late && len(clocks) < 2 && !strings.Contains(lower, "extra") {
		return HoursIntent{}, false
	}

	out := HoursIntent{
		Kind:          KindExtra,
		CaregiverName: cg.Name,
		Date:          date.Format("2006-01-02"),
	}

	switch {
	case len(clocks) >= 2:
		out.StartTime = clocks[0]
		out.EndTime = clocks[1]
		out.Hours = HoursBetweenClocks(out.StartTime, out.EndTime)
		out.Note = "extra hours"
	case early:
		start, end, hours, ok := ExtraEarlyWindow(cg.DefaultStart, clocks[0])
		if !ok {
			return HoursIntent{}, false
		}
		out.StartTime, out.EndTime, out.Hours = start, end, hours
		out.Note = "came early"
	case late:
		start, end, hours, ok := ExtraLateWindow(cg.DefaultEnd, clocks[0])
		if !ok {
			return HoursIntent{}, false
		}
		out.StartTime, out.EndTime, out.Hours = start, end, hours
		out.Note = "stayed late"
	default:
		return HoursIntent{}, false
	}
	if out.Hours <= 0 {
		return HoursIntent{}, false
	}
	return out, true
}

func isWorkedHolidayMessage(lower string) bool {
	if !workWordRe.MatchString(lower) {
		return false
	}
	if strings.Contains(lower, "early") || strings.Contains(lower, "late") {
		return false
	}
	return true
}

func isHolidayMessage(lower string) bool {
	if strings.Contains(lower, "holiday") || strings.Contains(lower, "day off") {
		return true
	}
	if strings.Contains(lower, "taking off") || strings.Contains(lower, "take off") {
		return true
	}
	if offWordRe.MatchString(lower) && (strings.Contains(lower, "tomorrow") || strings.Contains(lower, "today") || weekdayRe.MatchString(lower)) {
		return true
	}
	return false
}

func parseWorkDate(lower string, now time.Time) time.Time {
	if strings.Contains(lower, "tomorrow") {
		return now.AddDate(0, 0, 1)
	}
	if m := weekdayRe.FindString(lower); m != "" {
		if wd, ok := parseWeekday(m); ok {
			return nextWeekday(now, wd)
		}
	}
	return now
}

func parseWeekday(s string) (time.Weekday, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sun", "sunday":
		return time.Sunday, true
	case "mon", "monday":
		return time.Monday, true
	case "tue", "tues", "tuesday":
		return time.Tuesday, true
	case "wed", "wednesday":
		return time.Wednesday, true
	case "thu", "thur", "thurs", "thursday":
		return time.Thursday, true
	case "fri", "friday":
		return time.Friday, true
	case "sat", "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}

func nextWeekday(now time.Time, want time.Weekday) time.Time {
	delta := (int(want) - int(now.Weekday()) + 7) % 7
	return now.AddDate(0, 0, delta)
}

func resolveCaregiver(lower string, caregivers []Ref, genericOK bool) (Ref, bool) {
	bestIdx := -1
	bestLen := 0
	for i, c := range caregivers {
		name := strings.ToLower(strings.TrimSpace(c.Name))
		if name == "" {
			continue
		}
		if strings.Contains(lower, name) && len(name) > bestLen {
			bestIdx = i
			bestLen = len(name)
		}
	}
	if bestIdx >= 0 {
		return caregivers[bestIdx], true
	}
	generic := strings.Contains(lower, "nanny") || strings.Contains(lower, "babysitter") || strings.Contains(lower, "caregiver")
	if generic && len(caregivers) >= 1 {
		return caregivers[0], true
	}
	if genericOK && len(caregivers) == 1 {
		return caregivers[0], true
	}
	return Ref{}, false
}

func parseClocks(lower string, early, late bool) []string {
	matches := clockRe.FindAllStringSubmatch(lower, -1)
	var out []string
	for _, m := range matches {
		hour := atoi(m[1])
		min := 0
		if m[2] != "" {
			min = atoi(m[2])
		}
		if hour > 23 || min > 59 {
			continue
		}
		ampm := strings.ToLower(strings.ReplaceAll(m[3], ".", ""))
		if ampm == "am" {
			if hour == 12 {
				hour = 0
			}
		} else if ampm == "pm" {
			if hour < 12 {
				hour += 12
			}
		} else if hour <= 12 {
			if late && hour < 12 {
				hour += 12
			}
		}
		out = append(out, sprintfClock(hour, min))
	}
	return out
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

func sprintfClock(hour, min int) string {
	return time.Date(0, 1, 1, hour, min, 0, 0, time.UTC).Format("15:04")
}
