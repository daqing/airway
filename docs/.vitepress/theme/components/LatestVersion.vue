<script setup lang="ts">
import { onMounted, ref } from 'vue'

withDefaults(defineProps<{ label?: string }>(), {
  label: 'Latest release',
})

// Injected at docs build time from `git tag` (see config.mts), so the badge
// renders even when the GitHub API is unreachable from the browser.
declare const __AIRWAY_LATEST_TAG__: string

const version = ref(__AIRWAY_LATEST_TAG__)

function compareTags(a: string, b: string): number {
  const pa = a.replace(/^v/, '').split(/[.-]/).map((x) => parseInt(x, 10) || 0)
  const pb = b.replace(/^v/, '').split(/[.-]/).map((x) => parseInt(x, 10) || 0)
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const d = (pa[i] || 0) - (pb[i] || 0)
    if (d !== 0) return d
  }
  return 0
}

onMounted(async () => {
  try {
    const res = await fetch('https://api.github.com/repos/daqing/airway/tags?per_page=100')
    if (!res.ok) return
    const tags: { name: string }[] = await res.json()
    if (!Array.isArray(tags) || tags.length === 0) return
    const latest = tags.map((t) => t.name).sort(compareTags).at(-1) ?? ''
    // Only upgrade: the build-time value stays when the API is rate-limited
    // or reports nothing newer.
    if (latest && compareTags(latest, version.value || '0') > 0) {
      version.value = latest
    }
  } catch {
    // GitHub API unavailable (offline, rate limit): keep the build-time value.
  }
})
</script>

<template>
  <a
    v-if="version"
    class="latest-version"
    :href="`https://github.com/daqing/airway/tree/${version}`"
    target="_blank"
    rel="noopener"
    :title="`GitHub: ${version}`"
  >
    <span class="dot" aria-hidden="true"></span>
    {{ label }} <strong>{{ version }}</strong>
  </a>
</template>

<style scoped>
.latest-version {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 12px;
  padding: 3px 12px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  color: var(--vp-c-text-1);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.4;
  white-space: nowrap;
  text-decoration: none;
  transition: border-color 0.25s;
}

.latest-version:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.latest-version .dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: var(--vp-c-brand-1);
}
</style>
