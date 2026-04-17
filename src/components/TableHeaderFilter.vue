<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { Calendar, Filter } from '@element-plus/icons-vue'

const props = defineProps({
  label:      { type: String, required: true },
  type:       { type: String, default: null },  // 'input'|'id-search'|'select'|'daterange'
  modelValue: { default: null },
  placeholder:{ type: String, default: '' },
  options:    { type: Array,  default: () => [] },
  sortProp:   { type: String, default: null },
  sortState:  { type: Object, default: null },  // { prop, order }
  clearLabel: { type: String, default: '全部' },
})

const emit = defineEmits(['update:modelValue', 'change', 'sort'])

// ── 排序 ──────────────────────────────────────────────────
const isSortActive = computed(() =>
  !!props.sortProp && props.sortState?.prop === props.sortProp
)
const currentOrder = computed(() => isSortActive.value ? props.sortState.order : '')

const handleSortClick = () => {
  if (!props.sortProp) return
  let next
  if (!isSortActive.value || currentOrder.value === '') next = 'ascending'
  else if (currentOrder.value === 'ascending') next = 'descending'
  else next = ''
  emit('sort', { prop: props.sortProp, order: next })
}

// ── 筛选激活态 ────────────────────────────────────────────
const isActive = computed(() => {
  const v = props.modelValue
  if (v === null || v === undefined || v === '') return false
  if (Array.isArray(v)) return v.length > 0
  return true
})

// filter icon is always the funnel/filter icon (calendar handled separately in template)
const filterIcon = computed(() => Filter)

// ── Popover（input / id-search / select）──────────────────
const popoverVisible = ref(false)
const inputRef = ref(null)
watch(popoverVisible, (v) => {
  if (v) nextTick(() => inputRef.value?.focus?.())
})

// ── 日期范围（直接触发 el-date-picker）────────────────────
const datePickerRef = ref(null)
const openDatePicker = () => {
  nextTick(() => {
    const dp = datePickerRef.value
    if (!dp) return
    if (typeof dp.handleOpen === 'function') dp.handleOpen()
    else dp.focus?.()
  })
}

// ── 本地值 ────────────────────────────────────────────────
const localValue = ref(initLocal())
const idError = ref('')

function initLocal() {
  const v = props.modelValue
  return props.type === 'daterange'
    ? (Array.isArray(v) ? [...v] : [])
    : (v ?? '')
}

watch(() => props.modelValue, (v) => {
  localValue.value = props.type === 'daterange'
    ? (Array.isArray(v) ? [...v] : [])
    : (v ?? '')
  idError.value = ''
})

// ── 操作 ──────────────────────────────────────────────────
const validateId = (val) => {
  if (!val) return ''
  const len = val.length
  if (len > 0 && len < 8)  return '请输入至少 8 位 ID 前缀'
  if (len > 8 && len < 36) return '请输入 8 位前缀或完整 36 位 ID'
  if (len > 36) return 'ID 过长'
  return ''
}

const commit = (val) => {
  emit('update:modelValue', val)
  emit('change')
  popoverVisible.value = false
}
const handleConfirm = () => commit(localValue.value)
const handleIdConfirm = () => {
  const err = validateId(localValue.value)
  idError.value = err
  if (!err) commit(localValue.value)
}
const handleClear = () => {
  localValue.value = props.type === 'daterange' ? [] : ''
  idError.value = ''
  emit('update:modelValue', localValue.value)
  emit('change')
  popoverVisible.value = false
}
const handleSelectChange = (val) => { localValue.value = val; commit(val) }
const handleDateChange = (val) => {
  const v = val || []
  localValue.value = v
  emit('update:modelValue', v)
  emit('change')
}
const handleKeyEnter = () => {
  if (props.type === 'id-search') handleIdConfirm()
  else handleConfirm()
}
</script>

