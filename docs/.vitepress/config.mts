import { execSync } from 'node:child_process'
import { defineConfig } from 'vitepress'

// Latest release tag, read from git at docs build time and injected into the
// LatestVersion component. Falls back to an empty string (badge hidden) when
// git or tags are unavailable.
function latestTag(): string {
  try {
    const out = execSync('git tag --list', {
      stdio: ['ignore', 'pipe', 'ignore'],
    }).toString()

    const compare = (a: string, b: string): number => {
      const pa = a.replace(/^v/, '').split(/[.-]/).map((x) => parseInt(x, 10) || 0)
      const pb = b.replace(/^v/, '').split(/[.-]/).map((x) => parseInt(x, 10) || 0)
      for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
        const d = (pa[i] || 0) - (pb[i] || 0)
        if (d !== 0) return d
      }
      return 0
    }

    return out.split('\n').map((s) => s.trim()).filter(Boolean).sort(compare).at(-1) ?? ''
  } catch {
    return ''
  }
}

export default defineConfig({
  base: '/airway/',
  title: 'Airway',
  description: 'A full-stack API framework in Go, inspired by Ruby on Rails',
  cleanUrls: true,
  ignoreDeadLinks: true,

  vite: {
    define: {
      __AIRWAY_LATEST_TAG__: JSON.stringify(latestTag()),
    },
  },

  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/daqing/airway' },
    ],
  },

  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/cli-standalone' },
          { text: 'English', link: '/', activeMatch: '^/(?!zh-CN/)' },
          { text: '简体中文', link: '/zh-CN/' },
        ],
        sidebar: [
          {
            text: 'Getting Started',
            items: [
              { text: 'The Standalone CLI', link: '/cli-standalone' },
              { text: 'CLI Scaffolding Guide', link: '/cli' },
            ],
          },
          {
            text: 'Guide',
            items: [
              { text: 'Plugins', link: '/plugin' },
              { text: 'Storage', link: '/storage' },
              { text: 'Templates', link: '/template' },
            ],
          },
        ],
      },
    },
    'zh-CN': {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh-CN/',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh-CN/cli-standalone' },
          { text: 'English', link: '/' },
          { text: '简体中文', link: '/zh-CN/', activeMatch: '^/zh-CN/' },
        ],
        sidebar: [
          {
            text: '快速上手',
            items: [
              { text: '独立命令行工具', link: '/zh-CN/cli-standalone' },
              { text: 'CLI 脚手架指南', link: '/zh-CN/cli' },
            ],
          },
          {
            text: '指南',
            items: [
              { text: 'Plugin 扩展机制', link: '/zh-CN/plugin' },
              { text: 'SQL 构建器', link: '/zh-CN/sql-builder' },
              { text: '存储', link: '/zh-CN/storage' },
              { text: '模板', link: '/zh-CN/template' },
              { text: '私有仓库配置', link: '/zh-CN/setup-private-registry' },
            ],
          },
        ],
      },
    },
  },
})
