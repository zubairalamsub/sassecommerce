import { render, screen, fireEvent } from '@testing-library/react';
import FileUpload, { type UploadedFile } from '@/components/ui/file-upload';

const sampleFiles: UploadedFile[] = [
  { id: 'f1', name: 'photo.jpg', size: 1024 * 500, type: 'image/jpeg', url: '/photo.jpg' },
  { id: 'f2', name: 'document.pdf', size: 1024 * 1024 * 2, type: 'application/pdf' },
];

describe('FileUpload', () => {
  test('renders dropzone with instructions', () => {
    render(
      <FileUpload files={[]} onFilesAdded={() => {}} onFileRemoved={() => {}} />,
    );
    expect(screen.getByText('Drag & drop files here')).toBeInTheDocument();
    expect(screen.getByText(/browse/)).toBeInTheDocument();
  });

  test('renders file list', () => {
    render(
      <FileUpload files={sampleFiles} onFilesAdded={() => {}} onFileRemoved={() => {}} />,
    );
    expect(screen.getByText('photo.jpg')).toBeInTheDocument();
    expect(screen.getByText('document.pdf')).toBeInTheDocument();
    expect(screen.getByText('500.0 KB')).toBeInTheDocument();
    expect(screen.getByText('2.0 MB')).toBeInTheDocument();
  });

  test('calls onFileRemoved when remove button clicked', () => {
    const onRemoved = jest.fn();
    render(
      <FileUpload files={sampleFiles} onFilesAdded={() => {}} onFileRemoved={onRemoved} />,
    );
    // There should be 2 remove buttons (X icons)
    const removeButtons = screen.getAllByRole('button');
    fireEvent.click(removeButtons[0]);
    expect(onRemoved).toHaveBeenCalledWith('f1');
  });

  test('shows max size in dropzone', () => {
    render(
      <FileUpload
        files={[]}
        onFilesAdded={() => {}}
        onFileRemoved={() => {}}
        maxSize={5 * 1024 * 1024}
      />,
    );
    expect(screen.getByText(/max 5.0 MB/)).toBeInTheDocument();
  });

  test('hides dropzone when maxFiles reached', () => {
    render(
      <FileUpload
        files={sampleFiles}
        onFilesAdded={() => {}}
        onFileRemoved={() => {}}
        maxFiles={2}
      />,
    );
    expect(screen.queryByText('Drag & drop files here')).not.toBeInTheDocument();
  });

  test('shows dropzone when below maxFiles', () => {
    render(
      <FileUpload
        files={[sampleFiles[0]]}
        onFilesAdded={() => {}}
        onFileRemoved={() => {}}
        maxFiles={3}
      />,
    );
    expect(screen.getByText('Drag & drop files here')).toBeInTheDocument();
  });

  test('shows progress bar for uploading file', () => {
    const uploading: UploadedFile[] = [
      { id: 'u1', name: 'uploading.jpg', size: 1000, type: 'image/jpeg', progress: 45 },
    ];
    const { container } = render(
      <FileUpload files={uploading} onFilesAdded={() => {}} onFileRemoved={() => {}} />,
    );
    const progressBar = container.querySelector('[style*="width: 45%"]');
    expect(progressBar).toBeInTheDocument();
  });

  test('shows error message on file', () => {
    const errFile: UploadedFile[] = [
      { id: 'e1', name: 'bad.txt', size: 100, type: 'text/plain', error: 'Upload failed' },
    ];
    render(
      <FileUpload files={errFile} onFilesAdded={() => {}} onFileRemoved={() => {}} />,
    );
    expect(screen.getByText('Upload failed')).toBeInTheDocument();
  });

  test('renders image preview when url is provided', () => {
    const imgFile: UploadedFile[] = [
      { id: 'img1', name: 'pic.png', size: 500, type: 'image/png', url: '/test-img.png' },
    ];
    render(
      <FileUpload files={imgFile} onFilesAdded={() => {}} onFileRemoved={() => {}} />,
    );
    const img = screen.getByAltText('pic.png');
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute('src', '/test-img.png');
  });
});
