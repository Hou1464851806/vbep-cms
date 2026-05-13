<script setup>
import { formatShortId } from '@/utils/format.js'
import '@/assets/styles/universal.css'

defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>

<template>
  <el-tooltip effect="dark" :content="id" placement="top" :show-after="500">
    <span class="monospace-id">{{ formatShortId(id) }}</span>
  </el-tooltip>
</template>

<style scoped>
/* Scoped styles can be added here if needed, but currently uses global .monospace-id */
</style>
