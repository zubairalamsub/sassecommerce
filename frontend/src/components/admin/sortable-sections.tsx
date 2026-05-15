'use client';

import { useState, type ReactNode } from 'react';
import { ChevronDown, ChevronUp, GripVertical } from 'lucide-react';

export interface SectionDef {
  id: string;
  title: string;
  subtitle?: string;
  render: () => ReactNode;
}

interface SortableSectionsProps {
  order: string[];
  onOrderChange: (ids: string[]) => void;
  sections: SectionDef[];
}

export default function SortableSections({ order, onOrderChange, sections }: SortableSectionsProps) {
  const [dragId, setDragId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);

  const byId = new Map(sections.map((s) => [s.id, s]));
  const orderedIds = order.filter((id) => byId.has(id));
  // Append any sections not in the order (e.g. newly added) at the end.
  for (const s of sections) if (!orderedIds.includes(s.id)) orderedIds.push(s.id);

  const reorder = (fromId: string, toId: string) => {
    if (fromId === toId) return;
    const from = orderedIds.indexOf(fromId);
    const to = orderedIds.indexOf(toId);
    if (from < 0 || to < 0) return;
    const next = orderedIds.slice();
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    onOrderChange(next);
  };

  const move = (index: number, delta: number) => {
    const to = index + delta;
    if (to < 0 || to >= orderedIds.length) return;
    const next = orderedIds.slice();
    const [moved] = next.splice(index, 1);
    next.splice(to, 0, moved);
    onOrderChange(next);
  };

  return (
    <div className="space-y-6">
      {orderedIds.map((id, index) => {
        const section = byId.get(id);
        if (!section) return null;
        const isDragging = dragId === id;
        const isOver = overId === id && dragId !== id;
        return (
          <div
            key={id}
            onDragOver={(e) => { if (dragId) { e.preventDefault(); setOverId(id); } }}
            onDragLeave={() => setOverId((v) => (v === id ? null : v))}
            onDrop={(e) => { e.preventDefault(); if (dragId) reorder(dragId, id); setOverId(null); }}
            className={`rounded-xl border bg-white shadow-sm transition-all ${
              isOver ? 'border-primary ring-2 ring-primary/20' : 'border-gray-200'
            } ${isDragging ? 'opacity-50' : ''}`}
          >
            <div
              draggable
              onDragStart={(e) => { setDragId(id); e.dataTransfer.effectAllowed = 'move'; }}
              onDragEnd={() => { setDragId(null); setOverId(null); }}
              className="flex items-start gap-3 border-b border-gray-100 px-6 py-4"
            >
              <div className="cursor-grab pt-0.5 text-gray-300 hover:text-gray-500 active:cursor-grabbing" title="Drag to reorder section">
                <GripVertical className="h-5 w-5" />
              </div>
              <div className="flex flex-col">
                <button
                  type="button"
                  onClick={() => move(index, -1)}
                  disabled={index === 0}
                  className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                  title="Move up"
                  aria-label="Move section up"
                >
                  <ChevronUp className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => move(index, 1)}
                  disabled={index === orderedIds.length - 1}
                  className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                  title="Move down"
                  aria-label="Move section down"
                >
                  <ChevronDown className="h-4 w-4" />
                </button>
              </div>
              <div className="flex-1">
                <h2 className="text-lg font-semibold text-gray-900">{section.title}</h2>
                {section.subtitle && <p className="text-xs text-gray-500">{section.subtitle}</p>}
              </div>
            </div>
            <div className="px-6 py-5">{section.render()}</div>
          </div>
        );
      })}
    </div>
  );
}
