import { h, render } from "preact";
import type { ComponentChild } from "preact";
import { registry } from "airway-islands-registry";

// Island runtime: scans [data-island] mount points, parses the adjacent
// JSON script, and renders the registered component. The registry itself is
// generated at build time from app/assets/js/islands/ — see
// lib/jsbuild/registry.go.

type IslandComponent = (props: Record<string, unknown>) => ComponentChild;

function mountIslands(): void {
  const mounts = document.querySelectorAll<HTMLElement>("[data-island]");
  mounts.forEach((el) => {
    const name = el.dataset.island ?? "";
    const id = el.dataset.islandId ?? "";

    const component = (registry as Record<string, IslandComponent>)[name];
    if (!component) {
      console.error(`[island] no island registered for "${name}" — is there a matching file under app/assets/js/islands/?`);
      return;
    }

    let props: Record<string, unknown> = {};
    const dataEl = document.getElementById(`island-data-${id}`);
    if (dataEl && dataEl.textContent) {
      try {
        props = JSON.parse(dataEl.textContent) as Record<string, unknown>;
      } catch (err) {
        console.error(`[island] invalid JSON props for "${name}" (#island-data-${id})`, err);
        return;
      }
    }

    // h() builds the vnode without invoking the component; render() then
    // calls it inside Preact's component context, so hooks work.
    render(h(component, props), el);
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", mountIslands);
} else {
  mountIslands();
}
