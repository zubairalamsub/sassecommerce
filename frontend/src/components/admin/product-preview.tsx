'use client';

import { useState } from 'react';
import type { SpecRow } from './spec-list';
import type { FAQRow } from './faq-list';
import type { SeoData } from './seo-panel';
import { ChevronDown, ChevronRight, ShoppingCart, Tag } from 'lucide-react';

interface ProductPreviewProps {
  name: string;
  price: string;
  compareAtPrice: string;
  description: string; // HTML
  images: { url: string }[];
  specs: SpecRow[];
  faqs: FAQRow[];
  seo: SeoData;
  slug: string;
}

const fmtBDT = (n: number) =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency: 'BDT', maximumFractionDigits: 0 })
    .format(n)
    .replace('BDT', '৳');

export default function ProductPreview({
  name, price, compareAtPrice, description, images, specs, faqs, seo, slug,
}: ProductPreviewProps) {
  const [activeImg, setActiveImg] = useState(0);
  const [openFaqs, setOpenFaqs] = useState<Set<string>>(new Set());
  const toggleFaq = (id: string) =>
    setOpenFaqs((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });

  const priceN = Number(price) || 0;
  const compareN = Number(compareAtPrice) || 0;
  const hasDiscount = compareN > priceN && priceN > 0;

  return (
    <div className="space-y-8 bg-white">
      {/* Top hero */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Image gallery */}
        <div className="space-y-3">
          <div className="aspect-square w-full overflow-hidden rounded-xl bg-gray-100">
            {images[activeImg]?.url ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={images[activeImg].url} alt={name} className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full w-full items-center justify-center text-gray-300">
                <Tag className="h-16 w-16" />
              </div>
            )}
          </div>
          {images.length > 1 && (
            <div className="flex gap-2 overflow-x-auto">
              {images.map((img, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => setActiveImg(i)}
                  className={`h-16 w-16 flex-shrink-0 overflow-hidden rounded-lg border-2 transition-colors ${
                    activeImg === i ? 'border-primary' : 'border-transparent'
                  }`}
                >
                  {img.url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={img.url} alt="" className="h-full w-full object-cover" />
                  ) : null}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Right column */}
        <div>
          <h1 className="text-2xl font-bold text-gray-900 sm:text-3xl">{name || 'Product name'}</h1>
          <div className="mt-3 flex items-baseline gap-3">
            <span className="text-2xl font-semibold text-gray-900">{fmtBDT(priceN)}</span>
            {hasDiscount && (
              <>
                <span className="text-base text-gray-400 line-through">{fmtBDT(compareN)}</span>
                <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
                  {Math.round(((compareN - priceN) / compareN) * 100)}% off
                </span>
              </>
            )}
          </div>
          <button
            type="button"
            disabled
            className="mt-5 inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white opacity-90"
          >
            <ShoppingCart className="h-4 w-4" /> Add to cart
          </button>

          {/* Quick specs (first 6) */}
          {specs.filter((s) => s.key || s.value).length > 0 && (
            <dl className="mt-6 grid grid-cols-1 gap-2 sm:grid-cols-2">
              {specs.filter((s) => s.key || s.value).slice(0, 6).map((s) => (
                <div key={s.id} className="flex justify-between border-b border-gray-100 pb-1.5 text-sm">
                  <dt className="text-gray-500">{s.key || '—'}</dt>
                  <dd className="font-medium text-gray-900">{s.value || '—'}</dd>
                </div>
              ))}
            </dl>
          )}
        </div>
      </div>

      {/* Description */}
      {description && description.trim() && (
        <section>
          <h2 className="mb-3 border-b border-gray-200 pb-2 text-lg font-semibold text-gray-900">Description</h2>
          <div
            className="prose prose-sm max-w-none text-gray-700 [&_h2]:mb-2 [&_h2]:mt-4 [&_h2]:text-lg [&_h2]:font-semibold [&_h3]:mb-2 [&_h3]:mt-4 [&_h3]:text-base [&_h3]:font-semibold [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_blockquote]:border-l-4 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:italic [&_pre]:rounded-md [&_pre]:bg-gray-100 [&_pre]:px-3 [&_pre]:py-2 [&_a]:text-primary [&_a]:underline [&_img]:my-3 [&_img]:max-w-full [&_img]:rounded-md"
            dangerouslySetInnerHTML={{ __html: description }}
          />
        </section>
      )}

      {/* Full specifications */}
      {specs.filter((s) => s.key || s.value).length > 0 && (
        <section>
          <h2 className="mb-3 border-b border-gray-200 pb-2 text-lg font-semibold text-gray-900">Specifications</h2>
          <div className="overflow-hidden rounded-lg border border-gray-200">
            <table className="w-full text-sm">
              <tbody>
                {specs.filter((s) => s.key || s.value).map((s, i) => (
                  <tr key={s.id} className={i % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                    <td className="w-1/3 px-4 py-2 font-medium text-gray-600">{s.key || '—'}</td>
                    <td className="px-4 py-2 text-gray-900">{s.value || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* FAQs */}
      {faqs.filter((f) => f.question || f.answer).length > 0 && (
        <section>
          <h2 className="mb-3 border-b border-gray-200 pb-2 text-lg font-semibold text-gray-900">Frequently Asked Questions</h2>
          <div className="divide-y divide-gray-200 rounded-lg border border-gray-200">
            {faqs.filter((f) => f.question || f.answer).map((f) => {
              const open = openFaqs.has(f.id);
              return (
                <div key={f.id}>
                  <button
                    type="button"
                    onClick={() => toggleFaq(f.id)}
                    className="flex w-full items-center gap-2 px-4 py-3 text-left text-sm font-medium text-gray-900 hover:bg-gray-50"
                  >
                    {open ? <ChevronDown className="h-4 w-4 flex-shrink-0 text-gray-400" /> : <ChevronRight className="h-4 w-4 flex-shrink-0 text-gray-400" />}
                    <span>{f.question || 'Untitled question'}</span>
                  </button>
                  {open && f.answer && (
                    <div
                      className="prose prose-sm max-w-none px-10 pb-4 text-sm text-gray-700 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:list-decimal [&_ol]:pl-5 [&_a]:text-primary [&_a]:underline"
                      dangerouslySetInnerHTML={{ __html: f.answer }}
                    />
                  )}
                </div>
              );
            })}
          </div>
        </section>
      )}

      {/* SEO summary */}
      <section className="rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-xs text-gray-600">
        <h3 className="mb-2 text-sm font-semibold text-gray-700">SEO summary</h3>
        <div className="space-y-1">
          <div><span className="font-medium">URL:</span> /products/{slug || 'product-slug'}</div>
          <div><span className="font-medium">Meta title:</span> {seo.meta_title || name || '—'}</div>
          <div><span className="font-medium">Meta description:</span> {seo.meta_description || '—'}</div>
          <div><span className="font-medium">OG image:</span> {seo.og_image || images[0]?.url || '—'}</div>
        </div>
      </section>
    </div>
  );
}
