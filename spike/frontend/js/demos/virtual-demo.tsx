import { useRef } from "preact/hooks";
import { useVirtualizer } from "@tanstack/react-virtual";

export function VirtualDemo() {
  const parentRef = useRef<HTMLDivElement>(null);
  const rows = Array.from({ length: 10000 }, (_, i) => `row ${i}`);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 28,
    overscan: 8,
  });

  return (
    <div>
      <div
        ref={parentRef}
        style={{ height: "240px", overflow: "auto", border: "1px solid #ccc" }}
      >
        <div
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            position: "relative",
          }}
        >
          {virtualizer.getVirtualItems().map((vi) => (
            <div
              key={vi.key}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: `${vi.size}px`,
                transform: `translateY(${vi.start}px)`,
              }}
            >
              {rows[vi.index]}
            </div>
          ))}
        </div>
      </div>
      <p class="hint">10,000 rows, only visible ones in the DOM</p>
    </div>
  );
}
