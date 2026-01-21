<!-- src/views/dataset/DatasetList.vue -->
<script setup>
import { ref, onMounted, inject, reactive, watch } from 'vue'
import {
  Plus,
  Edit,
  Delete,
  Document,
  Upload,
  Download,
  Promotion,
  View,
  Refresh,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import {
  formatDateTime,
  formatShortId,
  formatFileSize,
} from '@/utils/format.js'

// --- API 函数 ---
const fetchDatasetListAPI = () => apiClient.get('/api/v1/datasets/')
const fetchDatasetDetailAPI = (uuid) => apiClient.get(`/api/v1/datasets/${uuid}`)
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
const uploadApplicationFileAPI = (applicationId, file, config = {}) => {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post(`/api/v1/applications/${applicationId}/files`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    ...config,
  })
}

const DATASET_FILE_LIMIT_MB = 500
const DATASET_FILE_LIMIT_BYTES = DATASET_FILE_LIMIT_MB * 1024 * 1024

const APPLICATION_FILE_LIMIT_MB = 20
const APPLICATION_FILE_LIMIT_BYTES = APPLICATION_FILE_LIMIT_MB * 1024 * 1024

const ALLOWED_EXTENSIONS = [
  // Documents
  '.pdf',
  '.txt',
  '.md',
  '.rtf',
  '.tex',
  '.doc',
  '.docx',
  '.wps',
  '.odt',
  '.ppt',
  '.pptx',
  '.dps',
  '.ods',
  '.xls',
  '.xlsx',
  '.et',
  '.odp',
  // Images
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.svg',
  '.bmp',
  '.tiff',
  // Compressed Files
  '.zip',
  '.rar',
  '.7z',
  '.tar',
  '.gz',
  '.tgz',
]

const DENIED_EXTENSIONS = [
  '.exe',
  '.msi',
  '.bat',
  '.cmd',
  '.sh',
  '.ps1',
  '.js',
  '.vbs',
  '.jar',
  '.dll',
  '.sys',
  '.app',
  '.dmg',
  '.pkg',
  '.scr',
  '.com',
  '.cpl',
  '.inf',
  '.lnk',
  '.reg',
  '.hta',
  '.msc',
]

// --- 状态定义 ---
const isAdmin = inject('isAdmin')
const datasetList = ref([])
const isDataLoading = ref(true)

// 状态1: 新增/编辑数据集对话框
const datasetFormDialog = reactive({
  visible: false,
  title: '',
  form: { id: null, name: '', description: '' },
})

// 状态2: 文件管理对话框
const fileDialog = reactive({
  visible: false,
  currentDataset: null,
  list: [],
  isLoading: false,
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
  mandatory: {
    file: null,
  },
  optional: {
    files: [],
    uploadTasks: [],
    isUploading: false,
    createdApplicationId: '',
  },
})

// 状态4: 数据集详情对话框 (普通用户查看)
const detailDialog = reactive({
  visible: false,
  loading: false,
  data: null,
  files: [],
})

const normalizeOneDataset = (item) => {
  return {
    id: item.id,
    name: item.title,
    description: item.description,
    fileCount: item.fileCount || (item.files ? item.files.length : 0),
    creator: item.creator,
    createdAt: formatDateTime(item.createdAt || item.created_at),
    hasAccess: item.hasAccess,
    raw: item,
  }
}
const normalizeDatasets = (datasets = []) => datasets.map(normalizeOneDataset)

const normalizeDatasetFiles = (files = []) =>
  files.map((item) => ({
    id: item.id || item.file_id || item.fileId || item.fileID || item.FileId || item.FileID,
    name: item.name || item.Name || '未命名文件',
    size: Number(item.size ?? item.Size ?? 0),
    createdAt: formatDateTime(item.createdAt || item.created_at),
    raw: item,
  }))

const updateUploadingState = () => {
  fileDialog.isUploading = fileDialog.uploadTasks.some((task) => task.status === 'uploading')
}

const ensureUploadTask = (file) => {
  const existing = fileDialog.uploadTasks.find((task) => task.uid === file.uid)
  if (existing) return existing
  const task = {
    uid: file.uid,
    name: file.name,
    percentage: 0,
    status: 'uploading',
  }
  fileDialog.uploadTasks.push(task)
  updateUploadingState()
  return task
}

const updateUploadTask = (uid, payload = {}) => {
  const task = fileDialog.uploadTasks.find((item) => item.uid === uid)
  if (task) {
    Object.assign(task, payload)
  }
}

