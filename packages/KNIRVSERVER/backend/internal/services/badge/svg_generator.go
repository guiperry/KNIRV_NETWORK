// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package badge

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
	"time"
)

// BadgeSVGParams controls badge SVG generation.
type BadgeSVGParams struct {
	Name           string   // badge name (displayed prominently)
	BadgeType      string   // "skill", "capability", "property"
	Description    string   // badge description
	ValueSignals   []string // selected values (displayed around border)
	OntologySignals []string // selected ontology elements (displayed around border)
	AITextTag      string   // configurable AI text tag placed within the badge design
	OutputSize     int      // SVG viewport size (default: 400)
	PrimaryColor   string   // hex e.g. "#FBBF24" (gold)
	SecondaryColor string   // hex e.g. "#D97706" (dark gold)
	BackgroundColor string  // hex e.g. "#02040a"
	BackgroundImage string  // base64-encoded background image (data:image/... or raw base64)
	PolicyName      string  // policy name to display on badge
	Tags            []string // tags to display on badge
}

// GenerateBadgeSVG produces an SVG string for a badge combining the selected
// value signals, ontology elements, and an AI text tag into a visually
// structured design suitable for NFT minting.
//
// The layout is:
//   - Outer ring with value signals radiating around the badge.
//   - Middle ring with ontology element labels.
//   - Center with a graphical element (custom icon based on badge type).
//   - Badge name text.
//   - AI text tag displayed below the name (if configured).
//   - KNIRV VERIFIED certification stamp.
func GenerateBadgeSVG(params BadgeSVGParams) string {
	if params.OutputSize <= 0 {
		params.OutputSize = 400
	}
	if params.PrimaryColor == "" {
		params.PrimaryColor = "#FBBF24"
	}
	if params.SecondaryColor == "" {
		params.SecondaryColor = "#D97706"
	}
	if params.BackgroundColor == "" {
		params.BackgroundColor = "#02040a"
	}

	size := params.OutputSize
	cx := size / 2
	cy := size / 2
	outerR := size / 2 - 10

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">
`, size, size, size, size))

	// Defs for gradients and filters
	buf.WriteString(fmt.Sprintf(`  <defs>
    <linearGradient id="primaryGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:%s"/>
      <stop offset="50%%" style="stop-color:%s"/>
      <stop offset="100%%" style="stop-color:%s"/>
    </linearGradient>
    <linearGradient id="darkGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#1F2937"/>
      <stop offset="100%%" style="stop-color:#111827"/>
    </linearGradient>
    <filter id="glow">
      <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
      <feMerge>
        <feMergeNode in="coloredBlur"/>
        <feMergeNode in="SourceGraphic"/>
      </feMerge>
    </filter>
  </defs>
`, params.PrimaryColor, params.SecondaryColor, params.PrimaryColor))

	// Background
	buf.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="%s"/>
`, size, size, params.BackgroundColor))

	// Background image (if provided, render as centered circular image with clip path)
	if params.BackgroundImage != "" {
		href := params.BackgroundImage
		if !strings.HasPrefix(href, "data:") {
			href = "data:image/png;base64," + href
		}
		buf.WriteString(fmt.Sprintf(`  <clipPath id="bgClip"><circle cx="%d" cy="%d" r="%d"/></clipPath>
`, cx, cy, outerR-8))
		buf.WriteString(fmt.Sprintf(`  <image href="%s" x="0" y="0" width="%d" height="%d" clip-path="url(#bgClip)" preserveAspectRatio="xMidYMid slice"/>
`, href, size, size))
	}

	// Outer rings
	buf.WriteString(fmt.Sprintf(`  <circle cx="%d" cy="%d" r="%d" fill="url(#darkGrad)" stroke="url(#primaryGrad)" stroke-width="6"/>
`, cx, cy, outerR))
	buf.WriteString(fmt.Sprintf(`  <circle cx="%d" cy="%d" r="%d" fill="none" stroke="url(#primaryGrad)" stroke-width="1.5" opacity="0.4"/>
`, cx, cy, outerR-20))

	// Center polygon icon (varies by badge type)
	centerIcon := badgeTypeIcon(params.BadgeType)
	buf.WriteString(fmt.Sprintf(`  <polygon points="%s" fill="url(#primaryGrad)" filter="url(#glow)"/>
`, centerIcon))

	// Value signals around the outer ring
	allSignals := append([]string{}, params.ValueSignals...)
	allSignals = append(allSignals, params.OntologySignals...)
	sort.Strings(allSignals)

	// Place up to 16 signal labels around the outer ring.
	renderSignalRing(&buf, allSignals, cx, cy, outerR-30, 14, params.PrimaryColor)

	// Badge name
	escapedName := html.EscapeString(truncate(params.Name, 24))
	buf.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" fill="%s" font-family="'Arial Black', Arial, sans-serif" font-size="%d" font-weight="bold" filter="url(#glow)">%s</text>
`, cx, cy+40, params.PrimaryColor, badgeNameFontSize(params.Name), escapedName))

	// AI text tag
	if params.AITextTag != "" {
		escapedTag := html.EscapeString(truncate(params.AITextTag, 32))
		buf.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" fill="%s" font-family="Arial, sans-serif" font-size="11" font-style="italic" opacity="0.7">%s</text>
`, cx, cy+60, params.PrimaryColor, escapedTag))
	}

	// Policy name (if provided, displayed prominently below badge name)
	if params.PolicyName != "" {
		escapedPolicy := html.EscapeString(truncate(params.PolicyName, 28))
		buf.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" fill="%s" font-family="'Arial', sans-serif" font-size="12" font-weight="bold" opacity="0.9">%s</text>
`, cx, cy+75, params.PrimaryColor, escapedPolicy))
	}

	// Tags (if provided, rendered as small rounded rectangles below policy name)
	if len(params.Tags) > 0 {
		tagStartX := cx - (len(params.Tags)*70)/2
		if tagStartX < 10 {
			tagStartX = 10
		}
		for i, tag := range params.Tags {
			if i >= 6 {
				break
			} // max 6 tags
			escapedTag := html.EscapeString(truncate(tag, 10))
			tx := tagStartX + i*70
			ty := cy + 90
			buf.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="60" height="16" rx="8" fill="%s" opacity="0.3"/>
  <text x="%d" y="%d" text-anchor="middle" fill="%s" font-family="Arial, sans-serif" font-size="8" font-weight="bold">%s</text>
`, tx, ty, params.PrimaryColor, tx+30, ty+12, params.PrimaryColor, escapedTag))
		}
	}

	// KNIRV VERIFIED certification
	buf.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" fill="#9CA3AF" font-family="Arial, sans-serif" font-size="10" font-weight="bold" letter-spacing="2">KNIRV VERIFIED</text>
