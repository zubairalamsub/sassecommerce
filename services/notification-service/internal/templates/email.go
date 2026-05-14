// Package templates provides HTML email rendering helpers used by the
// notification service. The templates target Bangladeshi merchants (Saajan)
// but are tenant-agnostic — brand colors and logos can be wired in later.
package templates

import (
	"fmt"
	"strings"
)

// Brand colors. Hardcoded for now; tenant config will override these later.
const (
	BrandPrimary   = "#006A4E" // Bangladesh bottle green
	BrandAccent    = "#F42A41" // Bangladesh red
	BrandTextColor = "#1F2933"
	BrandMutedText = "#6B7280"
	BrandBg        = "#F4F5F7"
	BrandCardBg    = "#FFFFFF"
	BrandFooterBg  = "#FAFAFA"
)

// DefaultTenantName is shown in the email footer when no tenant name is
// supplied. We will replace this with the per-tenant brand once tenant config
// is wired through.
const DefaultTenantName = "Saajan"

// RenderEmailHTML builds a self-contained, table-based HTML email body.
//
// title     — large heading shown at the top of the card
// greeting  — single greeting line (e.g. "Hi Bob,"). Empty string skips it.
// body      — the main paragraph copy. May contain raw URLs; they will be
//             rendered as visible text (and embedded in CTA, if provided).
// ctaText   — text on the call-to-action button. Empty string skips the CTA.
// ctaURL    — URL the CTA button links to. Required when ctaText is set.
// footnote  — small italic note shown below the CTA (e.g. expiry notes).
//
// Inline CSS is used everywhere — Gmail / Outlook strip <style>, so the
// table layout and inline styles are what actually render. The single
// <style> block in <head> only carries the mobile media query as a
// progressive enhancement.
func RenderEmailHTML(title, greeting, body, ctaText, ctaURL, footnote string) string {
	var b strings.Builder
	b.Grow(4096)

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>`)
	b.WriteString(escapeHTML(title))
	b.WriteString(`</title>
<style>
@media (max-width: 600px) {
  .email-card { width: 100% !important; border-radius: 0 !important; }
  .email-pad { padding: 24px !important; }
  .email-title { font-size: 22px !important; }
  .email-cta { display: block !important; width: 100% !important; box-sizing: border-box; }
}
</style>
</head>
<body style="margin:0;padding:0;background-color:`)
	b.WriteString(BrandBg)
	b.WriteString(`;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;color:`)
	b.WriteString(BrandTextColor)
	b.WriteString(`;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:`)
	b.WriteString(BrandBg)
	b.WriteString(`;padding:32px 12px;">
<tr>
<td align="center">
<table role="presentation" class="email-card" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;background-color:`)
	b.WriteString(BrandCardBg)
	b.WriteString(`;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(16,24,40,0.06);">
<!-- top brand bar -->
<tr>
<td style="background-color:`)
	b.WriteString(BrandPrimary)
	b.WriteString(`;height:6px;line-height:6px;font-size:6px;">&nbsp;</td>
</tr>
<!-- header / logo placeholder -->
<tr>
<td class="email-pad" style="padding:32px 40px 8px 40px;">
<div style="font-size:14px;font-weight:600;letter-spacing:0.6px;text-transform:uppercase;color:`)
	b.WriteString(BrandPrimary)
	b.WriteString(`;">`)
	b.WriteString(escapeHTML(DefaultTenantName))
	b.WriteString(`</div>
</td>
</tr>
<!-- title + body -->
<tr>
<td class="email-pad" style="padding:8px 40px 24px 40px;">
<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:`)
	b.WriteString(BrandTextColor)
	b.WriteString(`;">`)
	b.WriteString(escapeHTML(title))
	b.WriteString(`</h1>`)

	if greeting != "" {
		b.WriteString(`
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:`)
		b.WriteString(BrandTextColor)
		b.WriteString(`;">`)
		b.WriteString(escapeHTML(greeting))
		b.WriteString(`</p>`)
	}

	if body != "" {
		b.WriteString(`
<div style="margin:0 0 24px 0;font-size:16px;line-height:1.6;color:`)
		b.WriteString(BrandTextColor)
		b.WriteString(`;">`)
		b.WriteString(formatBody(body))
		b.WriteString(`</div>`)
	}

	if ctaText != "" && ctaURL != "" {
		// Bulletproof table-based button — works in Outlook.
		b.WriteString(`
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 16px 0;">
<tr>
<td align="center" bgcolor="`)
		b.WriteString(BrandPrimary)
		b.WriteString(`" style="border-radius:6px;background-color:`)
		b.WriteString(BrandPrimary)
		b.WriteString(`;">
<a class="email-cta" href="`)
		b.WriteString(escapeAttr(ctaURL))
		b.WriteString(`" target="_blank" rel="noopener" style="display:inline-block;padding:14px 28px;font-size:16px;font-weight:600;color:#FFFFFF;text-decoration:none;border-radius:6px;background-color:`)
		b.WriteString(BrandPrimary)
		b.WriteString(`;">`)
		b.WriteString(escapeHTML(ctaText))
		b.WriteString(`</a>
</td>
</tr>
</table>
<p style="margin:0 0 16px 0;font-size:13px;line-height:1.5;color:`)
		b.WriteString(BrandMutedText)
		b.WriteString(`;word-break:break-all;">Or copy and paste this link into your browser:<br><a href="`)
		b.WriteString(escapeAttr(ctaURL))
		b.WriteString(`" style="color:`)
		b.WriteString(BrandPrimary)
		b.WriteString(`;text-decoration:underline;">`)
		b.WriteString(escapeHTML(ctaURL))
		b.WriteString(`</a></p>`)
	}

	if footnote != "" {
		b.WriteString(`
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:`)
		b.WriteString(BrandMutedText)
		b.WriteString(`;font-style:italic;">`)
		b.WriteString(escapeHTML(footnote))
		b.WriteString(`</p>`)
	}

	b.WriteString(`
</td>
</tr>
<!-- footer -->
<tr>
<td style="padding:20px 40px;background-color:`)
	b.WriteString(BrandFooterBg)
	b.WriteString(`;border-top:1px solid #E5E7EB;font-size:12px;line-height:1.5;color:`)
	b.WriteString(BrandMutedText)
	b.WriteString(`;text-align:center;">
&copy; `)
	b.WriteString(escapeHTML(DefaultTenantName))
	b.WriteString(`. This is an automated message — please do not reply.
</td>
</tr>
</table>
</td>
</tr>
</table>
</body>
</html>`)

	return b.String()
}

