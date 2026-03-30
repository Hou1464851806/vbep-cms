import { reactive, ref, onUnmounted } from 'vue'
import { debounce } from '@/utils/debounce.js'

/**
 * 通用表格筛选/排序/分页 Hook
 * @param {Function} fetchApi - (params) => Promise，接收合并后的查询参数
 * @param {Object} options
 * @param {Function} options.normalize - 行数据标准化函数
 * @param {string} options.dataKey - 响应中列表字段名，默认 'list'
 * @param {Object} options.defaultSort - { prop, order }
 * @param {Object} options.filterSchema - 筛选字段初始值，如 { status: '', keyword: '' }
 */
export function useTableFilter(fetchApi, options = {}) {
  const {
    normalize = (v) => v,
    dataKey = 'list',
    defaultSort = { prop: '', order: '' },
    filterSchema = {},
    pageSize = 10,
    debounceDelay = 400,
  } = options

  const loading = ref(false)
  const data = ref([])

  const pagination = reactive({
    currentPage: 1,
    pageSize,
    total: 0,
  })

  const sortState = reactive({
    prop: defaultSort.prop,
    order: defaultSort.order,
  })

  // 深拷贝 schema 作为初始值
  const filterForm = reactive({ ...filterSchema })

  const buildParams = () => {
    const params = {
      page: pagination.currentPage,
      size: pagination.pageSize,
    }

    // 排序
    if (sortState.prop && sortState.order) {
      params.sort_by = sortState.prop
      params.order = sortState.order === 'ascending' ? 'asc' : 'desc'
    }

    // 筛选：只传有值的字段
    for (const [key, value] of Object.entries(filterForm)) {
      if (value === '' || value === null || value === undefined) continue;
      if (Array.isArray(value) && value.length === 0) continue;
      // 日期范围特殊处理
      if (key === 'date' && Array.isArray(value)) {
        params.date_start = value[0]
        params.date_end = value[1]
      } else {
        params[key] = value
      }
    }

    return params
  }

  const loadData = async () => {
    loading.value = true
    try {
      const res = await fetchApi(buildParams())
      if (res.status === 200) {
        const list = res.data[dataKey] || []
        data.value = list.map(normalize)
        pagination.total = res.data.total || 0
      }
    } catch (e) {
      console.error('useTableFilter loadData error:', e)
    } finally {
      loading.value = false
    }
  }

  // 排序变化（由 el-table @sort-change 触发）
  const handleSortChange = ({ prop, order }) => {
    sortState.prop = prop
    sortState.order = order || ''
    pagination.currentPage = 1
    loadData()
  }

  // 筛选提交
  const handleFilterChange = () => {
    pagination.currentPage = 1
    loadData()
  }

  // 自实现防抖
  const handleFilterInputChange = debounce(() => {
    handleFilterChange()
  }, debounceDelay)
  // 组件卸载时清理定时器
  onUnmounted(() => {
    handleFilterInputChange.cancel()
  })

  // 重置筛选
  const resetFilter = () => {
    for (const key of Object.keys(filterSchema)) {
      filterForm[key] = Array.isArray(filterSchema[key])
        ? []
        : filterSchema[key]
    }
    sortState.prop = defaultSort.prop
    sortState.order = defaultSort.order
    pagination.currentPage = 1
    loadData()
  }

  // 分页变化
  const handlePageChange = () => {
    loadData()
  }

  return {
    loading,
    data,
    pagination,
    sortState,
    filterForm,
    loadData,
    handleSortChange,
    handleFilterChange,
    handleFilterInputChange,
    resetFilter,
    handlePageChange,
  }
}
