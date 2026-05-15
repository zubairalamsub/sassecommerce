'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import {
  Bold, Italic, Underline, List, ListOrdered, Heading2, Heading3,
  Link2, Image as ImageIcon, Quote, Code, RotateCcw, RotateCw, Eraser,
} from 'lucide-react';

interface RichTextEditorProps {
  value: string;
  onChange: (html: string) => void;
  placeholder?: string;
  minHeight?: number;
  className?: string;
}

const exec = (cmd: string, arg?: string) => document.execCommand(cmd, false, arg);

export default function RichTextEditor({
  value,
  onChange,
  placeholder = 'Start typing...',
  minHeight = 200,
  className = '',
}: RichTextEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [showImageInput, setShowImageInput] = useState(false);
  const [linkUrl, setLinkUrl] = useState('');
  const [imageUrl, setImageUrl] = useState('');
  const savedRangeRef = useRef<Range | null>(null);

  useEffect(() => {
    if (editorRef.current && editorRef.current.innerHTML !== value) {
      editorRef.current.innerHTML = value || '';
    }
  }, [value]);

  const emit = useCallback(() => {
    if (editorRef.current) onChange(editorRef.current.innerHTML);
  }, [onChange]);

  const saveSelection = () => {
    const sel = window.getSelection();
    if (sel && sel.rangeCount > 0) savedRangeRef.current = sel.getRangeAt(0).cloneRange();
  };

  const restoreSelection = () => {
    const range = savedRangeRef.current;
    if (!range) return;
    const sel = window.getSelection();
    if (sel) { sel.removeAllRanges(); sel.addRange(range); }
  };

  const run = (cmd: string, arg?: string) => {
    editorRef.current?.focus();
    restoreSelection();
    exec(cmd, arg);
    emit();
  };

  const insertLink = () => {
    if (!linkUrl) { setShowLinkInput(false); return; }
    editorRef.current?.focus();
    restoreSelection();
    exec('createLink', linkUrl);
    // make links open in a new tab
    editorRef.current?.querySelectorAll('a[href]').forEach((a) => {
      if (!a.getAttribute('target')) {
        a.setAttribute('target', '_blank');
        a.setAttribute('rel', 'noopener noreferrer');
      }
    });
    emit();
    setLinkUrl('');
    setShowLinkInput(false);
  };

  const insertImage = () => {
    if (!imageUrl) { setShowImageInput(false); return; }
    editorRef.current?.focus();
    restoreSelection();
    exec('insertImage', imageUrl);
    // tag inserted images so styling can target them
    editorRef.current?.querySelectorAll('img:not([data-rte])').forEach((img) => {
      img.setAttribute('data-rte', '1');
    });
    emit();
    setImageUrl('');
    setShowImageInput(false);
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    // Strip styles on paste — keep plain text, let the toolbar apply formatting.
    e.preventDefault();
    const text = e.clipboardData.getData('text/plain');
    exec('insertText', text);
    emit();
  };

  return (
    <div className={`overflow-hidden rounded-lg border border-gray-300 bg-white ${className}`}>
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-0.5 border-b border-gray-200 bg-gray-50 px-2 py-1.5">
        <ToolbarBtn onMouseDown={() => run('undo')} title="Undo"><RotateCcw className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('redo')} title="Redo"><RotateCw className="h-4 w-4" /></ToolbarBtn>
        <Divider />
        <ToolbarBtn onMouseDown={() => run('formatBlock', 'H2')} title="Heading 2"><Heading2 className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('formatBlock', 'H3')} title="Heading 3"><Heading3 className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('formatBlock', 'P')} title="Paragraph"><span className="text-xs font-semibold">P</span></ToolbarBtn>
        <Divider />
        <ToolbarBtn onMouseDown={() => run('bold')} title="Bold"><Bold className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('italic')} title="Italic"><Italic className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('underline')} title="Underline"><Underline className="h-4 w-4" /></ToolbarBtn>
        <Divider />
        <ToolbarBtn onMouseDown={() => run('insertUnorderedList')} title="Bullet list"><List className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('insertOrderedList')} title="Numbered list"><ListOrdered className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('formatBlock', 'BLOCKQUOTE')} title="Quote"><Quote className="h-4 w-4" /></ToolbarBtn>
        <ToolbarBtn onMouseDown={() => run('formatBlock', 'PRE')} title="Code"><Code className="h-4 w-4" /></ToolbarBtn>
        <Divider />
        <ToolbarBtn
          onMouseDown={() => { saveSelection(); setShowLinkInput((s) => !s); setShowImageInput(false); }}
          title="Insert link"
          active={showLinkInput}
        >
          <Link2 className="h-4 w-4" />
        </ToolbarBtn>
        <ToolbarBtn
          onMouseDown={() => { saveSelection(); setShowImageInput((s) => !s); setShowLinkInput(false); }}
          title="Insert image"
          active={showImageInput}
        >
          <ImageIcon className="h-4 w-4" />
        </ToolbarBtn>
        <Divider />
        <ToolbarBtn onMouseDown={() => run('removeFormat')} title="Clear formatting"><Eraser className="h-4 w-4" /></ToolbarBtn>
      </div>

      {/* Inline link/image input row */}
      {showLinkInput && (
        <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50/60 px-2 py-1.5">
          <input
            type="url"
            autoFocus
            value={linkUrl}
            onChange={(e) => setLinkUrl(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); insertLink(); } if (e.key === 'Escape') setShowLinkInput(false); }}
            placeholder="https://..."
            className="flex-1 rounded-md border border-gray-300 px-2.5 py-1 text-xs focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <button type="button" onClick={insertLink} className="rounded-md bg-primary px-3 py-1 text-xs font-medium text-white hover:bg-primary-dark">Add</button>
          <button type="button" onClick={() => { setShowLinkInput(false); setLinkUrl(''); }} className="rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100">Cancel</button>
        </div>
      )}
      {showImageInput && (
        <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50/60 px-2 py-1.5">
          <input
            type="url"
            autoFocus
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); insertImage(); } if (e.key === 'Escape') setShowImageInput(false); }}
            placeholder="Image URL"
            className="flex-1 rounded-md border border-gray-300 px-2.5 py-1 text-xs focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <button type="button" onClick={insertImage} className="rounded-md bg-primary px-3 py-1 text-xs font-medium text-white hover:bg-primary-dark">Insert</button>
          <button type="button" onClick={() => { setShowImageInput(false); setImageUrl(''); }} className="rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100">Cancel</button>
        </div>
      )}

      {/* Editable surface */}
      <div
        ref={editorRef}
        contentEditable
        suppressContentEditableWarning
        data-placeholder={placeholder}
        onInput={emit}
        onPaste={handlePaste}
        onBlur={saveSelection}
        className="rte-surface prose prose-sm max-w-none px-4 py-3 text-sm leading-relaxed focus:outline-none [&_h2]:mb-1 [&_h2]:mt-3 [&_h2]:text-lg [&_h2]:font-semibold [&_h3]:mb-1 [&_h3]:mt-3 [&_h3]:text-base [&_h3]:font-semibold [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_blockquote]:border-l-4 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:italic [&_pre]:rounded-md [&_pre]:bg-gray-100 [&_pre]:px-3 [&_pre]:py-2 [&_pre]:font-mono [&_pre]:text-xs [&_a]:text-primary [&_a]:underline [&_img]:my-2 [&_img]:max-w-full [&_img]:rounded-md"
        style={{ minHeight }}
      />

      <style jsx>{`
        .rte-surface:empty::before {
          content: attr(data-placeholder);
          color: #9ca3af;
          pointer-events: none;
        }
      `}</style>
    </div>
  );
}

function ToolbarBtn({
  onMouseDown,
  title,
  active,
  children,
}: {
  onMouseDown: () => void;
  title: string;
  active?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      title={title}
      // mousedown (not click) so we don't lose the editor selection
      onMouseDown={(e) => { e.preventDefault(); onMouseDown(); }}
      className={`inline-flex h-7 w-7 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-200 hover:text-gray-900 ${active ? 'bg-gray-200 text-gray-900' : ''}`}
    >
      {children}
    </button>
  );
}

function Divider() {
  return <span className="mx-1 h-5 w-px bg-gray-300" />;
}
