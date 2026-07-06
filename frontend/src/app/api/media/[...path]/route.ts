import { NextRequest } from 'next/server';
import { readFile, stat } from 'fs/promises';
import path from 'path';

const STORAGE_PATH = process.env.MEDIA_STORAGE_PATH || path.join(process.cwd(), 'media');

// SVG is deliberately excluded: served as image/svg+xml it can execute
// embedded scripts in this origin if opened directly. Any legacy .svg file
// on disk falls through to application/octet-stream with the no-sniff and
// attachment headers below, which browsers will not render.
const MIME_TYPES: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.gif': 'image/gif',
  '.webp': 'image/webp',
  '.avif': 'image/avif',
};

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  try {
    const segments = (await params).path;
    // Prevent directory traversal
    const relativePath = segments.join('/');
    if (relativePath.includes('..')) {
      return new Response('Forbidden', { status: 403 });
    }

    const filePath = path.join(STORAGE_PATH, relativePath);

    // Check file exists
    const fileStat = await stat(filePath).catch(() => null);
    if (!fileStat || !fileStat.isFile()) {
      return new Response('Not found', { status: 404 });
    }

    const ext = path.extname(filePath).toLowerCase();
    const contentType = MIME_TYPES[ext];

    const buffer = await readFile(filePath);

    const headers: Record<string, string> = {
      'Content-Type': contentType || 'application/octet-stream',
      'X-Content-Type-Options': 'nosniff',
      'Cache-Control': 'public, max-age=31536000, immutable',
      'Content-Length': String(buffer.length),
    };
    if (!contentType) {
      // Unknown types download instead of rendering in this origin.
      headers['Content-Disposition'] = 'attachment';
    }

    return new Response(buffer, { headers });
  } catch {
    return new Response('Internal server error', { status: 500 });
  }
}
