import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/airway/',
  title: 'Airway',
  description: 'A full-stack API framework in Go, inspired by Ruby on Rails',
  cleanUrls: true,
  ignoreDeadLinks: true,

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
          { text: '简体中文', link: '/zh-CN/', ariaLabel: '切换到中文' },
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
              { text: 'Engines', link: '/engine' },
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
          { text: 'English', link: '/', ariaLabel: 'Switch to English' },
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
              { text: 'Engine 扩展机制', link: '/zh-CN/engine' },
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
