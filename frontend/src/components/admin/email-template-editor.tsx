'use client';

/**
 * EmailTemplateEditor renders the two-column edit layout used by both the
 * "new template" and "edit template" admin pages.
 *
 * Left column: form fields (name, type, channel, subject, body) plus an
 * "available variables" panel that updates as the type changes.
 *
 * Right column: a sandboxed iframe that re-renders the body HTML whenever it
 * changes (debounced via React's render cycle). The sandbox attribute keeps
 * admin HTML from breaking out of the admin shell.
 *
 * Test-send section: at the bottom of the form. Only enabled for saved
 * templates because the backend requires an ID to render with sample data.
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import { ArrowLeft, Loader2, Save, Send } from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import {
  notificationTemplateApi,
  type CreateNotificationTemplateRequest,
  type DefaultTemplate,
  type NotificationChannel,
  type NotificationTemplate,
  type NotificationTemplateType,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';

// Known template types with human-readable labels. Keep in sync with the
// backend's models.NotificationType constants.
export const TEMPLATE_TYPES: { value: NotificationTemplateType; label: string }[] = [
  { value: 'order_confirmation', label: 'Order confirmation' },
  { value: 'order_shipped', label: 'Order shipped' },
  { value: 'order_delivered', label: 'Order delivered' },
  { value: 'order_cancelled', label: 'Order cancelled' },
  { value: 'payment_confirmed', label: 'Payment confirmed' },
  { value: 'payment_failed', label: 'Payment failed' },
  { value: 'welcome', label: 'Welcome' },
  { value: 'email_verification', label: 'Email verification' },
  { value: 'password_reset', label: 'Password reset' },
  { value: 'receipt', label: 'Receipt' },
  { value: 'stock_alert', label: 'Low-stock alert (admin)' },
  { value: 'promotion', label: 'Promotion' },
  { value: 'custom', label: 'Custom' },
];

// Variables surfaced to admins for each type. The list matches what the
// consumer injects at runtime so admins can author against the same names.
const VARIABLES_FOR_TYPE: Record<string, string[]> = {
  order_confirmation: ['{{.OrderID}}', '{{.CustomerName}}', '{{.Total}}', '{{.Items}}'],
  order_shipped: ['{{.OrderID}}', '{{.TrackingNumber}}', '{{.Carrier}}'],
  order_delivered: ['{{.OrderID}}', '{{.CustomerName}}'],
  order_cancelled: ['{{.OrderID}}', '{{.Reason}}'],
  payment_confirmed: ['{{.OrderID}}', '{{.Total}}'],
  payment_failed: ['{{.OrderID}}'],
  welcome: ['{{.UserName}}'],
  email_verification: ['{{.VerifyURL}}', '{{.UserName}}'],
  password_reset: ['{{.ResetURL}}', '{{.UserName}}'],
  receipt: ['{{.OrderID}}', '{{.Items}}', '{{.Total}}', '{{.PaymentMethod}}'],
  stock_alert: ['{{.ProductName}}', '{{.SKU}}', '{{.CurrentQuantity}}'],
  promotion: ['{{.PromotionName}}', '{{.DiscountValue}}', '{{.ExpiresAt}}'],
  custom: [],
};

const ALWAYS_AVAILABLE = ['{{.TenantName}}', '{{.BrandColor}}', '{{.FrontendBaseURL}}'];

interface EmailTemplateEditorProps {
  // Existing template when editing, undefined for new templates.
  initial?: NotificationTemplate;
  // Called after a successful save. Pages typically push back to the list.
  onSaved?: (t: NotificationTemplate) => void;
}

export default function EmailTemplateEditor({ initial, onSaved }: EmailTemplateEditorProps) {
  const router = useRouter();
  const { tenantId, token } = useAuthStore();

  const [name, setName] = useState(initial?.name || '');
  const [type, setType] = useState<string>(initial?.type || 'order_confirmation');
  const [channel, setChannel] = useState<NotificationChannel>(initial?.channel || 'email');
  const [subjectTemplate, setSubjectTemplate] = useState(initial?.subject_template || '');
  const [bodyTemplate, setBodyTemplate] = useState(initial?.body_template || '');
  const [isActive, setIsActive] = useState(initial?.is_active ?? true);

  const [previewSubject, setPreviewSubject] = useState('');
  const [previewBody, setPreviewBody] = useState('');
  const [previewError, setPreviewError] = useState('');

  const [saving, setSaving] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [testSending, setTestSending] = useState(false);

  const iframeRef = useRef<HTMLIFrameElement>(null);
  const templateId = initial?.id;

  // Starter-pack catalogue used by the "Start from template" picker. We only
  // fetch it when authoring a brand-new template — editing an existing one
  // already has its own subject/body to work from.
  const [defaults, setDefaults] = useState<DefaultTemplate[]>([]);
  const [defaultsLoaded, setDefaultsLoaded] = useState(false);
  useEffect(() => {
    if (initial || !tenantId || !token) return;
    let cancelled = false;
    notificationTemplateApi
      .listDefaults(tenantId, token)
      .then((d) => {
        if (!cancelled) {
          setDefaults(d);
          setDefaultsLoaded(true);
        }
      })
      .catch(() => {
        // Non-fatal — the picker just won't appear. Admins can still author
        // a template from scratch.
        if (!cancelled) setDefaultsLoaded(true);
      });
    return () => {
      cancelled = true;
    };
  }, [initial, tenantId, token]);

  // applyDefault pre-fills the form from a starter-pack entry. We use the
  // type as the lookup key (one default per type) so the dropdown stays
  // simple. Channel is taken from the default too — every starter is email
  // today but that keeps the code future-proof.
  function applyDefault(typeKey: string) {
    if (!typeKey) return;
    const d = defaults.find((x) => x.type === typeKey);
    if (!d) return;
    setType(d.type);
    setChannel(d.channel as NotificationChannel);
    setSubjectTemplate(d.subject_template);
    setBodyTemplate(d.body_template);
    if (!name.trim()) {
      // Strip the " — starter" suffix so the admin's copy doesn't read like
      // boilerplate. They can rename further before saving.
      setName(d.name.replace(/\s*—\s*starter\s*$/i, ''));
    }
    toast.success(`Loaded "${d.name}"`);
  }

  // Render the preview locally when no saved template exists; once saved, we
  // hit the backend's /preview endpoint so admins see exactly what the
  // consumer will produce (including missingkey=zero behavior).
  useEffect(() => {
    let cancelled = false;
    async function run() {
      setPreviewError('');
      if (templateId && tenantId && token) {
        try {
          const res = await notificationTemplateApi.preview(templateId, {}, tenantId, token);
          if (!cancelled) {
            setPreviewSubject(res.subject);
            setPreviewBody(res.body);
          }
        } catch (err) {
          if (!cancelled) {
            // Fall back to a literal preview so admins still see *something*.
            setPreviewSubject(subjectTemplate);
            setPreviewBody(bodyTemplate);
            setPreviewError(err instanceof Error ? err.message : 'Preview failed');
          }
        }
      } else {
        setPreviewSubject(subjectTemplate);
        setPreviewBody(bodyTemplate);
      }
    }
    run();
    return () => {
      cancelled = true;
    };
    // Re-run whenever the saved version (templateId) changes; for unsaved
    // templates we also want live updates as the admin types.
  }, [templateId, tenantId, token, subjectTemplate, bodyTemplate]);

  // Write the rendered body into the iframe via srcdoc. The sandbox attribute
  // blocks scripts and same-origin escapes so admin HTML can't break out.
  useEffect(() => {
    const frame = iframeRef.current;
    if (!frame) return;
    frame.srcdoc = previewBody || '<p style="padding:24px;color:#9ca3af;font-family:sans-serif">Preview will appear here…</p>';
  }, [previewBody]);

  const variables = useMemo(() => {
    return [...(VARIABLES_FOR_TYPE[type] || []), ...ALWAYS_AVAILABLE];
  }, [type]);

  async function handleSave() {
    if (!tenantId || !token) {
      toast.error('Sign in required');
      return;
    }
    if (!name.trim()) {
      toast.error('Name is required');
      return;
    }
    if (!bodyTemplate.trim()) {
      toast.error('Body template is required');
      return;
    }

    const payload: CreateNotificationTemplateRequest = {
      type,
      channel,
      name: name.trim(),
      subject_template: subjectTemplate,
      body_template: bodyTemplate,
      is_active: isActive,
    };

    setSaving(true);
    try {
      const saved = templateId
        ? await notificationTemplateApi.update(templateId, payload, tenantId, token)
        : await notificationTemplateApi.create(payload, tenantId, token);
      toast.success(templateId ? 'Template updated' : 'Template created');
      if (onSaved) {
        onSaved(saved);
      } else {
        router.push('/admin/email-templates');
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  }

  async function handleTestSend() {
    if (!templateId) {
      toast.error('Save the template first before sending a test');
      return;
    }
    if (!testEmail.trim()) {
      toast.error('Test email address is required');
      return;
    }
    if (!tenantId || !token) {
      toast.error('Sign in required');
      return;
    }
    setTestSending(true);
    try {
      await notificationTemplateApi.testSend(templateId, testEmail.trim(), tenantId, token);
      toast.success(`Test email sent to ${testEmail.trim()}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Test send failed');
    } finally {
      setTestSending(false);
    }
  }

  function insertVariable(v: string) {
    setBodyTemplate((prev) => prev + (prev.endsWith(' ') || prev === '' ? '' : ' ') + v);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Link
            href="/admin/email-templates"
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
            aria-label="Back to templates"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-text">
              {templateId ? 'Edit template' : 'New template'}
            </h1>
            <p className="text-sm text-text-secondary">
              Templates override the built-in HTML used by the notification service.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Link
            href="/admin/email-templates"
            className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
          >
            Cancel
          </Link>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>

      {/*
        "Start from template" picker — only on the new-template page. We
        deliberately keep this above the editor grid so admins see it before
        they start typing, and it's hidden once defaults aren't available so
        the layout doesn't reflow on an empty list.
      */}
      {!initial && defaultsLoaded && defaults.length > 0 && (
        <div className="rounded-xl border border-border bg-surface p-4 shadow-sm">
          <label className="mb-1 block text-sm font-medium text-text" htmlFor="starter-picker">
            Start from a starter template
          </label>
          <p className="mb-2 text-xs text-text-muted">
            Pre-fill the editor with one of our pre-designed templates — then
            tweak the copy to match your brand.
          </p>
          <select
            id="starter-picker"
            defaultValue=""
            onChange={(e) => {
              applyDefault(e.target.value);
              // Reset to placeholder so picking the same entry twice triggers
              // a fresh apply.
              e.target.value = '';
            }}
            className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">Choose a starter…</option>
            {defaults.map((d) => (
              <option key={`${d.type}-${d.channel}`} value={d.type}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Left: editor */}
        <div className="space-y-5 rounded-xl border border-border bg-surface p-5 shadow-sm">
          <div>
            <label className="mb-1 block text-sm font-medium text-text">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Order confirmation – default"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-sm font-medium text-text">Type</label>
              <select
                value={type}
                onChange={(e) => setType(e.target.value)}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                {TEMPLATE_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-text">Channel</label>
              <div className="flex gap-3">
                {(['email', 'sms', 'push'] as NotificationChannel[]).map((ch) => (
                  <label
                    key={ch}
                    className="inline-flex items-center gap-1.5 text-sm text-text-secondary"
                  >
                    <input
                      type="radio"
                      name="channel"
                      value={ch}
                      checked={channel === ch}
                      onChange={() => setChannel(ch)}
                      className="text-primary focus:ring-primary"
                    />
                    <span className="capitalize">{ch}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-text">Subject template</label>
            <input
              type="text"
              value={subjectTemplate}
              onChange={(e) => setSubjectTemplate(e.target.value)}
              placeholder="e.g. Your order {{.OrderID}} is confirmed"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 font-mono text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <p className="mt-1 text-xs text-text-muted">
              Supports Go text/template syntax — e.g. <code>{'{{.OrderID}}'}</code>.
            </p>
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-text">Body template (HTML)</label>
            <textarea
              value={bodyTemplate}
              onChange={(e) => setBodyTemplate(e.target.value)}
              placeholder={'<!DOCTYPE html>\n<html><body>\n  <h1>Order {{.OrderID}} confirmed</h1>\n  <p>Thanks {{.CustomerName}}!</p>\n</body></html>'}
              spellCheck={false}
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 font-mono text-xs leading-relaxed text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              style={{ minHeight: 320 }}
            />
          </div>

          <div>
            <label className="inline-flex items-center gap-2 text-sm font-medium text-text">
              <input
                type="checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="rounded border-border text-primary focus:ring-primary"
              />
              Active (override the hardcoded HTML at send time)
            </label>
          </div>

          <div className="rounded-lg border border-border bg-surface-hover/50 p-3">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
              Available variables
            </p>
            <div className="flex flex-wrap gap-1.5">
              {variables.map((v) => (
                <button
                  key={v}
                  type="button"
                  onClick={() => insertVariable(v)}
                  className="rounded-md border border-border bg-surface px-2 py-1 font-mono text-[11px] text-text-secondary transition-colors hover:bg-surface hover:text-primary"
                  title="Click to insert into body"
                >
                  {v}
                </button>
              ))}
            </div>
            <p className="mt-2 text-xs text-text-muted">
              Click a chip to append it to the body template.
            </p>
          </div>
        </div>

        {/* Right: preview */}
        <div className="space-y-3 rounded-xl border border-border bg-surface p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-text">Live preview</h2>
            {previewError && (
              <span className="text-xs text-red-600" title={previewError}>
                Preview error
              </span>
            )}
          </div>
          <div className="rounded-lg border border-border bg-surface-hover/30 px-3 py-2">
            <p className="text-[10px] uppercase tracking-wider text-text-muted">Subject</p>
            <p className="text-sm font-medium text-text">{previewSubject || <span className="text-text-muted">(no subject)</span>}</p>
          </div>
          <iframe
            ref={iframeRef}
            title="Email preview"
            sandbox=""
            // Sandbox without "allow-scripts" / "allow-same-origin" blocks any
            // breakout the admin's HTML might attempt. We deliberately accept
            // raw HTML in the body field because admin authors are trusted.
            className="h-[480px] w-full rounded-lg border border-border bg-white"
          />
        </div>
      </div>

      {/* Test send */}
      <div className="rounded-xl border border-border bg-surface p-5 shadow-sm">
        <h2 className="mb-3 text-sm font-semibold text-text">Send a test email</h2>
        {!templateId ? (
          <p className="text-sm text-text-muted">Save the template first to enable test sending.</p>
        ) : (
          <div className="flex flex-wrap items-center gap-3">
            <input
              type="email"
              placeholder="test@yourdomain.com"
              value={testEmail}
              onChange={(e) => setTestEmail(e.target.value)}
              className="min-w-[260px] flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <button
              type="button"
              onClick={handleTestSend}
              disabled={testSending}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark disabled:opacity-50"
            >
              {testSending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              {testSending ? 'Sending…' : 'Send test'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
