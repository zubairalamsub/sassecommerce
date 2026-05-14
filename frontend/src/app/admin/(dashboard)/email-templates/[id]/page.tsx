'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Loader2 } from 'lucide-react';
import EmailTemplateEditor from '@/components/admin/email-template-editor';
import { notificationTemplateApi, type NotificationTemplate } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';

export default function EditEmailTemplatePage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;
  const { tenantId, token } = useAuthStore();

  const [template, setTemplate] = useState<NotificationTemplate | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function run() {
      if (!id || !tenantId || !token) return;
      setLoading(true);
      try {
        const t = await notificationTemplateApi.get(id, tenantId, token);
        if (!cancelled) setTemplate(t);
      } catch (err) {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : 'Template not found');
          router.replace('/admin/email-templates');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    run();
    return () => {
      cancelled = true;
    };
  }, [id, tenantId, token, router]);

  if (loading || !template) {
    return (
      <div className="flex justify-center py-16">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return <EmailTemplateEditor initial={template} />;
}