`, cx, size-20))

	// Generation watermark
	buf.WriteString(fmt.Sprintf(`  <text x="%d" y="%d" text-anchor="middle" fill="#374151" font-family="monospace" font-size="7">badge-lab | %s</text>
`, cx, size-8, time.Now().UTC().Format("2006-01-02")))

	buf.WriteString("</svg>")
	return buf.String()
}

// —— internal rendering helpers ——

func badgeTypeIcon(badgeType string) string {
	// 10-point star centered at (200,200) using relative coordinates.
	// The points are placed on inner (150) and outer (190) radii.
	outerR := 90.0
	innerR := 60.0
	cx := 200.0
	cy := 200.0
	points := 10

	var parts []string
	for i := 0; i < points*2; i++ {
		angle := math.Pi/2 - (float64(i) * math.Pi / float64(points))
		var r float64
		if i%2 == 0 {
			r = outerR
		} else {
			r = innerR
		}
		x := cx + r*math.Cos(angle)
		y := cy - r*math.Sin(angle)
		parts = append(parts, fmt.Sprintf("%.0f,%.0f", x, y))
	}
	return strings.Join(parts, " ")
}

func badgeNameFontSize(name string) int {
	l := len(name)
	switch {
	case l <= 8:
		return 22
	case l <= 16:
		return 18
	case l <= 24:
		return 14
	default:
		return 12
	}
}

// renderSignalRing places signal labels in a circle around the badge center.
func renderSignalRing(buf *bytes.Buffer, signals []string, cx, cy, radius, maxItems int, color string) {
	if len(signals) == 0 {
		return
	}
	items := signals
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	n := len(items)
	for i, label := range items {
		angle := float64(i)*2*math.Pi/float64(n) - math.Pi/2
		x := float64(cx) + float64(radius)*math.Cos(angle)
		y := float64(cy) + float64(radius)*math.Sin(angle)

		// Shorten signal labels.
		short := truncate(label, 12)
		escaped := html.EscapeString(short)
		rotation := angle * 180 / math.Pi

		buf.WriteString(fmt.Sprintf(
			`  <text x="%.0f" y="%.0f" text-anchor="middle" dominant-baseline="central" fill="%s" font-family="Arial, sans-serif" font-size="8" font-weight="bold" opacity="0.6" transform="rotate(%.0f, %.0f, %.0f)">%s</text>
`,
			x, y, color, rotation, x, y, escaped))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
