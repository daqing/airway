import { cx } from "./cx";

export function Spinner({ class: cls }: { class?: string }) {
  return <div class={cx("aw-spinner", cls)} aria-hidden="true" />;
}
