import { reactive, ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { debounce } from '@/utils/debounce.js'

/**
 * 通用表格筛选/排序/分页 Hook
 * @param {Function} fetchApi - (params) => Promise，接收合并后的查询参数
 * @param {Object}   options
 * @param {Function} options.normalize        - 行数据标准化函数
 * @param {string}   options.dataKey          - 响应中列表字段名，默认 'list'
 * @param {Object}   options.defaultSort      - { prop, order }
 * @param {Object}   options.filterSchema     - 筛选字段初始值
 * @param {number}   options.pageSize         - 每页条数，默认 10
 * @param {number}   options.debounceDelay    - 防抖延迟 ms，默认 400
 * @param {boolean}  options.syncUrl          - 是否双向同步 URL query，默认 false
 */
export function useTableFilter(fetchApi, options = {}) {
  const {
    normalize = (v) => v,
    dataKey = 'list',
    defaultSort = { prop: '', order: '' },
    filterSchema = {},
    pageSize = 10,
    debounceDelay = 400,
    syncUrl = false,
  } = options

  // 仅在启用 URL 同步时调用 router hooks（静态 bool，符合 composition API 规范）
  const route = syncUrl ? useRoute() : null
  const router = syncUrl ? useRouter() : null

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

  /** 是否有任意筛选条件激活 */
  const isFilterActive = computed(() =>
    Object.entries(filterForm).some(([, v]) =>
      Array.isArray(v) ? v.length > 0 : v !== '' && v !== null && v !== undefined
    )
  )

  /**
   * 根据 id_prefix / application_id 字段的规则决定是否加入 params：
   * - 恰好 8 位 → 前缀 LIKE（传给后端）
   * - 恰好 36 位 → 精确 UUID（传给后端）
   * - 其他长度 → 不传（前端 TableHeaderFilter 已做校验提示）
   */
  const isValidIdParam = (val) => {
    const len = String(val).length
    return len === 8 || len === 36
  }

  const ID_EXACT_KEYS = new Set(['id_prefix', 'application_id'])

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

    // 筛选：只传有値的字段
    for (const [key, value] of Object.entries(filterForm)) {
      if (value === '' || value === null || value === undefined) continue
      if (Array.isArray(value) && value.length === 0) continue

      if (Array.isArray(value) && value.length === 2) {
        // 日期范围字段统一处理：date → date_start/date_end；expire_date → expire_start/expire_end
        const base     = key === 'date' ? 'date' : key.replace(/_date$/, '')
        const startKey = key === 'date' ? 'date_start' : `${base}_start`
        const endKey   = key === 'date' ? 'date_end'   : `${base}_end`
        params[startKey] = value[0]
        params[endKey]   = value[1]
      } else if (ID_EXACT_KEYS.has(key)) {
        // ID 类字段：只有恰好 8 位或 36 位才发送
        if (isValidIdParam(value)) params[key] = value
      } else {
        params[key] = value
      }
    }

    return params
  }

  // ── URL 同步 ──────────────────────────────────────────────

  /** 从 URL query 初始化 filterForm / pagination / sortState */
  const initFromUrl = () => {
    if (!syncUrl || !route) return
    const q = route.query

    for (const key of Object.keys(filterSchema)) {
      if (Array.isArray(filterSchema[key])) {
        // 日期范围字段：date → date_start/date_end；expire_date → expire_start/expire_end
        const base = key === 'date' ? 'date' : key.replace(/_date$/, '')
        const startKey = key === 'date' ? 'date_start' : `${base}_start`
        const endKey   = key === 'date' ? 'date_end'   : `${base}_end`
        if (q[startKey] && q[endKey]) {
          filterForm[key] = [q[startKey], q[endKey]]
        }
      } else if (q[key] !== undefined && q[key] !== '') {
        filterForm[key] = q[key]
      }
    }
    if (q.page) pagination.currentPage = parseInt(q.page) || 1
    if (q.size) pagination.pageSize = parseInt(q.size) || pageSize
    if (q.sort_by) sortState.prop = q.sort_by
    if (q.order) {
      sortState.order = q.order === 'asc' ? 'ascending' : q.order === 'desc' ? 'descending' : ''
    }
  }

  /** 将当前状态写入 URL query（replace 不产生历史） */
  const syncToUrl = () => {
    if (!syncUrl || !router) return
    const filterParams = buildParams()
    const query = {}
    for (const [k, v] of Object.entries(filterParams)) {
      if (v !== undefined && v !== null && v !== '') query[k] = String(v)
    }
    router.replace({ query })
  }

  // ── 核心操作 ──────────────────────────────────────────────

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
      if (syncUrl) syncToUrl()
    }
  }

  /** 由 TableHeaderFilter @sort 事件（或 el-table @sort-change）触发 */
  const handleSortChange = ({ prop, order }) => {
    sortState.prop = prop
    sortState.order = order || ''
    pagination.currentPage = 1
    loadData()
  }

  /** 筛选提交（点击确认 / 选择下拉项后调用） */
  const handleFilterChange = () => {
    pagination.currentPage = 1
    loadData()
  }

  /** 防抖版筛选提交（文本框 input 事件） */
  const handleFilterInputChange = debounce(() => {
    handleFilterChange()
  }, debounceDelay)

  onUnmounted(() => {
    handleFilterInputChange.cancel()
  })

  /** 重置所有筛选/排序/分页，并可选地清空 URL */
  const resetFilter = () => {
    for (const key of Object.keys(filterSchema)) {
      filterForm[key] = Array.isArray(filterSchema[key]) ? [] : filterSchema[key]
    }
    sortState.prop = defaultSort.prop
    sortState.order = defaultSort.order
    pagination.currentPage = 1
    loadData()
  }

  const handlePageChange = () => {
    loadData()
  }

  // 若启用 URL 同步，在组件挂载时从 URL 还原状态
  onMounted(() => {
    if (syncUrl) initFromUrl()
  })

  return {
    loading,
    data,
    pagination,
    sortState,
    filterForm,
    isFilterActive,
    loadData,
    handleSortChange,
    handleFilterChange,
    handleFilterInputChange,
    resetFilter,
    handlePageChange,
    initFromUrl,
  }
}
