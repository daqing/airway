<script setup lang="ts">
import DefaultTheme from 'vitepress/theme'
import { useData } from 'vitepress'
import LatestVersion from './components/LatestVersion.vue'
import InstallCommand from './components/InstallCommand.vue'

const { Layout } = DefaultTheme
const { lang } = useData()
</script>

<template>
  <Layout>
    <template #nav-bar-content-after>
      <LatestVersion :label="lang.startsWith('zh') ? '最新版本' : 'Latest'" />
    </template>
    <template #home-hero-actions-after>
      <InstallCommand />
    </template>
  </Layout>
</template>
