package evidence

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizeText is the one text normalization used before hashing a snapshot, so
// the same document always yields the same snapshot id: NFKC, line endings folded,
// runs of blank space collapsed, no control characters.
func NormalizeText(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var b strings.Builder
	b.Grow(len(s))
	space := false
	nl := 0
	for _, r := range s {
		switch {
		case r == '\n':
			nl++
			space = false
			if nl <= 2 {
				b.WriteRune('\n')
			}
		case unicode.IsSpace(r):
			if !space && nl == 0 {
				b.WriteRune(' ')
			}
			space = true
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
			space, nl = false, 0
		}
	}
	return strings.TrimSpace(b.String())
}

// Canon is the canonical form of an entity or property name for keying: NFKC,
// lower case, accents kept (they distinguish names), punctuation reduced to single
// spaces, leading articles dropped.
func Canon(s string) string {
	s = strings.ToLower(norm.NFKC.String(s))
	var b strings.Builder
	space := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	out := strings.TrimSpace(b.String())
	for _, art := range []string{"the ", "a ", "an "} {
		out = strings.TrimPrefix(out, art)
	}
	return out
}

var (
	isoRE  = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	yearRE = regexp.MustCompile(`\b(1[0-9]{3}|20[0-9]{2})\b`)
	monRE  = regexp.MustCompile(`(?i)\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\.?\s+(\d{1,2})(?:st|nd|rd|th)?,?\s+(\d{4})\b`)
	dmyRE  = regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)?\s+(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\.?,?\s+(\d{4})\b`)
)

var months = map[string]int{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}

func monthNum(s string) int { return months[strings.ToLower(s)[:3]] }

func isoDate(y, m, d int) string {
	return strconv.Itoa(y) + "-" + pad2(m) + "-" + pad2(d)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// Dates finds calendar dates written as 2026-10-17, "October 17, 2026" or "17 October
// 2026" and returns them as ISO dates.
func Dates(s string) []string {
	var out []string
	// Cheap prefilter: a date needs a digit, and the three patterns need either a
	// "-" or a month name. Most passages have neither, and the regexps are slow.
	if !hasDigit(s) {
		return nil
	}
	dash := strings.IndexByte(s, '-') >= 0
	low := strings.ToLower(s)
	month := false
	for m := range months {
		if strings.Contains(low, m) {
			month = true
			break
		}
	}
	if !dash && !month {
		return nil
	}
	for _, m := range isoRE.FindAllStringSubmatch(s, -1) {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if mo >= 1 && mo <= 12 && d >= 1 && d <= 31 {
			out = append(out, isoDate(y, mo, d))
		}
	}
	for _, m := range monRE.FindAllStringSubmatch(s, -1) {
		y, _ := strconv.Atoi(m[3])
		d, _ := strconv.Atoi(m[2])
		if d >= 1 && d <= 31 {
			out = append(out, isoDate(y, monthNum(m[1]), d))
		}
	}
	for _, m := range dmyRE.FindAllStringSubmatch(s, -1) {
		y, _ := strconv.Atoi(m[3])
		d, _ := strconv.Atoi(m[1])
		if d >= 1 && d <= 31 {
			out = append(out, isoDate(y, monthNum(m[2]), d))
		}
	}
	return out
}

// Numbers returns the numeric values in s as canonical strings (thousands
// separators removed, trailing zeros trimmed). It is a hand-written scanner: this
// runs once per indexed passage, and the regexp it replaced was a quarter of ingest time.
func Numbers(s string) []string {
	var out []string
	n := len(s)
	for i := 0; i < n; {
		c := s[i]
		start := i
		if (c == '-' || c == '+') && i+1 < n && s[i+1] >= '0' && s[i+1] <= '9' {
			i++
		} else if c < '0' || c > '9' {
			i++
			continue
		}
		var b []byte
		if s[start] == '-' {
			b = append(b, '-')
		}
		for i < n && s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
			i++
			// a thousands separator: ",ddd" followed by a non-digit or another ",ddd"
			for i+3 < n+0 && s[i] == ',' && isDigit3(s, i+1) && (i+4 >= n || s[i+4] < '0' || s[i+4] > '9') {
				b = append(b, s[i+1], s[i+2], s[i+3])
				i += 4
			}
		}
		if i+1 < n && s[i] == '.' && s[i+1] >= '0' && s[i+1] <= '9' {
			b = append(b, '.')
			i++
			for i < n && s[i] >= '0' && s[i] <= '9' {
				b = append(b, s[i])
				i++
			}
		}
		if f, err := strconv.ParseFloat(string(b), 64); err == nil {
			out = append(out, strconv.FormatFloat(f, 'f', -1, 64))
		}
	}
	return out
}

func isDigit3(s string, i int) bool {
	return i+2 < len(s) && s[i] >= '0' && s[i] <= '9' && s[i+1] >= '0' && s[i+1] <= '9' && s[i+2] >= '0' && s[i+2] <= '9'
}

// Years returns four-digit years (1000-2099) mentioned in s.
func Years(s string) []string { return yearRE.FindAllString(s, -1) }

func hasDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}
