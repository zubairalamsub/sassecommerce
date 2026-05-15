'use client';

import { Globe, Image as ImageIcon } from 'lucide-react';

export interface SeoData {
  meta_title: string;
  meta_description: string;
  og_title: string;
  og_description: string;
  og_image: string;
}

export const emptySeo: SeoData = {
  meta_title: '',
  meta_description: '',
  og_title: '',
  og_description: '',
  og_image: '',
};

interface SeoPanelProps {
  value: SeoData;
  onChange: (v: SeoData) => void;
  productName: string;
  productSlug: string;
  productImageFallback?: string;
}

const META_TITLE_MAX = 60;
const META_DESC_MAX = 160;
const OG_TITLE_MAX = 60;
const OG_DESC_MAX = 200;

export default function SeoPanel({ value, onChange, productName, productSlug, productImageFallback }: SeoPanelProps) {
  const set = <K extends keyof SeoData>(key: K, v: SeoData[K]) => onChange({ ...value, [key]: v });

  const previewTitle = value.meta_title || productName || 'Product title';
  const previewDesc = value.meta_description || 'A short description of the product appears here in the search engine snippet.';
  const ogTitle = value.og_title || value.meta_title || productName || 'Product title';
  const ogDesc = value.og_description || value.meta_description || 'Product description for social previews.';
  const ogImage = value.og_image || productImageFallback || '';

  return (
    <div className="space-y-5">
      {/* Meta */}
      <div className="grid grid-cols-1 gap-4">
        <div>
          <div className="mb-1.5 flex items-center justify-between">
            <label className="text-sm font-medium text-gray-700">Meta Title</label>
            <CharCount value={value.meta_title} max={META_TITLE_MAX} />
          </div>
          <input
            type="text"
            value={value.meta_title}
            onChange={(e) => set('meta_title', e.target.value)}
            placeholder={productName || 'Product title for search engines'}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <div>
          <div className="mb-1.5 flex items-center justify-between">
            <label className="text-sm font-medium text-gray-700">Meta Description</label>
            <CharCount value={value.meta_description} max={META_DESC_MAX} />
          </div>
          <textarea
            rows={3}
            value={value.meta_description}
            onChange={(e) => set('meta_description', e.target.value)}
            placeholder="Short summary shown under the title in search results"
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {/* Google SERP preview */}
      <div className="rounded-lg border border-gray-200 bg-white p-4">
        <div className="mb-2 flex items-center gap-1.5 text-xs font-medium text-gray-500">
          <Globe className="h-3.5 w-3.5" />
          Google search preview
        </div>
        <div className="font-serif">
          <div className="flex items-center gap-2 text-xs text-gray-600">
            <span className="inline-block h-4 w-4 rounded-full bg-gray-200" />
            <span>yourstore.com</span>
            <span className="text-gray-400">› products › {productSlug || 'product-slug'}</span>
          </div>
          <div className="mt-1 truncate text-lg text-[#1a0dab] hover:underline">
            {previewTitle}
          </div>
          <div className="mt-0.5 line-clamp-2 text-sm text-gray-700">{previewDesc}</div>
        </div>
      </div>

      {/* Open Graph */}
      <div className="rounded-lg border border-dashed border-gray-200 p-4">
        <h3 className="mb-3 text-sm font-semibold text-gray-700">Social share (Open Graph)</h3>
        <div className="grid grid-cols-1 gap-4">
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <label className="text-sm font-medium text-gray-700">OG Title</label>
              <CharCount value={value.og_title} max={OG_TITLE_MAX} />
            </div>
            <input
              type="text"
              value={value.og_title}
              onChange={(e) => set('og_title', e.target.value)}
              placeholder={`Defaults to meta title${value.meta_title ? ` ("${value.meta_title.slice(0, 30)}${value.meta_title.length > 30 ? '...' : ''}")` : ''}`}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <label className="text-sm font-medium text-gray-700">OG Description</label>
              <CharCount value={value.og_description} max={OG_DESC_MAX} />
            </div>
            <textarea
              rows={2}
              value={value.og_description}
              onChange={(e) => set('og_description', e.target.value)}
              placeholder="Defaults to meta description"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700">OG Image URL</label>
            <input
              type="url"
              value={value.og_image}
              onChange={(e) => set('og_image', e.target.value)}
              placeholder="https://cdn.example.com/og.jpg (recommended 1200×630)"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <p className="mt-1 text-xs text-gray-500">Falls back to the first product image.</p>
          </div>
        </div>
      </div>

      {/* Facebook OG card preview */}
      <div className="rounded-lg border border-gray-200 bg-[#f0f2f5] p-4">
        <div className="mb-2 flex items-center gap-1.5 text-xs font-medium text-gray-500">
          <ImageIcon className="h-3.5 w-3.5" />
          Facebook share preview
        </div>
        <div className="overflow-hidden rounded-md border border-gray-300 bg-white">
          <div className="relative aspect-[1.91/1] w-full bg-gray-100">
            {ogImage ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={ogImage} alt="" className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full w-full items-center justify-center text-gray-300">
                <ImageIcon className="h-10 w-10" />
              </div>
            )}
          </div>
          <div className="border-t border-gray-200 bg-[#f0f2f5] px-3 py-2">
            <div className="text-[11px] uppercase tracking-wide text-gray-500">yourstore.com</div>
            <div className="mt-0.5 line-clamp-1 text-sm font-semibold text-gray-900">{ogTitle}</div>
            <div className="mt-0.5 line-clamp-2 text-xs text-gray-600">{ogDesc}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

function CharCount({ value, max }: { value: string; max: number }) {
  const len = value.length;
  const over = len > max;
  const warn = !over && len > max * 0.9;
  const cls = over ? 'text-red-600' : warn ? 'text-amber-600' : 'text-gray-400';
  return <span className={`text-[11px] font-mono ${cls}`}>{len} / {max}</span>;
}
