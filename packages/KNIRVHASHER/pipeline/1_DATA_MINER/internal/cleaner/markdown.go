package cleaner

import (
	"fmt"
	"regexp"
	"strings"
)

// Cleaner strips Markdown formatting and non-semantic noise from raw document
// text, producing clean prose suitable for NLP processing.
type Cleaner struct {
	codeBlock  *regexp.Regexp
	htmlTag    *regexp.Regexp
	mdHeading  *regexp.Regexp
	mdLink     *regexp.Regexp
	multiSpace *regexp.Regexp
}

// New returns a ready-to-use Cleaner.
func New() *Cleaner {
	return &Cleaner{
		codeBlock:  regexp.MustCompile("(?s)```.*?```|`[^`]+`"),
		htmlTag:    regexp.MustCompile("<[^>]+>"),
		mdHeading:  regexp.MustCompile(`(?m)^#{1,6}\s+`),
		mdLink:     regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`),
		multiSpace: regexp.MustCompile(`[ \t]+`),
	}
}

// CleanMarkdown removes Markdown markup and returns clean prose. Returns an
// error when the result is empty after cleaning.
func (c *Cleaner) CleanMarkdown(raw string) (string, error) {
	// Remove code blocks and inline code.
	text := c.codeBlock.ReplaceAllString(raw, " ")
	// Remove HTML tags.
	text = c.htmlTag.ReplaceAllString(text, " ")
	// Strip heading markers, keeping the heading text.
	text = c.mdHeading.ReplaceAllString(text, "")
	// Replace markdown links with their display text.
	text = c.mdLink.ReplaceAllStringFunc(text, func(m string) string {
		sub := c.mdLink.FindStringSubmatch(m)
		if len(sub) > 1 {
			return sub[1]
		}
		return ""
	})
	// Strip bold/italic emphasis markers.
	r := strings.NewReplacer("**", "", "__", "", "*", "", "_", "")
	text = r.Replace(text)
	// Normalise horizontal whitespace; preserve newlines for sentence breaks.
	text = c.multiSpace.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("document is empty after cleaning")
	}
	return text, nil
}
