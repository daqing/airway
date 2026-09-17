import { useState } from "preact/hooks";
import { useForm } from "react-hook-form";

type FormData = { name: string; email: string };

export function FormDemo() {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormData>();
  const [submitted, setSubmitted] = useState<FormData | null>(null);

  return (
    <form
      onSubmit={handleSubmit((d) => {
        setSubmitted(d);
        reset();
      })}
    >
      <p>
        <input {...register("name", { required: "name is required" })} placeholder="name" />
        {errors.name && <span class="err"> {errors.name.message}</span>}
      </p>
      <p>
        <input
          {...register("email", {
            required: "email is required",
            pattern: { value: /^\S+@\S+\.\S+$/, message: "invalid email" },
          })}
          placeholder="email"
        />
        {errors.email && <span class="err"> {errors.email.message}</span>}
      </p>
      <button type="submit">submit</button>
      {submitted && (
        <p class="hint">submitted: {JSON.stringify(submitted)}</p>
      )}
      <p class="hint">submit empty to see validation errors</p>
    </form>
  );
}