const removeUploadTask = (uid, delay = 0) => {
  const remover = () => {
    const index = fileDialog.uploadTasks.findIndex((item) => item.uid === uid)
    if (index !== -1) fileDialog.uploadTasks.splice(index, 1)
    updateUploadingState()
  }
  if (delay > 0) {
    setTimeout(remover, delay)
  } else {
    remover()
  }
}

const fetchDatasets = async () => {
  isDataLoading.value = true
  try {
    const res = await fetchDatasetListAPI()
    if (res && res.data?.message === 'success') {
      datasetList.value = normalizeDatasets(res.data.datasets)
      return
    }
    ElMessage.error(res?.data?.message || '数据集加载失败')
  } catch {
    ElMessage.error('数据集加载失败')
  } finally {
    isDataLoading.value = false
  }
}
// --- 生命周期钩子 ---
onMounted(() => {
  fetchDatasets()
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

// --- 数据集 CRUD 事件处理 ---
const handleAdd = () => {
  datasetFormDialog.title = '新增数据集'
  datasetFormDialog.form = { id: null, name: '', description: '' }
  datasetFormDialog.visible = true
  datasetFormDialog.loading = false
}
const handleEdit = async (row) => {
  datasetFormDialog.title = '编辑数据集'
  datasetFormDialog.visible = true
  datasetFormDialog.loading = true
  // 先重置表单防止闪烁，保留 ID 和 Name (列表里有) 作为占位
  datasetFormDialog.form = { id: row.id, name: row.name, description: '' }

  try {
    // 调用 GetOne 获取完整信息 (包含 Description)
    const res = await fetchDatasetDetailAPI(row.id)
    if (res.data?.message === 'success') {
      const detail = res.data.dataset
      datasetFormDialog.form = {
        id: detail.id,
        name: detail.title,
        description: detail.description || '',
      }
    } else {
      ElMessage.error('获取详情失败，无法编辑完整信息')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取详情失败')
  } finally {
    datasetFormDialog.loading = false
  }
}
const handleSaveDataset = async () => {
  const form = datasetFormDialog.form
  if (!form.name.trim()) {
    ElMessage.warning('数据集名称不能为空')
    return
  }

  try {
    if (form.id) {
      const res = await updateDatasetAPI(form.id, {
        title: form.name,
        description: form.description,
      })
      if (res.data?.message === 'success') {
        ElMessage.success('更新成功')
        await fetchDatasets()
        datasetFormDialog.visible = false
      } else {
        ElMessage.error(res.data?.message || '更新失败')
      }
    } else {
      // 新增
      const res = await createDatasetAPI({ title: form.name, description: form.description })
      if (res.data?.message === 'success') {
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
        if (res.data?.message === 'success') {
          ElMessage.success(`数据集 "${row.name}" 已删除`)
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
    if (res.data?.message === 'success') {
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
const handleApplyFromDetail = () => {
  if (!detailDialog.data) return

  // 1. 关闭详情弹窗
  detailDialog.visible = false

  // 2. 打开申请弹窗（复用已有的逻辑）
  handleApply(normalizeOneDataset(detailDialog.data))
}

// --- 文件管理事件处理 ---
const handleBeforeUpload = (rawFile) => {
  if (rawFile.size > DATASET_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${DATASET_FILE_LIMIT_MB}MB 限制，无法上传`)
    return false
  }
  return true
}

const handleFileDialogClose = () => {
  fileDialog.visible = false
  fileDialog.currentDataset = null
  fileDialog.list = []
  fileDialog.uploadTasks = []
  fileDialog.isUploading = false
}

const loadFilesForDataset = async (datasetId) => {
  fileDialog.isLoading = true
  fileDialog.list = []
  try {
    console.log('文件列表', datasetId)
    const res = await fetchFilesForDatasetAPI(datasetId)
    if (res?.data?.message === 'success') {
      fileDialog.list = normalizeDatasetFiles(res.data.files)
    } else {
      ElMessage.error('文件列表加载失败')
    }
  } catch {
    ElMessage.error('文件列表加载失败')
  } finally {
    fileDialog.isLoading = false
  }
}
const handleViewFiles = (row) => {
  fileDialog.currentDataset = row
  fileDialog.visible = true
  fileDialog.uploadTasks = []
  console.log(row.id)
  loadFilesForDataset(row.id)
}
const handleDatasetFileUpload = async (options) => {
  const { file, onError, onSuccess, onProgress } = options
  if (!fileDialog.currentDataset?.id) {
    const err = new Error('未选择数据集，无法上传文件')
    onError?.(err)
    ElMessage.error(err.message)
    return
  }
  const task = ensureUploadTask(file)
  onProgress?.({ percent: 0 })
  try {
    const res = await uploadDatasetFileAPI(fileDialog.currentDataset.id, file, {
      onUploadProgress: (event) => {
        const percentage = event.total ? Math.round((event.loaded / event.total) * 100) : 0
        updateUploadTask(task.uid, { percentage })
        onProgress?.({ percent: percentage })
      },
    })
    if (res?.data?.message === 'success') {
      ElMessage.success(`${file.name} 上传成功`)
      await loadFilesForDataset(fileDialog.currentDataset.id)
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
  if (!fileDialog.currentDataset?.id) return
  ElMessageBox.confirm(`确定要删除文件 "${file.name}" 吗？`, '提示', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(async () => {
      try {
        const res = await deleteDatasetFileAPI(fileDialog.currentDataset.id, file.id)
        if (res?.data?.message === 'success') {
          ElMessage.success('文件已删除')
          await loadFilesForDataset(fileDialog.currentDataset.id)
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
  applicationDialog.mandatory.file = null
  applicationDialog.optional.files = []
  applicationDialog.optional.uploadTasks = []
  applicationDialog.optional.createdApplicationId = ''
  applicationDialog.template.description = '加载中...'
  applicationDialog.template.files = []

  applicationDialog.visible = true

  try {
    const res = await fetchPublicTemplateFilesAPI()
    if (res.data?.message === 'success') {
      applicationDialog.template.description = res.data.description || '暂无说明'
      applicationDialog.template.files = normalizeDatasetFiles(res.data.files || [])
    } else {
      ElMessage.error('获取申请模板信息失败')
    }
  } catch (e) {
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

const handleMandatoryFileChange = (file) => {
  const rawFile = file.raw
  if (rawFile.type !== 'application/pdf') {
    ElMessage.error('仅支持上传 PDF 文件')
    // 清除文件列表
    applicationDialog.mandatory.file = null
    return
  }
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件大小不能超过 ${APPLICATION_FILE_LIMIT_MB}MB`)
    applicationDialog.mandatory.file = null
    return
  }
  applicationDialog.mandatory.file = rawFile
}

const handleOptionalFileChange = (file, fileList) => {
  const idx = fileList.indexOf(file)

  // 仅在文件存在于列表中时（新增时）进行校验
  if (idx !== -1) {
    const rawFile = file.raw
    const fileName = rawFile.name.toLowerCase()

    if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
      ElMessage.error(`文件 ${rawFile.name} 超过大小限制`)
      fileList.splice(idx, 1)
      return
    }

    const isDenied = DENIED_EXTENSIONS.some((ext) => fileName.endsWith(ext))
    if (isDenied) {
      ElMessage.error(`文件格式被拒绝："${rawFile.name}"`)
      fileList.splice(idx, 1)
      return
    }

    const isAllowed = ALLOWED_EXTENSIONS.some((ext) => fileName.endsWith(ext))
    if (!isAllowed) {
      ElMessage.error(`文件格式不受支持："${rawFile.name}"`)
      fileList.splice(idx, 1)
      return
    }
  }
  // 更新列表
  applicationDialog.optional.files = fileList.map((f) => f.raw)
}

const handleSubmitApplication = async () => {
  if (!applicationDialog.mandatory.file) {
    ElMessage.warning('请上传必填的申请文件')
    return
  }

  applicationDialog.isSubmitting = true
  try {
    let applicationId = applicationDialog.optional.createdApplicationId
    if (!applicationId) {
      // 提交申请，不再需要 reason
      const res = await submitApplicationAPI({
        datasetId: applicationDialog.dataset.id,
      })
      if (res.data?.message === 'success' && res.data.id) {
        applicationId = res.data.id
        applicationDialog.optional.createdApplicationId = res.data.id
      } else {
        ElMessage.error(res.data?.message || '申请提交失败')
        applicationDialog.isSubmitting = false
        return
      }
    }

    // 上传必填文件
    try {
      await uploadApplicationFileAPI(applicationId, applicationDialog.mandatory.file)
    } catch (e) {
      ElMessage.error('必填文件上传失败')
      applicationDialog.isSubmitting = false
      return
    }

    // 上传可选文件
    for (const file of applicationDialog.optional.files) {
      try {
        await uploadApplicationFileAPI(applicationId, file)
      } catch (e) {
        console.warn('Optional file upload failed', file.name)
      }
    }

    applicationDialog.currentStep = 3
  } catch (error) {
    console.error('Application submission failed:', error)
    ElMessage.error('提交申请时发生错误')
  } finally {
    applicationDialog.isSubmitting = false
  }
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
            <el-button :icon="Refresh" circle @click="fetchDatasets" :loading="isDataLoading" title="刷新列表" />
            <el-button v-if="isAdmin()" type="primary" :icon="Plus" @click="handleAdd">新增数据集</el-button>
          </div>
        </div>
      </template>

      <el-skeleton :rows="6" animated :loading="isDataLoading">
        <el-table :data="datasetList" stripe style="width: 100%">
          <el-table-column label="内部ID" width="100">
            <template #default="{ row }">
              <el-tooltip effect="dark" :content="row.id" placement="top" :show-after="500">
                <span class="monospace-id">
                  {{ formatShortId(row.id) }}
                </span>
              </el-tooltip>
            </template>
          </el-table-column>

          <el-table-column prop="name" label="数据集名称" min-width="300">
            <template #default="{ row }">
              <span style="font-weight: 500">{{ row.name }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="fileCount" label="文件数" width="90" align="center" />
          <el-table-column prop="creator" label="创建者" width="120" />
          <el-table-column prop="createdAt" label="创建时间" width="160" />
          <el-table-column label="操作" width="280" fixed="right" align="center">
            <template #default="scope">
              <template v-if="isAdmin()">
                <el-button size="small" :icon="Document" @click="handleViewFiles(scope.row)"
                  >管理文件</el-button
                >
                <el-button size="small" type="primary" :icon="Edit" @click="handleEdit(scope.row)"
                  >编辑</el-button
                >
                <el-button
                  size="small"
                  type="danger"
                  :icon="Delete"
                  @click="handleDelete(scope.row)"
                  >删除</el-button
                >
              </template>

              <template v-else>
                <el-button
                  v-if="scope.row.hasAccess"
                  type="success"
                  :icon="Download"
                  size="small"
                  @click="handleViewDetail(scope.row)"
                >
                  下载文件
                </el-button>

                <template v-else>
                  <el-button
                    type="primary"
                    link
                    :icon="View"
                    size="small"
                    @click="handleViewDetail(scope.row)"
                  >
                    查看详情
                  </el-button>

                  <el-button
                    type="primary"
                    :icon="Promotion"
                    size="small"
                    @click="handleApply(scope.row)"
                  >
                    申请使用
                  </el-button>
                </template>
              </template>
            </template>
          </el-table-column>
        </el-table>
      </el-skeleton>
    </el-card>

    <!-- ======== 所有对话框 ======== -->

    <!-- 对话框1：新增/编辑数据集 (管理员) -->
    <el-dialog v-model="datasetFormDialog.visible" :title="datasetFormDialog.title" width="50%">
      <el-form :model="datasetFormDialog.form" label-width="100px">
        <el-form-item label="数据集名称">
          <el-input v-model="datasetFormDialog.form.name" placeholder="请输入数据集名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="datasetFormDialog.form.description"
            type="textarea"
            :rows="5"
            placeholder="请输入详细的数据集描述信息..."
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="datasetFormDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveDataset">确定</el-button>
      </template>
    </el-dialog>

    <!-- 对话框2：文件列表 (管理员) -->
    <el-dialog
      v-model="fileDialog.visible"
      :title="`文件列表 - ${fileDialog.currentDataset?.name}`"
      width="70%"
      top="5vh"
      @close="handleFileDialogClose"
    >
      <div class="file-dialog-toolbar">
        <el-upload
          action="#"
          :show-file-list="false"
          :disabled="!fileDialog.currentDataset"
          multiple
          :before-upload="handleBeforeUpload"
          :http-request="handleDatasetFileUpload"
        >
          <el-button type="primary" :icon="Upload" :loading="fileDialog.isUploading"
            >上传文件</el-button
          >
          <template #tip>
            <div class="el-upload__tip">单个文件不超过 {{ DATASET_FILE_LIMIT_MB }}MB</div>
          </template>
        </el-upload>
      </div>
      <div v-if="fileDialog.uploadTasks.length" class="upload-progress-list">
        <div v-for="task in fileDialog.uploadTasks" :key="task.uid" class="upload-progress-item">
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
      <el-skeleton :rows="5" animated :loading="fileDialog.isLoading">
        <el-table :data="fileDialog.list" stripe height="450px">
          <el-table-column property="name" label="文件名">
            <template #default="scope">
              <el-icon style="vertical-align: middle; margin-right: 5px"><Document /></el-icon>
              <span>{{ scope.row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="140">
            <template #default="scope">
              {{ formatFileSize(scope.row.size) }}
            </template>
          </el-table-column>
          <el-table-column property="createdAt" label="上传时间" width="180">
            <template #default="scope">
              {{ scope.row.createdAt || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" align="center">
            <template #default="scope">
              <el-button
                type="success"
                :icon="Download"
                size="small"
                @click="handleDownloadFile(fileDialog.currentDataset.id, scope.row)"
                >下载</el-button
              >
              <el-button
                type="danger"
                :icon="Delete"
                size="small"
                @click="handleDeleteFile(scope.row)"
                >删除</el-button
              >
            </template>
          </el-table-column>
        </el-table>
      </el-skeleton>
      <template #footer>
        <el-button @click="handleFileDialogClose">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 对话框3：申请使用 (普通用户) -->
    <el-dialog
      v-model="applicationDialog.visible"
      :title="`申请使用数据集 - ${applicationDialog.dataset?.name}`"
      width="720px"
      :close-on-click-modal="false"
    >
      <div class="application-step-header">
        <div class="application-step-progress">
          <el-steps :active="applicationDialog.currentStep" align-center>
            <el-step title="申请说明" />
            <el-step title="资料上传" />
            <el-step title="提交成功" />
          </el-steps>
        </div>
      </div>

      <div v-if="applicationDialog.currentStep === 1" class="agreement-step">
        <div>
          <div class="upload-label">申请说明</div>
          <div class="template-description-box">
            <div class="template-text">{{ applicationDialog.template.description }}</div>
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

      <div v-else-if="applicationDialog.currentStep === 2" class="upload-step" v-loading="applicationDialog.isSubmitting">
        <div class="upload-section">
          <div class="upload-label required-label">1. 上传申请文件 (必填)</div>
          <div class="upload-desc">请上传填写并签署后的申请模板文件，仅支持 PDF 格式，限传 1 个。</div>
          <el-upload
            class="upload-demo"
            action="#"
            :auto-upload="false"
            :limit="1"
            accept=".pdf"
            :on-change="handleMandatoryFileChange"
            :on-remove="() => { applicationDialog.mandatory.file = null }"
            :file-list="applicationDialog.mandatory.file ? [applicationDialog.mandatory.file] : []"
          >
            <el-button type="primary" :icon="Upload">选择 PDF 文件</el-button>
          </el-upload>
        </div>

        <el-divider />

        <div class="upload-section">
          <div class="upload-label">2. 补充材料 (可选)</div>
          <div class="upload-desc">如有其他证明材料，可在此上传，不限数量。</div>
          <el-upload
            class="upload-demo"
            action="#"
            :auto-upload="false"
            multiple
            :on-change="handleOptionalFileChange"
            :on-remove="handleOptionalFileChange"
          >
            <el-button :icon="Upload">选择文件</el-button>
          </el-upload>
        </div>
      </div>

      <div v-else-if="applicationDialog.currentStep === 3" class="result-step">
        <el-result icon="success" title="申请已提交" sub-title="请等待1至5个工作日查看结果">
        </el-result>
      </div>

      <template #footer>
        <template v-if="applicationDialog.currentStep === 1">
          <el-button @click="applicationDialog.visible = false">取消</el-button>
          <el-button type="primary" @click="applicationDialog.currentStep = 2">下一步</el-button>
        </template>
        <template v-else-if="applicationDialog.currentStep === 2">
          <el-button @click="applicationDialog.visible = false" :disabled="applicationDialog.isSubmitting">取消</el-button>
          <el-button @click="applicationDialog.currentStep = 1" :disabled="applicationDialog.isSubmitting">上一步</el-button>
          <el-button
            type="primary"
            @click="handleSubmitApplication"
            :loading="applicationDialog.isSubmitting"
          >提交申请</el-button>
        </template>
        <template v-else-if="applicationDialog.currentStep === 3">
          <el-button type="primary" @click="applicationDialog.visible = false">确定</el-button>
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
              <div style="white-space: pre-wrap">
                {{ detailDialog.data.description || '暂无描述' }}
              </div>
            </el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 20px; font-weight: bold; margin-bottom: 10px">文件列表</div>

          <div v-if="detailDialog.data.hasAccess">
            <el-table :data="detailDialog.files" stripe border height="300px">
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
                  <el-button
                    type="primary"
                    :icon="Promotion"
                    size="large"
                    @click="handleApplyFromDetail"
                  >
                    发起申请
                  </el-button>
                </div>
              </template>
            </el-empty>
          </div>
        </div>
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
.template-text {
  white-space: pre-wrap;
  line-height: 1.6;
  color: #606266;
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
  height: 300px; /* 保持和有权限时的表格高度一致，避免弹窗抖动 */
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
</style>
