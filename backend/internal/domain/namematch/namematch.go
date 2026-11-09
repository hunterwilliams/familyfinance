package namematch

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
)

type Result struct {
	MemberID *uuid.UUID
	IsJoint  bool
}

var titles = regexp.MustCompile(`(?i)\b(mr|mrs|ms|miss|khun)\.?(\b|$)|นางสาว|นาย|นาง|คุณ`)
var noise = regexp.MustCompile(`(?i)\b(scb|ktc|kbank|bbl|ttb|gsb|bay|uob|visa|mastercard|debit|credit|card|account|savings|joint|bank|thai)\b|ธนาคารไทยพาณิชย์|ไทยพาณิชย์|ธนาคาร|บัญชี|บัตร`)

func MatchAll(texts []string, members []models.Member) Result {
	seen := map[uuid.UUID]struct{}{}
	joint := false
	for _, t := range texts {
		r := Match(t, members)
		if r.IsJoint {
			joint = true
		}
		if r.MemberID != nil {
			seen[*r.MemberID] = struct{}{}
		}
	}
	if joint || len(seen) >= 2 {
		return Result{IsJoint: true}
	}
	for id := range seen {
		id := id
		return Result{MemberID: &id}
	}
	return Result{}
}

func Match(holder string, members []models.Member) Result {
	holderN := normalize(holder)
	holderC := compact(holderN)
	if holderC == "" || len(members) == 0 {
		return Result{}
	}

	type hit struct {
		id      uuid.UUID
		aliases []string
	}
	var hits []hit
	for _, m := range members {
		aliases := aliasesOf(m)
		if matchesMember(holderN, holderC, aliases) {
			hits = append(hits, hit{id: m.ID, aliases: aliases})
		}
	}
	if len(hits) >= 2 {
		return Result{IsJoint: true}
	}
	if len(hits) == 1 {
		left := leftoverCompact(holderN, hits[0].aliases)
		if utf8.RuneCountInString(left) >= 4 {
			return Result{IsJoint: true}
		}
		id := hits[0].id
		return Result{MemberID: &id}
	}
	return Result{}
}

func aliasesOf(m models.Member) []string {
	out := make([]string, 0, 2)
	for _, s := range []string{m.Name, m.NameTh} {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func matchesMember(holderN, holderC string, aliases []string) bool {
	ht := tokens(holderN)
	for _, a := range aliases {
		an := normalize(a)
		ac := compact(a)
		if utf8.RuneCountInString(ac) < 3 {
			continue
		}
		at := tokens(an)
		if strings.Contains(holderC, ac) {
			return true
		}
		if tokensContained(at, ht) {
			return true
		}
		// Slip shows a shorter form of the member name (first name only).
		if significant(holderC) && (strings.Contains(ac, holderC) || tokensContained(ht, at)) {
			return true
		}
		if tokensClose(ht, at) {
			return true
		}
	}
	return false
}

func significant(s string) bool {
	return utf8.RuneCountInString(s) >= 4
}

func tokensClose(need, have []string) bool {
	if len(need) == 0 || len(have) == 0 {
		return false
	}
	bag := append([]string(nil), have...)
	for _, n := range need {
		i := indexCloseToken(bag, n)
		if i < 0 {
			return false
		}
		bag = append(bag[:i], bag[i+1:]...)
	}
	return true
}

func indexCloseToken(have []string, need string) int {
	for i, h := range have {
		if h == need {
			return i
		}
		if significant(need) && significant(h) && damerau(need, h) <= 1 {
			return i
		}
	}
	return -1
}

func damerau(a, b string) int {
	ar, br := []rune(a), []rune(b)
	al, bl := len(ar), len(br)
	if al == 0 {
		return bl
	}
	if bl == 0 {
		return al
	}
	const maxDist = 2
	if abs(al-bl) > maxDist {
		return maxDist + 1
	}
	prev := make([]int, bl+1)
	cur := make([]int, bl+1)
	prev2 := make([]int, bl+1)
	for j := 0; j <= bl; j++ {
		prev[j] = j
	}
	for i := 1; i <= al; i++ {
		cur[0] = i
		for j := 1; j <= bl; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				if t := prev2[j-2] + 1; t < cur[j] {
					cur[j] = t
				}
			}
		}
		copy(prev2, prev)
		copy(prev, cur)
	}
	return prev[bl]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func min(xs ...int) int {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func leftoverCompact(holderN string, aliases []string) string {
	sorted := append([]string(nil), aliases...)
	sort.Slice(sorted, func(i, j int) bool {
		return utf8.RuneCountInString(compact(sorted[i])) > utf8.RuneCountInString(compact(sorted[j]))
	})

	toks := tokens(holderN)
	for _, a := range sorted {
		for _, t := range tokens(normalize(a)) {
			toks = removeCloseToken(toks, t)
		}
	}
	tokenLeft := compact(noise.ReplaceAllString(strings.Join(toks, " "), " "))

	s := holderN
	for _, a := range sorted {
		s = replaceCI(s, a, " ")
		s = replaceCI(s, compact(a), " ")
	}
	s = titles.ReplaceAllString(s, " ")
	s = noise.ReplaceAllString(s, " ")
	substrLeft := compact(s)

	if utf8.RuneCountInString(tokenLeft) <= utf8.RuneCountInString(substrLeft) {
		return tokenLeft
	}
	return substrLeft
}

func removeCloseToken(toks []string, want string) []string {
	i := indexCloseToken(toks, want)
	if i < 0 {
		return toks
	}
	return append(toks[:i], toks[i+1:]...)
}

func normalize(s string) string {
	s = titles.ReplaceAllString(s, " ")
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '&' || r == '|' || r == '+' {
			return ' '
		}
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func compact(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func tokens(s string) []string {
	return strings.Fields(s)
}

func tokensContained(need, have []string) bool {
	if len(need) == 0 {
		return false
	}
	bag := map[string]int{}
	for _, t := range have {
		bag[t]++
	}
	for _, t := range need {
		if bag[t] == 0 {
			return false
		}
		bag[t]--
	}
	return true
}

func replaceCI(s, old, neu string) string {
	if old == "" {
		return s
	}
	lower := strings.ToLower(s)
	o := strings.ToLower(old)
	i := strings.Index(lower, o)
	if i < 0 {
		return s
	}
	return s[:i] + neu + s[i+len(old):]
}
