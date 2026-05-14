// Package templates - default_templates.go bundles the "starter pack" of
// pre-designed notification templates that ship with every Saajan tenant. The
// admin UI's "Install starter pack" button writes these into a tenant's
// notification_templates collection so they can be customized in the editor.
//
// Each entry pairs a Saajan-branded HTML body with a sensible default subject
// line and metadata (Type, Channel, Name). The bodies use Go text/template
// syntax against the same variables the consumer injects at runtime
// (see service.MergeSampleVars), so admins can author against the same names.
//
// Design conventions baked into every email body:
//   - Inline CSS only (clients strip <style>)
//   - 600px max card on soft grey page bg, 6px brand-color stripe on top
//   - 26px bold title, 16px/1.6 body in #1F2933 text
//   - Bulletproof table-based CTA buttons (Outlook compatible)
//   - Footer with tenant name + "automated message" copy
//   - Mobile-friendly via a single <style>@media block (progressive)
//   - Bangla-safe (UTF-8, no escaping of non-ASCII like ৳ or Bangla script)
//   - {{.BrandColor}} drives the stripe + CTA so each tenant looks distinct
package templates

import "github.com/ecommerce/notification-service/internal/models"

// DefaultTemplate is the registry entry used by the admin "install defaults"
// endpoint. It mirrors the persisted NotificationTemplate but without
// tenant/id/timestamps — those are filled in at install time.
type DefaultTemplate struct {
	Type            models.NotificationType
	Channel         models.Channel
	Name            string
	SubjectTemplate string
	BodyTemplate    string
}

// Defaults returns the canonical list of starter-pack templates. It is a
// function (not a package-level var) so callers always get a fresh slice — the
// install path may mutate per-tenant copies and we don't want to leak that
// into the registry.
func Defaults() []DefaultTemplate {
	return []DefaultTemplate{
		welcomeTemplate(),
		emailVerificationTemplate(),
		passwordResetTemplate(),
		passwordChangedTemplate(),
		orderConfirmationTemplate(),
		orderShippedTemplate(),
		orderCancelledTemplate(),
		paymentConfirmedTemplate(),
		paymentFailedTemplate(),
		receiptTemplate(),
		stockAlertTemplate(),
	}
}

// emailShell wraps body content with the standard Saajan email chrome:
// soft-grey page bg, white 600px card, brand-color stripe, footer. The body
// fragment is dropped into the main <td> so each template can compose its own
// layout (headings, items table, etc.) inside the chrome.
//
// Defined as a function so we can fmt-style substitute the variable
// placeholders (which themselves are Go template syntax) without escaping.
func emailShell(bodyFragment, footerExtra string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.TenantName}}</title>
<style>
@media (max-width: 600px) {
  .email-card { width: 100% !important; border-radius: 0 !important; }
  .email-pad { padding: 24px !important; }
  .email-title { font-size: 22px !important; }
  .email-cta { display: block !important; width: 100% !important; box-sizing: border-box; }
}
</style>
</head>
<body style="margin:0;padding:0;background-color:#F4F5F7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,'SolaimanLipi','Kalpurush',sans-serif;color:#1F2933;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#F4F5F7;padding:32px 12px;">
<tr><td align="center">
<table role="presentation" class="email-card" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;background-color:#FFFFFF;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(16,24,40,0.06);">
<tr><td style="background-color:{{.BrandColor}};height:6px;line-height:6px;font-size:6px;">&nbsp;</td></tr>
<tr><td class="email-pad" style="padding:32px 40px 8px 40px;">
<div style="font-size:14px;font-weight:600;letter-spacing:0.6px;text-transform:uppercase;color:{{.BrandColor}};">{{.TenantName}}</div>
</td></tr>
<tr><td class="email-pad" style="padding:8px 40px 24px 40px;">
` + bodyFragment + `
</td></tr>
<tr><td style="padding:20px 40px;background-color:#FAFAFA;border-top:1px solid #E5E7EB;font-size:12px;line-height:1.5;color:#6B7280;text-align:center;">
` + footerExtra + `&copy; {{.TenantName}}. This is an automated message — please do not reply.<br>
Need help? Contact <a href="mailto:support@{{.TenantName}}.com" style="color:{{.BrandColor}};text-decoration:underline;">support@{{.TenantName}}.com</a>.<br>
You're receiving this because you have an account with {{.TenantName}}.
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`
}

// ctaButton builds a bulletproof table-based CTA button. Outlook respects the
// bgcolor attr on <td> even when CSS background fails. Both the wrapper and
// the anchor get the brand color so the button renders solid everywhere.
func ctaButton(text, hrefTemplate string) string {
	return `<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 16px 0;">
