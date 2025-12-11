<!-- src/views/dataset/DatasetList.vue -->
<script setup>
import { ref, onMounted, inject, reactive, watch } from 'vue'
import { Plus, Edit, Delete, Document, Upload, Download, Promotion } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '../../request/axios.js'

// --- API 函数 ---
const fetchDatasetListAPI = () => apiClient.get('/api/v1/datasets/')
const createDatasetAPI = (data) => apiClient.post('/api/v1/datasets/', data)
const updateDatasetAPI = (uuid, data) => apiClient.patch(`/api/v1/datasets/${uuid}`, data)
const deleteDatasetAPI = (uuid) => apiClient.delete(`/api/v1/datasets/${uuid}`)

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

const submitApplicationAPI = (applicationData) => apiClient.post('/api/v1/applications', applicationData)
const uploadApplicationFileAPI = (applicationId, file, config = {}) => {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post(`/api/v1/applications/${applicationId}/files`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    ...config,
  })
}

const FILE_SIZE_LIMIT_MB = 50
const FILE_SIZE_LIMIT_BYTES = FILE_SIZE_LIMIT_MB * 1024 * 1024

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
const applicationFormRef = ref(null)
const uploadRef = ref(null)
const applicationDialog = reactive({
  visible: false,
  isSubmitting: false,
  dataset: null,
  form: { reason: '' },
  rules: { reason: [{ required: true, message: '申请理由不能为空', trigger: 'blur' }] },
  attachments: {
    files: [],
    uploadTasks: [],
    isUploading: false,
    createdApplicationId: '',
  },
})

const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

