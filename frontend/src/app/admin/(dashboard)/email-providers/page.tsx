'use client';

/**
 * Admin page for configuring email delivery.
 *
 * Providers form an ordered chain: the lowest priority carries normal traffic
 * and the rest are fallbacks, tried in order when a send fails. A tenant that
 * configures any provider of its own uses exactly those; a tenant that
 * configures none inherits the platform default, shown here read-only so the
 * precedence is visible rather than mysterious.
 *
 * Secrets are write-only. The backend never returns a stored credential, only
 * whether one is set plus a hint derived from the ciphertext — so the "Secret"
 * field is always blank on load, and leaving it blank on save keeps the
 * existing key.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Mail,
  Plus,
  Loader2,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Send,
  ShieldAlert,
  ShieldCheck,
  Info,
  KeyRound,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { emailProviderApi, type EmailProviderConfig } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';
import { useConfirm } from '@/stores/confirm';

/** Vendors whose relay coordinates the backend already knows, so only
 *  credentials are needed. Mirrors service.smtpPresets. */
const PRESET_HINTS: Record<string, { label: string; userLabel: string; secretLabel: string; note?: string }> = {
  mailjet: {
    label: 'Mailjet',
    userLabel: 'API Key',
    secretLabel: 'Secret Key',
    note: 'Mailjet uses your API Key as the SMTP username and the Secret Key as the password. The sender address must be validated in Mailjet first.',
  },
  brevo: { label: 'Brevo', userLabel: 'SMTP login', secretLabel: 'SMTP key' },
  resend: { label: 'Resend', userLabel: 'Username (usually "resend")', secretLabel: 'API key' },
  smtp2go: { label: 'SMTP2GO', userLabel: 'SMTP username', secretLabel: 'SMTP password' },
  elasticemail: { label: 'Elastic Email', userLabel: 'Username', secretLabel: 'API key' },
  zeptomail: { label: 'ZeptoMail', userLabel: 'SMTP username', secretLabel: 'SMTP password' },
  mailgun: { label: 'Mailgun', userLabel: 'SMTP login', secretLabel: 'SMTP password' },
  postmark: { label: 'Postmark', userLabel: 'Server token', secretLabel: 'Server token' },
  ses: { label: 'Amazon SES', userLabel: 'SMTP username', secretLabel: 'SMTP password', note: 'SES is region-scoped — set the Host to your region\'s endpoint, e.g. email-smtp.ap-southeast-1.amazonaws.com' },
  sendgrid: { label: 'SendGrid', userLabel: 'unused', secretLabel: 'API key', note: 'SendGrid uses its REST API, so no username or host is needed.' },
  smtp: { label: 'Custom SMTP', userLabel: 'SMTP username', secretLabel: 'SMTP password', note: 'Host is required for a custom relay.' },
  simulated: { label: 'Simulated (logs only)', userLabel: '', secretLabel: '', note: 'Delivers nothing — it logs the message. Useful as the last entry in a chain so an outage is recorded rather than lost. Remove it in production.' },
};

function providerLabel(key: string): string {
  return PRESET_HINTS[key]?.label ?? key;
}

/** Providers that need neither username nor host. */
function isCredentialOnly(provider: string): boolean {
  return provider === 'sendgrid';
}

function needsNothing(provider: string): boolean {
  return provider === 'simulated';
}

interface DraftState {
  enabled: boolean;
  priority: number;
  host: string;
  port: string;
  username: string;
  secret: string;
  from_email: string;
  from_name: string;
}

function draftFrom(cfg: EmailProviderConfig): DraftState {
  return {
    enabled: cfg.enabled,
    priority: cfg.priority,
    host: cfg.host ?? '',
    port: cfg.port ? String(cfg.port) : '',
    username: cfg.username ?? '',
    // Always blank: the backend does not return stored secrets.
    secret: '',
    from_email: cfg.from_email ?? '',
    from_name: cfg.from_name ?? '',
  };
}