<tr><td align="center" bgcolor="{{.BrandColor}}" style="border-radius:6px;background-color:{{.BrandColor}};">
<a class="email-cta" href="` + hrefTemplate + `" target="_blank" rel="noopener" style="display:inline-block;padding:14px 28px;font-size:16px;font-weight:600;color:#FFFFFF;text-decoration:none;border-radius:6px;background-color:{{.BrandColor}};">` + text + `</a>
</td></tr>
</table>`
}

// === Individual template definitions ===
//
// Each builder returns a DefaultTemplate. The body fragment is inserted into
// emailShell, which adds the surrounding card/footer chrome.

func welcomeTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Welcome, {{.CustomerName}}!</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Your {{.TenantName}} account is ready. We're delighted to have you with us.</p>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Browse our latest collections, save favourites to your wishlist, and enjoy fast delivery across Bangladesh.</p>
` + ctaButton("Start shopping", "{{.FrontendBaseURL}}") + `
<p style="margin:16px 0 8px 0;font-size:14px;line-height:1.6;color:#6B7280;">Popular right now:</p>
<ul style="margin:0 0 16px 20px;padding:0;font-size:14px;line-height:1.6;color:#1F2933;">
  <li><a href="{{.FrontendBaseURL}}/categories/clothing" style="color:{{.BrandColor}};text-decoration:none;">Clothing</a></li>
  <li><a href="{{.FrontendBaseURL}}/categories/home-living" style="color:{{.BrandColor}};text-decoration:none;">Home &amp; living</a></li>
  <li><a href="{{.FrontendBaseURL}}/categories/electronics" style="color:{{.BrandColor}};text-decoration:none;">Electronics</a></li>
</ul>`
	return DefaultTemplate{
		Type:            models.TypeWelcome,
		Channel:         models.ChannelEmail,
		Name:            "Welcome — starter",
		SubjectTemplate: "Welcome to {{.TenantName}}, {{.CustomerName}}!",
		BodyTemplate:    emailShell(body, ""),
	}
}

func emailVerificationTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Verify your email</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Welcome to {{.TenantName}}! Please confirm this is your email address so we can secure your account and send you order updates.</p>
` + ctaButton("Verify email", "{{.VerifyURL}}") + `
<p style="margin:0 0 16px 0;font-size:13px;line-height:1.5;color:#6B7280;word-break:break-all;">Or copy and paste this link: <a href="{{.VerifyURL}}" style="color:{{.BrandColor}};text-decoration:underline;">{{.VerifyURL}}</a></p>
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;font-style:italic;">This link expires in 24 hours. If you didn't create a {{.TenantName}} account, you can safely ignore this email.</p>`
	return DefaultTemplate{
		Type:            models.TypeEmailVerification,
		Channel:         models.ChannelEmail,
		Name:            "Email verification — starter",
		SubjectTemplate: "Verify your email — {{.TenantName}}",
		BodyTemplate:    emailShell(body, ""),
	}
}

func passwordResetTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Reset your password</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Hi {{.UserName}},</p>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">We received a request to reset the password on your {{.TenantName}} account. Click the button below to choose a new one.</p>
` + ctaButton("Reset password", "{{.ResetURL}}") + `
<p style="margin:0 0 16px 0;font-size:13px;line-height:1.5;color:#6B7280;word-break:break-all;">Or copy and paste this link: <a href="{{.ResetURL}}" style="color:{{.BrandColor}};text-decoration:underline;">{{.ResetURL}}</a></p>
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;font-style:italic;">This link expires in 1 hour. If you didn't request a password reset, you can safely ignore this email — your password won't change.</p>`
	return DefaultTemplate{
		Type:            models.TypePasswordReset,
		Channel:         models.ChannelEmail,
		Name:            "Password reset — starter",
		SubjectTemplate: "Reset your {{.TenantName}} password",
		BodyTemplate:    emailShell(body, ""),
	}
}