const formatFileSize = (value) => {
  const size = Number(value) || 0
  if (size <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let idx = 0
  let current = size
  while (current >= 1024 && idx < units.length - 1) {
    current /= 1024
    idx += 1
  }
  const formatted = idx === 0 ? current.toFixed(0) : current.toFixed(2)
  return `${formatted} ${units[idx]}`
}

const normalizeDatasets = (datasets = []) =>
  datasets.map((item) => {
    const files = Array.isArray(item.Files) ? item.Files : []
    const creatorName =
      item.Creator?.NickName || item.creator || (item.CreatedBy ? `用户 ${item.CreatedBy}` : '未知')
    const datasetId = item.UUID || item.uuid || item.id || item.ID
    const createdAt =
      item.Time?.CreatedAt || item.CreatedAt || 
      item.Time?.createdAt || item.createdAt || 
      item.Time?.created_at || item.created_at
    return {
      id: datasetId ? String(datasetId) : `dataset-${Date.now()}`,
      name: item.Title || item.title || item.name || '未命名数据集',
      description: item.Description || item.description || '暂无描述',
      fileCount: item.fileCount ?? files.length,
      creator: creatorName,
      createdAt: formatDateTime(createdAt),
      raw: item,
    }
  })

const normalizeDatasetFiles = (files = []) =>
  files.map((item) => ({
    id: item.file_id || item.fileId || item.fileID || item.FileId || item.FileID,
    name: item.name || item.Name || '未命名文件',
    size: Number(item.size ?? item.Size ?? 0),
    createdAt: formatDateTime(item.createdAt || item.created_at || item.Time?.CreatedAt),
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
onMounted(fetchDatasets)

watch(
  () => applicationDialog.visible,
  (visible) => {
    if (!visible) {
      resetApplicationAttachments()
      applicationDialog.dataset = null
      applicationDialog.form.reason = ''
    }
  },
)

// --- 数据集 CRUD 事件处理 ---
const handleAdd = () => {
  datasetFormDialog.title = '新增数据集'
  datasetFormDialog.form = { id: null, name: '', description: '' }
  datasetFormDialog.visible = true
}
const handleEdit = (row) => {
  datasetFormDialog.title = '编辑数据集'
  datasetFormDialog.form = { ...row }
  datasetFormDialog.visible = true
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
        ElMessage.success('数据集更新成功！')
        await fetchDatasets()
      } else {
        ElMessage.error(res.data?.message || '更新失败')
      }
    } else {
      // 新增逻辑
      const res = await createDatasetAPI({ title: form.name, description: form.description })
      if (res.data?.message === 'success') {
        ElMessage.success('新增数据集成功！')
        await fetchDatasets() // 重新获取列表
      } else {
        ElMessage.error(res.data.message || '新增失败')
      }
    }
    datasetFormDialog.visible = false
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

// --- 文件管理事件处理 ---
const handleBeforeUpload = (rawFile) => {
  if (rawFile.size > FILE_SIZE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${FILE_SIZE_LIMIT_MB}MB 限制，无法上传`)
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

const loadFilesForDataset = async (dataset) => {
  if (!dataset?.id) return
  fileDialog.isLoading = true
  fileDialog.list = []
  try {
    const res = await fetchFilesForDatasetAPI(dataset.id)
    if (res?.data?.message === 'success') {
      fileDialog.list = normalizeDatasetFiles(res.data.files)
      dataset.fileCount = fileDialog.list.length
    } else {
      ElMessage.error(res?.data?.message || '文件列表加载失败')
    }
  } catch (error) {
    console.error('Failed to load files', error)
    ElMessage.error('文件列表加载失败')
  } finally {
    fileDialog.isLoading = false
  }
}
const handleViewFiles = (row) => {
  fileDialog.currentDataset = row
  fileDialog.visible = true
  fileDialog.uploadTasks = []
  loadFilesForDataset(row)
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
      await loadFilesForDataset(fileDialog.currentDataset)
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
const handleDownloadFile = async (file) => {
  if (!fileDialog.currentDataset?.id) {
    ElMessage.error('未找到对应的数据集')
    return
  }
  try {
    const res = await downloadDatasetFileAPI(fileDialog.currentDataset.id, file.id)
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
          await loadFilesForDataset(fileDialog.currentDataset)
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
const handleApply = (row) => {
  applicationDialog.dataset = row
  applicationDialog.form.reason = ''
  resetApplicationAttachments()
  applicationDialog.visible = true
}
const resetApplicationAttachments = () => {
  applicationDialog.attachments.files = []
  applicationDialog.attachments.uploadTasks = []
  applicationDialog.attachments.isUploading = false
  applicationDialog.attachments.createdApplicationId = ''
  if (uploadRef.value) {
    uploadRef.value.clearFiles()
  }
}
const ensureApplicationAttachmentTask = (file) => {
  const existing = applicationDialog.attachments.uploadTasks.find((task) => task.uid === file.uid)
  if (existing) return existing
  const task = {
    uid: file.uid,
    name: file.name,
    percentage: 0,
    status: 'pending',
    uploaded: false,
  }
  applicationDialog.attachments.uploadTasks.push(task)
  return task
}
const updateApplicationAttachmentTask = (uid, payload = {}) => {
  const task = applicationDialog.attachments.uploadTasks.find((item) => item.uid === uid)
  if (task) {
    Object.assign(task, payload)
  }
}
const handleApplicationAttachmentBeforeUpload = (rawFile) => {
  if (rawFile.size > FILE_SIZE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${FILE_SIZE_LIMIT_MB}MB 限制，无法上传`)
    return false
  }
  return true
}
const handleQueueApplicationAttachment = (options) => {
  const { file, onError, onSuccess } = options
  if (!handleApplicationAttachmentBeforeUpload(file)) {
    onError?.(new Error('文件大小超出限制'))
    return
  }
  const exists = applicationDialog.attachments.files.some((item) => item.uid === file.uid)
  if (!exists) {
    applicationDialog.attachments.files.push({
      uid: file.uid,
      name: file.name,
      size: file.size,
      raw: file,
    })
    ensureApplicationAttachmentTask(file)
    updateApplicationAttachmentTask(file.uid, { status: 'ready', percentage: 100, uploaded: false })
  }
  onSuccess?.({}, file)
}
const handleRemoveQueuedAttachment = (file) => {
  if (applicationDialog.attachments.isUploading) return
  const index = applicationDialog.attachments.files.findIndex((item) => item.uid === file.uid)
  if (index !== -1) {
    applicationDialog.attachments.files.splice(index, 1)
  }
  const taskIndex = applicationDialog.attachments.uploadTasks.findIndex((item) => item.uid === file.uid)
  if (taskIndex !== -1) {
    applicationDialog.attachments.uploadTasks.splice(taskIndex, 1)
  }
}
const uploadAttachmentsForApplication = async (applicationId) => {
  const pendingFiles = applicationDialog.attachments.files.filter((file) => {
    const task = applicationDialog.attachments.uploadTasks.find((item) => item.uid === file.uid)
    return !task || !task.uploaded
  })
  if (!pendingFiles.length) {
    return { failed: false }
  }
  applicationDialog.attachments.isUploading = true
  let hasError = false
  for (const file of pendingFiles) {
    updateApplicationAttachmentTask(file.uid, { status: 'uploading' })
    try {
      await uploadApplicationFileAPI(applicationId, file.raw)
      updateApplicationAttachmentTask(file.uid, { status: 'success', uploaded: true })
    } catch (error) {
      console.error('Failed to upload application attachment', error)
      updateApplicationAttachmentTask(file.uid, { status: 'error', uploaded: false })
      hasError = true
    }
  }
  applicationDialog.attachments.isUploading = false
  if (!hasError) {
    ElMessage.success('上传附件成功')
  }
  return { failed: hasError }
}
const handleSubmitApplication = async () => {
  if (!applicationFormRef.value) return
  const submitAction = async () => {
    applicationDialog.isSubmitting = true
    try {
      let applicationId = applicationDialog.attachments.createdApplicationId
      if (!applicationId) {
        const res = await submitApplicationAPI({
          datasetId: applicationDialog.dataset.id,
          reason: applicationDialog.form.reason,
        })
        if (res.data?.message === 'success' && res.data.id) {
          applicationId = res.data.id
          applicationDialog.attachments.createdApplicationId = res.data.id
        } else {
          ElMessage.error(res.data?.message || '申请提交失败')
          return
        }
      }

      const { failed } = await uploadAttachmentsForApplication(applicationId)
      if (failed) {
        ElMessage.error('部分附件上传失败，请检查后重试')
        return
      }

      ElMessage.success('申请提交成功，请等待管理员审批')
      applicationDialog.visible = false
      resetApplicationAttachments()
    } catch (error) {
      console.error('Application submission failed:', error)
      ElMessage.error('提交申请时发生错误')
    } finally {
      applicationDialog.isSubmitting = false
    }
  }

  if (applicationDialog.attachments.createdApplicationId) {
    await submitAction()
    return
  }

  await applicationFormRef.value.validate(async (valid) => {
    if (valid) {
      await submitAction()
    } else {
      ElMessage.warning('请填写必要的申请信息')
    }
  })
}
</script>

<template>
  <div>
    <!-- 主体：数据集列表 -->
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>数据集列表</span>
          <el-button v-if="isAdmin()" type="primary" :icon="Plus" @click="handleAdd"
            >新增数据集</el-button
          >
        </div>
      </template>

      <el-skeleton :rows="6" animated :loading="isDataLoading">
        <el-table :data="datasetList" stripe style="width: 100%">
          <el-table-column v-if="isAdmin()" type="expand">
            <template #default="props">
              <div class="dataset-detail-description">
                <p>{{ props.row.description }}</p>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="数据集名称" min-width="200" />
          <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
          <el-table-column prop="fileCount" label="文件数" width="90" align="center" />
          <el-table-column prop="creator" label="创建者" width="120" />
          <el-table-column prop="createdAt" label="创建时间" width="160" />
          <el-table-column label="操作" width="280" fixed="right" align="center">
            <template #default="scope">
              <!-- 管理员操作 -->
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
              <!-- 普通用户操作 -->
              <template v-else>
                <el-button type="primary" :icon="Promotion" @click="handleApply(scope.row)"
                  >申请使用</el-button
                >
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
                @click="handleDownloadFile(scope.row)"
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
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="applicationFormRef"
        :model="applicationDialog.form"
        :rules="applicationDialog.rules"
        label-position="top"
        v-loading="applicationDialog.isSubmitting"
        element-loading-text="正在提交申请..."
      >
        <el-form-item label="申请理由" prop="reason">
          <el-input
            v-model="applicationDialog.form.reason"
            type="textarea"
            :rows="5"
            placeholder="请详细说明您申请此数据集的用途、目的以及期望的使用方式。"
          />
        </el-form-item>
        <el-form-item label="上传附件 (可选)">
          <div class="application-attachment-wrapper">
            <div class="application-upload-controls">
              <el-upload
                ref="uploadRef"
                action="#"
                :show-file-list="false"
                multiple
                :before-upload="handleApplicationAttachmentBeforeUpload"
                :http-request="handleQueueApplicationAttachment"
                :disabled="applicationDialog.isSubmitting || applicationDialog.attachments.isUploading"
              >
                <el-button
                  type="primary"
                  :icon="Upload"
                  :disabled="applicationDialog.isSubmitting || applicationDialog.attachments.isUploading"
                  >选择文件</el-button
                >
                <template #tip>
                  <div class="el-upload__tip">
                    可上传项目计划书、需求文档等补充材料，单个文件不超过 {{ FILE_SIZE_LIMIT_MB }}MB
                  </div>
                </template>
              </el-upload>
            </div>
            <div class="application-attachment-info">
              <el-empty
                v-if="!applicationDialog.attachments.files.length"
                description="尚未选择附件"
                :image-size="100"
              />
              <div v-else class="application-attachment-list">
                <div
                  v-for="file in applicationDialog.attachments.files"
                  :key="file.uid"
                  class="application-attachment-item"
                >
                  <div>
                    <div class="attachment-name">{{ file.name }}</div>
                    <div class="attachment-size">{{ formatFileSize(file.size) }}</div>
                  </div>
                  <el-button
                    text
                    type="danger"
                    size="small"
                    :disabled="applicationDialog.attachments.isUploading"
                    @click="handleRemoveQueuedAttachment(file)"
                    >移除</el-button
                  >
                </div>
              </div>
              <div v-if="applicationDialog.attachments.uploadTasks.length" class="upload-progress-list">
                <div
                  v-for="task in applicationDialog.attachments.uploadTasks"
                  :key="task.uid"
                  class="upload-progress-item"
                >
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
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button
          @click="applicationDialog.visible = false"
          :disabled="applicationDialog.isSubmitting || applicationDialog.attachments.isUploading"
          >取消</el-button
        >
        <el-button
          type="primary"
          @click="handleSubmitApplication"
          :loading="applicationDialog.isSubmitting"
          :disabled="applicationDialog.attachments.isUploading"
          >{{ applicationDialog.attachments.createdApplicationId ? '继续上传附件' : '提交申请' }}</el-button
        >
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 18px;
}
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
.application-attachment-wrapper {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.application-attachment-info {
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
</style>
