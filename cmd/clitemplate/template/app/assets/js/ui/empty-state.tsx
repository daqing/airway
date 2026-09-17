import type { ComponentChildren } from "react";

export type EmptyStateProps = {
  icon?: ComponentChildren;
  title: ComponentChildren;
  description?: ComponentChildren;
  action?: ComponentChildren;
};

export function EmptyState(props: EmptyStateProps) {
  return (
    <div class="aw-empty">
      {props.icon && <div class="aw-empty-icon" aria-hidden="true">{props.icon}</div>}
      <div class="aw-empty-title">{props.title}</div>
      {props.description && <div>{props.description}</div>}
      {props.action && <div>{props.action}</div>}
    </div>
  );
}
