import { NextRequest } from 'next/server';
import { readFile, stat } from 'fs/promises';
import path from 'path';

const STORAGE_PATH = process.env.MEDIA_STORAGE_PATH || path.join(process.cwd(), 'media');

const MIME_TYPES: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.gif': 'image/gif',
  '.webp': 'image/webp',
  '.svg': 'image/svg+xml',
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
    const contentType = MIME_TYPES[ext] || 'application/octet-stream';

    const buffer = await readFile(filePath);

    return new Response(buffer, {
      headers: {
        'Content-Type': contentType,
        'Cache-Control': 'public, max-age=31536000, immutable',
        'Content-Length': String(buffer.length),
      },
    });
  } catch {
    return new Response('Internal server error', { status: 500 });
  }
}
