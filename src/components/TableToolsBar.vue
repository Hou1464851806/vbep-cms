<script setup>
import { Search, Refresh } from '@element-plus/icons-vue'

const props = defineProps({
  /** 筛选配置项数组 */
  filters: {
    type: Array,
    default: () => [],
    // 每项结构: { key, type, label, options?, placeholder?, ... }
    // type: 'select' | 'input' | 'daterange'
  },
  modelValue: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['update:modelValue', 'search', 'reset'])

const updateField = (key, value) => {
  const newValue = { ...props.modelValue }
  newValue[key] = value
  emit('update:modelValue', newValue)
}

</script>

<template>
  <div class="table-filter-bar">
    <template v-for="filter in filters" :key="filter.key">
      <!-- 下拉选择 -->
      <el-select
        v-if="filter.type === 'select'"
        v-model="modelValue[filter.key]"
        @change="$emit('search')"
        :placeholder="filter.placeholder || filter.label"
        clearable
        style="width: 160px"
      >
        <el-option
          v-for="opt in filter.options"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>

      <!-- 文本搜索 -->
      <el-input
        v-else-if="filter.type === 'input'"
        v-model="modelValue[filter.key]"
        @input="$emit('input-change')"
        @keyup.enter="$emit('search')"
        :placeholder="filter.placeholder || `搜索${filter.label}...`"
        clearable
        style="width: 220px"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <!-- 日期范围 -->
      <el-date-picker
        v-else-if="filter.type === 'daterange'"
        :model-value="modelValue[filter.key]"
        @update:model-value="(v) => updateField(filter.key, v || [])"
        @change="$emit('search')"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        value-format="YYYY-MM-DD"
        style="width: 260px"
      />
    </template>

    <el-button :icon="Refresh" @click="$emit('reset')">重置</el-button>

    <slot name="extra" />
  </div>
</template>

<style scoped>
.table-filter-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
}
</style>
