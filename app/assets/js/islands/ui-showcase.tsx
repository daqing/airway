import { useState } from "react";
import type { ComponentChildren } from "react";
import { useForm } from "react-hook-form";
import type { ColumnDef } from "@tanstack/react-table";

import {
  Button,
  Checkbox,
  DataTable,
  EmptyState,
  Field,
  Form,
  Input,
  Modal,
  Pagination,
  Radio,
  Select,
  Spinner,
  Tabs,
  Textarea,
  useApiQuery,
  useToast,
} from "../ui";

// The airway-ui showcase: every component, interactive, plus a CRUD demo
// built only from library components (list + create/edit modal + delete).

type Post = { id: number; title: string; author: string; views: number };

const seedPosts: Post[] = [
  { id: 1, title: "Shipping without Node", author: "daqing", views: 812 },
  { id: 2, title: "Islands over SPAs", author: "ada", views: 534 },
  { id: 3, title: "Generics all the way down", author: "lin", views: 401 },
  { id: 4, title: "One binary to deploy", author: "daqing", views: 276 },
  { id: 5, title: "templ + Preact", author: "grace", views: 190 },
];

export default function UIShowcase() {
  return (
    <div>
      <ButtonsDemo />
      <FormDemo />
      <CrudDemo />
      <OverlaysDemo />
      <TabsDemo />
      <MiscDemo />
      <DataLayerDemo />
    </div>
  );
}

function Section({ title, children }: { title: string; children: ComponentChildren }) {
  return (
    <section class="aw-showcase-section">
      <h2>{title}</h2>
      {children}
    </section>
  );
}

function ButtonsDemo() {
  const toast = useToast();
  return (
    <Section title="Buttons">
      <div class="aw-showcase-row">
        <Button variant="primary" onClick={() => toast.success("Primary clicked")}>Primary</Button>
        <Button onClick={() => toast.show("Secondary clicked")}>Secondary</Button>
        <Button variant="danger" onClick={() => toast.error("Danger clicked")}>Danger</Button>
        <Button variant="ghost">Ghost</Button>
        <Button size="sm" variant="primary">Small</Button>
        <Button loading>Loading</Button>
        <Button disabled>Disabled</Button>
      </div>
    </Section>
  );
}

type FormData = { title: string; email: string };

function FormDemo() {
  const toast = useToast();
  const form = useForm<FormData>();

  return (
    <Section title="Form (react-hook-form)">
      <div style="max-width: 420px;">
        <Form
          form={form}
          submitLabel="Subscribe"
          onSubmit={async (values) => {
            toast.success(`Subscribed ${values.email}`);
            form.reset();
          }}
        >
          <Field label="Title" required error={form.formState.errors.title?.message}>
            {(id) => (
              <Input
                id={id}
                placeholder="Ms / Mr / Dr"
                invalid={!!form.formState.errors.title}
                {...form.register("title", { required: "title is required" })}
              />
            )}
          </Field>
          <Field label="Email" required error={form.formState.errors.email?.message} hint="We never share it.">
            {(id) => (
              <Input
                id={id}
                type="email"
                placeholder="you@example.com"
                invalid={!!form.formState.errors.email}
                {...form.register("email", {
                  required: "email is required",
                  pattern: { value: /^\S+@\S+\.\S+$/, message: "invalid email" },
                })}
              />
            )}
          </Field>
        </Form>
      </div>
    </Section>
  );
}

function CrudDemo() {
  const toast = useToast();
  const [posts, setPosts] = useState<Post[]>(seedPosts);
  const [editing, setEditing] = useState<Post | "new" | null>(null);
  const form = useForm<{ title: string; author: string }>();

  const openNew = () => {
    form.reset({ title: "", author: "" });
    setEditing("new");
  };
  const openEdit = (post: Post) => {
    form.reset({ title: post.title, author: post.author });
    setEditing(post);
  };

  const save = (values: { title: string; author: string }) => {
    setPosts((prev) => {
      if (editing === "new") {
        const id = Math.max(0, ...prev.map((p) => p.id)) + 1;
        return [...prev, { id, title: values.title, author: values.author, views: 0 }];
      }
      const target = editing as Post;
      return prev.map((p) => (p.id === target.id ? { ...p, title: values.title, author: values.author } : p));
    });
    toast.success(editing === "new" ? "Post created" : "Post updated");
    setEditing(null);
  };

  const remove = (post: Post) => {
    setPosts((prev) => prev.filter((p) => p.id !== post.id));
    toast.show(`Deleted “${post.title}”`, "error");
  };

  const columns: ColumnDef<Post, any>[] = [
    { accessorKey: "title", header: "Title" },
    { accessorKey: "author", header: "Author" },
    { accessorKey: "views", header: "Views" },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <span style="display: inline-flex; gap: 6px;">
          <Button size="sm" onClick={() => openEdit(row.original)}>Edit</Button>
          <Button size="sm" variant="danger" onClick={() => remove(row.original)}>Delete</Button>
        </span>
      ),
    },
  ];

  return (
    <Section title="CRUD (DataTable + Modal + Form + Toast)">
      <div style="margin-bottom: 10px;">
        <Button variant="primary" onClick={openNew}>New post</Button>
      </div>
      <DataTable
        columns={columns}
        data={posts}
        pageSize={4}
        selectable
        empty={{ title: "No posts yet", description: "Create the first one." }}
      />

      <Modal
        open={editing !== null}
        onClose={() => setEditing(null)}
        title={editing === "new" ? "New post" : editing ? `Edit #${(editing as Post).id}` : ""}
      >
        <Form
          form={form}
          submitLabel={editing === "new" ? "Create" : "Save"}
          onSubmit={save}
          secondary={<Button onClick={() => setEditing(null)}>Cancel</Button>}
        >
          <Field label="Title" required error={form.formState.errors.title?.message}>
            {(id) => (
              <Input id={id} invalid={!!form.formState.errors.title} {...form.register("title", { required: "title is required" })} />
            )}
          </Field>
          <Field label="Author" required error={form.formState.errors.author?.message}>
            {(id) => (
              <Input id={id} invalid={!!form.formState.errors.author} {...form.register("author", { required: "author is required" })} />
            )}
          </Field>
        </Form>
      </Modal>
    </Section>
  );
}

