<!-- src/views/dataset/DatasetList.vue -->
<script setup>
import { ref, onMounted, inject, reactive, watch, nextTick, onBeforeUnmount, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import {
  Plus,
  Delete,
  Document,
  Upload,
  Download,
  Promotion,
  View,
  Refresh,
  DataAnalysis,
  QuestionFilled,
  Setting,
  Sort,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import {
  formatDateTime,
  formatShortId,
  formatFileSize,
} from '@/utils/format.js'
import { DATASET_FILE_LIMIT_MB } from '@/utils/constrains.js'
import MarkdownRenderer from "@/components/MarkdownRenderer.vue";
import ListPagination from "@/components/ListPagination.vue";
import { useTableFilter } from "@/hooks/useTable.js";
import TableHeaderFilter from "@/components/TableHeaderFilter.vue";
import IdCell from "@/components/IdCell.vue";

// --- API 函数 ---
const fetchDatasetListAPI = (params) => apiClient.get('/api/v1/datasets/', { params })
const fetchDatasetDetailAPI = (uuid) => apiClient.get(`/api/v1/datasets/${uuid}`)
const fetchDatasetStatisticsAPI = (uuid) => apiClient.get(`/api/v1/datasets/${uuid}/statistics`)
const createDatasetAPI = (data) => apiClient.post('/api/v1/datasets/', data)
const updateDatasetAPI = (uuid, data) => apiClient.patch(`/api/v1/datasets/${uuid}`, data)
const deleteDatasetAPI = (uuid) => apiClient.delete(`/api/v1/datasets/${uuid}`)
const fetchPublicTemplateFilesAPI = () => apiClient.get('/api/v1/files/public/templates')

const fetchFilesForDatasetAPI = (datasetId) => apiClient.get(`/api/v1/datasets/${datasetId}/files`)
const uploadDatasetFileAPI = (datasetId, file, config = {}) => {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post(`/api/v1/datasets/${datasetId}/files`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    ...config,
  })
}
const deleteDatasetFileAPI = (datasetId, fileId) =>
  apiClient.delete(`/api/v1/datasets/${datasetId}/files/${fileId}`)
const downloadDatasetFileAPI = (datasetId, fileId) =>
  apiClient.get(`/api/v1/datasets/${datasetId}/files/${fileId}`, { responseType: 'blob' })
const downloadPublicFileAPI = (fileId) =>
  apiClient.get(`/api/v1/files/public/${fileId}/download`, { responseType: 'blob' })

const submitApplicationAPI = (applicationData) =>
  apiClient.post('/api/v1/applications', applicationData)

const DATASET_FILE_LIMIT_BYTES = DATASET_FILE_LIMIT_MB * 1024 * 1024

// --- 状态定义 ---
const router = useRouter()
const isAdmin = inject('isAdmin')
// const datasetList = ref([])
// const isDataLoading = ref(true)

// // 分页相关
// const pagination = reactive({
//   currentPage: 1,
//   pageSize: 10,
//   total: 0
// })

// 获取全局 echarts 实例
const { proxy } = getCurrentInstance()
const echarts = proxy.$echarts

// ECharts 实例
const chartRef = ref(null)
let chartInstance = null

// 状态1: 新增/编辑数据集对话框
const datasetFormDialog = reactive({
  visible: false,
  title: '',
  form: { id: null, name: '', description: '' },
})

// 状态2: 管理对话框 (合并了编辑、删除、文件管理)
const manageDialog = reactive({
  visible: false,
  loading: false,
  id: null,
  form: { title: '', description: '' },
  fileList: [],
  isFilesLoading: false,
  isUploading: false,
  uploadTasks: [],
})

// 状态3: 申请使用对话框
const applicationDialog = reactive({
  visible: false,
  isSubmitting: false,
  currentStep: 1,
  dataset: null,
  template: {
    description: '',
    files: [],
  },
})

// 状态4: 数据集详情对话框 (普通用户查看)
const detailDialog = reactive({
  visible: false,
  loading: false,
  data: null,
  files: [],
})

// 状态5: 统计信息对话框 (管理员)
const statisticsDialog = reactive({
  visible: false,
  loading: false,
  data: null,
})

const normalizeOneDataset = (item) => {
  return {
    id: item.id,
    name: item.title,
    description: item.description,
    fileCount: item.fileCount || (item.files ? item.files.length : 0),
    creator: item.creator,
    createdAt: formatDateTime(item.createdAt),
    hasAccess: item.hasAccess,
    raw: item,
  }
}
const normalizeDatasets = (datasets = []) => datasets.map(normalizeOneDataset)

const normalizeDatasetFiles = (files = []) =>
  files.map((item) => ({
    id: item.fileID,
    name: item.name || '未命名文件',
    size: Number(item.size ?? 0),
    createdAt: formatDateTime(item.createdAt),
    raw: item,
  }))

const updateUploadingState = () => {
  manageDialog.isUploading = manageDialog.uploadTasks.some((task) => task.status === 'uploading')
}

const ensureUploadTask = (file) => {
  const existing = manageDialog.uploadTasks.find((task) => task.uid === file.uid)
  if (existing) return existing
  const task = {
    uid: file.uid,
    name: file.name,
    percentage: 0,
    status: 'uploading',
  }
  manageDialog.uploadTasks.push(task)
  updateUploadingState()
  return task
}

const updateUploadTask = (uid, payload = {}) => {
  const task = manageDialog.uploadTasks.find((item) => item.uid === uid)
  if (task) {
    Object.assign(task, payload)
  }
}

const removeUploadTask = (uid, delay = 0) => {
  const remover = () => {
    const index = manageDialog.uploadTasks.findIndex((item) => item.uid === uid)
    if (index !== -1) manageDialog.uploadTasks.splice(index, 1)
    updateUploadingState()
  }
  if (delay > 0) {
    setTimeout(remover, delay)
  } else {
    remover()
  }
}

// const fetchDatasets = async () => {
//   isDataLoading.value = true
//   try {
//     const params = {
//       page: pagination.currentPage,
//       size: pagination.pageSize
//     }
//     // 仅获取数据集列表，不再预加载用户的申请历史，提升性能
//     const dsRes = await fetchDatasetListAPI(params)
//
//     if (dsRes && dsRes.status === 200) {
//       datasetList.value = normalizeDatasets(dsRes.data.datasets)
//       pagination.total = dsRes.data.total || 0
//       return
//     }
//     ElMessage.error(dsRes?.data?.message || '数据集加载失败')
//   } catch {
//     ElMessage.error('数据集加载失败')
//   } finally {
//     isDataLoading.value = false
//   }
// }

const {
  loading: isDataLoading,
  data: datasetList,
  pagination,
  filterForm,
  sortState,
  isFilterActive,
  loadData: fetchDatasets,
  handleSortChange,
  handleFilterChange,
  handleFilterInputChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/datasets/', { params }),
  {
    normalize: normalizeOneDataset,
    dataKey: 'datasets',
    defaultSort: { prop: 'createdAt', order: 'descending' },
    filterSchema: { id_prefix: '', name: '', creator: '', date: [] },
    pageSize: 10,
    syncUrl: true,
  }
)

// --- 生命周期钩子 ---
onMounted(() => {
  fetchDatasets()
  window.addEventListener('resize', handleResize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (chartInstance) {
    chartInstance.dispose()
  }
})

watch(
  () => applicationDialog.visible,
  (visible) => {
    if (!visible) {
      applicationDialog.dataset = null
      applicationDialog.currentStep = 1
    }
  },
)

// // 分页回调函数
// const handlePageChange = () => {
//   fetchDatasets()
// }

const handleResize = () => {
  if (chartInstance) {
    chartInstance.resize()
  }
}

// --- 数据集 CRUD 事件处理 ---
const handleAdd = () => {
  datasetFormDialog.title = '新增数据集'
  datasetFormDialog.form = { id: null, name: '', description: '' }
  datasetFormDialog.visible = true
  datasetFormDialog.loading = false
}
const handleSaveDataset = async () => {
  const form = datasetFormDialog.form
  if (!form.name.trim()) {
    ElMessage.warning('数据集名称不能为空')
    return
  }

  try {
    // 仅处理新增，编辑逻辑已移至 manageDialog
    if (!form.id) {
      // 新增
      const res = await createDatasetAPI({ title: form.name, description: form.description })
      if (res.status === 200) {
        ElMessage.success('创建成功')
        await fetchDatasets()
        datasetFormDialog.visible = false
      } else {
        ElMessage.error(res.data.message || '创建失败')
      }
    }
  } catch {
    ElMessage.error('操作失败')
  }
}
const handleDelete = (row) => {
  ElMessageBox.confirm(`确定要删除数据集 "${row.name}" 吗？`, '警告', {
    confirmButtonText: '确定删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(async () => {
      try {
        const res = await deleteDatasetAPI(row.id)
        if (res.status === 200) {
          ElMessage.success(`数据集已删除`)
          await fetchDatasets()
        } else {
          ElMessage.error(res.data?.message || '删除失败')
        }
      } catch {
        ElMessage.error('删除失败')
      }
    })
    .catch(() => {
      ElMessage.info('已取消删除')
    })
}

const handleViewDetail = async (row) => {
  detailDialog.visible = true
  detailDialog.loading = true
  detailDialog.data = null
  detailDialog.files = []

  try {
    const res = await fetchDatasetDetailAPI(row.id)
    // 后端现在返回结构为: { message: "success", dataset: {...}, hasAccess: true/false }
    if (res.status === 200) {
      const data = res.data.dataset

      // 关键：将后端返回的 hasAccess 状态合并到 dialog 数据中
      const accessStatus = res.data.hasAccess !== undefined ? res.data.hasAccess : row.hasAccess

      detailDialog.data = { ...data, hasAccess: accessStatus }

      // 只有有权限时才处理文件列表
      if (accessStatus) {
        detailDialog.files = normalizeDatasetFiles(data.files)
      } else {
        detailDialog.files = []
      }
    } else {
      ElMessage.error(res.data?.message || '获取详情失败')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取详情失败')
    detailDialog.visible = false
  } finally {
    detailDialog.loading = false
  }
}

const handleViewStatistics = async (row) => {
  statisticsDialog.visible = true
  statisticsDialog.loading = true
  statisticsDialog.data = null

  try {
    const res = await fetchDatasetStatisticsAPI(row.id)
    if (res.status === 200) {
      statisticsDialog.data = res.data.statistics
      // 显式设置 loading 为 false，确保 DOM 更新（Skeleton 消失，内容显示）后再初始化图表
      statisticsDialog.loading = false
      nextTick(() => {
        if (chartRef.value) {
          if (chartInstance) chartInstance.dispose()
          chartInstance = echarts.init(chartRef.value)

          const distribution = statisticsDialog.data.submissionDistribution || {}
          // 确保按版本号排序 (key 是字符串 "0", "1" 等)
          const versions = Object.keys(distribution).sort((a, b) => Number(a) - Number(b))
          const counts = versions.map((v) => distribution[v])
          const categories = versions.map((v) => (v == 0 ? '草稿' : `${v}`))

          chartInstance.setOption({
            tooltip: {
              trigger: 'axis',
              axisPointer: { type: 'shadow' },
            },
            grid: {
              top: '15%',
              left: '3%',
              right: '16%',
              bottom: '3%',
              containLabel: true,
            },
            xAxis: { type: 'category', data: categories, name: '提交次数' },
            yAxis: { type: 'value', minInterval: 1, name: '计数' },
            series: [{ data: counts, type: 'bar', barWidth: '40%', color: '#409eff' }],
          })
        }
      })
    } else {
      ElMessage.error(res.data?.message || '获取统计数据失败')
      statisticsDialog.loading = false
    }
  } catch (error) {
    ElMessage.error('获取统计数据失败')
    statisticsDialog.loading = false
  }
}

// --- 文件管理事件处理 ---
const handleBeforeUpload = (rawFile) => {
  if (rawFile.size > DATASET_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(DATASET_FILE_LIMIT_BYTES)} 限制，无法上传`)
    return false
  }
  return true
}

const handleManage = async (row) => {
  manageDialog.visible = true
  manageDialog.id = row.id
  manageDialog.form = { title: row.name, description: '' }
  manageDialog.loading = true

  // 重置文件状态
  manageDialog.fileList = []
  manageDialog.uploadTasks = []
  manageDialog.isUploading = false

  // 1. 获取详情 (为了描述)
  try {
    const res = await fetchDatasetDetailAPI(row.id)
    if (res.status === 200) {
      const d = res.data.dataset
      manageDialog.form.title = d.title
      manageDialog.form.description = d.description || ''
    }
  } catch (e) {
    console.error(e)
  } finally {
    manageDialog.loading = false
  }

  // 2. 加载文件
  loadFilesForManageDialog(row.id)
}

const handleUpdateDatasetInfo = async () => {
  if (!manageDialog.form.title.trim()) {
    ElMessage.warning('数据集名称不能为空')
    return
  }
  try {
    const res = await updateDatasetAPI(manageDialog.id, manageDialog.form)
    if (res.status === 200) {
      ElMessage.success('基本信息更新成功')
      await fetchDatasets()
    } else {
      ElMessage.error(res.data?.message || '更新失败')
    }
  } catch {
    ElMessage.error('更新失败')
  }
}

const loadFilesForManageDialog = async (datasetId) => {
  manageDialog.isFilesLoading = true
  try {
    const res = await fetchFilesForDatasetAPI(datasetId)
    if (res?.status === 200) {
      manageDialog.fileList = normalizeDatasetFiles(res.data.files)
    } else {
      ElMessage.error('文件列表加载失败')
    }
  } catch {
    ElMessage.error('文件列表加载失败')
  } finally {
    manageDialog.isFilesLoading = false
  }
}
const handleDatasetFileUpload = async (options) => {
  const { file, onError, onSuccess, onProgress } = options
  if (!manageDialog.id) {
    const err = new Error('未选择数据集，无法上传文件')
    onError?.(err)
    ElMessage.error(err.message)
    return
  }
  const task = ensureUploadTask(file)
  onProgress?.({ percent: 0 })
  try {
    const res = await uploadDatasetFileAPI(manageDialog.id, file, {
      onUploadProgress: (event) => {
        const percentage = event.total ? Math.round((event.loaded / event.total) * 100) : 0
        updateUploadTask(task.uid, { percentage })
        onProgress?.({ percent: percentage })
      },
    })
    if (res?.status === 200) {
      ElMessage.success(`${file.name} 上传成功`)
      await loadFilesForManageDialog(manageDialog.id)
      onSuccess?.(res.data)
      updateUploadTask(task.uid, { status: 'success', percentage: 100 })
      updateUploadingState()
      removeUploadTask(task.uid, 2000)
    } else {
      const message = res?.data?.message || '上传失败'
      ElMessage.error(message)
      updateUploadTask(task.uid, { status: 'error' })
      updateUploadingState()
      removeUploadTask(task.uid, 4000)
      onError?.(new Error(message))
    }
  } catch (error) {
    console.error('Failed to upload file', error)
    ElMessage.error('上传文件失败')
    updateUploadTask(task.uid, { status: 'error' })
    updateUploadingState()
    removeUploadTask(task.uid, 4000)
    onError?.(error)
  } finally {
    updateUploadingState()
  }
}
const handleDownloadFile = async (datasetId, file) => {
  if (!datasetId) {
    ElMessage.error('未找到对应的数据集 ID')
    return
  }
  if (!file || !file.id) {
    ElMessage.error('文件信息不完整')
    return
  }

  try {
    const res = await downloadDatasetFileAPI(datasetId, file.id)

    // 创建 Blob URL 并触发下载
    const blob = new Blob([res.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()

    // 清理资源
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to download file', error)
    ElMessage.error('下载文件失败')
  }
}
const handleDeleteFile = (file) => {
  if (!manageDialog.id) return
  ElMessageBox.confirm(`确定要删除文件 "${file.name}" 吗？`, '提示', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(async () => {
      try {
        const res = await deleteDatasetFileAPI(manageDialog.id, file.id)
        if (res?.status === 200) {
          ElMessage.success('文件已删除')
          await loadFilesForManageDialog(manageDialog.id)
        } else {
          ElMessage.error(res?.data?.message || '删除失败')
        }
      } catch (error) {
        console.error('Failed to delete file', error)
        ElMessage.error('删除文件失败')
      }
    })
    .catch(() => {})
}

// --- 申请功能事件处理 ---
const handleApply = async (row) => {
  applicationDialog.dataset = row
  applicationDialog.currentStep = 1
  applicationDialog.template.description = '加载中...'
  applicationDialog.template.files = []

  applicationDialog.visible = true

  try {
    const res = await fetchPublicTemplateFilesAPI()
    if (res.status === 200) {
      applicationDialog.template.description = res.data.description || '暂无说明'
      applicationDialog.template.files = normalizeDatasetFiles(res.data.files || [])
    } else {
      ElMessage.error('获取申请模板信息失败')
    }
  } catch (error) {
    ElMessage.error('获取申请模板信息失败')
  }
}

const handleDownloadTemplateFile = async (file) => {
  if (!file || !file.id) return
  try {
    const res = await downloadPublicFileAPI(file.id)
    const blob = new Blob([res.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to download template', error)
    ElMessage.error('下载模板文件失败')
  }
}

const handleCreateDraft = async () => {
  applicationDialog.isSubmitting = true
  try {
    const res = await submitApplicationAPI({
      datasetID: applicationDialog.dataset.id,
    })
    if (res.status === 200 && res.data.applicationID) {
      applicationDialog.visible = false
      await fetchDatasets() // Refresh list to update button status

      ElMessageBox.confirm(
        '您已成功创建申请表草稿。请前往“我的申请”页面完善资料并提交。',
        '草稿创建成功',
        {
          confirmButtonText: '立即前往',
          cancelButtonText: '稍后处理',
          type: 'success',
        },
      )
        .then(() => {
          router.push('/home/applications')
        })
        .catch(() => {
          // Stay on current page
        })
    } else {
      ElMessage.error(res.data?.message || '创建申请失败')
    }
  } catch (error) {
    console.error('Failed to create draft application:', error)
    const msg = error.response?.data?.message || '创建申请失败'
    ElMessage.error(msg)
  } finally {
    applicationDialog.isSubmitting = false
  }
}

const handleCancelApplication = async () => {
  applicationDialog.visible = false
}
</script>

<template>
  <div>
    <!-- 主体：数据集列表 -->
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
        <span>数据集列表</span>
        <div class="header-actions">
          <el-button
            v-if="isFilterActive"
            class="reset-filter-btn"
            plain
            size="small"
            @click="resetFilter"
          >重置筛选</el-button>
          <el-button :icon="Refresh" circle @click="fetchDatasets" :loading="isDataLoading" title="刷新列表" />
          <el-button v-if="isAdmin()" type="primary" :icon="Plus" @click="handleAdd">新增数据集</el-button>
        </div>
      </div>
      </template>

      <div v-loading="isDataLoading">
        <el-table
          :data="datasetList"
          style="width: 100%"
        >
          <el-table-column label="数据集ID" width="110">
            <template #header>
              <table-header-filter
                label="数据集ID"
                type="id-search"
                v-model="filterForm.id_prefix"
                @change="handleFilterChange"
              />
            </template>
            <template #default="{ row }">
              <id-cell :id="row.id" />
            </template>
          </el-table-column>

          <el-table-column label="数据集名称" min-width="300">
            <template #header>
              <table-header-filter
                label="数据集名称"
                type="input"
                v-model="filterForm.name"
                @change="handleFilterChange"
                sort-prop="name"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">
              <span class="table-name-text">{{ row.name }}</span>
            </template>
          </el-table-column>

          <el-table-column label="文件数" width="90" align="center">
            <template #header>
              <table-header-filter
                label="文件数"
                sort-prop="fileCount"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">{{ row.fileCount }}</template>
          </el-table-column>

          <el-table-column label="创建者" width="140">
            <template #header>
              <table-header-filter
                label="创建者"
                type="input"
                v-model="filterForm.creator"
                @change="handleFilterChange"
                sort-prop="creator"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">{{ row.creator }}</template>
          </el-table-column>

          <el-table-column label="创建时间" width="180">
            <template #header>
              <table-header-filter
                label="创建时间"
                type="daterange"
                v-model="filterForm.date"
                @change="handleFilterChange"
                sort-prop="createdAt"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">
              <span class="table-time-text">{{ row.createdAt }}</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" :width="isAdmin() ? 260 : 260" fixed="right" align="center">
            <template #default="scope">
              <template v-if="isAdmin()">
                <el-button link type="primary" :icon="Setting" @click="handleManage(scope.row)">管理</el-button>
                <el-button link type="primary" :icon="DataAnalysis" @click="handleViewStatistics(scope.row)">统计</el-button>
                <el-button
                  link
                  type="danger"
                  :icon="Delete"
                  @click="handleDelete(scope.row)"
                  >删除</el-button
                >
              </template>
              <template v-else>
                <el-button
                  v-if="scope.row.hasAccess"
                  type="primary"
                  link
                  :icon="Download"
                  @click="handleViewDetail(scope.row)"
                >
                  下载
                </el-button>
                <el-button
                  v-else
                  type="primary"
                  link
                  :icon="View"
                  @click="handleViewDetail(scope.row)"
                >
                  查看详情
                </el-button>
                <template v-if="!scope.row.hasAccess">
                  <el-button type="primary" link :icon="Promotion" @click="handleApply(scope.row)">
                    申请使用
                  </el-button>
                </template>
              </template>
            </template>
          </el-table-column>
        </el-table>

        <list-pagination
          v-model="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          @change="handlePageChange"
        />
      </div>
    </el-card>

    <!-- ======== 所有对话框 ======== -->

    <!-- 对话框1：新增/编辑数据集 (管理员) -->
    <el-dialog v-model="datasetFormDialog.visible" :title="datasetFormDialog.title" width="80%" top="5vh">
      <el-form :model="datasetFormDialog.form" label-width="100px">
        <el-form-item label="数据集名称">
          <el-input v-model="datasetFormDialog.form.name" placeholder="请输入数据集名称" style="width: 90%" />
        </el-form-item>
        <el-form-item label="简介">
          <div class="markdown-editor-container" style="height: 300px; width: 90%;">
            <div class="editor-pane">
              <el-input
                v-model="datasetFormDialog.form.description"
                type="textarea"
                class="h-100-input"
                placeholder="请输入详细的数据集简介信息..."
              />
            </div>
            <div class="editor-divider">
              <el-icon class="divider-icon"><Sort /></el-icon>
            </div>
            <div class="preview-pane">
              <div class="preview-content">
                <markdown-renderer :model-value="datasetFormDialog.form.description" />
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="datasetFormDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveDataset">确定</el-button>
      </template>
    </el-dialog>

    <!-- 对话框2：管理数据集 (管理员) -->
    <el-dialog
      v-model="manageDialog.visible"
      title="管理数据集"
      width="80%"
      top="5vh"
    >
      <!-- 上半部分：基本信息管理 -->
      <div class="manage-section">
        <div class="manage-section-header">
          <span class="manage-section-title">基本信息管理</span>
          <el-button type="primary" size="small" @click="handleUpdateDatasetInfo">保存</el-button>
        </div>
        <div class="manage-content-wrapper">
          <el-form :model="manageDialog.form" label-width="80px" v-loading="manageDialog.loading">
            <el-form-item label="名称">
              <el-input v-model="manageDialog.form.title" placeholder="数据集名称" style="width: 90%" />
            </el-form-item>
            <el-form-item label="描述">
              <div class="markdown-editor-container" style="height: 300px; width: 90%;">
                <div class="editor-pane">
                  <el-input
                    v-model="manageDialog.form.description"
                    type="textarea"
                    class="h-100-input"
                    placeholder="数据集描述"
                  />
                </div>
                <div class="editor-divider">
                  <el-icon class="divider-icon"><Sort /></el-icon>
                </div>
                <div class="preview-pane">
                  <div class="preview-content">
                    <markdown-renderer :model-value="manageDialog.form.description" />
                  </div>
                </div>
              </div>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <el-divider />

      <!-- 下半部分：文件管理 -->
      <div class="manage-section">
        <div class="manage-section-header">
          <span class="manage-section-title">文件管理</span>
          <div class="header-actions">
            <span class="upload-tip">限 {{ formatFileSize(DATASET_FILE_LIMIT_BYTES) }}/个</span>
                <el-upload
                  action="#"
                  :show-file-list="false"
                  :disabled="!manageDialog.id"
                  multiple
                  :before-upload="handleBeforeUpload"
                  :http-request="handleDatasetFileUpload"
                >
                  <el-button type="primary" size="small" :icon="Upload" :loading="manageDialog.isUploading">上传文件</el-button>
                </el-upload>
          </div>
        </div>
        <div class="manage-content-wrapper">
        <div v-if="manageDialog.uploadTasks.length" class="upload-progress-list">
        <div v-for="task in manageDialog.uploadTasks" :key="task.uid" class="upload-progress-item">
          <span class="upload-progress-name">{{ task.name }}</span>
          <el-progress
            :percentage="task.percentage"
            :status="
              task.status === 'error'
                ? 'exception'
                : task.status === 'success'
                  ? 'success'
                  : undefined
            "
          />
        </div>
        </div>
        <el-skeleton :rows="5" animated :loading="manageDialog.isFilesLoading">
        <el-table :data="manageDialog.fileList" max-height="250px">
          <el-table-column property="name" label="文件名">
            <template #default="scope">
              <div class="file-name-cell">
                <el-icon class="file-icon"><Document /></el-icon>
                <span class="file-name-text" :title="scope.row.name">{{ scope.row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="100">
            <template #default="scope">
              {{ formatFileSize(scope.row.size) }}
            </template>
          </el-table-column>
          <el-table-column property="createdAt" label="上传时间" width="160">
            <template #default="scope">
              {{ scope.row.createdAt || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right" align="center">
            <template #default="scope">
              <el-button
                type="primary"
                link
                :icon="Download"
                @click="handleDownloadFile(manageDialog.id, scope.row)"
                >下载</el-button
              >
              <el-button
                type="danger"
                link
                :icon="Delete"
                @click="handleDeleteFile(scope.row)"
                >删除</el-button
              >
            </template>
          </el-table-column>
        </el-table>
        </el-skeleton>
        </div>
      </div>
    </el-dialog>

    <!-- 对话框3：申请使用 (普通用户) -->
    <el-dialog
      v-model="applicationDialog.visible"
      :title="`申请使用数据集 - ${applicationDialog.dataset?.name}`"
      width="720px"
      :close-on-click-modal="false"
    >
      <div v-if="applicationDialog.currentStep === 1" class="agreement-step">
        <div>
          <div class="upload-label">申请说明</div>
          <div class="template-description-box">
            <markdown-renderer :model-value="applicationDialog.template.description" />
          </div>
        </div>
        <div class="template-download-area" v-if="applicationDialog.template.files && applicationDialog.template.files.length">
          <div class="upload-label">模版下载</div>
          <div v-for="file in applicationDialog.template.files" :key="file.id">
            <el-button type="primary" link :icon="Download" @click="handleDownloadTemplateFile(file)">
              下载“{{ file.name }}”
            </el-button>
          </div>
        </div>
      </div>

      <template #footer>
        <template v-if="applicationDialog.currentStep === 1">
          <el-button @click="handleCancelApplication">取消</el-button>
          <el-button type="primary" @click="handleCreateDraft" :loading="applicationDialog.isSubmitting">开始申请</el-button>
        </template>
      </template>
    </el-dialog>

    <!-- 对话框4：查看信息 (普通用户) -->
    <el-dialog v-model="detailDialog.visible" title="数据集详情" width="60%">
      <el-skeleton :rows="5" animated :loading="detailDialog.loading">
        <div v-if="detailDialog.data">
          <el-descriptions border :column="1">
            <el-descriptions-item label="名称">{{ detailDialog.data.title }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{
              formatDateTime(detailDialog.data.created_at)
            }}</el-descriptions-item>
            <el-descriptions-item label="描述">
              <markdown-renderer
                v-if="detailDialog.data.description"
                :model-value="detailDialog.data.description" />
              <span v-else>暂无描述</span>
            </el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 20px; font-weight: bold; margin-bottom: 10px">文件列表</div>

          <div v-if="detailDialog.data.hasAccess">
            <el-table :data="detailDialog.files" border max-height="300px">
              <el-table-column prop="name" label="文件名" />
              <el-table-column prop="size" label="大小" width="120">
                <template #default="{ row }">{{ formatFileSize(row.size) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="100" align="center">
                <template #default="{ row }">
                  <el-button
                    type="primary"
                    link
                    :icon="Download"
                    @click="handleDownloadFile(detailDialog.data.id, row)"
                    >下载</el-button
                  >
                </template>
              </el-table-column>
            </el-table>
          </div>

          <div v-else class="no-access-placeholder">
            <el-empty description="您暂时没有权限查看或下载该数据集的文件内容。" :image-size="80">
              <template #default>
                <div class="no-access-actions">
                  <p class="no-access-tip">请提交申请，通过审批后即可访问所有文件。</p>
                </div>
              </template>
            </el-empty>
          </div>
        </div>
      </el-skeleton>
    </el-dialog>

    <!-- 对话框5：统计数据 (管理员) -->
    <el-dialog v-model="statisticsDialog.visible" title="数据集统计概览" width="700px">
      <el-skeleton :rows="4" animated :loading="statisticsDialog.loading">
        <div v-if="statisticsDialog.data" class="statistics-container">
          <el-row :gutter="40">
            <!-- 左侧：申请人次、提交次数、图表 -->
            <el-col :span="16" class="stats-left-col">
              <div class="stats-top-row">
                <div class="stat-item">
                  <el-statistic title="申请(人次)" :value="statisticsDialog.data.totalApplications" />
                </div>
                <div class="stat-item">
                  <el-statistic :value="statisticsDialog.data.totalAudits">
                    <template #title>
                      <div class="statistic-header-wrapper">
                        <span>申请提交(次)</span>
                        <el-tooltip content="包括退回重新提交" placement="top">
                          <el-icon class="help-icon"><QuestionFilled /></el-icon>
                        </el-tooltip>
                      </div>
                    </template>
                  </el-statistic>
                </div>
              </div>

              <div class="chart-wrapper">
                <div class="chart-title el-statistic__head">申请提交分布</div>
                <div ref="chartRef" style="width: 100%; height: 220px;"></div>
              </div>
            </el-col>

            <!-- 右侧：被批准、通过率 -->
            <el-col :span="8" class="stats-right-col">
              <div class="stat-item-right">
                <el-statistic title="被批准的申请(人次)" :value="statisticsDialog.data.approvedCount" />
              </div>
              <div class="stat-item-right">
                <el-statistic title="通过率" :value="statisticsDialog.data.approvalRate * 100" :precision="1" suffix="%" />
              </div>
            </el-col>
          </el-row>
        </div>
        <el-empty v-else description="暂无数据" />
      </el-skeleton>
    </el-dialog>
  </div>
</template>

<style scoped>
.file-dialog-toolbar {
  margin-bottom: 20px;
}
.dataset-detail-description {
  padding: 10px 20px 10px 60px;
  background-color: #fafafa;
  border-radius: 4px;
}
.dataset-detail-description p {
  margin: 0;
  line-height: 1.7;
  color: #606266;
  white-space: pre-wrap;
}
.upload-progress-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}
.upload-progress-item {
  display: flex;
  align-items: center;
  gap: 12px;
}
.upload-progress-name {
  flex: 0 0 220px;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.upload-progress-item :deep(.el-progress) {
  flex: 1;
}
.application-attachment-form-item :deep(.el-form-item__content) {
  width: 100%;
}
.application-attachment-wrapper {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.application-upload-controls {
  width: 100%;
}
.application-attachment-info {
  width: 100%;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
  background-color: #f9fafc;
}
.application-attachment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.application-attachment-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background-color: #fff;
}
.attachment-name {
  font-size: 14px;
  color: #303133;
  word-break: break-all;
}
.attachment-badge {
  margin-left: 8px;
}
.attachment-size {
  font-size: 12px;
  color: #909399;
}
.el-upload__tip {
  color: #909399;
  font-size: 12px;
  margin-left: 0;
}
.application-attachment-info :deep(.el-empty__description) {
  color: #909399;
}
.application-step-header {
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.application-step-progress {
  flex: 1;
}
.step-return-button {
  flex-shrink: 0;
}
.template-description-box {
  padding: 16px;
  background-color: #f9fafc;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  min-height: 100px;
  margin-bottom: 16px;
}
.upload-section {
  margin-bottom: 20px;
}
.upload-label {
  font-weight: 500;
  margin-bottom: 4px;
}
.required-label::after {
  content: " *";
  color: #f56c6c;
}
.upload-desc {
  font-size: 12px;
  color: #909399;
  margin-bottom: 12px;
}
.agreement-step {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.agreement-viewer-wrapper {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: auto;
  height: 360px;
  background-color: #fff;
}
.agreement-viewer-frame {
  width: 100%;
  height: 100%;
  border: none;
  display: block;
}
.agreement-upload-section {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  background-color: #f9fafc;
}
.agreement-upload-title {
  font-weight: 500;
  margin-bottom: 12px;
}
.agreement-usage-section {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  background-color: #fff;
}
.agreement-usage-title {
  font-weight: 500;
  margin-bottom: 12px;
}
.agreement-usage-box {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.agreement-usage-input {
  flex: 1;
}
.agreement-usage-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}
.agreement-image-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
}
.agreement-image-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background-color: #fff;
}
.agreement-image-item img {
  width: 80px;
  height: 80px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid #ebeef5;
}
.agreement-image-meta {
  flex: 1;
  overflow: hidden;
}
.agreement-image-name {
  font-size: 14px;
  color: #303133;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}
.agreement-image-size {
  font-size: 12px;
  color: #909399;
}
.agreement-upload-empty :deep(.el-empty__description) {
  color: #909399;
}
.no-access-placeholder {
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  background-color: #fafafa;
  height: 220px; /* 保持和有权限时的表格高度一致，避免弹窗抖动 */
  display: flex;
  justify-content: center;
  align-items: center;
}

.no-access-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
}

.no-access-tip {
  color: #909399;
  font-size: 14px;
  margin: 0;
}

.statistics-container {
  padding: 10px 20px;
}

.statistic-header-wrapper {
  display: flex;
  align-items: center;
  gap: 4px;
}
.help-icon {
  cursor: pointer;
  font-size: 14px;
}

.chart-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.chart-title {
  margin-bottom: 8px;
}

.stats-left-col {
  border-right: 1px solid #ebeef5;
}
.stats-top-row {
  display: flex;
  gap: 40px;
  margin-bottom: 20px;
}
.stats-right-col {
  display: flex;
  flex-direction: column;
  justify-content: space-around;
}

.manage-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.manage-section-title {
  font-size: 14px;
  font-weight: normal;
  color: #303133;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.upload-tip {
  font-size: 12px;
  color: #909399;
}
.manage-content-wrapper {
  padding: 10px 0;
}
.file-name-text {
  font-weight: normal;
}

.markdown-editor-container {
  display: flex;
  gap: 10px;
}
.editor-pane, .preview-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.h-100-input {
  height: 100%;
}
.h-100-input :deep(.el-textarea__inner) {
  height: 100%;
  resize: none;
  font-family: monospace;
}
.preview-content {
  flex: 1;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 10px;
  overflow-y: auto;
  background-color: #fafafa;
}
.editor-divider {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
}
.divider-icon {
  transform: rotate(90deg);
  font-size: 20px;
}
</style>
