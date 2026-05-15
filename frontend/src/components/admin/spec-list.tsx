'use client';

import { useState } from 'react';
import { ChevronDown, ChevronUp, GripVertical, Plus, X } from 'lucide-react';

export interface SpecRow {
  id: string;
  key: string;
  value: string;
}

interface SpecListProps {
  rows: SpecRow[];
  onChange: (rows: SpecRow[]) => void;
}

const newRow = (): SpecRow => ({
  id: `s-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
  key: '',
  value: '',
});

export default function SpecList({ rows, onChange }: SpecListProps) {
  const [dragId, setDragId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);

  const update = (id: string, field: keyof SpecRow, value: string) =>
    onChange(rows.map((r) => (r.id === id ? { ...r, [field]: value } : r)));

  const remove = (id: string) => onChange(rows.filter((r) => r.id !== id));

  const add = () => onChange([...rows, newRow()]);

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
          <p className="text-sm text-gray-500">No specifications yet</p>
          <button type="button" onClick={add} className="mt-2 inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-primary-dark">
            <Plus className="h-4 w-4" /> Add specification
          </button>
        </div>
      ) : (
        <>
          {rows.map((row, index) => (
            <div
              key={row.id}
              draggable
              onDragStart={(e) => { setDragId(row.id); e.dataTransfer.effectAllowed = 'move'; }}
              onDragEnd={() => { setDragId(null); setOverId(null); }}
              onDragOver={(e) => { e.preventDefault(); setOverId(row.id); }}
              onDragLeave={() => setOverId((v) => (v === row.id ? null : v))}
              onDrop={(e) => { e.preventDefault(); if (dragId) reorder(dragId, row.id); setOverId(null); }}
              className={`group flex items-center gap-2 rounded-lg border bg-white px-2 py-1.5 transition-all ${
                overId === row.id && dragId !== row.id ? 'border-primary ring-2 ring-primary/20' : 'border-gray-200'
              } ${dragId === row.id ? 'opacity-50' : ''}`}
            >
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
              <input
                type="text"
                value={row.key}
                onChange={(e) => update(row.id, 'key', e.target.value)}
                placeholder="Material"
                className="w-1/3 rounded-md border border-gray-200 bg-gray-50 px-2 py-1.5 text-sm focus:border-primary focus:bg-white focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="text"
                value={row.value}
                onChange={(e) => update(row.id, 'value', e.target.value)}
                placeholder="100% pure cotton"
                className="flex-1 rounded-md border border-gray-200 bg-gray-50 px-2 py-1.5 text-sm focus:border-primary focus:bg-white focus:outline-none focus:ring-1 focus:ring-primary"
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
          ))}
          <button
            type="button"
            onClick={add}
            className="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:border-primary hover:text-primary"
          >
            <Plus className="h-4 w-4" /> Add specification
          </button>
        </>
      )}
    </div>
  );
}
