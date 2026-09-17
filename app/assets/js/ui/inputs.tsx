import { forwardRef } from "react";
import type { ComponentChildren } from "react";

import { cx } from "./cx";

// Inputs are wrapped in forwardRef: react-hook-form's register() passes a
// ref that must reach the underlying DOM element, and compat (React
// semantics) does not forward refs through plain function components.

type ControlProps = {
  name?: string;
  value?: string;
  invalid?: boolean;
  disabled?: boolean;
  placeholder?: string;
  class?: string;
  id?: string;
};

export const Input = forwardRef<HTMLInputElement, ControlProps & {
  type?: string;
  onInput?: (e: Event) => void;
  onChange?: (e: Event) => void;
  [key: string]: unknown; // register() spread from react-hook-form
}>((props, ref) => {
  const { invalid, class: cls, ...rest } = props;
  return (
    <input
      ref={ref}
      class={cx("aw-input", invalid && "aw-input-invalid", cls)}
      aria-invalid={invalid ? "true" : undefined}
      {...rest}
    />
  );
});

export const Textarea = forwardRef<HTMLTextAreaElement, ControlProps & {
  rows?: number;
  onInput?: (e: Event) => void;
  [key: string]: unknown;
}>((props, ref) => {
  const { invalid, class: cls, ...rest } = props;
  return (
    <textarea
      ref={ref}
      class={cx("aw-textarea", invalid && "aw-textarea-invalid", cls)}
      aria-invalid={invalid ? "true" : undefined}
      {...rest}
    />
  );
});

export const Select = forwardRef<HTMLSelectElement, ControlProps & {
  children?: ComponentChildren;
  onChange?: (e: Event) => void;
  [key: string]: unknown;
}>((props, ref) => {
  const { invalid, class: cls, children, ...rest } = props;
  return (
    <select
      ref={ref}
      class={cx("aw-select", invalid && "aw-select-invalid", cls)}
      aria-invalid={invalid ? "true" : undefined}
      {...rest}
    >
      {children}
    </select>
  );
});

export const Checkbox = forwardRef<HTMLInputElement, {
  name?: string;
  checked?: boolean;
  disabled?: boolean;
  label?: ComponentChildren;
  onChange?: (e: Event) => void;
  [key: string]: unknown;
}>((props, ref) => {
  const { label, ...rest } = props;
  return (
    <label class="aw-check">
      <input ref={ref} type="checkbox" {...rest} />
      {label}
    </label>
  );
});

export const Radio = forwardRef<HTMLInputElement, {
  name?: string;
  checked?: boolean;
  disabled?: boolean;
  label?: ComponentChildren;
  onChange?: (e: Event) => void;
  [key: string]: unknown;
}>((props, ref) => {
  const { label, ...rest } = props;
  return (
    <label class="aw-check">
      <input ref={ref} type="radio" {...rest} />
      {label}
    </label>
  );
});
