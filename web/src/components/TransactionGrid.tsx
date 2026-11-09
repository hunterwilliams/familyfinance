import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AgGridReact } from 'ag-grid-react';
import type { ColDef, CellValueChangedEvent } from 'ag-grid-community';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import {
  api,
  formatTHB,
  satangToTHB,
  thbToSatang,
  type Account,
  type Category,
  type Member,
  type Transaction,
} from '../api';

ModuleRegistry.registerModules([AllCommunityModule]);

interface Row extends Transaction {
  amount_thb: number;
}

interface Props {
  members: Member[];
  accounts: Account[];
  categories: Category[];
}

export default function TransactionGrid({ members, accounts, categories }: Props) {
  const safeMembers = members ?? [];
  const safeAccounts = accounts ?? [];
  const safeCategories = categories ?? [];
  const [rows, setRows] = useState<Row[]>([]);
  const [pending, setPending] = useState<Map<string, Partial<Row>>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const gridRef = useRef<AgGridReact<Row>>(null);

  const memberMap = useMemo(() => Object.fromEntries(safeMembers.map((m) => [m.id, m.name])), [safeMembers]);
  const accountMap = useMemo(() => Object.fromEntries(safeAccounts.map((a) => [a.id, a.name])), [safeAccounts]);
  const categoryMap = useMemo(() => Object.fromEntries(safeCategories.map((c) => [c.id, c.name])), [safeCategories]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const txns = await api.getTransactions();
      setRows(
        txns.map((t) => ({
          ...t,
          transaction_date: t.transaction_date.slice(0, 10),
          amount_thb: satangToTHB(t.amount_satang),
        })),
      );
      setPending(new Map());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const columnDefs = useMemo<ColDef<Row>[]>(
    () => [
      { field: 'transaction_date', headerName: 'Date', editable: true, width: 120 },
      { field: 'description', headerName: 'Description', editable: true, flex: 1, minWidth: 160 },
      {
        field: 'amount_thb',
        headerName: 'Amount (THB)',
        editable: true,
        width: 130,
        valueFormatter: (p) => (p.value != null ? formatTHB(thbToSatang(p.value)) : ''),
      },
      {
        field: 'paid_by_member_id',
        headerName: 'Paid By',
        editable: true,
        width: 130,
        cellEditor: 'agSelectCellEditor',
        cellEditorParams: { values: ['joint', ...safeMembers.map((m) => m.id)] },
        valueFormatter: (p) => {
          const row = p.data;
          if (row?.is_joint || p.value === 'joint') return 'Joint';
          return memberMap[p.value] || p.value;
        },
      },
      {
        field: 'account_id',
        headerName: 'Payment method',
        editable: true,
        width: 160,
        cellEditor: 'agSelectCellEditor',
        cellEditorParams: { values: ['', ...safeAccounts.map((a) => a.id)] },
        valueFormatter: (p) => (p.value ? accountMap[p.value] || p.value : '—'),
      },
      {
        field: 'category_id',
        headerName: 'Category',
        editable: true,
        width: 130,
        cellEditor: 'agSelectCellEditor',
        cellEditorParams: { values: safeCategories.map((c) => c.id) },
        valueFormatter: (p) => (p.value ? categoryMap[p.value] || p.value : ''),
      },
    ],
    [safeMembers, safeAccounts, safeCategories, memberMap, accountMap, categoryMap],
  );

  const onCellValueChanged = (e: CellValueChangedEvent<Row>) => {
    const id = e.data.id;
    setPending((prev) => {
      const next = new Map(prev);
      next.set(id, { ...next.get(id), ...e.data });
      return next;
    });
    setMessage('');
  };

  const saveChanges = async () => {
    if (pending.size === 0) return;
    setError('');
    try {
      const updates = Array.from(pending.entries()).map(([id, row]) => {
        const joint = row.paid_by_member_id === 'joint' || row.is_joint;
        return {
          id,
          transaction_date: row.transaction_date,
          description: row.description,
          amount_satang: row.amount_thb != null ? thbToSatang(row.amount_thb) : undefined,
          paid_by_member_id: joint ? null : row.paid_by_member_id,
          is_joint: joint,
          account_id: row.account_id || null,
          category_id: row.category_id || null,
        };
      });
      await api.bulkUpdateTransactions(updates);
      setMessage(`Saved ${updates.length} change(s)`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed');
    }
  };

  const addRow = async () => {
    if (safeMembers.length === 0) return;
    setError('');
    try {
      const today = new Date().toISOString().slice(0, 10);
      await api.createTransaction({
        account_id: null,
        paid_by_member_id: safeMembers[0].id,
        is_joint: false,
        transaction_date: today,
        description: 'New expense',
        amount_satang: 0,
        category_id: safeCategories[0]?.id,
      });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Create failed');
    }
  };

  if (loading) return <div className="panel">Loading transactions...</div>;

  return (
    <div className="panel">
      <h2>Transactions</h2>
      <div className="toolbar">
        <button type="button" onClick={addRow}>Add row</button>
        <button type="button" onClick={saveChanges} disabled={pending.size === 0}>
          Save changes ({pending.size})
        </button>
        <button type="button" className="secondary" onClick={load}>Refresh</button>
      </div>
      {error && <p className="error">{error}</p>}
      {message && <p className="success">{message}</p>}
      <div className="grid-wrap ag-theme-quartz">
        <AgGridReact
          ref={gridRef}
          rowData={rows}
          columnDefs={columnDefs}
          defaultColDef={{ sortable: true, filter: true, resizable: true }}
          onCellValueChanged={onCellValueChanged}
          stopEditingWhenCellsLoseFocus
        />
      </div>
    </div>
  );
}
