import { useState } from "preact/hooks";
// react-table v9 replaced the v8 hooks API; "./legacy" keeps it intact.
// The v8 API is what the support matrix in PLAN.md commits to.
import { flexRender, type ColumnDef, type SortingState } from "@tanstack/react-table";
import {
  useLegacyTable,
  getCoreRowModel,
  getSortedRowModel,
  getPaginationRowModel,
} from "@tanstack/react-table/legacy";

type Row = { id: number; name: string; score: number };

const data: Row[] = [
  { id: 1, name: "alpha", score: 42 },
  { id: 2, name: "bravo", score: 17 },
  { id: 3, name: "charlie", score: 88 },
  { id: 4, name: "delta", score: 23 },
  { id: 5, name: "echo", score: 65 },
  { id: 6, name: "foxtrot", score: 9 },
  { id: 7, name: "golf", score: 71 },
  { id: 8, name: "hotel", score: 34 },
];

const columns: ColumnDef<Row>[] = [
  { accessorKey: "id", header: "ID" },
  { accessorKey: "name", header: "Name" },
  { accessorKey: "score", header: "Score" },
];

export function TableDemo() {
  const [sorting, setSorting] = useState<SortingState>([]);
  const table = useLegacyTable({
    data,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: 4 } },
  });

  const arrow: Record<string, string> = { asc: " ▲", desc: " ▼" };

  return (
    <div>
      <table>
        <thead>
          {table.getHeaderGroups().map((hg) => (
            <tr>
              {hg.headers.map((h) => (
                <th key={h.id} onClick={h.column.getToggleSortingHandler()}>
                  {flexRender(h.column.columnDef.header, h.getContext())}
                  {arrow[h.column.getIsSorted() as string] ?? ""}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <p>
        <button
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          prev
        </button>{" "}
        page {table.getState().pagination.pageIndex + 1} / {table.getPageCount()}{" "}
        <button onClick={() => table.nextPage()} disabled={!table.getCanNextPage()}>
          next
        </button>
      </p>
      <p class="hint">click a header to sort, use prev/next to paginate</p>
    </div>
  );
}
