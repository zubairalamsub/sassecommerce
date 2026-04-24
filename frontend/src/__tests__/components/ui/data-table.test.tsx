import { render, screen, fireEvent } from '@testing-library/react';
import DataTable, { type Column } from '@/components/ui/data-table';

interface TestRow {
  id: string;
  name: string;
  price: number;
  status: string;
}

const columns: Column<TestRow>[] = [
  { key: 'name', header: 'Name', sortable: true },
  { key: 'price', header: 'Price', sortable: true },
  {
    key: 'status',
    header: 'Status',
    cell: (row) => <span data-testid={`status-${row.id}`}>{row.status}</span>,
  },
];

const data: TestRow[] = [
  { id: '1', name: 'Alpha', price: 100, status: 'active' },
  { id: '2', name: 'Charlie', price: 300, status: 'inactive' },
  { id: '3', name: 'Bravo', price: 200, status: 'active' },
];

describe('DataTable', () => {
  test('renders column headers', () => {
    render(<DataTable columns={columns} data={data} rowKey={(r) => r.id} />);
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Price')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
  });

  test('renders row data', () => {
    render(<DataTable columns={columns} data={data} rowKey={(r) => r.id} />);
    expect(screen.getByText('Alpha')).toBeInTheDocument();
    expect(screen.getByText('Charlie')).toBeInTheDocument();
    expect(screen.getByText('Bravo')).toBeInTheDocument();
  });

  test('renders custom cell via cell prop', () => {
    render(<DataTable columns={columns} data={data} rowKey={(r) => r.id} />);
    expect(screen.getByTestId('status-1')).toHaveTextContent('active');
  });

  test('shows loading state', () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        rowKey={(r) => r.id}
        loading
        loadingText="Fetching..."
      />,
    );
    expect(screen.getByText('Fetching...')).toBeInTheDocument();
  });

  test('shows empty state', () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        rowKey={(r) => r.id}
        emptyTitle="Nothing here"
        emptyDescription="Add some items"
      />,
    );
    expect(screen.getByText('Nothing here')).toBeInTheDocument();
    expect(screen.getByText('Add some items')).toBeInTheDocument();
  });

  test('sorts by column ascending then descending', () => {
    render(
      <DataTable columns={columns} data={data} rowKey={(r) => r.id} pageSize={0} />
    );

    const nameHeader = screen.getByText('Name');
    fireEvent.click(nameHeader);

    const rows = screen.getAllByRole('row');
    // header row + 3 data rows
    expect(rows).toHaveLength(4);
    // After ascending sort: Alpha, Bravo, Charlie
    const cells = rows.slice(1).map((r) => r.querySelector('td')?.textContent);
    expect(cells).toEqual(['Alpha', 'Bravo', 'Charlie']);

    // Click again for descending
    fireEvent.click(nameHeader);
    const cells2 = screen
      .getAllByRole('row')
      .slice(1)
      .map((r) => r.querySelector('td')?.textContent);
    expect(cells2).toEqual(['Charlie', 'Bravo', 'Alpha']);
  });

  test('sorts numbers correctly', () => {
    render(
      <DataTable columns={columns} data={data} rowKey={(r) => r.id} pageSize={0} />
    );

    fireEvent.click(screen.getByText('Price'));

    const rows = screen.getAllByRole('row').slice(1);
    const prices = rows.map((r) => r.querySelectorAll('td')[1]?.textContent);
    expect(prices).toEqual(['100', '200', '300']);
  });

  test('search filters rows', () => {
    render(
      <DataTable
        columns={columns}
        data={data}
        rowKey={(r) => r.id}
        searchable
        pageSize={0}
      />,
    );

    const searchInput = screen.getByPlaceholderText('Search...');
    fireEvent.change(searchInput, { target: { value: 'alpha' } });

    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(2); // header + 1 match
    expect(screen.getByText('Alpha')).toBeInTheDocument();
    expect(screen.queryByText('Charlie')).not.toBeInTheDocument();
  });

  test('pagination shows correct page info', () => {
    const manyRows: TestRow[] = Array.from({ length: 25 }, (_, i) => ({
      id: String(i),
      name: `Item ${i}`,
      price: i * 10,
      status: 'active',
    }));

    render(
      <DataTable columns={columns} data={manyRows} rowKey={(r) => r.id} pageSize={10} />,
    );

    expect(screen.getByText(/Showing 1–10 of 25/)).toBeInTheDocument();
  });

  test('page navigation works', () => {
    const manyRows: TestRow[] = Array.from({ length: 25 }, (_, i) => ({
      id: String(i),
      name: `Item ${i}`,
      price: i * 10,
      status: 'active',
    }));

    render(
      <DataTable columns={columns} data={manyRows} rowKey={(r) => r.id} pageSize={10} />,
    );

    // Go to page 2
    fireEvent.click(screen.getByText('2'));
    expect(screen.getByText(/Showing 11–20 of 25/)).toBeInTheDocument();

    // Go to page 3
    fireEvent.click(screen.getByText('3'));
    expect(screen.getByText(/Showing 21–25 of 25/)).toBeInTheDocument();
  });

  test('row click handler fires', () => {
    const onClick = jest.fn();
    render(
      <DataTable
        columns={columns}
        data={data}
        rowKey={(r) => r.id}
        onRowClick={onClick}
        pageSize={0}
      />,
    );

    fireEvent.click(screen.getByText('Alpha'));
    expect(onClick).toHaveBeenCalledWith(data[0]);
  });
});