// password_changed is not in the models.NotificationType set today; we install
// it as a "custom" type with a recognisable name so admins can rename/repoint
// it if/when the type is added to the consumer. Falling back to "custom" keeps
// validateCreate happy.
func passwordChangedTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Your password was changed</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Hi {{.UserName}},</p>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">This is a confirmation that the password for your {{.TenantName}} account was successfully changed.</p>
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:14px;line-height:1.5;color:#1F2933;"><strong>Wasn't you?</strong> Please <a href="{{.FrontendBaseURL}}/account/security" style="color:{{.BrandColor}};text-decoration:underline;">secure your account</a> or contact support immediately.</p>`
	return DefaultTemplate{
		Type:            models.NotificationType("password_changed"),
		Channel:         models.ChannelEmail,
		Name:            "Password changed — starter",
		SubjectTemplate: "Your {{.TenantName}} password was changed",
		BodyTemplate:    emailShell(body, ""),
	}
}

func orderConfirmationTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Thanks, {{.CustomerName}}!</h1>
<p style="margin:0 0 8px 0;font-size:16px;line-height:1.6;color:#1F2933;">Your order <strong>{{.OrderID}}</strong> is confirmed. We'll let you know as soon as it ships.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:20px 0;border-collapse:collapse;">
  <thead>
    <tr>
      <th align="left" style="padding:10px 8px;border-bottom:2px solid #E5E7EB;font-size:13px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Item</th>
      <th align="center" style="padding:10px 8px;border-bottom:2px solid #E5E7EB;font-size:13px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Qty</th>
      <th align="right" style="padding:10px 8px;border-bottom:2px solid #E5E7EB;font-size:13px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Subtotal</th>
    </tr>
  </thead>
  <tbody>
    {{range .Items}}
    <tr>
      <td align="left" style="padding:10px 8px;border-bottom:1px solid #F1F3F5;font-size:14px;color:#1F2933;">{{.Name}}</td>
      <td align="center" style="padding:10px 8px;border-bottom:1px solid #F1F3F5;font-size:14px;color:#1F2933;">{{.Quantity}}</td>
      <td align="right" style="padding:10px 8px;border-bottom:1px solid #F1F3F5;font-size:14px;color:#1F2933;">{{.Subtotal}}</td>
    </tr>
    {{end}}
    <tr>
      <td align="left" colspan="2" style="padding:14px 8px;font-size:15px;font-weight:600;color:#1F2933;">Total</td>
      <td align="right" style="padding:14px 8px;font-size:15px;font-weight:700;color:#1F2933;">{{.Total}}</td>
    </tr>
  </tbody>
</table>
` + ctaButton("Track your order", "{{.FrontendBaseURL}}/account/orders/{{.OrderID}}") + `
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;">Expected delivery: 3–5 business days. We'll send a separate email when your parcel is on its way.</p>`
	return DefaultTemplate{
		Type:            models.TypeOrderConfirmation,
		Channel:         models.ChannelEmail,
		Name:            "Order confirmation — starter",
		SubjectTemplate: "Order {{.OrderID}} confirmed — thanks, {{.CustomerName}}",
		BodyTemplate:    emailShell(body, ""),
	}
}

func orderShippedTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Your order is on its way</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Hi {{.CustomerName}}, good news — order <strong>{{.OrderID}}</strong> has just shipped.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0;background-color:#F4F5F7;border-radius:8px;">
  <tr>
    <td style="padding:16px 20px;">
      <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">Carrier</p>
      <p style="margin:2px 0 12px 0;font-size:16px;font-weight:600;color:#1F2933;">{{.Carrier}}</p>
      <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">Tracking number</p>
      <p style="margin:2px 0 0 0;font-size:16px;font-weight:600;color:#1F2933;">{{.TrackingNumber}}</p>
    </td>
  </tr>
