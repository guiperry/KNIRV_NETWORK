package locator

import (
	"regexp"
	"strings"

	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Owner decision Q83: the public view of an ErrorNode never carries a
// checkout's location or its source lines. The full record is kept in the
// store; the public listing and the commit reply are cut down by
// PublicErrorNode. The backend's internal-token listing gets the full record
// (see internal/network getAllErrors).

// publicDropKeys are context keys removed from the public view at any depth.
// code_context holds source lines and project_path the absolute checkout
// location, as errorscan builds them.
var publicDropKeys = map[string]bool{
	"code_context": true,
	"project_path": true,
}

// pathStop is the set of characters that end a path token.
const pathStop = `\s"'` + "`" + `<>|;{}()\[\],=`

// pathTokenRe finds filesystem path tokens: file:// URIs, Windows UNC paths,
// rooted Windows paths (\Users\me), Windows profile paths with no drive
// (Users\me\x), drive paths, ~ and ~user home paths, and slash-rooted paths.
// A path that starts glued to a word is only accepted under a known root (see
// attachOK).
var pathTokenRe = regexp.MustCompile(
	`file://[^` + pathStop + `]*` +
		`|\\\\[^` + pathStop + `]+` +
		`|\\[^` + pathStop + `\\]+\\[^` + pathStop + `]*` +
		`|(?:Users|Windows|ProgramData|Temp)\\[^` + pathStop + `\\]+\\[^` + pathStop + `]*` +
		`|[A-Za-z]:[\\/][^` + pathStop + `]*` +
		`|~[A-Za-z0-9_.-]*/[^` + pathStop + `]*` +
		`|/[^` + pathStop + `]+`)

// rootPathRe recognises a slash-rooted path under a conventional root. It is
// one of the ways a slash attached to a word ("in/home/u/x") counts as a path.
// Only system and home roots are listed: repo directories such as app or data
// are common in relative paths ("internal/app/app.go") and must be kept.
var rootPathRe = regexp.MustCompile(`^/(?:home|Users|root|etc|var|usr|opt|tmp|srv|mnt|media|private|Volumes)(?:/|$)`)

// winRootRe recognises a backslash-rooted path under a conventional Windows
// root. It is the only way a backslash attached to a word counts as a path.
var winRootRe = regexp.MustCompile(`^\\(?:Users|Windows|ProgramData|Program|Temp|Documents)\b`)

// pathWordRe matches one space-free word of a path, used to pick up the tail of
// a path that contains spaces ("C:\Program Files\app\x.exe").
var pathWordRe = regexp.MustCompile(`^[^` + pathStop + `]+`)

// redactText reduces every filesystem path in s to its final path element. A
// path with no element left becomes "[path]". Text with no path is returned
// unchanged.
func redactText(s string) string {
	var b strings.Builder
	last, pos := 0, 0
	for pos < len(s) {
		loc := pathTokenRe.FindStringIndex(s[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		attached := false
		if start > 0 && isWordByte(s[start-1]) {
			// Glued to a word: accept only the forms attachOK allows. Otherwise
			// resume one byte on, so a path that starts inside this match is
			// still found.
			if !attachOK(s[start:end]) {
				pos = start + 1
				continue
			}
			attached = true
		}
		end = extendOverSpaces(s, end)
		b.WriteString(s[last:start])
		if attached {
			b.WriteByte(' ')
		}
		b.WriteString(pathBase(s[start:end]))
		last, pos = end, end
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// attachOK reports whether a path token that starts right after a word
// character is a path. A slash form needs a known root. A backslash form needs
// a known Windows root. A drive letter glued to a word ("fooC:\Users") is a
// drive, but "https:/" is a scheme: the slash form needs an uppercase drive
// letter and a single slash.
func attachOK(tok string) bool {
	switch {
	case tok[0] == '/':
		return rootPathRe.MatchString(tok)
	case tok[0] == '\\':
		return winRootRe.MatchString(tok)
	case len(tok) > 3 && isASCIILetter(tok[0]) && tok[1] == ':':
		if tok[2] == '\\' {
			return true
		}
		return tok[2] == '/' && tok[0] >= 'A' && tok[0] <= 'Z' && tok[3] != '/'
	}
	return false
}

// extendOverSpaces returns the end of a path that continues after spaces. A
// following space-free word belongs to the path while it holds a separator, so
// a path with spaces does not leave its tail behind. A word without a separator
// ends the path.
func extendOverSpaces(s string, end int) int {
	for end < len(s) && s[end] == ' ' {
		m := pathWordRe.FindString(s[end+1:])
		if m == "" || !strings.ContainsAny(m, `/\`) {
			break
		}
		end += 1 + len(m)
	}
	return end
}

// isWordByte reports whether c is part of a name, so a slash after it joins the
// name rather than starting a path.
func isWordByte(c byte) bool {
	return c == '_' || c == '.' || c == '-' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// isASCIILetter reports whether c is a letter a-z or A-Z.
func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// pathBase returns the last element of a path token, or "[path]" if none.
func pathBase(tok string) string {
	i := strings.LastIndexAny(tok, `/\`)
	base := tok[i+1:]
	if base == "" {
		return "[path]"
	}
	return base
}

// PublicStruct returns a copy of s with the public-view redactions applied. It
// returns nil for a nil input and never modifies s. Keys that name a path are
// dropped; string values are reduced to their path-free form.
func PublicStruct(s *structpb.Struct) *structpb.Struct {
	if s == nil {
		return nil
	}
	out := &structpb.Struct{Fields: make(map[string]*structpb.Value, len(s.Fields))}
	for k, v := range s.Fields {
		if publicDropKeys[k] || redactText(k) != k {
			continue
		}
		out.Fields[k] = publicValue(v)
	}
	return out
}

// publicValue returns the public copy of v. Strings are reduced; nested structs
// and lists are walked.
func publicValue(v *structpb.Value) *structpb.Value {
	if v == nil {
		return nil
	}
	switch kind := v.Kind.(type) {
	case *structpb.Value_StringValue:
		return structpb.NewStringValue(redactText(kind.StringValue))
	case *structpb.Value_StructValue:
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: PublicStruct(kind.StructValue)}}
	case *structpb.Value_ListValue:
		list := &structpb.ListValue{Values: make([]*structpb.Value, 0, len(kind.ListValue.GetValues()))}
		for _, item := range kind.ListValue.GetValues() {
			list.Values = append(list.Values, publicValue(item))
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: list}}
	default:
		return gproto.Clone(v).(*structpb.Value)
	}
}
