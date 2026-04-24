import { render, screen, fireEvent } from '@testing-library/react';
import Modal, { ConfirmDialog } from '@/components/ui/modal';

// Framer Motion AnimatePresence needs this
jest.mock('framer-motion', () => {
  const actual = jest.requireActual('framer-motion');
  return {
    ...actual,
    AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
    motion: {
      div: ({
        children,
        className,
        onClick,
        ...rest
      }: React.HTMLAttributes<HTMLDivElement> & Record<string, unknown>) => (
        <div className={className} onClick={onClick} {...filterDomProps(rest)}>
          {children}
        </div>
      ),
    },
  };
});

// Filter out framer-motion specific props that aren't valid DOM attributes
function filterDomProps(props: Record<string, unknown>) {
  const invalid = ['initial', 'animate', 'exit', 'transition', 'variants'];
  const clean: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(props)) {
    if (!invalid.includes(k)) clean[k] = v;
  }
  return clean;
}

describe('Modal', () => {
  test('renders when open', () => {
    render(
      <Modal open onClose={() => {}} title="Test Modal">
        <p>Modal content</p>
      </Modal>,
    );
    expect(screen.getByText('Test Modal')).toBeInTheDocument();
    expect(screen.getByText('Modal content')).toBeInTheDocument();
  });

  test('does not render when closed', () => {
    render(
      <Modal open={false} onClose={() => {}} title="Hidden">
        <p>Hidden content</p>
      </Modal>,
    );
    expect(screen.queryByText('Hidden')).not.toBeInTheDocument();
  });

  test('calls onClose when close button clicked', () => {
    const onClose = jest.fn();
    render(
      <Modal open onClose={onClose} title="Closable">
        <p>Body</p>
      </Modal>,
    );
    // The X button
    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[0]);
    expect(onClose).toHaveBeenCalled();
  });

  test('calls onClose on Escape key', () => {
    const onClose = jest.fn();
    render(
      <Modal open onClose={onClose} title="Escapable">
        <p>Body</p>
      </Modal>,
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalled();
  });

  test('renders description', () => {
    render(
      <Modal open onClose={() => {}} title="Title" description="Some description">
        <p>Body</p>
      </Modal>,
    );
    expect(screen.getByText('Some description')).toBeInTheDocument();
  });

  test('renders footer', () => {
    render(
      <Modal open onClose={() => {}} title="With Footer" footer={<button>Save</button>}>
        <p>Body</p>
      </Modal>,
    );
    expect(screen.getByText('Save')).toBeInTheDocument();
  });

  test('hides header when hideHeader is true', () => {
    render(
      <Modal open onClose={() => {}} title="Should be hidden" hideHeader>
        <p>Only body</p>
      </Modal>,
    );
    expect(screen.queryByText('Should be hidden')).not.toBeInTheDocument();
    expect(screen.getByText('Only body')).toBeInTheDocument();
  });
});

describe('ConfirmDialog', () => {
  test('renders message and buttons', () => {
    render(
      <ConfirmDialog
        open
        onClose={() => {}}
        onConfirm={() => {}}
        title="Please Confirm"
        message="Are you sure?"
        confirmLabel="Yes, do it"
      />,
    );
    expect(screen.getByText('Are you sure?')).toBeInTheDocument();
    expect(screen.getByText('Yes, do it')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  test('calls onConfirm when confirm button clicked', () => {
    const onConfirm = jest.fn();
    render(
      <ConfirmDialog
        open
        onClose={() => {}}
        onConfirm={onConfirm}
        message="Delete this?"
        confirmLabel="Delete"
      />,
    );
    fireEvent.click(screen.getByText('Delete'));
    expect(onConfirm).toHaveBeenCalled();
  });

  test('calls onClose when cancel button clicked', () => {
    const onClose = jest.fn();
    render(
      <ConfirmDialog
        open
        onClose={onClose}
        onConfirm={() => {}}
        message="Cancel me"
      />,
    );
    fireEvent.click(screen.getByText('Cancel'));
    expect(onClose).toHaveBeenCalled();
  });
});
