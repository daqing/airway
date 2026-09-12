<script setup lang="ts">
import { ref } from 'vue'
import { useData } from 'vitepress'

const command = 'go install github.com/daqing/airway@latest'

const { lang } = useData()
const copied = ref(false)

let timer: ReturnType<typeof setTimeout> | undefined

async function copy() {
  try {
    await navigator.clipboard.writeText(command)
  } catch {
    // Clipboard API unavailable (non-secure context): legacy fallback.
    const ta = document.createElement('textarea')
    ta.value = command
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }

  copied.value = true
  clearTimeout(timer)
  timer = setTimeout(() => (copied.value = false), 1600)
}
</script>

<template>
  <div class="install-command">
    <code>{{ command }}</code>
    <button
      type="button"
      class="copy"
      :aria-label="lang.startsWith('zh') ? '复制安装命令' : 'Copy install command'"
      :title="lang.startsWith('zh') ? (copied ? '已复制' : '复制') : (copied ? 'Copied' : 'Copy')"
      @click="copy"
    >
      <svg v-if="!copied" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
        <rect x="8" y="8" width="12" height="13" rx="2"></rect>
        <path d="M15 8V3H3v13h5"></path>
      </svg>
      <svg v-else width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
        <path d="m5 12 4 4L19 6"></path>
      </svg>
    </button>
  </div>
</template>

<style scoped>
.install-command {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  margin: 6px;
  padding: 0 6px 0 16px;
  height: 40px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 20px;
  background-color: var(--vp-c-bg-alt);
  transition: border-color 0.25s;
}

.install-command:hover {
  border-color: var(--vp-c-brand-1);
}

.install-command code {
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  color: var(--vp-c-text-1);
  white-space: nowrap;
  user-select: all;
}

.install-command .copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  color: var(--vp-c-text-2);
  cursor: pointer;
  transition: color 0.25s, background-color 0.25s;
}

.install-command .copy:hover {
  color: var(--vp-c-brand-1);
  background-color: var(--vp-c-bg-soft);
}
</style>
