---
layout: home

hero:
  name: Airway
  text: Go 全栈 API 框架
  tagline: 灵感来自 Ruby on Rails —— Gin、方言感知的 SQL 构建器、基于泛型的 repo/ORM、SQL 迁移、统一存储与可插拔的 Engine。
  actions:
    - theme: brand
      text: 快速上手
      link: /zh-CN/cli-standalone
    - theme: alt
      text: GitHub 仓库
      link: https://github.com/daqing/airway

features:
  - title: 一个 CLI 搞定一切
    details: 通过 `go install github.com/daqing/airway@latest` 安装 —— 脚手架项目与 Engine、生成代码、执行迁移。
  - title: PostgreSQL / MySQL / SQLite
    details: 一套代码支持三种数据库，驱动在运行时根据 DSN 自动推断。
  - title: Engine 扩展机制
    details: Rails Engine 风格的功能模块，以独立 Go module 分发，一行空白导入即可启用。
  - title: 服务端渲染视图
    details: 基于 templ 的类型安全 HTML，内置 WebSocket Hub 与统一的本地 / S3 / R2 / COS 存储。
---
