package connector

import (
	"crypto/sha256"
	"golang.org/x/text/unicode/norm"
	"html"
	"regexp"
	"strings"
	"unicode"
)

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// CleanRecord normalizes source text and applies the pipeline's bounded input
// policy. The bool is false when a record should be skipped.
func CleanRecord(record RawRecord) (RawRecord, bool) {
	record.Text = norm.NFC.String(html.UnescapeString(htmlTag.ReplaceAllString(record.Text, " ")))
	record.Text = strings.Join(strings.FieldsFunc(record.Text, unicode.IsSpace), " ")
	if len(record.Text) < 32 || len(record.Text) > 16384 {
		return RawRecord{}, false
	}
	return record, true
}

func TextHash(record RawRecord) [32]byte { return sha256.Sum256([]byte(record.Text)) }
