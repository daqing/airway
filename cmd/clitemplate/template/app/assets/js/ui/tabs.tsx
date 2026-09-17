import { useState } from "react";
import type { ComponentChildren } from "react";

import { cx } from "./cx";

export type TabItem = {
  id: string;
  label: ComponentChildren;
  content: ComponentChildren;
};

export function Tabs({ items }: { items: TabItem[] }) {
  const [active, setActive] = useState(items[0]?.id);

  const move = (dir: 1 | -1) => {
    const idx = items.findIndex((i) => i.id === active);
    const next = (idx + dir + items.length) % items.length;
    setActive(items[next].id);
    document.getElementById(`aw-tab-${items[next].id}`)?.focus();
  };

  return (
    <div class="aw-tabs">
      <div class="aw-tabs-list" role="tablist" onKeyDown={(e) => {
        if (e.key === "ArrowRight") { e.preventDefault(); move(1); }
        if (e.key === "ArrowLeft") { e.preventDefault(); move(-1); }
      }}>
        {items.map((item) => (
          <button
            key={item.id}
            id={`aw-tab-${item.id}`}
            type="button"
            role="tab"
            aria-selected={item.id === active}
            aria-controls={`aw-tabpanel-${item.id}`}
            tabindex={item.id === active ? 0 : -1}
            class={cx("aw-tab", item.id === active && "aw-tab-active")}
            onClick={() => setActive(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>
      {items.map((item) => (
        <div
          key={item.id}
          id={`aw-tabpanel-${item.id}`}
          role="tabpanel"
          aria-labelledby={`aw-tab-${item.id}`}
          hidden={item.id !== active}
          class="aw-tab-panel"
        >
          {item.content}
        </div>
      ))}
    </div>
  );
}