<template>
  <div class="th-filter-wrap" @click.stop>

    <!-- 1. 列标题（点击排序） -->
    <span
      class="th-label"
      :class="{ 'is-sortable': !!sortProp }"
      @click="handleSortClick"
    >{{ label }}</span>

    <!-- 2. 排序指示（纯 CSS 三角，独立 flex 元素） -->
    <span
      v-if="sortProp"
      class="th-sort-carets"
      @click.stop="handleSortClick"
      title="点击排序"
    >
      <!-- 上三角：升序 -->
      <i class="th-caret th-caret-asc" :class="{ 'is-active': isSortActive && currentOrder === 'ascending' }"></i>
      <!-- 下三角：降序 -->
      <i class="th-caret th-caret-desc" :class="{ 'is-active': isSortActive && currentOrder === 'descending' }"></i>
    </span>

    <!-- 3a. 日期范围：图标直接触发，el-date-picker 放在不可见容器中作为弹出锚点 -->
    <div v-if="type === 'daterange'" class="th-date-outer">
      <el-icon
        class="th-filter-icon"
        :class="{ 'is-active': isActive }"
        @click.stop="openDatePicker"
        title="日期筛选"
      >
        <Calendar />
      </el-icon>
      <!-- 完全隐藏的 date-picker（只用于弹出日历，不参与视觉渲染） -->
      <span class="th-date-anchor" aria-hidden="true">
        <el-date-picker
          ref="datePickerRef"
          v-model="localValue"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          clearable
          @change="handleDateChange"
        />
      </span>
    </div>

    <!-- 3b. 其他筛选类型：popover -->
    <el-popover
      v-else-if="type"
      v-model:visible="popoverVisible"
      trigger="click"
      placement="bottom"
      :width="type === 'select' ? 130 : 240"
      :teleported="true"
      popper-class="th-filter-popover"
    >
      <template #reference>
        <el-icon
          class="th-filter-icon"
          :class="{ 'is-active': isActive }"
          @click.stop
        >
          <component :is="filterIcon" />
        </el-icon>
      </template>

      <!-- 文本搜索 -->
      <div v-if="type === 'input'" class="th-popover-content">
        <el-input
          ref="inputRef"
          v-model="localValue"
          :placeholder="placeholder || `搜索 ${label}...`"
          clearable size="small"
          @keyup.enter="handleKeyEnter"
          @clear="handleClear"
        />
        <div class="th-popover-actions">
          <el-button size="small" @click="handleClear">清空</el-button>
          <el-button size="small" type="primary" @click="handleConfirm">确认</el-button>
        </div>
      </div>

      <!-- ID 精确搜索 -->
      <div v-else-if="type === 'id-search'" class="th-popover-content">
        <el-input
          ref="inputRef"
          v-model="localValue"
          placeholder="8位ID前缀或完整36位ID"
          clearable size="small"
          @keyup.enter="handleKeyEnter"
          @clear="handleClear"
        />
        <div v-if="idError" class="th-id-error">{{ idError }}</div>
        <div class="th-popover-actions">
          <el-button size="small" @click="handleClear">清空</el-button>
          <el-button size="small" type="primary" @click="handleIdConfirm">确认</el-button>
        </div>
      </div>

      <!-- 状态选择：列表 hover 样式，无圆形单选 -->
      <div v-else-if="type === 'select'" class="th-select-list">
        <div
          class="th-select-item"
          :class="{ 'is-active': modelValue === '' || modelValue === null || modelValue === undefined }"
          @click="handleSelectChange('')"
        >{{ clearLabel }}</div>
        <div class="th-select-divider" />
        <div
          v-for="opt in options"
          :key="opt.value"
          class="th-select-item"
          :class="{ 'is-active': modelValue === opt.value }"
          @click="handleSelectChange(opt.value)"
        >
          <el-tag
            v-if="opt.tagType !== undefined"
            :type="opt.tagType"
            size="small"
            :effect="modelValue === opt.value ? 'dark' : 'light'"
            class="th-opt-tag"
          >
            {{ opt.label }}
          </el-tag>
          <span v-else>{{ opt.label }}</span>
        </div>
      </div>

    </el-popover>
  </div>
</template>

<style scoped>
/* ── 整体 Wrapper ──────────────────────────────────────── */
.th-filter-wrap {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  max-width: 100%;
}

/* ── 标题文字 ──────────────────────────────────────────── */
.th-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  line-height: 1.4;
}
.th-label.is-sortable { cursor: pointer; }
.th-label.is-sortable:hover { color: var(--el-color-primary); }

/* ── 排序三角（El Plus 内部同款 border 实现）────────────── */
.th-sort-carets {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  cursor: pointer;
  flex-shrink: 0;
  padding: 0 1px;
}
.th-caret {
  display: block;
  width: 0;
  height: 0;
}
.th-caret-asc {
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-bottom: 5px solid #c8c9cc;
  transition: border-bottom-color 0.2s;
}
.th-caret-asc.is-active { border-bottom-color: var(--el-color-primary); }

.th-caret-desc {
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid #c8c9cc;
  transition: border-top-color 0.2s;
}
.th-caret-desc.is-active { border-top-color: var(--el-color-primary); }

/* ── 筛选图标 ──────────────────────────────────────────── */
.th-filter-icon {
  font-size: 15px;
  color: #909399;
  cursor: pointer;
  flex-shrink: 0;
  padding: 1px;
  border-radius: 3px;
  transition: color 0.2s, background 0.2s;
}
.th-filter-icon:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.th-filter-icon.is-active { color: var(--el-color-primary); }

/* ── 日期触发区 ────────────────────────────────────────── */
.th-date-outer {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}
/* 完全不可见的锚点容器：宽高为0，overflow裁剪，opacity透明 */
.th-date-anchor {
  position: absolute;
  top: 0;
  left: 0;
  width: 0;
  height: 0;
  overflow: hidden;
  opacity: 0;
  pointer-events: none;
  display: block;
}
</style>

<style>
/* ── Popover 全局样式（teleported 到 body）────────────── */
.th-filter-popover.el-popover {
  padding: 8px !important;
  min-width: 0 !important;
}
.th-filter-popover .th-popover-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.th-filter-popover .th-popover-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
.th-filter-popover .th-id-error {
  color: var(--el-color-danger);
  font-size: 12px;
  margin-top: -4px;
}

/* ── 状态选择列表 ─────────────────────────────────────── */
.th-filter-popover .th-select-list { padding: 2px 0; }
.th-filter-popover .th-select-item {
  padding: 6px 12px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 13px;
  line-height: 1.4;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s;
  color: var(--el-text-color-regular);
}
.th-filter-popover .th-select-item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.th-filter-popover .th-select-item.is-active {
  color: var(--el-color-primary);
  font-weight: 500;
}
.th-filter-popover .th-select-divider {
  height: 1px;
  background: var(--el-border-color-lighter);
  margin: 4px 6px;
}

/* 标签样式微调 */
.th-filter-popover .th-opt-tag {
  border-radius: 4px;
  height: 20px;
  line-height: 18px;
  padding: 0 6px;
  font-weight: normal;
}
</style>