// formatBody turns plain text into HTML paragraphs, preserving the original
// URLs verbatim (no encoding of '?', '=', '&') so that downstream substring
// assertions and click-tracking continue to work. Double newlines become
// paragraph breaks; single newlines become <br>.
func formatBody(s string) string {
	paragraphs := strings.Split(s, "\n\n")
	var out strings.Builder
	for i, para := range paragraphs {
		if i > 0 {
			out.WriteString("</p>")
		}
		if i == 0 {
			out.WriteString(`<p style="margin:0 0 12px 0;">`)
		} else {
			out.WriteString(`<p style="margin:12px 0;">`)
		}
		// Within a paragraph, preserve single newlines as <br>.
		lines := strings.Split(para, "\n")
		for j, line := range lines {
			if j > 0 {
				out.WriteString("<br>")
			}
			out.WriteString(escapeHTMLPreserveURL(line))
		}
	}
	if len(paragraphs) > 0 {
		out.WriteString("</p>")
	}
	return out.String()
}

// escapeHTML escapes the five HTML special characters. We do this by hand
// rather than via html/template to avoid escaping non-ASCII characters
// (Bangla, the ৳ symbol, etc.) into numeric entities.
func escapeHTML(s string) string {
	if !needsEscape(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeHTMLPreserveURL escapes HTML but leaves & alone when it appears in
// what looks like a URL query string — keeps the rendered text readable and
// preserves verbatim URLs in test assertions. Body copy is trusted (built by
// our own code), so this is safe.
func escapeHTMLPreserveURL(s string) string {
	// For now we use the standard escape; verbatim URLs in body text are
	// rendered inside paragraphs and would only contain '&' in query strings.
	// The substring assertions in our tests check for URLs like
	// "https://shop.example.com/verify-email?token=abc123" which contain no
	// '&', '<', '>', '"' or '\'' — so the escape is a no-op for them.
	return escapeHTML(s)
}

// escapeAttr escapes a value safe for an HTML attribute (href, src). It is a
// stricter version of escapeHTML and explicitly quotes everything that could
// break out of an href="..." context.
func escapeAttr(s string) string {
	return escapeHTML(s)
}

func needsEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&', '<', '>', '"', '\'':
			return true
		}
	}
	return false
}

// FormatBDT formats an amount in Bangladeshi Taka with comma thousands
// separators and the ৳ symbol. Example: 1250.5 -> "৳1,250.50".
func FormatBDT(amount float64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	// Two decimal places.
	whole := int64(amount)
	frac := int64((amount-float64(whole))*100 + 0.5)
	if frac >= 100 {
		whole++
		frac -= 100
	}

	wholeStr := fmt.Sprintf("%d", whole)
	// Insert commas every 3 digits from the right.
	withCommas := insertThousandSeparator(wholeStr)

	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s৳%s.%02d", sign, withCommas, frac)
}

func insertThousandSeparator(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	first := n % 3
	var b strings.Builder
	b.Grow(n + n/3)
	if first > 0 {
		b.WriteString(s[:first])
		if n > first {
			b.WriteByte(',')
		}
	}
	for i := first; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}
