import { useQuery, useQueryClient } from "@tanstack/preact-query";

type ItemsResp = {
  ok: boolean;
  data: { items: { id: number; name: string }[]; requestCount: number };
};

function ItemList() {
  const q = useQuery({
    queryKey: ["items"],
    queryFn: async (): Promise<ItemsResp> => {
      const r = await fetch("/api/items");
      return await r.json();
    },
  });

  if (q.isPending) return <p>loading…</p>;
  if (q.isError) return <p>error: {String(q.error)}</p>;

  return (
    <div>
      <ul>
        {q.data.data.items.map((it) => (
          <li key={it.id}>{it.name}</li>
        ))}
      </ul>
      <p class="hint">server requestCount = {q.data.data.requestCount}</p>
    </div>
  );
}

export function QueryDemo() {
  const qc = useQueryClient();
  return (
    <div>
      <ItemList />
      <ItemList />
      <p class="hint">
        two identical components render from one request (shared cache) —
        requestCount stays 1 until invalidated
      </p>
      <button onClick={() => qc.invalidateQueries({ queryKey: ["items"] })}>
        invalidate and refetch
      </button>
    </div>
  );
}