export default function EmailProvidersPage() {
  const { tenantId, token, user } = useAuthStore();
  const confirm = useConfirm();

  const isSuperAdmin = user?.role === 'super_admin';
  // A super_admin has no tenant of its own, so tenant scope would just return
  // "tenant_required". Land it on the platform default instead.
  const [scope, setScope] = useState<'tenant' | 'platform'>(isSuperAdmin ? 'platform' : 'tenant');
  const apiScope = scope === 'platform' ? ('platform' as const) : undefined;

  const [configs, setConfigs] = useState<EmailProviderConfig[]>([]);
  const [available, setAvailable] = useState<string[]>([]);
  const [inherited, setInherited] = useState(false);
  const [encrypted, setEncrypted] = useState(true);
  const [loading, setLoading] = useState(true);

  const [drafts, setDrafts] = useState<Record<string, DraftState>>({});
  const [saving, setSaving] = useState<string | null>(null);
  const [testing, setTesting] = useState<string | null>(null);
  const [testTo, setTestTo] = useState('');
  const [addProvider, setAddProvider] = useState('');

  const load = useCallback(async () => {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await emailProviderApi.list(tenantId, token, apiScope);
      setConfigs(res.data);
      setAvailable(res.available);
      setInherited(res.inherited);
      setEncrypted(res.encrypted_at_rest);
      setDrafts(Object.fromEntries(res.data.map((c) => [c.provider, draftFrom(c)])));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load email providers');
    } finally {
      setLoading(false);
    }
  }, [tenantId, token, apiScope]);

  useEffect(() => {
    void load();
  }, [load]);

  const unconfigured = useMemo(
    () => available.filter((p) => !configs.some((c) => c.provider === p)).sort(),
    [available, configs],
  );

  function patchDraft(provider: string, patch: Partial<DraftState>) {
    setDrafts((prev) => ({ ...prev, [provider]: { ...prev[provider], ...patch } }));
  }

  async function save(provider: string) {
    if (!tenantId || !token) return;
    const draft = drafts[provider];
    if (!draft) return;

    setSaving(provider);
    try {
      await emailProviderApi.upsert(
        {
          provider,
          enabled: draft.enabled,
          priority: draft.priority,
          host: draft.host || undefined,
          port: draft.port ? Number(draft.port) : undefined,
          username: draft.username || undefined,
          // Blank means "keep the stored credential".
          secret: draft.secret || undefined,
          from_email: draft.from_email || undefined,
          from_name: draft.from_name || undefined,
        },
        tenantId,
        token,
        apiScope,
      );
      toast.success(`${providerLabel(provider)} saved`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save provider');
    } finally {
      setSaving(null);
    }
  }

  async function add() {
    if (!addProvider || !tenantId || !token) return;
    setSaving(addProvider);
    try {
      await emailProviderApi.upsert(
        { provider: addProvider, enabled: false, priority: configs.length + 1 },
        tenantId,
        token,
        apiScope,
      );
      toast.success(`${providerLabel(addProvider)} added — add credentials, then enable it`);
      setAddProvider('');
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to add provider');
    } finally {
      setSaving(null);
    }
  }

  async function remove(provider: string) {
    if (!tenantId || !token) return;
    const ok = await confirm({
      title: `Remove ${providerLabel(provider)}?`,
      description: 'Its stored credentials will be deleted. This cannot be undone.',
      variant: 'danger',
      confirmLabel: 'Remove',
    });
    if (!ok) return;

    try {
      await emailProviderApi.remove(provider, tenantId, token, apiScope);
      toast.success(`${providerLabel(provider)} removed`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to remove provider');
    }
  }

  async function test(provider: string) {
    if (!tenantId || !token) return;
    if (!testTo.trim()) {
      toast.error('Enter an address to send the test to');
      return;
    }
    setTesting(provider);
    try {
      const res = await emailProviderApi.test(provider, testTo.trim(), tenantId, token, apiScope);
      // "Accepted", not "sent": all we know is that the relay completed the
      // SMTP transaction. Vendors apply sender validation and policy
      // asynchronously, so an accepted message can still be dropped after the
      // fact — most often because the From address is not a validated sender.
      // Claiming delivery here sends operators hunting for a bug in us.
      toast.success(
        `${res.provider} accepted the message for ${res.sent_to}. Check the provider's own delivery log to confirm it arrived.`,
      );
    } catch (err) {
      // The backend tests one provider deliberately, so this is the real
      // vendor response rather than a fallback masking the failure.
      toast.error(err instanceof Error ? err.message : 'Test failed');
    } finally {
      setTesting(null);
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold text-gray-900">
            <Mail className="h-6 w-6 text-primary" />
            Email providers
          </h1>
          <p className="mt-1 max-w-2xl text-sm text-gray-500">
            Providers are tried in priority order — the lowest number carries normal traffic, the
            rest are fallbacks used when a send fails.
          </p>
        </div>

        {isSuperAdmin && (
          <div className="flex rounded-lg border border-gray-200 p-1">
            {(['tenant', 'platform'] as const).map((s) => (
              <button
                key={s}
                onClick={() => setScope(s)}
                className={cn(
                  'rounded-md px-3 py-1.5 text-sm font-medium transition',
                  scope === s ? 'bg-primary text-white' : 'text-gray-600 hover:bg-gray-50',
                )}
              >
                {s === 'tenant' ? 'This store' : 'Platform default'}
              </button>
            ))}
          </div>
        )}
      </header>

      {!encrypted && (
        <div className="flex items-start gap-3 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">
          <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0" />
          <div>
            <p className="font-medium">Credentials are not encrypted at rest</p>
            <p className="mt-1">
              <code className="rounded bg-amber-100 px-1">EMAIL_ENCRYPTION_KEY</code> is not set on
              notification-service, so secrets are only base64-encoded. Set a 32-byte key before
              storing production credentials.
            </p>
          </div>
        </div>
      )}

      {encrypted && (
        <div className="flex items-center gap-2 text-xs text-gray-500">
          <ShieldCheck className="h-4 w-4 text-green-600" />
          Credentials are AES-GCM encrypted at rest and are never sent back to this page.
        </div>
      )}

      {inherited && (
        <div className="flex items-start gap-3 rounded-lg border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
          <Info className="mt-0.5 h-5 w-5 shrink-0" />
          <div>
            <p className="font-medium">Using the platform default</p>
            <p className="mt-1">
              This store has no providers of its own, so it inherits the platform configuration
              below. Add a provider here to override it — your own chain replaces the platform&apos;s
              entirely rather than merging with it.
            </p>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16 text-gray-400">
          <Loader2 className="h-6 w-6 animate-spin" />
        </div>
      ) : (
        <>
          {configs.length === 0 && (
            <div className="rounded-lg border border-dashed border-gray-300 p-10 text-center">
              <Mail className="mx-auto h-8 w-8 text-gray-300" />
              <p className="mt-3 text-sm font-medium text-gray-900">No providers configured</p>
              <p className="mt-1 text-sm text-gray-500">
                Add one below. Until then, delivery falls back to whatever{' '}
                <code className="rounded bg-gray-100 px-1">EMAIL_PROVIDERS</code> is set to.
              </p>
            </div>
          )}

          <div className="space-y-4">
            {configs.map((cfg) => {
              const draft = drafts[cfg.provider];
              const hint = PRESET_HINTS[cfg.provider];
              const readOnly = cfg.inherited;
              if (!draft) return null;

              return (
                <section
                  key={cfg.provider}
                  className={cn(
                    'rounded-lg border bg-white p-5',
                    cfg.enabled ? 'border-gray-200' : 'border-gray-200 bg-gray-50/60',
                  )}
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <h2 className="flex items-center gap-2 font-medium text-gray-900">
                        {providerLabel(cfg.provider)}
                        <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-normal text-gray-600">
                          priority {draft.priority}
                        </span>
                        {readOnly && (
                          <span className="rounded bg-blue-100 px-2 py-0.5 text-xs font-normal text-blue-800">
                            inherited
                          </span>
                        )}
                      </h2>
                      {cfg.secret_set ? (
                        <p className="mt-1 flex items-center gap-1.5 text-xs text-gray-500">
                          <KeyRound className="h-3.5 w-3.5" />
                          Credential stored {cfg.secret_hint && <code>{cfg.secret_hint}</code>}
                        </p>
                      ) : (
                        !needsNothing(cfg.provider) && (
                          <p className="mt-1 text-xs text-amber-700">No credential stored yet</p>
                        )
                      )}
                    </div>

                    {!readOnly && (
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => patchDraft(cfg.provider, { enabled: !draft.enabled })}
                          className="flex items-center gap-1.5 text-sm text-gray-600 hover:text-gray-900"
                        >
                          {draft.enabled ? (
                            <ToggleRight className="h-5 w-5 text-green-600" />
                          ) : (
                            <ToggleLeft className="h-5 w-5 text-gray-400" />
                          )}
                          {draft.enabled ? 'Enabled' : 'Disabled'}
                        </button>
                        <button
                          onClick={() => remove(cfg.provider)}
                          className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                          aria-label={`Remove ${providerLabel(cfg.provider)}`}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    )}
                  </div>

                  {hint?.note && (
                    <p className="mt-3 rounded bg-gray-50 p-2.5 text-xs text-gray-600">{hint.note}</p>
                  )}

                  {!readOnly && !needsNothing(cfg.provider) && (
                    <div className="mt-4 grid gap-4 sm:grid-cols-2">
                      {!isCredentialOnly(cfg.provider) && (
                        <>
                          <Field
                            label="Host"
                            value={draft.host}
                            placeholder={cfg.provider === 'smtp' ? 'required' : 'preset default'}
                            onChange={(v) => patchDraft(cfg.provider, { host: v })}
                          />
                          <Field
                            label="Port"
                            value={draft.port}
                            placeholder="587"
                            onChange={(v) => patchDraft(cfg.provider, { port: v })}
                          />
                          <Field
                            label={hint?.userLabel || 'Username'}
                            value={draft.username}
                            onChange={(v) => patchDraft(cfg.provider, { username: v })}
                          />
                        </>
                      )}
                      <Field
                        label={hint?.secretLabel || 'Secret'}
                        type="password"
                        value={draft.secret}
                        placeholder={cfg.secret_set ? 'unchanged — enter a value to replace' : ''}
                        onChange={(v) => patchDraft(cfg.provider, { secret: v })}
                        autoComplete="new-password"
                      />
                      <Field
                        label="From address"
                        value={draft.from_email}
                        placeholder="noreply@yourdomain.com"
                        onChange={(v) => patchDraft(cfg.provider, { from_email: v })}
                      />
                      <Field
                        label="From name"
                        value={draft.from_name}
                        onChange={(v) => patchDraft(cfg.provider, { from_name: v })}
                      />
                      <Field
                        label="Priority"
                        value={String(draft.priority)}
                        onChange={(v) => patchDraft(cfg.provider, { priority: Number(v) || 0 })}
                      />
                    </div>
                  )}

                  {!readOnly && (
                    <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-4">
                      <button
                        onClick={() => save(cfg.provider)}
                        disabled={saving === cfg.provider}
                        className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-50"
                      >
                        {saving === cfg.provider ? 'Saving…' : 'Save'}
                      </button>
                      <div className="ml-auto flex items-center gap-2">
                        <input
                          type="email"
                          value={testTo}
                          onChange={(e) => setTestTo(e.target.value)}
                          placeholder="you@example.com"
                          className="w-52 rounded-lg border border-gray-300 px-3 py-2 text-sm"
                        />
                        <button
                          onClick={() => test(cfg.provider)}
                          disabled={testing === cfg.provider}
                          className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                        >
                          {testing === cfg.provider ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Send className="h-4 w-4" />
                          )}
                          Send test
                        </button>
                      </div>
                    </div>
                  )}
                </section>
              );
            })}
          </div>

          {!inherited && unconfigured.length > 0 && (
            <div className="flex flex-wrap items-center gap-2 rounded-lg border border-gray-200 bg-white p-4">
              <label htmlFor="add-provider" className="text-sm font-medium text-gray-700">
                Add a provider
              </label>
              <select
                id="add-provider"
                value={addProvider}
                onChange={(e) => setAddProvider(e.target.value)}
                className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="">Choose…</option>
                {unconfigured.map((p) => (
                  <option key={p} value={p}>
                    {providerLabel(p)}
                  </option>
                ))}
              </select>
              <button
                onClick={add}
                disabled={!addProvider || saving === addProvider}
                className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-50"
              >
                <Plus className="h-4 w-4" />
                Add
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  type = 'text',
  autoComplete,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: string;
  autoComplete?: string;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600">{label}</label>
      <input
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none"
      />
    </div>
  );
}
