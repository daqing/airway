import { render } from "preact";
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { QueryDemo } from "./demos/query-demo";
import { TableDemo } from "./demos/table-demo";
import { VirtualDemo } from "./demos/virtual-demo";
import { FormDemo } from "./demos/form-demo";

const client = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={client}>
      <h1>Airway frontend spike — Phase 0</h1>
      <section>
        <h2>@tanstack/preact-query (official Preact adapter)</h2>
        <QueryDemo />
      </section>
      <section>
        <h2>@tanstack/react-table via preact/compat</h2>
        <TableDemo />
      </section>
      <section>
        <h2>@tanstack/react-virtual via preact/compat</h2>
        <VirtualDemo />
      </section>
      <section>
        <h2>react-hook-form via preact/compat</h2>
        <FormDemo />
      </section>
    </QueryClientProvider>
  );
}

render(<App />, document.getElementById("app")!);
