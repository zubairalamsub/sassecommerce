import { NextRequest } from 'next/server';
import { writeFile, mkdir } from 'fs/promises';
import path from 'path';
import { jwtVerify } from 'jose';

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

async function authorize(request: NextRequest): Promise<boolean> {
  const secret = process.env.JWT_SECRET;
  if (!secret) return false;

  const header = request.headers.get('authorization') || '';
  const token = header.startsWith('Bearer ') ? header.slice(7) : '';
  if (!token) return false;

  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(secret));
    return UPLOADER_ROLES.has(String(payload.role ?? ''));
  } catch {
    return false;
  }
}

export async function POST(request: NextRequest) {
  if (!(await authorize(request))) {
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
    const uploadDir = path.join(STORAGE_PATH, folder);
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
      paths.push(`${folder}/${filename}`);
    }

    return Response.json({ paths });
  } catch (err) {
    console.error('Upload error:', err);
    return Response.json({ error: 'Upload failed' }, { status: 500 });
  }
}
