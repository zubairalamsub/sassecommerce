import { NextRequest } from 'next/server';
import { writeFile, mkdir } from 'fs/promises';
import path from 'path';
import { jwtVerify } from 'jose';
import { readAuthCookie } from '@/lib/auth-cookie';

// Configurable storage path — defaults to ./media in project root
// In Docker: set MEDIA_STORAGE_PATH=/app/media (mounted to a named volume)
const STORAGE_PATH = process.env.MEDIA_STORAGE_PATH || path.join(process.cwd(), 'media');

// Uploads may only land in one of these fixed subfolders. The client-supplied
// value is never used in the path unless it matches exactly — this is what
// prevents `folder=../../etc`-style traversal.
const ALLOWED_FOLDERS = new Set(['products', 'avatars', 'tenants']);

// Extension is derived from the (server-checked) MIME type, never from the
// client filename. SVG is deliberately absent: it can carry scripts and would
// be served back as image/svg+xml.
const MIME_EXTENSIONS: Record<string, string> = {
  'image/jpeg': '.jpg',
  'image/png': '.png',
  'image/gif': '.gif',
  'image/webp': '.webp',
  'image/avif': '.avif',
};

const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5 MB

// Roles allowed to upload media (staff-only feature in the admin dashboard).
const UPLOADER_ROLES = new Set(['super_admin', 'admin', 'moderator']);

// Platform-level uploads (super_admin, whose JWT carries no tenant) land under
// this reserved bucket so they never collide with a real tenant segment.
const PLATFORM_TENANT = '_platform';

interface Uploader {
  role: string;
  tenantId: string;
}

// Derives the on-disk tenant segment from the verified JWT. Tenant ids are
// sanitised to a strict charset so the value can never escape its subtree
// (defence in depth on top of the traversal check below).
function tenantSegment(rawTenantId: unknown): string {
  const t = String(rawTenantId ?? '').trim();
  if (!t) return PLATFORM_TENANT;
  return /^[A-Za-z0-9_-]+$/.test(t) ? t : '';
}

// Returns the authenticated uploader (role + tenant) or null if the request is
// not a valid staff session. B03: the JWT lives in the HttpOnly auth cookie,
// not an Authorization header the browser sets from localStorage.
async function authorize(request: NextRequest): Promise<Uploader | null> {
  const secret = process.env.JWT_SECRET;
  if (!secret) return null;

  const token = readAuthCookie(request);
  if (!token) return null;

  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(secret));
    const role = String(payload.role ?? '');
    if (!UPLOADER_ROLES.has(role)) return null;

    const tenantId = tenantSegment(payload.tenant_id);
    if (!tenantId) return null; // present but malformed → reject

    return { role, tenantId };
  } catch {
    return null;
  }
}

export async function POST(request: NextRequest) {
  const uploader = await authorize(request);
  if (!uploader) {
    return Response.json({ error: 'Unauthorized' }, { status: 401 });
  }

  try {
    const formData = await request.formData();
    const files = formData.getAll('files') as File[];

    if (files.length === 0) {
      return Response.json({ error: 'No files provided' }, { status: 400 });
    }

    const folder = formData.get('folder')?.toString() || 'products';
    if (!ALLOWED_FOLDERS.has(folder)) {
      return Response.json({ error: 'Invalid folder' }, { status: 400 });
    }
    // Media is physically partitioned by tenant: {tenant}/{folder}/{file}. The
    // tenant segment comes from the JWT (never the request body), so a caller
    // cannot write into another tenant's subtree.
    const relDir = path.posix.join(uploader.tenantId, folder);
    const uploadDir = path.join(STORAGE_PATH, uploader.tenantId, folder);
    await mkdir(uploadDir, { recursive: true });

    const paths: string[] = [];

    for (const file of files) {
      const ext = MIME_EXTENSIONS[file.type];
      if (!ext) {
        return Response.json(
          { error: `File type ${file.type || 'unknown'} is not allowed` },
          { status: 400 },
        );
      }

      if (file.size > MAX_FILE_SIZE) {
        return Response.json(
          { error: `File ${file.name} exceeds 5MB limit` },
          { status: 400 },
        );
      }

      const bytes = await file.arrayBuffer();
      const buffer = Buffer.from(bytes);

      const filename = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}${ext}`;
      const filepath = path.join(uploadDir, filename);

      await writeFile(filepath, buffer);

      // Return relative path only — the frontend prepends NEXT_PUBLIC_MEDIA_URL
      paths.push(path.posix.join(relDir, filename));
    }

    return Response.json({ paths });
  } catch (err) {
    console.error('Upload error:', err);
    return Response.json({ error: 'Upload failed' }, { status: 500 });
  }
}
