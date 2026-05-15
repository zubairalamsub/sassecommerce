'use client';

import { useState } from 'react';
import { GripVertical, Plus, X, ChevronDown, ChevronRight, ChevronUp } from 'lucide-react';
import RichTextEditor from './rich-text-editor';

export interface FAQRow {
  id: string;
  question: string;
  answer: string;
}

interface FAQListProps {
  rows: FAQRow[];
  onChange: (rows: FAQRow[]) => void;
}

const newRow = (): FAQRow => ({
  id: `f-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
  question: '',
  answer: '',
});

export default function FAQList({ rows, onChange }: FAQListProps) {
  const [dragId, setDragId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);
  const [openIds, setOpenIds] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    setOpenIds((s) => {
      const n = new Set(s);
      if (n.has(id)) n.delete(id); else n.add(id);
      return n;
    });
  };

  const update = (id: string, field: keyof FAQRow, value: string) =>
    onChange(rows.map((r) => (r.id === id ? { ...r, [field]: value } : r)));

  const remove = (id: string) => {
    onChange(rows.filter((r) => r.id !== id));
    setOpenIds((s) => { const n = new Set(s); n.delete(id); return n; });
  };

  const add = () => {
    const row = newRow();
    onChange([...rows, row]);
    setOpenIds((s) => new Set(s).add(row.id));
  };

  const reorder = (fromId: string, toId: string) => {
    if (fromId === toId) return;
    const from = rows.findIndex((r) => r.id === fromId);
    const to = rows.findIndex((r) => r.id === toId);
    if (from < 0 || to < 0) return;
    const next = rows.slice();
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    onChange(next);
  };

  const move = (index: number, delta: number) => {
    const to = index + delta;
    if (to < 0 || to >= rows.length) return;
    const next = rows.slice();
    const [moved] = next.splice(index, 1);
    next.splice(to, 0, moved);
    onChange(next);
  };

  return (
    <div className="space-y-2">
      {rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-8 text-center">
          <p className="text-sm text-gray-500">No FAQs yet</p>
          <button type="button" onClick={add} className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-primary-dark">
            <Plus className="h-4 w-4" /> Add question
          </button>
        </div>
      ) : (
        <>
          {rows.map((row, index) => {
            const open = openIds.has(row.id);
            return (
              <div
                key={row.id}
                draggable
                onDragStart={(e) => { setDragId(row.id); e.dataTransfer.effectAllowed = 'move'; }}
                onDragEnd={() => { setDragId(null); setOverId(null); }}
                onDragOver={(e) => { e.preventDefault(); setOverId(row.id); }}
                onDragLeave={() => setOverId((v) => (v === row.id ? null : v))}
                onDrop={(e) => { e.preventDefault(); if (dragId) reorder(dragId, row.id); setOverId(null); }}
                className={`rounded-lg border bg-white transition-all ${
                  overId === row.id && dragId !== row.id ? 'border-primary ring-2 ring-primary/20' : 'border-gray-200'
                } ${dragId === row.id ? 'opacity-50' : ''}`}
              >
                <div className="flex items-center gap-2 px-2 py-1.5">
                  <button
                    type="button"
                    className="cursor-grab touch-none rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 active:cursor-grabbing"
                    title="Drag to reorder"
                    aria-label="Drag handle"
                  >
                    <GripVertical className="h-4 w-4" />
                  </button>
                  <div className="flex flex-col">
                    <button
                      type="button"
                      onClick={() => move(index, -1)}
                      disabled={index === 0}
                      className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                      title="Move up"
                      aria-label="Move up"
                    >
                      <ChevronUp className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => move(index, 1)}
                      disabled={index === rows.length - 1}
                      className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                      title="Move down"
                      aria-label="Move down"
                    >
                      <ChevronDown className="h-3.5 w-3.5" />
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => toggle(row.id)}
                    className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                    title={open ? 'Collapse' : 'Expand'}
                  >
                    {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                  </button>
                  <input
                    type="text"
                    value={row.question}
                    onChange={(e) => update(row.id, 'question', e.target.value)}
                    onFocus={() => setOpenIds((s) => new Set(s).add(row.id))}
                    placeholder="Question (e.g., What is the return policy?)"
                    className="flex-1 rounded-md border border-gray-200 bg-gray-50 px-2 py-1.5 text-sm font-medium focus:border-primary focus:bg-white focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                  <button
                    type="button"
                    onClick={() => remove(row.id)}
                    className="rounded p-1 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                    title="Remove"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
                {open && (
                  <div className="border-t border-gray-100 px-2 pb-2 pt-2">
                    <RichTextEditor
                      value={row.answer}
                      onChange={(v) => update(row.id, 'answer', v)}
                      placeholder="Answer..."
                      minHeight={120}
                    />
                  </div>
                )}
              </div>
            );
          })}
          <button
            type="button"
            onClick={add}
            className="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:border-primary hover:text-primary"
          >
            <Plus className="h-4 w-4" /> Add question
          </button>
        </>
      )}
    </div>
  );
}
