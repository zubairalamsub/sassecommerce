package templates

import (
	"strings"
	"testing"
)

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text is untouched", in: "Hello there", want: "Hello there"},
		{name: "empty string", in: "", want: ""},
		{name: "ampersand", in: "Tom & Jerry", want: "Tom &amp; Jerry"},
		{name: "angle brackets", in: "a < b > c", want: "a &lt; b &gt; c"},
		{name: "double quote", in: `say "hi"`, want: "say &quot;hi&quot;"},
		{name: "single quote", in: "it's", want: "it&#39;s"},
		{name: "all five at once", in: `&<>"'`, want: "&amp;&lt;&gt;&quot;&#39;"},
		{
			name: "script tag is neutralised",
			in:   `<script>alert('xss')</script>`,
			want: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			// Escaping is hand-rolled precisely so Bangla and the taka sign
			// survive as literal runes instead of numeric entities.
			name: "non-ASCII survives verbatim",
			in:   "সাজান ৳500",
			want: "সাজান ৳500",
		},
		{
			name: "non-ASCII mixed with an escapable character",
			in:   "সাজান & Co ৳500",
			want: "সাজান &amp; Co ৳500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeHTML(tt.in); got != tt.want {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNeedsEscape(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "", want: false},
		{in: "plain", want: false},
		{in: "সাজান ৳500", want: false},
		{in: "a&b", want: true},
		{in: "a<b", want: true},
		{in: "a>b", want: true},
		{in: `a"b`, want: true},
		{in: "a'b", want: true},
	}

	for _, tt := range tests {
		if got := needsEscape(tt.in); got != tt.want {
			t.Errorf("needsEscape(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestEscapeAttrAndPreserveURLMatchEscapeHTML(t *testing.T) {
	inputs := []string{"", "plain", `a"b`, "<script>", "সাজান ৳500", "x & y"}

	for _, in := range inputs {
		if got, want := escapeAttr(in), escapeHTML(in); got != want {
			t.Errorf("escapeAttr(%q) = %q, want %q", in, got, want)
		}
		if got, want := escapeHTMLPreserveURL(in), escapeHTML(in); got != want {
			t.Errorf("escapeHTMLPreserveURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatBody(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single paragraph",
			in:   "Hello there",
			want: `<p style="margin:0 0 12px 0;">Hello there</p>`,
		},
		{
			name: "double newline starts a new paragraph",
			in:   "First para\n\nSecond para",
			want: `<p style="margin:0 0 12px 0;">First para</p><p style="margin:12px 0;">Second para</p>`,
		},
		{
			name: "single newline becomes a line break",
			in:   "Line one\nLine two",
			want: `<p style="margin:0 0 12px 0;">Line one<br>Line two</p>`,
		},
		{
			name: "breaks and paragraphs together",
			in:   "A\nB\n\nC",
			want: `<p style="margin:0 0 12px 0;">A<br>B</p><p style="margin:12px 0;">C</p>`,
		},
		{
			name: "empty input still yields a closed paragraph",
			in:   "",
			want: `<p style="margin:0 0 12px 0;"></p>`,
		},
		{
			name: "markup in body copy is escaped",
			in:   "<b>bold</b>",
			want: `<p style="margin:0 0 12px 0;">&lt;b&gt;bold&lt;/b&gt;</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBody(tt.in); got != tt.want {
				t.Errorf("formatBody(%q) =\n  %q\nwant\n  %q", tt.in, got, tt.want)
			}
		})
	}
}

// A verification link must survive body copy verbatim — an escaped '?' or '='
// would produce a link the recipient cannot use.
func TestFormatBodyKeepsURLsUsable(t *testing.T) {
	url := "https://shop.example.com/verify-email?token=abc123"

	got := formatBody("Click here:\n" + url)

	if !strings.Contains(got, url) {
		t.Errorf("formatBody dropped or mangled the URL:\n%s", got)
	}
}

func TestInsertThousandSeparator(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "1", want: "1"},
		{in: "12", want: "12"},
		{in: "123", want: "123"},
		{in: "1234", want: "1,234"},
		{in: "12345", want: "12,345"},
		{in: "123456", want: "123,456"},
		{in: "1234567", want: "1,234,567"},
		{in: "1234567890", want: "1,234,567,890"},
	}

	for _, tt := range tests {
		if got := insertThousandSeparator(tt.in); got != tt.want {
			t.Errorf("insertThousandSeparator(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatBDT(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{name: "zero", amount: 0, want: "৳0.00"},
		{name: "whole taka", amount: 500, want: "৳500.00"},
		{name: "with paisa", amount: 1250.5, want: "৳1,250.50"},
		{name: "thousands separator", amount: 1234567.89, want: "৳1,234,567.89"},
		{name: "negative", amount: -99.5, want: "-৳99.50"},
		{name: "rounds paisa up", amount: 10.005, want: "৳10.01"},
		{name: "rounds paisa down", amount: 10.004, want: "৳10.00"},
		{
			// 9.999 rounds to 100 paisa, which has to carry into the taka.
			name:   "paisa carry rolls into taka",
			amount: 9.999,
			want:   "৳10.00",
		},
		{name: "carry across a separator boundary", amount: 999.999, want: "৳1,000.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBDT(tt.amount); got != tt.want {
				t.Errorf("FormatBDT(%v) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestRenderEmailHTMLStructure(t *testing.T) {
	html := RenderEmailHTML("Order confirmed", "Hi Bob,", "Your order is on its way.", "View order", "https://shop.example.com/orders/1", "This link expires in 24 hours.")

	mustContain := []string{
		"<!DOCTYPE html>",
		"<title>Order confirmed</title>",
		"Order confirmed",
		"Hi Bob,",
		"Your order is on its way.",
		"View order",
		"https://shop.example.com/orders/1",
		"This link expires in 24 hours.",
		DefaultTenantName,
		BrandPrimary,
		"</html>",
	}
	for _, want := range mustContain {
		if !strings.Contains(html, want) {
			t.Errorf("rendered email is missing %q", want)
		}
	}

	// The CTA has to open in a new tab without leaking the referrer.
	if !strings.Contains(html, `target="_blank" rel="noopener"`) {
		t.Error("CTA link is missing target/rel hardening")
	}
	// Mail clients strip <style>, so the layout must not depend on it.
	if !strings.Contains(html, "max-width:600px") {
		t.Error("card width should be set inline, not only in the media query")
	}
}

// Every optional section is genuinely optional.
func TestRenderEmailHTMLOmitsEmptySections(t *testing.T) {
	html := RenderEmailHTML("Title only", "", "", "", "", "")

	if strings.Contains(html, "Or copy and paste this link") {
		t.Error("CTA block rendered without cta text/url")
	}
	if strings.Contains(html, "font-style:italic") {
		t.Error("footnote block rendered without a footnote")
	}
	// The shell must still be complete.
	if !strings.Contains(html, "<!DOCTYPE html>") || !strings.Contains(html, "</html>") {
		t.Error("shell is incomplete when every optional section is empty")
	}
}

// The CTA needs both halves; text without a URL would render a dead button.
func TestRenderEmailHTMLRequiresBothCTAHalves(t *testing.T) {
	tests := []struct {
		name    string
		ctaText string
		ctaURL  string
	}{
		{name: "text without url", ctaText: "Click me", ctaURL: ""},
		{name: "url without text", ctaText: "", ctaURL: "https://shop.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := RenderEmailHTML("Title", "", "Body", tt.ctaText, tt.ctaURL, "")

			// The class also appears in the <style> media query, so match the
			// anchor itself.
			if strings.Contains(html, `class="email-cta" href=`) {
				t.Error("CTA button rendered with only half of the pair supplied")
			}
		})
	}
}

// Recipient-controlled values (names, product titles) reach these fields, so
// nothing may pass through as live markup.
func TestRenderEmailHTMLEscapesInjectedMarkup(t *testing.T) {
	payload := `<script>alert('xss')</script>`

	html := RenderEmailHTML(payload, payload, payload, payload, "https://shop.example.com", payload)

	if strings.Contains(html, "<script>") {
		t.Error("an injected <script> tag survived into the rendered email")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("the injected markup should appear escaped, not dropped")
	}
}

// A quote in the CTA URL must not be able to close the href attribute and add
// its own onclick handler.
func TestRenderEmailHTMLEscapesCTAAttribute(t *testing.T) {
	html := RenderEmailHTML("Title", "", "Body", "Click", `https://shop.example.com/" onclick="alert(1)`, "")

	if strings.Contains(html, `onclick="alert(1)"`) {
		t.Error("a quote in the CTA URL broke out of the href attribute")
	}
	if !strings.Contains(html, "&quot;") {
		t.Error("the quote in the CTA URL should have been escaped")
	}
}