function OverlaysDemo() {
  const [open, setOpen] = useState(false);
  const toast = useToast();

  return (
    <Section title="Modal & Toast">
      <div class="aw-showcase-row">
        <Button variant="primary" onClick={() => setOpen(true)}>Open modal</Button>
        <Button onClick={() => toast.success("Saved successfully")}>Success toast</Button>
        <Button variant="danger" onClick={() => toast.error("Something went wrong")}>Error toast</Button>
      </div>
      <Modal
        open={open}
        onClose={() => setOpen(false)}
        title="Confirm"
        footer={
          <>
            <Button onClick={() => setOpen(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => { setOpen(false); toast.success("Confirmed"); }}>Confirm</Button>
          </>
        }
      >
        <p style="margin: 0; line-height: 1.6;">
          Try the keyboard: <kbd>Tab</kbd> cycles inside the dialog, <kbd>Esc</kbd> closes it,
          and focus returns to the opener.
        </p>
      </Modal>
    </Section>
  );
}

function TabsDemo() {
  return (
    <Section title="Tabs">
      <Tabs
        items={[
          { id: "inputs", label: "Inputs", content: <InputsTab /> },
          { id: "notes", label: "Notes", content: <p style="margin: 0">Arrow keys move between tabs; the panels are plain children.</p> },
          { id: "third", label: "Third", content: <p style="margin: 0">Any ComponentChildren works as tab content.</p> },
        ]}
      />
    </Section>
  );
}

function InputsTab() {
  const [select, setSelect] = useState("b");
  return (
    <div style="display: grid; gap: 12px; max-width: 420px;">
      <Field label="Text" hint="Plain input">
        <Input placeholder="type here" />
      </Field>
      <Field label="Select">
        <Select value={select} onChange={(e) => setSelect((e.target as HTMLSelectElement).value)}>
          <option value="a">Option A</option>
          <option value="b">Option B</option>
          <option value="c">Option C</option>
        </Select>
      </Field>
      <Field label="Textarea">
        <Textarea rows={3} placeholder="multi-line" />
      </Field>
      <div class="aw-showcase-row">
        <Checkbox label="Checkbox" defaultChecked />
        <Radio name="demo-radio" label="Radio one" checked />
        <Radio name="demo-radio" label="Radio two" />
      </div>
    </div>
  );
}

function MiscDemo() {
  const [page, setPage] = useState(2);
  return (
    <Section title="Spinner / EmptyState / Pagination">
      <div class="aw-showcase-row">
        <Spinner />
        <div style="flex: 1; min-width: 260px; border: 1px dashed var(--aw-border, #d7dae0); border-radius: 8px;">
          <EmptyState icon="◎" title="Nothing here yet" description="EmptyState fills tables and lists." action={<Button size="sm">Add one</Button>} />
        </div>
      </div>
      <Pagination page={page} pageCount={5} onPageChange={setPage} />
    </Section>
  );
}

type DemoItems = { items: { id: number; name: string; stock: number }[] };

function DataLayerDemo() {
  const q = useApiQuery<DemoItems>(["ui-demo-items"], "/api/v1/ui-demo/items");

  return (
    <Section title="Data layer (TanStack Query + render envelope)">
      {q.isPending && <Spinner />}
      {q.isError && <EmptyState icon="✖" title="Request failed" description={String(q.error)} />}
      {q.isSuccess && (
        <ul style="margin: 0; padding-left: 18px; line-height: 1.9;">
          {q.data.items.map((it) => (
            <li key={it.id}>{it.name} — {it.stock} in stock</li>
          ))}
        </ul>
      )}
    </Section>
  );
}
