import { createContext } from "react";
import { useCallback, useContext, useState } from "react";
import type { ComponentChildren } from "react";

type ToastKind = "info" | "success" | "error";

type ToastItem = {
  id: number;
  kind: ToastKind;
  message: string;
};

type ToastApi = {
  show: (message: string, kind?: ToastKind) => void;
  success: (message: string) => void;
  error: (message: string) => void;
};

const ToastContext = createContext<ToastApi | null>(null);
let nextToastId = 1;

/** ToastProvider mounts once around every island (see the runtime entry). */
export function ToastProvider(props: { children?: ComponentChildren }) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: number) => {
    setItems((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const show = useCallback((message: string, kind: ToastKind = "info") => {
    const id = nextToastId++;
    setItems((prev) => [...prev, { id, kind, message }]);
    setTimeout(() => dismiss(id), 3500);
  }, [dismiss]);

  const api: ToastApi = {
    show,
    success: (m) => show(m, "success"),
    error: (m) => show(m, "error"),
  };

  return (
    <ToastContext.Provider value={api}>
      {props.children}
      <div class="aw-toast-container" aria-live="polite">
        {items.map((t) => (
          <div key={t.id} class={`aw-toast aw-toast-${t.kind}`} role="status">
            <span>{t.message}</span>
            <button type="button" class="aw-toast-close" aria-label="dismiss" onClick={() => dismiss(t.id)}>
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (ctx) return ctx;
  // islands are always mounted inside ToastProvider by the runtime; the
  // no-op fallback keeps standalone mounts from crashing
  return { show: () => {}, success: () => {}, error: () => {} };
}
