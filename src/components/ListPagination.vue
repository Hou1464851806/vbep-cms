<template>
  <div class="common-pagination">
    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :page-sizes="[10, 20, 50, 100]"
      background
      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Number, default: 1 }, // 当前页
  pageSize: { type: Number, default: 10 },  // 每页条数
  total: { type: Number, default: 0 }       // 总条数
})

const emit = defineEmits(['update:modelValue', 'update:pageSize', 'change'])

const currentPage = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const pageSize = computed({
  get: () => props.pageSize,
  set: (val) => emit('update:pageSize', val)
})

const handleSizeChange = (val) => {
  emit('change') // 触发重新加载数据
}

const handleCurrentChange = (val) => {
  emit('change') // 触发重新加载数据
}
</script>

<style scoped>
.common-pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
