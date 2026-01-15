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
import VuePdfEmbed from 'vue-pdf-embed'
import { PDFDocument, StandardFonts, rgb } from 'pdf-lib'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import {
  formatDateTime,
  formatShortId,
  formatFileSize,
  textToTransparentImage,
} from '@/utils/format.js'

// --- API 函数 ---
const fetchDatasetListAPI = () => apiClient.get('/api/v1/datasets/')
const fetchDatasetDetailAPI = (uuid) => apiClient.get(`/api/v1/datasets/${uuid}`)
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
const AGREEMENT_PDF_STATIC_PATH = '/static/agreement.pdf'

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
const agreementPdfUrl = ref(AGREEMENT_PDF_STATIC_PATH)

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
const agreementUploadRef = ref(null)
const uploadRef = ref(null)
const applicationDialog = reactive({
  visible: false,
  isSubmitting: false,
  currentStep: 1,
  dataset: null,
  form: { reason: '' },
  rules: { reason: [{ required: true, message: '申请理由不能为空', trigger: 'blur' }] },
  agreement: {
    files: [],
    uploadTasks: [],
    isUploading: false,
    intendedUsage: '',
    generatedAttachmentUid: '',
    isGeneratingPdf: false,
  },
  attachments: {
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
  if (typeof window !== 'undefined') {
    agreementPdfUrl.value = `${window.location.origin}${AGREEMENT_PDF_STATIC_PATH}`
  }
})

watch(
  () => applicationDialog.visible,
  (visible) => {
    if (!visible) {
      resetApplicationAttachments()
      resetAgreementUploads()
      applicationDialog.dataset = null
      applicationDialog.form.reason = ''
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
const handleApply = (row) => {
  applicationDialog.dataset = row
  applicationDialog.form.reason = ''
  resetApplicationAttachments()
  resetAgreementUploads()
  applicationDialog.visible = true
  applicationDialog.currentStep = 1
}
const resetApplicationAttachments = () => {
  applicationDialog.attachments.files = []
  applicationDialog.attachments.uploadTasks = []
  applicationDialog.attachments.isUploading = false
  applicationDialog.attachments.createdApplicationId = ''
  applicationDialog.agreement.generatedAttachmentUid = ''
  if (uploadRef.value) {
    uploadRef.value.clearFiles()
  }
}
const resetAgreementUploads = () => {
  applicationDialog.agreement.files.forEach((file) => {
    if (file.preview) {
      URL.revokeObjectURL(file.preview)
    }
  })
  applicationDialog.agreement.files = []
  applicationDialog.agreement.uploadTasks = []
  applicationDialog.agreement.isUploading = false
  applicationDialog.agreement.intendedUsage = ''
  applicationDialog.agreement.generatedAttachmentUid = ''
  applicationDialog.agreement.isGeneratingPdf = false
  if (agreementUploadRef.value) {
    agreementUploadRef.value.clearFiles()
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
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${APPLICATION_FILE_LIMIT_MB}MB 限制，无法上传`)
    return false
  }

  const fileName = rawFile.name.toLowerCase()

  const isDenied = DENIED_EXTENSIONS.some((ext) => fileName.endsWith(ext))
  if (isDenied) {
    ElMessage.error(`文件格式被拒绝："${rawFile.name}"`)
    return false
  }

  const isAllowed = ALLOWED_EXTENSIONS.some((ext) => fileName.endsWith(ext))
  if (!isAllowed) {
    ElMessage.error(`文件格式不受支持："${rawFile.name}"`)
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
    appendAttachmentEntry({
      uid: file.uid,
      name: file.name,
      size: file.size,
      raw: file,
      locked: false,
    })
  }
  onSuccess?.({}, file)
}
const handleRemoveQueuedAttachment = (file) => {
  if (applicationDialog.attachments.isUploading) return
  if (file.locked) {
    ElMessage.warning('系统生成的知情同意书不可删除')
    return
  }
  const index = applicationDialog.attachments.files.findIndex((item) => item.uid === file.uid)
  if (index !== -1) {
    applicationDialog.attachments.files.splice(index, 1)
  }
  const taskIndex = applicationDialog.attachments.uploadTasks.findIndex(
    (item) => item.uid === file.uid,
  )
  if (taskIndex !== -1) {
    applicationDialog.attachments.uploadTasks.splice(taskIndex, 1)
  }
}
const removeGeneratedAgreementAttachment = () => {
  const generatedUid = applicationDialog.agreement.generatedAttachmentUid
  if (!generatedUid) return
  const fileIndex = applicationDialog.attachments.files.findIndex(
    (item) => item.uid === generatedUid,
  )
  if (fileIndex !== -1) {
    applicationDialog.attachments.files.splice(fileIndex, 1)
  }
  const taskIndex = applicationDialog.attachments.uploadTasks.findIndex(
    (item) => item.uid === generatedUid,
  )
  if (taskIndex !== -1) {
    applicationDialog.attachments.uploadTasks.splice(taskIndex, 1)
  }
  applicationDialog.agreement.generatedAttachmentUid = ''
}
const appendAttachmentEntry = (entry) => {
  applicationDialog.attachments.files.push(entry)
  ensureApplicationAttachmentTask(entry)
  updateApplicationAttachmentTask(entry.uid, { status: 'ready', percentage: 100, uploaded: false })
}
const addGeneratedAgreementAttachment = (file) => {
  removeGeneratedAgreementAttachment()
  const entry = {
    uid: `agreement-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    name: file.name,
    size: file.size,
    raw: file,
    locked: true,
    systemGenerated: true,
  }
  appendAttachmentEntry(entry)
  applicationDialog.agreement.generatedAttachmentUid = entry.uid
}
const handleAgreementImageBeforeUpload = (rawFile) => {
  if (!rawFile.type.startsWith('image/')) {
    ElMessage.error('仅支持上传图片文件，请重新选择')
    return false
  }
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`图片 "${rawFile.name}" 超过 ${APPLICATION_FILE_LIMIT_MB}MB 限制，无法上传`)
    return false
  }
  return true
}
const handleQueueAgreementImage = (options) => {
  const { file, onError, onSuccess } = options
  if (!handleAgreementImageBeforeUpload(file)) {
    onError?.(new Error('图片校验失败'))
    return
  }
  applicationDialog.agreement.files.forEach((item) => {
    if (item.preview) URL.revokeObjectURL(item.preview)
  })
  applicationDialog.agreement.files = []
  const previewUrl = URL.createObjectURL(file)
  applicationDialog.agreement.files.push({
    uid: file.uid,
    name: file.name,
    size: file.size,
    raw: file,
    preview: previewUrl,
  })
  onSuccess?.({}, file)
}
const handleRemoveAgreementImage = (file) => {
  if (applicationDialog.agreement.isUploading) return
  const index = applicationDialog.agreement.files.findIndex((item) => item.uid === file.uid)
  if (index !== -1) {
    const [removed] = applicationDialog.agreement.files.splice(index, 1)
    if (removed?.preview) {
      URL.revokeObjectURL(removed.preview)
    }
  }
}
const generateSignedAgreementPdf = async () => {
  if (!agreementPdfUrl.value) {
    throw new Error('无法读取知情同意书模板')
  }
  const usage = applicationDialog.agreement.intendedUsage.trim()
  const signatureEntry = applicationDialog.agreement.files[0]
  if (!signatureEntry?.raw) {
    throw new Error('未找到签名图片')
  }
  const response = await fetch(agreementPdfUrl.value, { cache: 'no-store' })
  if (!response.ok) {
    throw new Error('加载知情同意书模板失败')
  }
  const pdfBytes = await response.arrayBuffer()
  const pdfDoc = await PDFDocument.load(pdfBytes)
  const font = await pdfDoc.embedFont(StandardFonts.Helvetica)
  const pages = pdfDoc.getPages()
  if (!pages.length) {
    throw new Error('知情同意书模板内容异常')
  }

  const scale = 3
  const usageImageBuffer = await textToTransparentImage(usage, { scale: scale })
  const usageImage = await pdfDoc.embedPng(usageImageBuffer)
  const usageImageWidth = usageImage.width / scale
  const usageImageHeight = usageImage.height / scale

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    timeZone: 'Asia/Shanghai',
  })
  const now = new Date()
  const parts = formatter.formatToParts(now)
  const year = parts.find((part) => part.type === 'year')?.value
  const month = parts.find((part) => part.type === 'month')?.value
  const day = parts.find((part) => part.type === 'day')?.value
  const yearText = `${year}`
  const monthText = `${month}`
  const dayText = `${day}`

  const coordinates = {
    usage: { x: 215, y: 350 },
    signature: { x: 420, y: 120 },
    year2: { x: 360, y: 99 },
    month2: { x: 406, y: 99 },
    day2: { x: 436, y: 99 },
    year1: { x: 129, y: 263 },
    month1: { x: 170, y: 263 },
    day1: { x: 198, y: 263 },
  }

  const page = pages[0]
  const page2 = pages[1]

  page.drawImage(usageImage, {
    x: coordinates.usage.x,
    y: coordinates.usage.y,
    width: usageImageWidth,
    height: usageImageHeight,
  })

  page2.drawText(yearText, {
    x: coordinates.year1.x,
    y: coordinates.year1.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })
  page2.drawText(monthText, {
    x: coordinates.month1.x,
    y: coordinates.month1.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })
  page2.drawText(dayText, {
    x: coordinates.day1.x,
    y: coordinates.day1.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })
  page2.drawText(yearText, {
    x: coordinates.year2.x,
    y: coordinates.year2.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })
  page2.drawText(monthText, {
    x: coordinates.month2.x,
    y: coordinates.month2.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })
  page2.drawText(dayText, {
    x: coordinates.day2.x,
    y: coordinates.day2.y,
    size: 12,
    font,
    color: rgb(0, 0, 0),
  })

  const signatureFile = signatureEntry.raw
  const signatureBuffer = await signatureFile.arrayBuffer()
  const lowerType = signatureFile.type?.toLowerCase() || ''
  const signatureImage = lowerType.includes('png')
    ? await pdfDoc.embedPng(signatureBuffer)
    : await pdfDoc.embedJpg(signatureBuffer)
  // 签名的显示区域最大值
  const maxWidth = 80
  const maxHeight = 20
  // 等比例缩放
  let signatureWidth = maxWidth
  let signatureHeight = (signatureImage.height / signatureImage.width) * signatureWidth
  if (signatureHeight > maxHeight) {
    const ratio = maxHeight / signatureHeight
    signatureWidth *= ratio
    signatureHeight = maxHeight
  }
  page2.drawImage(signatureImage, {
    x: coordinates.signature.x,
    y: coordinates.signature.y,
    width: signatureWidth,
    height: signatureHeight,
  })

  const modifiedBytes = await pdfDoc.save()
  const fileName = `知情同意书-${new Date().toISOString().slice(0, 10)}.pdf`
  return new File([modifiedBytes], fileName, { type: 'application/pdf' })
}
const handleConfirmAgreementStep = async () => {
  if (applicationDialog.agreement.isGeneratingPdf) return
  if (!applicationDialog.agreement.intendedUsage.trim()) {
    ElMessage.warning('请填写“仅将数据用于 XX 工作”的内容')
    return
  }
  if (!applicationDialog.agreement.files.length) {
    ElMessage.warning('请先上传个人签名图片')
    return
  }
  applicationDialog.agreement.isGeneratingPdf = true
  try {
    const generatedFile = await generateSignedAgreementPdf()
    addGeneratedAgreementAttachment(generatedFile)
    applicationDialog.currentStep = 2
  } catch (error) {
    console.error('Failed to generate agreement PDF', error)
    ElMessage.error(error.message || '生成知情同意书失败，请重试')
  } finally {
    applicationDialog.agreement.isGeneratingPdf = false
  }
}
const handleBackToAgreementStep = () => {
  if (applicationDialog.currentStep !== 1) {
    removeGeneratedAgreementAttachment()
    applicationDialog.currentStep = 1
  }
}
const uploadAttachmentsForApplication = async (applicationId) => {
  const groups = [
    {
      group: applicationDialog.attachments,
      updateTask: updateApplicationAttachmentTask,
    },
  ]

  let hasError = false
  let hasUploads = false

  for (const { group, updateTask } of groups) {
    const pendingFiles = group.files.filter((file) => {
      const task = group.uploadTasks.find((item) => item.uid === file.uid)
      return !task || !task.uploaded
    })
    if (!pendingFiles.length) continue
    hasUploads = true
    group.isUploading = true
    for (const file of pendingFiles) {
      updateTask(file.uid, { status: 'uploading' })
      try {
        await uploadApplicationFileAPI(applicationId, file.raw)
        updateTask(file.uid, { status: 'success', uploaded: true })
      } catch (error) {
        console.error('Failed to upload application attachment', error)
        updateTask(file.uid, { status: 'error', uploaded: false })
        hasError = true
      }
    }
    group.isUploading = false
  }

  if (!hasError && hasUploads) {
    ElMessage.success('上传附件成功')
  }
  return { failed: hasError }
}
const buildApplicationReason = () => {
  // const usage = applicationDialog.agreement.intendedUsage.trim()
  const baseReason = applicationDialog.form.reason.trim()
  // const declaration = usage ? `承诺：仅将数据用于${usage}工作` : ''
  return baseReason
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
          reason: buildApplicationReason(),
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

      applicationDialog.currentStep = 3
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
            <el-step title="知情同意书" />
            <el-step title="补充资料" />
            <el-step title="提交成功" />
          </el-steps>
        </div>
      </div>

      <div v-if="applicationDialog.currentStep === 1" class="agreement-step">
        <div class="agreement-viewer-wrapper">
          <vue-pdf-embed class="agreement-viewer-frame" :source="agreementPdfUrl" />
        </div>
        <div class="agreement-usage-section">
          <div class="agreement-usage-title">承诺声明</div>
          <div class="agreement-usage-box">
            <span>本人承诺：仅将数据用于</span>
            <el-input
              v-model="applicationDialog.agreement.intendedUsage"
              placeholder="请输入简要数据用途"
              class="agreement-usage-input"
              :disabled="applicationDialog.agreement.isUploading || applicationDialog.isSubmitting"
            />
            <span>工作。</span>
          </div>
          <div class="agreement-usage-tip">你可以在下一步中详细阐述数据用途。</div>
        </div>
        <div class="agreement-upload-section">
          <div class="agreement-upload-title">请上传个人手写签名照片</div>
          <el-upload
            ref="agreementUploadRef"
            action="#"
            accept="image/*"
            :show-file-list="false"
            :before-upload="handleAgreementImageBeforeUpload"
            :http-request="handleQueueAgreementImage"
            :disabled="applicationDialog.agreement.isUploading || applicationDialog.isSubmitting"
          >
            <el-button
              type="primary"
              :icon="Upload"
              :disabled="applicationDialog.agreement.isUploading || applicationDialog.isSubmitting"
              >选择图片</el-button
            >
            <template #tip>
              <div class="el-upload__tip">
                仅支持单个不超过{{ APPLICATION_FILE_LIMIT_MB }}MB的图片文件。建议使用 JPG/PNG 格式。
              </div>
            </template>
          </el-upload>
          <el-empty
            v-if="!applicationDialog.agreement.files.length"
            description="尚未上传签名图片"
            :image-size="100"
            class="agreement-upload-empty"
          />
          <div v-else class="agreement-image-list">
            <div
              v-for="file in applicationDialog.agreement.files"
              :key="file.uid"
              class="agreement-image-item"
            >
              <img :src="file.preview" :alt="file.name" />
              <div class="agreement-image-meta">
                <div class="agreement-image-name">{{ file.name }}</div>
                <div class="agreement-image-size">{{ formatFileSize(file.size) }}</div>
              </div>
              <el-button
                text
                type="danger"
                size="small"
                :disabled="applicationDialog.agreement.isUploading"
                @click="handleRemoveAgreementImage(file)"
                >移除</el-button
              >
            </div>
          </div>
        </div>
      </div>
      <el-form
        v-else-if="applicationDialog.currentStep === 2"
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
        <el-form-item label="上传附件 (可选)" class="application-attachment-form-item">
          <div class="application-attachment-wrapper">
            <div class="application-upload-controls">
              <el-upload
                ref="uploadRef"
                action="#"
                :show-file-list="false"
                multiple
                :before-upload="handleApplicationAttachmentBeforeUpload"
                :http-request="handleQueueApplicationAttachment"
                :disabled="
                  applicationDialog.isSubmitting ||
                  applicationDialog.attachments.isUploading ||
                  applicationDialog.agreement.isUploading
                "
              >
                <el-button
                  type="primary"
                  :icon="Upload"
                  :disabled="
                    applicationDialog.isSubmitting ||
                    applicationDialog.attachments.isUploading ||
                    applicationDialog.agreement.isUploading
                  "
                  >选择文件</el-button
                >
                <template #tip>
                  <div class="el-upload__tip">
                    仅支持常见文件格式；单个文件不得超过 {{ APPLICATION_FILE_LIMIT_MB }}MB。
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
                    <div class="attachment-name">
                      {{ file.name }}
                      <el-tag
                        v-if="file.locked"
                        type="success"
                        size="small"
                        class="attachment-badge"
                        effect="light"
                        >系统生成</el-tag
                      >
                    </div>
                    <div class="attachment-size">{{ formatFileSize(file.size) }}</div>
                  </div>
                  <el-button
                    v-if="!file.locked"
                    text
                    type="danger"
                    size="small"
                    :disabled="applicationDialog.attachments.isUploading"
                    @click="handleRemoveQueuedAttachment(file)"
                    >移除</el-button
                  >
                </div>
              </div>
              <div
                v-if="applicationDialog.attachments.uploadTasks.length"
                class="upload-progress-list"
              >
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
      <div v-else-if="applicationDialog.currentStep === 3" class="result-step">
        <el-result icon="success" title="申请已提交" sub-title="请等待1至5个工作日查看结果">
        </el-result>
      </div>
      <template #footer>
        <template v-if="applicationDialog.currentStep === 1">
          <el-button
            @click="applicationDialog.visible = false"
            :disabled="applicationDialog.agreement.isUploading"
            >取消</el-button
          >
          <el-button
            type="primary"
            @click="handleConfirmAgreementStep"
            :disabled="
              !applicationDialog.agreement.files.length ||
              !applicationDialog.agreement.intendedUsage.trim() ||
              applicationDialog.agreement.isUploading ||
              applicationDialog.agreement.isGeneratingPdf
            "
            :loading="applicationDialog.agreement.isGeneratingPdf"
            >同意并继续</el-button
          >
        </template>
        <template v-else-if="applicationDialog.currentStep === 2">
          <el-button
            @click="applicationDialog.visible = false"
            :disabled="
              applicationDialog.isSubmitting ||
              applicationDialog.attachments.isUploading ||
              applicationDialog.agreement.isUploading
            "
            >取消</el-button
          >
          <el-button
            @click="handleBackToAgreementStep"
            :disabled="
              applicationDialog.isSubmitting ||
              applicationDialog.attachments.isUploading ||
              applicationDialog.agreement.isUploading
            "
          >
            上一步
          </el-button>
          <el-button
            type="primary"
            @click="handleSubmitApplication"
            :loading="applicationDialog.isSubmitting"
            :disabled="
              applicationDialog.attachments.isUploading || applicationDialog.agreement.isUploading
            "
            >{{
              applicationDialog.attachments.createdApplicationId ? '继续上传附件' : '提交申请'
            }}</el-button
          >
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
