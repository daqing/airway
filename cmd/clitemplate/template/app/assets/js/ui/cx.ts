// className merge helper — the single convention airway-ui uses for the
// `class` prop (Preact style): library classes first, caller's class last.
export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}
