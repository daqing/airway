import type { ComponentChild, ComponentChildren } from "react";
import { useId } from "react";

import { cx } from "./cx";

export type FieldProps = {
  label?: ComponentChild;
  // link the label to a control id you pass; when omitted, a generated id
  // is applied to the control via render prop
  htmlFor?: string;
  error?: ComponentChild;
  hint?: ComponentChild;
  required?: boolean;
  class?: string;
  children?: ComponentChild | ((id: string) => ComponentChild);
};

export function Field(props: FieldProps) {
  const autoId = useId();
  const id = props.htmlFor ?? `aw-field-${autoId}`;

  return (
    <div class={cx("aw-field", props.class)}>
      {props.label && (
        <label class="aw-field-label" htmlFor={id}>
          {props.label}
          {props.required && <span aria-hidden="true"> *</span>}
        </label>
      )}
      {typeof props.children === "function" ? props.children(id) : props.children}
      {props.error ? (
        <p class="aw-field-error" role="alert">{props.error}</p>
      ) : props.hint ? (
        <p class="aw-field-hint">{props.hint}</p>
      ) : null}
    </div>
  );
}
