'use client';

import { useState, useRef, useCallback, type DragEvent } from 'react';
import { Upload, X, FileImage, File as FileIcon, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface UploadedFile {
  id: string;
  name: string;
  size: number;
  type: string;
  /** URL for preview (images) — either a data URL or server URL */
  url?: string;
  /** Relative path for storage in DB (e.g. "products/abc123.jpg"). Falls back to url if unset. */
  path?: string;
  /** Upload progress 0–100, undefined if not yet started */
  progress?: number;
  /** Error message if upload failed */
  error?: string;
}

export interface FileUploadProps {
  /** Currently uploaded/selected files */
  files: UploadedFile[];
  /** Called when files are added via drop or browse */
  onFilesAdded: (files: File[]) => void;
  /** Called when a file is removed */
  onFileRemoved: (id: string) => void;
  /** Accepted MIME types, e.g. "image/*" or "image/png,image/jpeg" */
  accept?: string;
  /** Allow multiple files */
  multiple?: boolean;
  /** Max file size in bytes */
  maxSize?: number;
  /** Max number of files */
  maxFiles?: number;
  /** Disabled state */
  disabled?: boolean;
  /** Custom class for the dropzone */
  className?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function isImage(type: string): boolean {
  return type.startsWith('image/');
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function FileUpload({
  files,
  onFilesAdded,
  onFileRemoved,
  accept,
  multiple = false,
  maxSize,
  maxFiles,
  disabled = false,
  className,
}: FileUploadProps) {
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const validate = useCallback(
    (incoming: File[]): File[] => {
      setError(null);
      let valid = incoming;

      // Check max files
      if (maxFiles && files.length + valid.length > maxFiles) {
        setError(`Maximum ${maxFiles} file${maxFiles > 1 ? 's' : ''} allowed`);
        valid = valid.slice(0, maxFiles - files.length);
        if (valid.length === 0) return [];
      }

      // Check sizes
      if (maxSize) {
        const oversized = valid.filter((f) => f.size > maxSize);
        if (oversized.length > 0) {
          setError(`File too large. Maximum size: ${formatSize(maxSize)}`);
          valid = valid.filter((f) => f.size <= maxSize);
        }
      }

      return valid;
    },
    [files.length, maxFiles, maxSize],
  );

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    if (!disabled) setDragging(true);
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    if (disabled) return;

    const droppedFiles = Array.from(e.dataTransfer.files);
    const valid = validate(droppedFiles);
    if (valid.length > 0) onFilesAdded(valid);
  }

  function handleBrowse() {
    if (!disabled) inputRef.current?.click();
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files || []);
    const valid = validate(selected);
    if (valid.length > 0) onFilesAdded(valid);
    // Reset so the same file can be re-selected
    e.target.value = '';
  }

  const atLimit = maxFiles ? files.length >= maxFiles : false;

  return (
    <div className={cn('space-y-3', className)}>
      {/* Drop Zone */}
      {!atLimit && (
        <div
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={handleBrowse}
          className={cn(
            'flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-8 transition-colors',
            dragging
              ? 'border-primary bg-primary/5'
              : 'border-border hover:border-primary/40 hover:bg-surface-hover',
            disabled && 'cursor-not-allowed opacity-50',
          )}
        >
          <Upload
            className={cn(
              'mb-3 h-8 w-8',
              dragging ? 'text-primary' : 'text-text-muted',
            )}
          />
          <p className="text-sm font-medium text-text">
            {dragging ? 'Drop files here' : 'Drag & drop files here'}
          </p>
          <p className="mt-1 text-xs text-text-muted">
            or{' '}
            <span className="font-medium text-primary">browse</span>
            {maxSize && ` (max ${formatSize(maxSize)})`}
          </p>

          <input
            ref={inputRef}
            type="file"
            accept={accept}
            multiple={multiple}
            onChange={handleInputChange}
            className="hidden"
          />
        </div>
      )}

      {/* Error */}
      {error && (
        <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
      )}

      {/* File List */}
      {files.length > 0 && (
        <ul className="space-y-2">
          {files.map((file) => (
            <li
              key={file.id}
              className="flex items-center gap-3 rounded-xl border border-border bg-surface p-3"
            >
              {/* Preview / icon */}
              {file.url && isImage(file.type) ? (
                <img
                  src={file.url}
                  alt={file.name}
                  className="h-12 w-12 flex-shrink-0 rounded-lg object-cover"
                />
              ) : (
                <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-lg bg-surface-hover">
                  {isImage(file.type) ? (
                    <FileImage className="h-5 w-5 text-text-muted" />
                  ) : (
                    <FileIcon className="h-5 w-5 text-text-muted" />
                  )}
                </div>
              )}

              {/* Info */}
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-text">{file.name}</p>
                <p className="text-xs text-text-muted">{formatSize(file.size)}</p>

                {/* Progress bar */}
                {file.progress !== undefined && file.progress < 100 && !file.error && (
                  <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                    <div
                      className="h-full rounded-full bg-primary transition-all duration-300"
                      style={{ width: `${file.progress}%` }}
                    />
                  </div>
                )}

                {/* Error */}
                {file.error && (
                  <p className="mt-0.5 text-xs text-red-600 dark:text-red-400">{file.error}</p>
                )}
              </div>

              {/* Status / remove */}
              {file.progress !== undefined && file.progress < 100 && !file.error ? (
                <Loader2 className="h-4 w-4 flex-shrink-0 animate-spin text-primary" />
              ) : (
                <button
                  onClick={() => onFileRemoved(file.id)}
                  disabled={disabled}
                  className="flex-shrink-0 rounded-lg p-1.5 text-text-muted transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 disabled:opacity-50"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
