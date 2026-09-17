import { useEffect, useRef } from "react";
import type { ComponentChildren } from "react";

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])';

export type ModalProps = {
  open: boolean;
  onClose: () => void;
  title?: ComponentChildren;
  footer?: ComponentChildren;
  children?: ComponentChildren;
};

export function Modal(props: ModalProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!props.open) return;

    const panel = panelRef.current;
    const previouslyFocused = document.activeElement as HTMLElement | null;
    // focus the first control (or the panel) on open
    const first = panel?.querySelector<HTMLElement>(FOCUSABLE);
    (first ?? panel)?.focus();

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        props.onClose();
        return;
      }
      if (e.key !== "Tab" || !panel) return;
      // keep Tab cycling inside the dialog
      const focusables = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE));
      if (focusables.length === 0) return;
      const firstEl = focusables[0];
      const lastEl = focusables[focusables.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === firstEl || active === panel)) {
        e.preventDefault();
        lastEl.focus();
      } else if (!e.shiftKey && active === lastEl) {
        e.preventDefault();
        firstEl.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = "";
      previouslyFocused?.focus?.();
    };
  }, [props.open]);

  if (!props.open) return null;

  return (
    <div
      class="aw-modal-overlay"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) props.onClose();
      }}
    >
      <div
        ref={panelRef}
        class="aw-modal"
        role="dialog"
        aria-modal="true"
        aria-label={typeof props.title === "string" ? props.title : "dialog"}
        tabindex={-1}
      >
        {props.title && (
          <div class="aw-modal-header">
            <div class="aw-modal-title">{props.title}</div>
            <button type="button" class="aw-modal-close" aria-label="close" onClick={props.onClose}>
              ×
            </button>
          </div>
        )}
        <div class="aw-modal-body">{props.children}</div>
        {props.footer && <div class="aw-modal-footer">{props.footer}</div>}
      </div>
    </div>
  );
}
