import { useState } from "react";
import type { ComponentChildren } from "react";
// v8 hooks API lives in the ./legacy entry of react-table v9 (see the
// support matrix in PLAN.md); types and flexRender come from the main entry.
import { flexRender, type ColumnDef, type SortingState, type RowSelectionState } from "@tanstack/react-table";
import {
  useLegacyTable,
  getCoreRowModel,
  getSortedRowModel,
  getPaginationRowModel,
} from "@tanstack/react-table/legacy";

import { cx } from "./cx";
import { EmptyState } from "./empty-state";
import { Pagination } from "./pagination";

export type DataTableProps<T> = {
  columns: ColumnDef<T, any>[];
  data: T[];
  /** page size; omit to show everything on one page */
  pageSize?: number;
  loading?: boolean;
  /** row selection via a leading checkbox column */
  selectable?: boolean;
  onSelectionChange?: (selected: Record<string, boolean>) => void;
  onRowClick?: (row: T) => void;
  empty?: { title: ComponentChildren; description?: ComponentChildren };
};

export function DataTable<T>(props: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [selection, setSelection] = useState<RowSelectionState>({});

  const selectable = props.selectable ?? false;
  const columns = selectable ? [selectColumn<T>(), ...props.columns] : props.columns;

  const table = useLegacyTable<T>({
    data: props.data,
    columns,
    state: { sorting, rowSelection: selection },
    onSortingChange: setSorting,
    onRowSelectionChange: (updater) => {
      setSelection((prev) => {
        const next = typeof updater === "function" ? updater(prev) : updater;
        props.onSelectionChange?.(next);
        return next;
      });
    },
    enableRowSelection: selectable,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    ...(props.pageSize ? { getPaginationRowModel: getPaginationRowModel() } : {}),
    initialState: props.pageSize ? { pagination: { pageSize: props.pageSize } } : undefined,
  });

  const rows = table.getRowModel().rows;
  const pageCount = props.pageSize ? table.getPageCount() : 1;
  const pageIndex = props.pageSize ? table.getState().pagination.pageIndex : 0;

  return (
    <div>
      <div class="aw-table-wrap">
        <table class="aw-table">
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr>
                {hg.headers.map((h) => {
                  const sortable = h.column.getCanSort();
                  const dir = h.column.getIsSorted();
                  return (
                    <th
                      key={h.id}
                      class={cx(sortable && "aw-sortable")}
                      onClick={sortable ? h.column.getToggleSortingHandler() : undefined}
                      aria-sort={dir === "asc" ? "ascending" : dir === "desc" ? "descending" : undefined}
                    >
                      {flexRender(h.column.columnDef.header, h.getContext())}
                      {dir === "asc" ? " ▲" : dir === "desc" ? " ▼" : ""}
                    </th>
                  );
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={row.id}
                onClick={props.onRowClick ? () => props.onRowClick!(row.original) : undefined}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {!props.loading && rows.length === 0 && (
          <EmptyState
            icon="◎"
            title={props.empty?.title ?? "Nothing here yet"}
            description={props.empty?.description}
          />
        )}
        {props.loading && (
          <div class="aw-table-loading">
            <div class="aw-spinner" aria-label="loading" />
          </div>
        )}
      </div>
      {props.pageSize && pageCount > 1 && (
        <Pagination
          page={pageIndex + 1}
          pageCount={pageCount}
          onPageChange={(p) => table.setPageIndex(p - 1)}
        />
      )}
    </div>
  );
}

function selectColumn<T>(): ColumnDef<T, any> {
  return {
    id: "__select",
    enableSorting: false,
    header: ({ table }) => (
      <input
        type="checkbox"
        aria-label="select all rows"
        checked={table.getIsAllRowsSelected()}
        onChange={table.getToggleAllRowsSelectedHandler()}
      />
    ),
    cell: ({ row }) => (
      <input
        type="checkbox"
        aria-label="select row"
        checked={row.getIsSelected()}
        onChange={row.getToggleSelectedHandler()}
        onClick={(e) => e.stopPropagation()}
      />
    ),
  };
}
