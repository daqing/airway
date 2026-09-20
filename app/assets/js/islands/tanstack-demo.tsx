import type { ComponentChild } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import { DataTable, useToast } from "../ui";

// Homepage demo island for the frontend section: the interactive DataTable
// (sorting, pagination, row toasts) is built on TanStack Table, showing how
// islands lean on the React ecosystem through airway-ui.
type Layer = { layer: string; component: string; from: string };

const stack: Layer[] = [
  { layer: "Tables", component: "DataTable", from: "@tanstack/react-table" },
  { layer: "Queries", component: "useApiQuery", from: "@tanstack/preact-query" },
  { layer: "Forms", component: "Form + Field", from: "react-hook-form" },
  { layer: "Runtime", component: "islands", from: "preact/compat" },
];

const columns: ColumnDef<Layer, any>[] = [
  { accessorKey: "layer", header: "Layer" },
  { accessorKey: "component", header: "airway-ui" },
  { accessorKey: "from", header: "Built on" },
];

export default function TanStackDemo(props: Record<string, unknown>): ComponentChild {
  const caption = typeof props.caption === "string" ? props.caption : "";
  const toast = useToast();

  return (
    <div class="tanstack-demo">
      <DataTable
        columns={columns}
        data={stack}
        pageSize={3}
        onRowClick={(row) => toast.show(`${row.component} — from ${row.from}`)}
      />
      {caption && <p class="tanstack-demo-note">{caption}</p>}
    </div>
  );
}
