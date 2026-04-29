import { NextRequest } from 'next/server';
import { writeFile, mkdir } from 'fs/promises';
import path from 'path';

// Configurable storage path — defaults to ./media in project root
// In Docker: set MEDIA_STORAGE_PATH=/app/media (mounted to a named volume)
const STORAGE_PATH = process.env.MEDIA_STORAGE_PATH || path.join(process.cwd(), 'media');

export async function POST(request: NextRequest) {
  try {
    const formData = await request.formData();
    const files = formData.getAll('files') as File[];

    if (files.length === 0) {
      return Response.json({ error: 'No files provided' }, { status: 400 });
    }

    // Organize by subfolder (default: products)
    const folder = formData.get('folder')?.toString() || 'products';
    const uploadDir = path.join(STORAGE_PATH, folder);
    await mkdir(uploadDir, { recursive: true });

    const paths: string[] = [];

    for (const file of files) {
      if (!file.type.startsWith('image/')) {
        continue;
      }

      if (file.size > 5 * 1024 * 1024) {
        return Response.json(
          { error: `File ${file.name} exceeds 5MB limit` },
          { status: 400 },
        );
      }

      const bytes = await file.arrayBuffer();
      const buffer = Buffer.from(bytes);

      const ext = path.extname(file.name) || '.jpg';
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
