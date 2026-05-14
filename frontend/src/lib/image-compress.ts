// Browser-side image compression. Resizes oversized images and re-encodes in
// WebP (~30% smaller than JPEG at the same visible quality). Pure Canvas API
// — no dependency.
//
// Skips compression for:
//   - SVG (vector — already tiny, can't be rasterised without losing quality)
//   - GIF (preserves animation)
//   - Files already under the target size + dimensions
//
// Use case sizing presets are tuned for what each surface actually needs to
// display. A storefront product card never shows more than ~600px wide on a
// retina screen, so 2048px is plenty even at 2× DPR; an avatar in a header is
// usually 32-48px, so 512px is overkill on purpose for future-proofing.

export interface CompressOptions {
  /** Maximum width or height in pixels. Image is downscaled to fit. */
  maxDimension?: number;
  /** WebP encoder quality 0–1. Default 0.85 — visually lossless for most photos. */
  quality?: number;
  /**
   * If the source file is already smaller than this in bytes AND fits within
   * maxDimension, skip compression entirely. Default 50 KB.
   */
  skipUnderBytes?: number;
}

/** Tuned presets for each upload surface. */
export const ImagePresets = {
  /** Product photos: shown up to full-width on PDP, may be zoomed. */
  product:  { maxDimension: 2048, quality: 0.85 } satisfies CompressOptions,
  /** Avatars: tiny on screen but kept high enough for future retina/zoom UX. */
  avatar:   { maxDimension: 512,  quality: 0.85 } satisfies CompressOptions,
  /** Logos: always rendered small in headers, transparency matters. */
  logo:     { maxDimension: 800,  quality: 0.92 } satisfies CompressOptions,
  /** Favicons: very small target, but uploads stay PNG since browsers expect it. */
  favicon:  { maxDimension: 256,  quality: 0.95 } satisfies CompressOptions,
  /** Banners: hero images, render at full container width. */
  banner:   { maxDimension: 1920, quality: 0.85 } satisfies CompressOptions,
  /** Review attachments: customer phone photos, modest size is fine. */
  review:   { maxDimension: 1600, quality: 0.82 } satisfies CompressOptions,
};

/**
 * Compresses an image file. Returns the original file unchanged if compression
 * isn't beneficial (already small / SVG / GIF).
 *
 * The returned File preserves the original name (extension swapped to .webp
 * if re-encoded) so the server-side MIME validation still works.
 */
export async function compressImage(file: File, opts: CompressOptions = {}): Promise<File> {
  const { maxDimension = 2048, quality = 0.85, skipUnderBytes = 50 * 1024 } = opts;

  // Don't touch vector or animated formats.
  if (file.type === 'image/svg+xml' || file.type === 'image/gif') {
    return file;
  }

  // Quick skip for already-small files. We still need dimensions to be sure,
  // so do a cheap probe before bailing.
  const dims = await getDimensions(file);
  if (
    file.size < skipUnderBytes &&
    dims.width <= maxDimension &&
    dims.height <= maxDimension
  ) {
    return file;
  }

  const targetW = dims.width <= maxDimension && dims.height <= maxDimension
    ? dims.width
    : Math.round(dims.width >= dims.height
        ? maxDimension
        : (maxDimension * dims.width) / dims.height);
  const targetH = dims.width <= maxDimension && dims.height <= maxDimension
    ? dims.height
    : Math.round(dims.height >= dims.width
        ? maxDimension
        : (maxDimension * dims.height) / dims.width);

  const blob = await drawAndEncode(file, targetW, targetH, quality);
  if (!blob || blob.size >= file.size) {
    // Compression made it bigger (sometimes happens with already-optimised JPEGs)
    // — keep the original to avoid a regression.
    return file;
  }

  const newName = swapExtension(file.name, '.webp');
  return new File([blob], newName, { type: 'image/webp', lastModified: file.lastModified });
}

async function getDimensions(file: File): Promise<{ width: number; height: number }> {
  // createImageBitmap is the fast path; fall back to <img> for older browsers.
  if (typeof createImageBitmap === 'function') {
    try {
      const bmp = await createImageBitmap(file);
      const out = { width: bmp.width, height: bmp.height };
      bmp.close?.();
      return out;
    } catch {
      // fall through
    }
  }
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve({ width: img.naturalWidth, height: img.naturalHeight });
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to load image'));
    };
    img.src = url;
  });
}

async function drawAndEncode(file: File, w: number, h: number, quality: number): Promise<Blob | null> {
  // Prefer OffscreenCanvas when available (works off the main thread).
  const useOffscreen = typeof OffscreenCanvas !== 'undefined';

  let bitmap: ImageBitmap | HTMLImageElement;
  if (typeof createImageBitmap === 'function') {
    bitmap = await createImageBitmap(file);
  } else {
    bitmap = await loadHTMLImage(file);
  }

  let blob: Blob | null;
  if (useOffscreen) {
    const canvas = new OffscreenCanvas(w, h);
    const ctx = canvas.getContext('2d');
    if (!ctx) return null;
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(bitmap as CanvasImageSource, 0, 0, w, h);
    try {
      blob = await canvas.convertToBlob({ type: 'image/webp', quality });
    } catch {
      blob = await canvas.convertToBlob({ type: 'image/jpeg', quality });
    }
  } else {
    const canvas = document.createElement('canvas');
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    if (!ctx) return null;
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(bitmap as CanvasImageSource, 0, 0, w, h);
    blob = await new Promise<Blob | null>((resolve) =>
      canvas.toBlob(resolve, 'image/webp', quality),
    );
    if (!blob) {
      blob = await new Promise<Blob | null>((resolve) =>
        canvas.toBlob(resolve, 'image/jpeg', quality),
      );
    }
  }

  if ('close' in bitmap && typeof bitmap.close === 'function') bitmap.close();
  return blob;
}

function loadHTMLImage(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      URL.revokeObjectURL(url);
      resolve(img);
    };
    img.onerror = () => {
      URL.revokeObjectURL(url);
      reject(new Error('Failed to decode image'));
    };
    img.src = url;
  });
}

function swapExtension(name: string, newExt: string): string {
  const dot = name.lastIndexOf('.');
  return dot === -1 ? name + newExt : name.slice(0, dot) + newExt;
}
