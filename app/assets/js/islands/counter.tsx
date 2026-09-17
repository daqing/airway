import { useState } from "react";
import type { ComponentChild } from "react";

// Reference island: props arrive as JSON from the server-rendered page
// (see the Counter mount in app/views/home). Islands default-export their
// component; the registry name is this file's path without extension.
export default function Counter(props: Record<string, unknown>): ComponentChild {
  const label = typeof props.label === "string" ? props.label : "counter";
  const start = typeof props.start === "number" ? props.start : 0;
  const [count, setCount] = useState(start);

  return (
    <div class="island-counter">
      <span class="island-label">{label}</span>
      <button type="button" onClick={() => setCount(count - 1)} aria-label="decrease">−</button>
      <strong>{count}</strong>
      <button type="button" onClick={() => setCount(count + 1)} aria-label="increase">+</button>
    </div>
  );
}
