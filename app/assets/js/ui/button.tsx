import type { ComponentChildren } from "react";

import { cx } from "./cx";
import { Spinner } from "./spinner";

export type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";

export type ButtonProps = {
  variant?: ButtonVariant;
  size?: "md" | "sm";
  type?: "button" | "submit" | "reset";
  loading?: boolean;
  disabled?: boolean;
  class?: string;
  children?: ComponentChildren;
  onClick?: (e: MouseEvent) => void;
};

export function Button(props: ButtonProps) {
  const {
    variant = "secondary",
    size = "md",
    type = "button",
    loading = false,
    disabled = false,
  } = props;

  return (
    <button
      type={type}
      class={cx("aw-button", `aw-button-${variant}`, size === "sm" && "aw-button-sm", props.class)}
      disabled={disabled || loading}
      onClick={props.onClick}
    >
      {loading && <Spinner />}
      {props.children}
    </button>
  );
}