</table>
` + ctaButton("Track shipment", "{{.FrontendBaseURL}}/account/orders/{{.OrderID}}") + `
<p style="margin:16px 0 0 0;font-size:14px;line-height:1.6;color:#6B7280;"><strong>What happens next?</strong> Your parcel is with the carrier and should arrive within 3–5 business days. We'll let you know once it's delivered.</p>`
	return DefaultTemplate{
		Type:            models.TypeOrderShipped,
		Channel:         models.ChannelEmail,
		Name:            "Order shipped — starter",
		SubjectTemplate: "Your order {{.OrderID}} is on its way",
		BodyTemplate:    emailShell(body, ""),
	}
}

func orderCancelledTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Order {{.OrderID}} cancelled</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Hi {{.CustomerName}}, your order <strong>{{.OrderID}}</strong> has been cancelled.</p>
{{if .Reason}}<p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#6B7280;"><strong>Reason:</strong> {{.Reason}}</p>{{end}}
{{if .RefundAmount}}<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0;background-color:#F4F5F7;border-radius:8px;">
  <tr><td style="padding:16px 20px;">
    <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">Refund</p>
    <p style="margin:4px 0 0 0;font-size:18px;font-weight:700;color:#1F2933;">{{.RefundAmount}}</p>
    <p style="margin:6px 0 0 0;font-size:13px;color:#6B7280;">Refunds typically reach your account in 5–10 business days.</p>
  </td></tr>
</table>{{end}}
` + ctaButton("Shop again", "{{.FrontendBaseURL}}") + `
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;">Questions? Reach our support team at <a href="mailto:support@{{.TenantName}}.com" style="color:{{.BrandColor}};text-decoration:underline;">support@{{.TenantName}}.com</a>.</p>`
	return DefaultTemplate{
		Type:            models.TypeOrderCancelled,
		Channel:         models.ChannelEmail,
		Name:            "Order cancelled — starter",
		SubjectTemplate: "Order {{.OrderID}} cancelled",
		BodyTemplate:    emailShell(body, ""),
	}
}

func paymentConfirmedTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Payment received</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Thanks, {{.CustomerName}} — we've successfully received your payment for order <strong>{{.OrderID}}</strong>.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0;background-color:#F4F5F7;border-radius:8px;">
  <tr><td style="padding:16px 20px;">
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="left" style="padding:4px 0;font-size:13px;color:#6B7280;">Order</td>
        <td align="right" style="padding:4px 0;font-size:13px;color:#1F2933;font-weight:600;">{{.OrderID}}</td>
      </tr>
      <tr>
        <td align="left" style="padding:4px 0;font-size:13px;color:#6B7280;">Payment method</td>
        <td align="right" style="padding:4px 0;font-size:13px;color:#1F2933;font-weight:600;">{{.PaymentMethod}}</td>
      </tr>
      <tr>
        <td align="left" style="padding:8px 0 4px 0;font-size:14px;color:#1F2933;font-weight:600;border-top:1px solid #E5E7EB;">Amount paid</td>
        <td align="right" style="padding:8px 0 4px 0;font-size:16px;color:#1F2933;font-weight:700;border-top:1px solid #E5E7EB;">{{.Total}}</td>
      </tr>
    </table>
  </td></tr>
</table>
` + ctaButton("View order", "{{.FrontendBaseURL}}/account/orders/{{.OrderID}}") + ``
	return DefaultTemplate{
		Type:            models.TypePaymentConfirmed,
		Channel:         models.ChannelEmail,
		Name:            "Payment confirmed — starter",
		SubjectTemplate: "Payment received — order {{.OrderID}}",
		BodyTemplate:    emailShell(body, ""),
	}
}

func paymentFailedTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Payment didn't go through</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Hi {{.CustomerName}}, we couldn't process the payment for order <strong>{{.OrderID}}</strong>.</p>
<p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#6B7280;">Common reasons:</p>
<ul style="margin:0 0 16px 20px;padding:0;font-size:14px;line-height:1.6;color:#1F2933;">
  <li>Insufficient balance on the card or mobile wallet</li>
  <li>Incorrect card details or expired card</li>
  <li>Network timeout — try again in a few minutes</li>
</ul>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">Don't worry — no money has been deducted. You can retry checkout using a different payment method.</p>
` + ctaButton("Retry checkout", "{{.FrontendBaseURL}}/checkout") + `
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;">Still stuck? Email <a href="mailto:support@{{.TenantName}}.com" style="color:{{.BrandColor}};text-decoration:underline;">support@{{.TenantName}}.com</a> and we'll help you out.</p>`
	return DefaultTemplate{
		Type:            models.TypePaymentFailed,
		Channel:         models.ChannelEmail,
		Name:            "Payment failed — starter",
		SubjectTemplate: "Payment failed for order {{.OrderID}}",
		BodyTemplate:    emailShell(body, ""),
	}
}

func receiptTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Your receipt</h1>
<p style="margin:0 0 8px 0;font-size:16px;line-height:1.6;color:#1F2933;">Thank you for shopping with {{.TenantName}}. Here's a summary of your purchase.</p>
<p style="margin:0 0 16px 0;font-size:13px;color:#6B7280;">Order <strong>{{.OrderID}}</strong></p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 16px 0;border-collapse:collapse;">
  <thead>
    <tr>
      <th align="left" style="padding:8px 6px;border-bottom:2px solid #E5E7EB;font-size:12px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Item</th>
      <th align="center" style="padding:8px 6px;border-bottom:2px solid #E5E7EB;font-size:12px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Qty</th>
      <th align="right" style="padding:8px 6px;border-bottom:2px solid #E5E7EB;font-size:12px;color:#6B7280;text-transform:uppercase;letter-spacing:0.4px;">Subtotal</th>
    </tr>
  </thead>
  <tbody>
    {{range .Items}}
    <tr>
      <td align="left" style="padding:8px 6px;border-bottom:1px solid #F1F3F5;font-size:13px;color:#1F2933;">{{.Name}}</td>
      <td align="center" style="padding:8px 6px;border-bottom:1px solid #F1F3F5;font-size:13px;color:#1F2933;">{{.Quantity}}</td>
      <td align="right" style="padding:8px 6px;border-bottom:1px solid #F1F3F5;font-size:13px;color:#1F2933;">{{.Subtotal}}</td>
    </tr>
    {{end}}
    <tr>
      <td align="left" colspan="2" style="padding:10px 6px;font-size:13px;color:#6B7280;">Payment method</td>
      <td align="right" style="padding:10px 6px;font-size:13px;color:#1F2933;">{{.PaymentMethod}}</td>
    </tr>
    <tr>
      <td align="left" colspan="2" style="padding:10px 6px;font-size:15px;font-weight:600;color:#1F2933;border-top:1px solid #E5E7EB;">Total</td>
      <td align="right" style="padding:10px 6px;font-size:16px;font-weight:700;color:#1F2933;border-top:1px solid #E5E7EB;">{{.Total}}</td>
    </tr>
  </tbody>
</table>
<p style="margin:16px 0 0 0;padding-top:16px;border-top:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#6B7280;">Thank you for shopping with {{.TenantName}}. Keep this receipt for your records.</p>`
	return DefaultTemplate{
		Type:            models.TypeReceipt,
		Channel:         models.ChannelEmail,
		Name:            "Receipt — starter",
		SubjectTemplate: "Your receipt from {{.TenantName}}",
		BodyTemplate:    emailShell(body, ""),
	}
}

// stock_alert targets admins, so the copy is plain and direct. We still use
// the shell so the visual style stays consistent with the rest of the suite.
func stockAlertTemplate() DefaultTemplate {
	body := `<h1 class="email-title" style="margin:0 0 16px 0;font-size:26px;line-height:1.25;font-weight:700;color:#1F2933;">Low stock alert</h1>
<p style="margin:0 0 16px 0;font-size:16px;line-height:1.6;color:#1F2933;">A product in your catalog is running low.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0;background-color:#F4F5F7;border-radius:8px;">
  <tr><td style="padding:16px 20px;">
    <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">Product</p>
    <p style="margin:2px 0 12px 0;font-size:16px;font-weight:600;color:#1F2933;">{{.ProductName}}</p>
    <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">SKU</p>
    <p style="margin:2px 0 12px 0;font-size:14px;color:#1F2933;font-family:monospace;">{{.SKU}}</p>
    <p style="margin:0;font-size:12px;text-transform:uppercase;letter-spacing:0.4px;color:#6B7280;">Current quantity</p>
    <p style="margin:2px 0 0 0;font-size:18px;font-weight:700;color:#1F2933;">{{.CurrentQuantity}}</p>
  </td></tr>
</table>
` + ctaButton("Open inventory", "{{.FrontendBaseURL}}/admin/inventory") + `
<p style="margin:16px 0 0 0;font-size:13px;line-height:1.5;color:#6B7280;">Restock soon to avoid out-of-stock listings on your storefront.</p>`
	return DefaultTemplate{
		Type:            models.TypeStockAlert,
		Channel:         models.ChannelEmail,
		Name:            "Low-stock alert — starter",
		SubjectTemplate: "Low stock alert — {{.SKU}}",
		BodyTemplate:    emailShell(body, "This is an admin notification.<br>"),
	}
}
