import type { ComponentChildren } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";

import { cx } from "./cx";

/**
 * Form wires a react-hook-form instance (official Preact support) to the
 * form element and renders a submitting-aware submit button. Callers own
 * the useForm() instance and the field layout:
 *
 *   const form = useForm<PostData>();
 *   <Form form={form} onSubmit={save} submitLabel="Save">
 *     <Field label="Title" error={form.formState.errors.title?.message}>
 *       <Input {...form.register("title", { required: "required" })} />
 *     </Field>
 *   </Form>
 */
export function Form<T extends FieldValues>(props: {
  form: UseFormReturn<T>;
  onSubmit: (values: T) => void | Promise<void>;
  submitLabel?: ComponentChildren;
  secondary?: ComponentChildren;
  class?: string;
  children?: ComponentChildren;
}) {
  const { form } = props;
  const submitting = form.formState.isSubmitting;
  const handleSubmit = form.handleSubmit(async (values) => {
    await props.onSubmit(values as T);
  }) as unknown as (e: Event) => void;

  return (
    <form class={cx("aw-form", props.class)} novalidate onSubmit={handleSubmit}>
      {props.children}
      <div class="aw-form-actions">
        {props.secondary}
        <button type="submit" class={cx("aw-button", "aw-button-primary")} disabled={submitting}>
          {submitting ? "Saving…" : (props.submitLabel ?? "Submit")}
        </button>
      </div>
    </form>
  );
}
