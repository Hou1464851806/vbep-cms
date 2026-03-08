<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject, watch, computed } from 'vue'
import MarkdownRenderer from "@/components/MarkdownRenderer.vue"
import {
  Check,
  Close,
  View,
  Download,
  RefreshLeft,
  Upload,
  Delete,
  Refresh,
  Timer,
  Edit,
  Plus,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize, formatShortId } from '@/utils/format.js'
import { APPLICATION_FILE_LIMIT_MB } from '@/utils/constrains.js'

const fetchDatasetListAPI = () => apiClient.get('/api/v1/datasets/')
const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const auditApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/audit`, data)
const resubmitApplicationAPI = (id, data, config = {}) =>
  apiClient.post(`/api/v1/applications/${id}/resubmit`, data, config)
const fetchAuditDetailAPI = (appId, auditId) => apiClient.get(`/api/v1/applications/${appId}/audits/${auditId}`)
const fetchApplicationFilesAPI = (id) => apiClient.get(`/api/v1/applications/${id}/files`)
const downloadApplicationFileAPI = (applicationId, fileId) =>
  apiClient.get(`/api/v1/applications/${applicationId}/files/${fileId}`, { responseType: 'blob' })
const deleteApplicationAPI = (id) => apiClient.delete(`/api/v1/applications/${id}`)
const confirmApplicationAPI = (id) => apiClient.post(`/api/v1/applications/${id}/submit`)
const createApplicationAPI = (data) => apiClient.post('/api/v1/applications/', data)
const fetchPublicTemplateFilesAPI = () => apiClient.get('/api/v1/files/public/templates')
const downloadPublicFileAPI = (fileId) => apiClient.get(`/api/v1/files/public/${fileId}/download`, { responseType: 'blob' })
const uploadApplicationFileAPI = (applicationId, file, config = {}) => {
  const formData = new FormData()
  formData.append('file', file)
  if (config.type) formData.append('type', config.type)
  return apiClient.post(`/api/v1/applications/${applicationId}/files`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    ...config,
  })
}

const APPLICATION_FILE_LIMIT_BYTES = APPLICATION_FILE_LIMIT_MB * 1024 * 1024
const ALLOWED_EXTENSIONS = [
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
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.svg',
  '.bmp',
  '.tiff',
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

const IDENTITY_EXTENSIONS = [
  '.pdf',
  '.doc',
  '.docx',
  '.wps',
  '.jpg',
  '.jpeg',
  '.png',
  '.bmp',
  '.heif',
]

const readStoredUserId = () => {
  if (typeof window === 'undefined') return ''
  try {
    return window.localStorage.getItem('userID') || ''
  } catch {
    return ''
  }
}

const currentUserId = ref(readStoredUserId())

const isAdmin = inject('isAdmin')
const applicationList = ref([])
const isLoading = ref(true)
const datasetList = ref([])
const resubmitUploadRef = ref(null)
const detailDialog = reactive({
  visible: false,
  loading: false,
  data: null,
  audit: {
    result: '',
    loading: false,
    selectedId: '',
    detailLoading: false,
  },
  attachments: {
    isLoading: false,
    list: [],
    originalList: [],
    currentApplicationId: '',
  },
  resubmit: {
    loading: false,
  },
  timeline: [],
})

// 状态：草稿编辑对话框
const draftDialog = reactive({
  visible: false,
  isSubmitting: false,
  data: null,
  mode: 'create', // 'create' | 'resubmit'
  mandatory: { list: [] },
  identity: { list: [] },
  optional: { list: [] },
})

// 状态：新增申请流程
const newAppDialog = reactive({
  visible: false,
  loading: false,
  selectedDatasetId: '',
  description: '',
  files: [],
})

const activeNames = ref(['2'])

const statusMetaMap = {
  draft: { label: '待提交', type: 'info' },
  pending: { label: '待审批', type: '' },
  approved: { label: '已批准', type: 'success' },
  rejected: { label: '已拒绝', type: 'danger' },
  returned: { label: '已退回', type: 'warning' },
}

const normalizeApplication = (item = {}) => {
  let meta = statusMetaMap[item.status] || { label: '未知', type: 'info' }

  return {
    id: item.id,
    status: item.status,
    statusMeta: meta,
    version: item.version,
    dataset: {
      id: item.dataset?.id,
      title: item.dataset?.title,
    },
    applicant: {
      id: item.applicant?.id,
      nickname: item.applicant?.nickname,
      email: item.applicant?.email,
    },
    createdAt: formatDateTime(item.createdAt),
    auditedAt: formatDateTime(item.updatedAt),
  }
}

const auditActionMap = {
  draft: { label: '草稿编辑中', color: '#909399', operatorLabel: '申请人', resultLabel: '备注' },
  pending: { label: '申请已提交', color: '#C0C4CC', operatorLabel: '提交人', resultLabel: '备注' },
  approved: {
    label: '管理员审批',
    color: '#C0C4CC',
    operatorLabel: '审批人',
    actionDetail: '通过申请',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  rejected: {
    label: '管理员审批',
    color: '#C0C4CC',
    operatorLabel: '审批人',
    actionDetail: '拒绝申请',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  returned: {
    label: '管理员审批',
    color: '#C0C4CC',
    operatorLabel: '审批人',
    actionDetail: '退回修改',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
}

const normalizeAudit = (item = {}) => {
  const meta = auditActionMap[item.action] || { label: '未知操作', color: '#909399' }
  return {
    id: item.id,
    action: item.action || 'unknown',
    color: meta.color,
    operator: item.operator
      ? {
          id: item.operator.id,
          nickname: item.operator.nickname,
          email: item.operator.email,
        }
      : null,
    result: item.result || '',
    time: formatDateTime(item.updatedAt || item.createdAt),
    createdAt: formatDateTime(item.createdAt),
    actionDetail: meta.actionDetail || '',
  }
}

const normalizeApplicationFiles = (files = []) =>
  files.map((item) => ({
    id: item.fileID,
    name: item.name || '未命名文件',
    size: Number(item.size ?? 0),
    type: item.type || '',
    createdAt: formatDateTime(item.createdAt),
  }))

const normalizeTemplateFiles = (files = []) =>
  files.map((item) => ({
    id: item.fileID,
    name: item.name || '未命名文件',
    size: Number(item.size ?? 0),
    createdAt: formatDateTime(item.createdAt),
  }))

const toEditableAttachment = (item = {}) => ({
  id: item.id || '',
  uid: item.id || `local-${Date.now()}-${Math.random().toString(16).slice(2)}`,
  name: item.name || '未命名文件',
  size: Number(item.size ?? 0),
  type: item.type || '',
  createdAt: item.createdAt || '-',
  isNew: !item.id,
  isRemoved: Boolean(item.isRemoved),
  file: item.file || null,
})

const canViewApplicationAttachments = (application) => {
  if (!application) return false
  if (isAdmin()) return true
  return application.applicant?.id === currentUserId.value
}

const canViewAttachments = computed(() => canViewApplicationAttachments(detailDialog.data))
const canResubmitApplication = computed(() => {
  const data = detailDialog.data
  if (!data) return false
  return data.status === 'returned' && data.applicant?.id === currentUserId.value
})

const resetAttachmentState = () => {
  detailDialog.attachments.isLoading = false
  detailDialog.attachments.list = []
  detailDialog.attachments.originalList = []
  detailDialog.attachments.currentApplicationId = ''
  // 清理 resubmitUploadRef 相关的逻辑可以移除了，因为我们不再在详情页上传
  if (resubmitUploadRef.value) {
    resubmitUploadRef.value.clearFiles?.()
  }
}

const resetResubmitState = () => {
  detailDialog.resubmit.loading = false
}

// 这是一个通用的加载文件方法，我们可以复用它来获取 draftDialog 需要的数据
const loadApplicationFiles = async (applicationId) => {
  if (!applicationId) return
  detailDialog.attachments.isLoading = true
  try {
    const res = await fetchApplicationFilesAPI(applicationId)
    if (res?.status === 200) {
      const normalized = normalizeApplicationFiles(res.data.files)
      detailDialog.attachments.originalList = normalized.map((item) => ({ ...item }))
      detailDialog.attachments.list = normalized.map((item) => toEditableAttachment(item))
      return normalized // 返回文件列表供调用者使用
      detailDialog.attachments.currentApplicationId = applicationId
    } else if (res?.data?.message) {
      ElMessage.error(res.data.message)
    } else {
      ElMessage.error('附件列表加载失败')
    }
  } catch (error) {
    console.error('Failed to load application files', error)
    ElMessage.error('附件列表加载失败')
  } finally {
    detailDialog.attachments.isLoading = false
  }
}

const handleAuditClick = async (auditItem) => {
  if (!auditItem || !auditItem.id) return
  detailDialog.audit.selectedId = auditItem.id

  detailDialog.audit.detailLoading = true
  resetAttachmentState()

  try {
    const res = await fetchAuditDetailAPI(detailDialog.data.id, auditItem.id)
    if (res.status === 200) {
      const files = normalizeApplicationFiles(res.data.audit.files || [])
      detailDialog.attachments.list = files.map(toEditableAttachment)
      detailDialog.attachments.currentApplicationId = detailDialog.data.id
    } else {
      ElMessage.error('获取详情失败')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取详情失败')
  } finally {
    detailDialog.audit.detailLoading = false
  }
}

const getAttachmentStatusMeta = (item = {}) => {
  if (item.isRemoved) return { label: '被移除', type: 'danger' }
  if (!item.id) return { label: '待提交', type: 'info' }
  return { label: '已上传', type: 'success' }
}

const getAttachmentRowClass = ({ row } = {}) => {
  if (!row) return ''
  if (row.isRemoved) return 'attachment-row-removed'
  if (!row.id) return 'attachment-row-new'
  return ''
}

const validateAttachmentFile = (rawFile) => {
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(APPLICATION_FILE_LIMIT_BYTES)} 限制，无法上传`)
    return false
  }
  const fileName = rawFile.name?.toLowerCase?.() || ''
  if (DENIED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
    ElMessage.error(`文件类型暂不支持："${rawFile.name}"`)
    return false
  }
  if (!ALLOWED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
    ElMessage.error(`文件格式不受支持："${rawFile.name}"`)
    return false
  }
  return true
}

const addLocalAttachment = (rawFile, uid) => {
  const entry = toEditableAttachment({
    id: '',
    uid: uid || `local-${Date.now()}`,
    name: rawFile.name,
    size: rawFile.size,
    createdAt: formatDateTime(new Date()),
    file: rawFile,
    isRemoved: false,
  })
  detailDialog.attachments.list.push(entry)
}

const handleAttachmentFileQueued = (uploadFile) => {
  const rawFile = uploadFile?.raw
  if (!rawFile) return
  if (!validateAttachmentFile(rawFile)) {
    ElMessage.warning('文件未通过校验')
    return
  }
  addLocalAttachment(rawFile, uploadFile.uid)
  ElMessage.success(`已添加附件「${rawFile.name}」，提交后生效`)
}

const handleRemoveAttachmentLocal = async (file) => {
  if (!file) return
  if (file.id && file.isRemoved) {
    file.isRemoved = false
    ElMessage.success(`已撤销移除「${file.name}」`)
    return
  }
  if (file.id) {
    try {
      await ElMessageBox.confirm(`确定移除已上传的附件「${file.name}」吗？`, '提示', {
        type: 'warning',
        confirmButtonText: '移除',
        cancelButtonText: '取消',
      })
    } catch {
      return
    }
    file.isRemoved = true
    ElMessage.info('附件已标记删除，重新提交后生效')
    return
  }
  const index = detailDialog.attachments.list.findIndex((item) => item.uid === file.uid)
  if (index !== -1) {
    detailDialog.attachments.list.splice(index, 1)
    ElMessage.success(`已撤销新增附件「${file.name}」`)
  }
}

const getAttachmentDiff = () => {
  const keptFileIds = []
  const newFiles = []
  ;(detailDialog.attachments.list || []).forEach((item) => {
    if (item.id && !item.isRemoved) {
      if (!keptFileIds.includes(item.id)) keptFileIds.push(item.id)
    } else if (!item.id && item.file && !item.isRemoved) {
      newFiles.push(item)
    }
  })
  return { keptFileIds, newFiles }
}

const handleDownloadAttachment = async (file) => {
  const applicationId = detailDialog.attachments.currentApplicationId
  if (!applicationId) {
    ElMessage.error('未找到对应的申请')
    return
  }
  if (!file?.id) {
    ElMessage.warning('请提交后再下载此附件')
    return
  }
  if (file.isRemoved) {
    ElMessage.warning('该附件已标记为移除，如需下载请撤销移除后再尝试')
    return
  }
  try {
    const res = await downloadApplicationFileAPI(applicationId, file.id)
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
    console.error('Failed to download attachment', error)
    ElMessage.error('下载附件失败')
  }
}

const fetchApplications = async () => {
  isLoading.value = true
  try {
    const res = await fetchApplicationListAPI()
    if (res.status === 200) {
      applicationList.value = res.data.applications.map((item) => normalizeApplication(item))
    } else {
      ElMessage.error(res.data?.message || '申请列表加载失败')
    }
  } catch (error) {
    console.error('Failed to load applications', error)
    ElMessage.error('申请列表加载失败')
  } finally {
    isLoading.value = false
  }
}

const handleViewDetail = async (row) => {
  detailDialog.visible = true
  detailDialog.loading = true
  detailDialog.data = null
  detailDialog.timeline = []
  detailDialog.audit.result = '' // 重置
  activeNames.value = ['2']
  resetAttachmentState()
  try {
    const detailRes = await fetchApplicationDetailAPI(row.id)

    if (detailRes.status === 200) {
      detailDialog.data = normalizeApplication(detailRes.data.application)

      const normalizedAudits = (detailRes.data.application.audits || []).map((item) => normalizeAudit(item))
      detailDialog.timeline = normalizedAudits

      // 默认选中最新的一条（最后一条）
      if (detailDialog.timeline.length > 0) {
        handleAuditClick(detailDialog.timeline[detailDialog.timeline.length - 1])
      }
    } else {
      ElMessage.error(detailRes.data?.message || '获取申请详情失败')
    }
  } catch (error) {
    console.error('Failed to load application detail', error)
    ElMessage.error('获取申请详情失败')
  } finally {
    detailDialog.loading = false
  }
}

const updateApplicationEntry = (nextData) => {
  const normalized = normalizeApplication(nextData)
  const index = applicationList.value.findIndex((item) => item.id === normalized.id)
  if (index !== -1) {
    applicationList.value.splice(index, 1, normalized)
  } else {
    applicationList.value.unshift(normalized)
  }

  if (detailDialog.data && detailDialog.data.id === normalized.id) {
    detailDialog.data = normalized
  }
}

const auditActionSuccessMessage = {
  approve: '申请已批准',
  reject: '申请已拒绝',
  return: '申请已退回',
}

const auditConfirmMessages = {
  approve: '您即将执行批准操作，批准后用户将有权访问该数据集下的所有文件',
  reject: '您即将执行拒绝操作，拒绝后用户将无法修改并重新提交申请',
  return: '您即将执行退回操作，退回后用户可以修改并重新提交申请',
}

const handleDialogAudit = async (action) => {
  const current = detailDialog.data
  if (!current?.id) return
  const result = detailDialog.audit.result.trim()
  if (!result) {
    ElMessage.warning('请填写审核意见')
    return
  }

  try {
    await ElMessageBox.confirm(auditConfirmMessages[action], '操作确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: action === 'reject' ? 'error' : 'warning',
    })
  } catch {
    return
  }

  detailDialog.audit.loading = true
  try {
    const res = await auditApplicationAPI(current.id, {
      action: action === 'approve' ? 'approved' : action === 'reject' ? 'rejected' : 'returned',
      result,
    })
    if (res.status === 200 && res.data.application) {
      const normalized = normalizeApplication(res.data.application)
      updateApplicationEntry(res.data.application)
      detailDialog.audit.result = normalized.result

      const normalizedAudits = (res.data.application.audits || []).map((item) => normalizeAudit(item))
      detailDialog.timeline = normalizedAudits

      ElMessage.success(auditActionSuccessMessage[action] || '操作成功')
    } else {
      ElMessage.error(res.data?.message || '操作失败')
    }
  } catch (error) {
    console.error('Failed to process application', error)
    ElMessage.error('操作失败')
  } finally {
    detailDialog.audit.loading = false
  }
}

// --- 草稿与重填管理逻辑 ---

// 打开草稿/重填窗口的通用方法
const openEditDialog = async (row, mode = 'create') => {
  draftDialog.data = row
  draftDialog.mode = mode
  draftDialog.mandatory.list = []
  draftDialog.identity.list = []
  draftDialog.optional.list = []
  draftDialog.visible = true

  // 如果是重填模式，需要加载已有的文件
  if (mode === 'resubmit') {
    draftDialog.isSubmitting = true // 复用 loading 状态显示加载中
    try {
      const res = await fetchApplicationFilesAPI(row.id)
      if (res?.status === 200) {
        const files = normalizeApplicationFiles(res.data.files)
        // 分发文件到对应类别
        draftDialog.mandatory.list = files.filter(f => f.type === 'form')
        draftDialog.identity.list = files.filter(f => f.type === 'identity')
        draftDialog.optional.list = files.filter(f => f.type === 'general' || !f.type)
      }
    } catch (e) {
      ElMessage.error('加载已有文件失败')
    } finally {
      draftDialog.isSubmitting = false
    }
  }
}

const performResubmit = async (applicationId, keptFileIDs, newFiles) => {
  const formData = new FormData()
  formData.append('keptFileIDs', JSON.stringify(keptFileIDs))
  newFiles.forEach((item) => {
    if (item.file) {
      formData.append('files', item.file, item.name)
      formData.append('types', item.type || 'general')
    }
  })

  return await resubmitApplicationAPI(applicationId, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

const handleContinueDraft = (row) => {
  openEditDialog(row, 'create')
}

const handleEditReturned = (row) => {
  // 关闭详情页（如果在详情页触发）
  detailDialog.visible = false
  openEditDialog(row, 'resubmit')
}

// 辅助函数：收集所有新上传的文件
const collectNewFiles = () => {
  const files = []
  
  const processList = (list, type, defaultName) => {
    list.forEach(item => {
      if (item.raw) { // 只有包含 raw 属性的才是新上传的文件
        files.push({ 
          name: defaultName || item.name, 
          file: item.raw, 
          type: type 
        })
      }
    })
  }

  processList(draftDialog.mandatory.list, 'form', 'application_form.pdf')
  processList(draftDialog.identity.list, 'identity', 'identity_proof') // 名字其实后端不依赖，主要靠文件内容
  processList(draftDialog.optional.list, 'general')

  return files
}

const handleDeleteDraft = async (row) => {
  try {
    await ElMessageBox.confirm('确定要删除该申请草稿吗？此操作不可恢复。', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteApplicationAPI(row.id)
    ElMessage.success('草稿已删除')
    await fetchApplications()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleMandatoryFileChange = (file, fileList) => {
  // 强制单选：保留最后一个（最新的）
  if (fileList.length > 1) {
    fileList.splice(0, fileList.length - 1)
  }
  
  const current = fileList[0]
  if (!current || !current.raw) {
    // 如果是空或者已有文件(没有raw)，直接更新列表
    draftDialog.mandatory.list = fileList
    return
  }

  const rawFile = current.raw
  if (rawFile.type !== 'application/pdf') {
    ElMessage.error('仅支持上传 PDF 文件')
    fileList.pop() // 移除无效文件
    return
  }
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(APPLICATION_FILE_LIMIT_BYTES)} 限制，无法上传`)
    fileList.pop()
    return
  }
  draftDialog.mandatory.list = fileList
}

const handleIdentityFileChange = (file, fileList) => {
  if (fileList.length > 1) {
    fileList.splice(0, fileList.length - 1)
  }

  const current = fileList[0]
  if (!current || !current.raw) {
    draftDialog.identity.list = fileList
    return
  }

  const rawFile = current.raw
  const fileName = rawFile.name.toLowerCase()

  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(APPLICATION_FILE_LIMIT_BYTES)} 限制，无法上传`)
    fileList.pop()
    return
  }
  if (!IDENTITY_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
    ElMessage.error(`文件格式不支持："${rawFile.name}"，请上传 PDF/Word/图片 格式`)
    fileList.pop()
    return
  }
  draftDialog.identity.list = fileList
}

const handleOptionalFileChange = (file, fileList) => {
  // 过滤掉不合法的新文件
  const validList = fileList.filter(item => {
    if (!item.raw) return true // 已有文件直接保留
    
    const rawFile = item.raw
    const fileName = rawFile.name.toLowerCase()
    if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
      ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(APPLICATION_FILE_LIMIT_BYTES)} 限制，无法上传`)
      return false
    }
    if (DENIED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
      ElMessage.error(`文件格式被拒绝："${rawFile.name}"`)
      return false
    }
    if (!ALLOWED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
      ElMessage.error(`文件格式不受支持："${rawFile.name}"`)
      return false
    }
    return true
  })
  
  draftDialog.optional.list = validList
}

const handleSubmitDraft = async () => {
  // 校验逻辑：检查列表中是否有文件（无论是新上传的还是已有的）
  const hasMandatory = draftDialog.mandatory.list.length > 0
  const hasIdentity = draftDialog.identity.list.length > 0
  const hasOptional = draftDialog.optional.list.length > 0

  if (draftDialog.mode === 'create') {
    if (!hasMandatory || !hasIdentity) {
      ElMessage.warning('请上传必填的申请文件和身份证明文件')
      return
    }
  } else if (!hasMandatory && !hasIdentity && !hasOptional) {
    ElMessage.warning('请至少保留或上传一个文件')
    return
  }

  try {
    const confirmMessage = draftDialog.mode === 'resubmit'
      ? '您即将提交申请。申请提交后，您将不能修改其中的任何信息。'
      : '您即将提交申请。申请提交后，您将不能修改其中的任何信息，也不再能撤回此次申请。'

    await ElMessageBox.confirm(confirmMessage, '提交确认', {
      confirmButtonText: '确认提交',
      cancelButtonText: '再检查一下',
      type: 'warning',
    })
  } catch {
    return
  }

  draftDialog.isSubmitting = true
  const applicationId = draftDialog.data.id

  try {
    if (draftDialog.mode === 'resubmit') {
      // 重填模式：一次性提交所有变更
      const keptFileIDs = []
      const collectKeptIDs = (list) => {
        list.forEach(item => { if (item.id && !item.raw) keptFileIDs.push(item.id) })
      }
      collectKeptIDs(draftDialog.mandatory.list)
      collectKeptIDs(draftDialog.identity.list)
      collectKeptIDs(draftDialog.optional.list)

      const newFiles = collectNewFiles()
      
      await performResubmit(applicationId, keptFileIDs, newFiles)
      ElMessage.success('申请已重新提交')
    } else {
      // 草稿模式：逐个上传后确认
      await uploadApplicationFileAPI(applicationId, draftDialog.mandatory.list[0].raw, { type: 'form' })
      await uploadApplicationFileAPI(applicationId, draftDialog.identity.list[0].raw, { type: 'identity' })
      for (const item of draftDialog.optional.list) {
        await uploadApplicationFileAPI(applicationId, item.raw, { type: 'general' }).catch(console.warn)
      }
      await confirmApplicationAPI(applicationId)
      ElMessage.success('申请已提交')
    }
    
    draftDialog.visible = false
    await fetchApplications()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '提交申请失败')
  } finally {
    draftDialog.isSubmitting = false
  }
}

// --- 新增申请流程逻辑 ---
const handleNewApplication = async () => {
  newAppDialog.selectedDatasetId = ''
  newAppDialog.visible = true
  newAppDialog.loading = true
  try {
    const [dsRes, tplRes] = await Promise.all([
      fetchDatasetListAPI(),
      fetchPublicTemplateFilesAPI(),
    ])
    if (dsRes.status === 200) {
      datasetList.value = dsRes.data.datasets || []
    }
    if (tplRes.status === 200) {
      newAppDialog.description = tplRes.data.description || '暂无说明'
      newAppDialog.files = normalizeTemplateFiles(tplRes.data.files || [])
    }
  } catch (e) {
    ElMessage.error('加载数据失败')
  } finally {
    newAppDialog.loading = false
  }
}

const handleCreateAndStart = async () => {
  if (!newAppDialog.selectedDatasetId) {
    ElMessage.warning('请选择数据集')
    return
  }
  newAppDialog.loading = true
  try {
    // 1. 创建草稿
    const res = await createApplicationAPI({ datasetID: newAppDialog.selectedDatasetId })
    if (res.status === 200) {
      const appId = res.data.applicationID
      
      // 2. 获取完整申请详情以填充 draftDialog
      const detailRes = await fetchApplicationDetailAPI(appId)
      if (detailRes.status === 200) {
        const appData = normalizeApplication(detailRes.data.application)
        newAppDialog.visible = false
        await fetchApplications() // 刷新列表
        handleContinueDraft(appData)
      } else {
        ElMessage.error('无法加载申请详情')
      }
    } else {
      ElMessage.error(res.data?.message || '创建申请失败')
    }
  } catch (error) {
    const msg = error.response?.data?.message || '创建申请失败'
    ElMessage.error(msg)
  } finally {
    newAppDialog.loading = false
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

const getActionTagType = (action) => {
  if (action === 'approved') return 'success'
  if (action === 'rejected') return 'danger'
  if (action === 'returned') return 'warning'
  return 'info'
}

watch(
  () => detailDialog.data,
  (value) => {
    detailDialog.audit.result = ''
    if (!value) {
      resetAttachmentState()
      resetResubmitState()
      return
    }
    // 如果申请被退回，加载最新的文件列表以供用户修改和重新提交
    if (value.status === 'returned' && value.id) {
      loadApplicationFiles(value.id)
    }
    // 不再自动加载文件，改为点击时间轴加载
  },
)

watch(
  () => detailDialog.visible,
  (visible) => {
    if (!visible) {
      resetAttachmentState()
      resetResubmitState()
    }
  },
)

onMounted(fetchApplications)
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>申请列表</span>
        <div class="header-actions">
          <el-button :icon="Refresh" circle @click="fetchApplications" :loading="isLoading" title="刷新列表" />
          <el-button type="primary" :icon="Plus" @click="handleNewApplication">新增申请</el-button>
        </div>
      </div>
    </template>

    <el-skeleton :rows="5" animated :loading="isLoading">
      <el-table :data="applicationList" style="width: 100%">
        <el-table-column label="内部ID" width="100">
          <template #default="{ row }">
            <el-tooltip effect="dark" :content="row.id" placement="top" :show-after="500">
              <span class="monospace-id">
                {{ formatShortId(row.id) }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="申请人" width="120">
          <template #default="scope">
            <div>{{ scope.row.applicant.nickname }}</div>
          </template>
        </el-table-column>
        <el-table-column label="目标数据集" min-width="150" show-overflow-tooltip>
          <template #default="scope">
            {{ scope.row.dataset.title || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="auditedAt" label="最后更新时间" width="180" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.statusMeta.type">{{ scope.row.statusMeta.label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="scope">
            <template v-if="scope.row.status === 'draft'">
              <el-button type="danger" link :icon="Delete" @click="handleDeleteDraft(scope.row)">
                删除草稿
              </el-button>
              <el-button v-if="!isAdmin()" type="primary" link :icon="Edit" @click="handleContinueDraft(scope.row)">
                修改表单
              </el-button>
            </template>
            <template v-else>
              <el-button type="primary" link :icon="View" @click="handleViewDetail(scope.row)">
                查看详情
              </el-button>
              <el-button
                v-if="scope.row.status === 'returned' && !isAdmin()"
                type="primary"
                link
                :icon="Edit"
                @click="handleEditReturned(scope.row)"
              >
                修改表单
              </el-button>
              <el-button
                v-else-if="scope.row.status === 'pending' && !isAdmin()"
                type="info"
                link
                :icon="Edit"
                disabled
              >
                修改表单
              </el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </el-skeleton>
  </el-card>

  <!-- 1. 新增申请对话框 (合并说明与选择) -->
  <el-dialog v-model="newAppDialog.visible" title="新增申请" width="720px" :close-on-click-modal="false">
    <div v-loading="newAppDialog.loading">
      <div class="upload-label">申请说明</div>
      <div class="template-description-box">
        <markdown-renderer :model-value="newAppDialog.description" />
      </div>
      <div class="template-download-area" v-if="newAppDialog.files && newAppDialog.files.length">
        <div class="upload-label">模版下载</div>
        <div v-for="file in newAppDialog.files" :key="file.id">
          <el-button type="primary" link :icon="Download" @click="handleDownloadTemplateFile(file)">
            下载“{{ file.name }}”
          </el-button>
        </div>
      </div>
      <el-divider />
      <el-form label-position="top">
        <el-form-item label="选择目标数据集" required>
          <el-select v-model="newAppDialog.selectedDatasetId" placeholder="请选择要申请的数据集" style="width: 100%" filterable>
            <el-option
              v-for="item in datasetList"
              :key="item.id"
              :label="item.title"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
    </div>
    <template #footer>
      <el-button @click="newAppDialog.visible = false">取消</el-button>
      <el-button type="primary" @click="handleCreateAndStart" :loading="newAppDialog.loading" :disabled="!newAppDialog.selectedDatasetId">下一步</el-button>
    </template>
  </el-dialog>

  <!-- 草稿编辑对话框 -->
  <el-dialog
    v-model="draftDialog.visible"
    :title="draftDialog.mode === 'create' ? `完善申请资料 - ${draftDialog.data?.dataset?.title}` : `修改并重新提交 - ${draftDialog.data?.dataset?.title}`"
    width="720px"
    :close-on-click-modal="false"
  >
    <div v-loading="draftDialog.isSubmitting">
      <div class="upload-section">
        <div class="upload-label required-label">1. 上传申请文件 (必填)</div>
        <div class="upload-desc">请上传填写并签署后的申请模板文件，仅支持 PDF 格式，限传 1 个，文件大小不能超过 {{ formatFileSize(APPLICATION_FILE_LIMIT_BYTES) }}。</div>
        <el-upload
          class="upload-demo"
          action="#"
          :auto-upload="false"
          accept=".pdf"
          :on-change="handleMandatoryFileChange"
          :on-remove="handleMandatoryFileChange"
          v-model:file-list="draftDialog.mandatory.list"
        >
          <el-button type="primary" :icon="Upload">{{ draftDialog.mandatory.list.length > 0 ? '替换文件' : '选择 PDF 文件' }}</el-button>
        </el-upload>
      </div>

      <el-divider />

      <div class="upload-section">
        <div class="upload-label required-label">2. 上传身份证明文件 (必填)</div>
        <div class="upload-desc">请上传身份证、学生证或工作证等有效身份证明文件，支持 PDF、Word、图片格式，限传 1 个。</div>
        <el-upload
          class="upload-demo"
          action="#"
          :auto-upload="false"
          :accept="IDENTITY_EXTENSIONS.join(',')"
          :on-change="handleIdentityFileChange"
          :on-remove="handleIdentityFileChange"
          v-model:file-list="draftDialog.identity.list"
        >
          <el-button :icon="Upload">{{ draftDialog.identity.list.length > 0 ? '替换文件' : '选择文件' }}</el-button>
        </el-upload>
      </div>

      <el-divider />

      <div class="upload-section">
        <div class="upload-label">3. 补充材料 (可选)</div>
        <div class="upload-desc">如有其他证明材料，可在此上传，不限数量，单个文件大小不能超过 {{ formatFileSize(APPLICATION_FILE_LIMIT_BYTES) }}。</div>
        <el-upload
          class="upload-demo"
          action="#"
          :auto-upload="false"
          multiple
          :on-change="handleOptionalFileChange"
          :on-remove="handleOptionalFileChange"
          v-model:file-list="draftDialog.optional.list"
        >
          <el-button :icon="Upload">选择文件</el-button>
        </el-upload>
      </div>
    </div>
    <template #footer>
      <el-button @click="draftDialog.visible = false" :disabled="draftDialog.isSubmitting">取消</el-button>
      <el-button 
        type="primary" 
        @click="handleSubmitDraft" 
        :loading="draftDialog.isSubmitting"
      >
        {{ draftDialog.mode === 'resubmit' ? '确认修改并提交' : '提交申请' }}
      </el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="detailDialog.visible" title="申请详情" width="1000px">
    <el-skeleton :rows="5" animated :loading="detailDialog.loading">
      <template #default>
        <el-collapse v-model="activeNames">
          <el-collapse-item title="基本信息" name="1">
            <el-descriptions border :column="2" v-if="detailDialog.data" class="custom-descriptions">
              <el-descriptions-item label="当前状态">
                <el-tag :type="detailDialog.data.statusMeta.type">{{
                  detailDialog.data.statusMeta.label
                }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="提交次数">
                {{ detailDialog.data.version }}
              </el-descriptions-item>
              <el-descriptions-item label="目标数据集">
                {{ detailDialog.data.dataset.title }}
                <el-tooltip :content="detailDialog.data.dataset.id" placement="top">
                  <span class="sub-text">
                    (ID: {{ formatShortId(detailDialog.data.dataset.id) }})
                  </span>
                </el-tooltip>
              </el-descriptions-item>
              <el-descriptions-item label="申请人">
                {{ detailDialog.data.applicant.nickname }}
                <span v-if="detailDialog.data.applicant.email" class="sub-text">
                  ({{ detailDialog.data.applicant.email }})
                </span>
              </el-descriptions-item>
              <el-descriptions-item label="创建时间">
                {{ detailDialog.data.createdAt }}
              </el-descriptions-item>
              <el-descriptions-item label="最后修改时间">
                {{ detailDialog.data.auditedAt || '-' }}
              </el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
          
          <el-collapse-item title="时间线" name="2">
            <el-row :gutter="20">
              <!-- 左侧：时间轴 -->
              <el-col :span="12" class="audit-timeline-col">
                <el-empty v-if="!detailDialog.timeline.length" description="暂无审批记录" />
                <el-timeline v-else class="audit-log-steps">
                  <el-timeline-item
                    v-for="(item, index) in detailDialog.timeline"
                    :key="item.id"
                    :timestamp="item.time"
                    :color="item.color"
                    placement="top"
                    hide-timestamp
                  >
                    <el-card
                      class="audit-record-card"
                      :class="{ 'is-selected': item.id === detailDialog.audit.selectedId }"
                      shadow="hover"
                      :body-style="{ padding: '0' }"
                      @click="handleAuditClick(item)"
                    >
                      <!-- 上半部分：用户提交信息 -->
                      <div class="audit-cell user-cell">
                        <div class="cell-header">
                          <div class="subject-wrapper">
                            <span class="prefix">申请人：</span>
                            <el-tooltip :content="detailDialog.data.applicant.email" placement="top">
                              <span class="subject">{{ detailDialog.data.applicant.nickname }}</span>
                            </el-tooltip>
                          </div>
                          <div class="meta-wrapper">
                            <span class="time">{{ item.createdAt }}</span>
                            <el-tag type="primary" size="small" effect="dark" class="action-tag">
                              {{ index === 0 ? '初次提交' : '重新提交' }}
                            </el-tag>
                          </div>
                        </div>
                      </div>

                      <!-- 分割线 -->
                      <div class="audit-divider"></div>

                      <!-- 下半部分：管理员审批信息 -->
                      <div class="audit-cell admin-cell">
                        <template v-if="item.action === 'pending'">
                          <div class="placeholder-content">
                            <el-icon><Timer /></el-icon>
                            <span>等待管理员审批</span>
                          </div>
                        </template>
                        <template v-else>
                          <div class="cell-header">
                            <div class="subject-wrapper">
                              <span class="prefix">经办人：</span>
                              <el-tooltip :content="item.operator?.email || '未知'" placement="top">
                                <span class="subject">{{ item.operator?.nickname || '管理员' }}</span>
                              </el-tooltip>
                            </div>
                            <div class="meta-wrapper">
                              <span class="time">{{ item.time }}</span>
                              <el-tag :type="getActionTagType(item.action)" size="small" effect="dark" class="action-tag">
                                {{ item.actionDetail }}
                              </el-tag>
                            </div>
                          </div>
                        </template>
                      </div>
                    </el-card>
                  </el-timeline-item>
                </el-timeline>
              </el-col>

              <!-- 右侧：详情与操作 -->
              <el-col :span="12" class="audit-detail-col" v-loading="detailDialog.audit.detailLoading">
                <div v-if="!detailDialog.audit.selectedId" class="detail-placeholder">
                  请选择左侧记录查看详情
                </div>
                <div v-else class="detail-content">
                  <!-- 附件列表 -->
                  <div v-if="canViewAttachments" class="info-block">
                    <div class="info-block-header">
                      <span>申请附件</span>
                    </div>
                    <el-skeleton :rows="3" animated :loading="detailDialog.attachments.isLoading">
                      <template #default>
                        <el-empty v-if="!detailDialog.attachments.list.length" description="暂无附件" :image-size="60" class="empty-attachment" />
                        <el-table
                          v-else
                          :data="detailDialog.attachments.list"
                          size="small"
                          class="attachment-table"
                        >
                          <el-table-column
                            prop="name"
                            label="文件名"
                            min-width="150"
                            show-overflow-tooltip
                          />
                          <el-table-column label="类型" width="90">
                            <template #default="{ row }">
                              <span v-if="row.type === 'form'">申请表</span>
                              <span v-else-if="row.type === 'identity'">身份证明</span>
                              <span v-else-if="row.type === 'general'">补充材料</span>
                              <span v-else>-</span>
                            </template>
                          </el-table-column>
                          <el-table-column label="大小" width="90">
                            <template #default="scope">{{ formatFileSize(scope.row.size) }}</template>
                          </el-table-column>
                          <el-table-column label="下载" width="60" align="center">
                            <template #default="scope">
                              <el-button
                                link
                                :icon="Download"
                                @click="handleDownloadAttachment(scope.row)"
                              />
                            </template>
                          </el-table-column>
                        </el-table>
                      </template>
                    </el-skeleton>
                  </div>

                  <!-- 审批意见展示 -->
                  <div v-if="detailDialog.timeline.find(i => i.id === detailDialog.audit.selectedId)?.result" class="info-block">
                    <div class="info-block-header">
                      <span>审批意见</span>
                    </div>
                    <div class="result-content">
                      {{ detailDialog.timeline.find(i => i.id === detailDialog.audit.selectedId)?.result }}
                    </div>
                  </div>

                  <!-- 管理员操作区域 -->
                  <div
                    v-if="isAdmin() && detailDialog.data?.status === 'pending' && detailDialog.timeline.length > 0 && detailDialog.timeline[detailDialog.timeline.length - 1]?.id === detailDialog.audit.selectedId"
                    class="audit-action-area"
                  >
                    <div class="action-divider">审批操作</div>
                    <el-input
                      v-model="detailDialog.audit.result"
                      type="textarea"
                      :rows="3"
                      maxlength="200"
                      show-word-limit
                      placeholder="请填写审核意见"
                    />
                    <div class="audit-action-buttons">
                      <el-button
                        type="warning"
                        :loading="detailDialog.audit.loading"
                        :icon="RefreshLeft"
                        @click="handleDialogAudit('return')"
                        >退回修改</el-button
                      >
                      <el-button
                        type="danger"
                        :loading="detailDialog.audit.loading"
                        :icon="Close"
                        @click="handleDialogAudit('reject')"
                        >拒绝申请</el-button
                      >
                      <el-button
                        type="success"
                        :loading="detailDialog.audit.loading"
                        :icon="Check"
                        @click="handleDialogAudit('approve')"
                        >通过申请</el-button
                      >
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-skeleton>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="detailDialog.visible = false" style="margin-right: auto">返回</el-button>
        <el-button
          v-if="canResubmitApplication"
          type="primary"
          @click="handleEditReturned(detailDialog.data)"
          >修改并重新提交</el-button
        >
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.reason-block {
  white-space: pre-wrap;
  line-height: 1.6;
}
.info-block {
  padding: 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background-color: #f9fafc;
}
.info-block-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: bold;
  color: #303133;
}
.attachment-upload-area {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed #dcdfe6;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.attachment-upload-header {
  display: flex;
  flex-direction: column;
  font-size: 13px;
  color: #909399;
}
.attachment-upload-header span {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
}
.attachment-upload-tip {
  font-size: 12px;
  color: #a8abb2;
}
.attachment-table {
  margin-top: 8px;
}
.attachment-table :deep(.attachment-row-removed) {
  background-color: #fff1f0;
}
.attachment-table :deep(.attachment-row-new) {
  background-color: #f0f9ff;
}
.audit-action-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.audit-action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
.resubmit-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  border: 1px dashed #f3d19e;
  border-radius: 8px;
  background-color: #fff7e6;
}
.resubmit-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.resubmit-block {
  border: 1px solid #f5dab1;
  border-radius: 8px;
  padding: 16px;
  background-color: #fffdf6;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.4);
}
.resubmit-block-title {
  font-weight: 600;
  color: #b05b00;
  margin-bottom: 12px;
  font-size: 14px;
}
.audit-log-steps {
  max-height: 420px;
  padding-right: 8px;
  overflow-y: auto;
  padding-top: 10px;
}

.audit-record-card {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background-color: #fff;
  cursor: pointer;
  transition: all 0.3s;
}
.audit-record-card:hover {
  border-color: #c0c4cc;
}
.audit-record-card.is-selected {
  border-color: #409EFF;
  background-color: #ecf5ff;
}

.audit-cell {
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.audit-divider {
  height: 1px;
  background-color: #ebeef5;
  margin: 0;
}

.cell-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}

.subject-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cell-header .subject {
  font-weight: 600;
  color: #303133;
}

.cell-header .time {
  color: #909399;
  font-size: 12px;
}

.prefix {
  color: #909399;
  font-size: 12px;
}

.cell-content {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
}

.result-text {
  color: #303133;
}

.placeholder-content {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #909399;
  font-size: 13px;
  padding: 4px 0;
}

.custom-descriptions {
  width: 90%;
  margin: 0 auto;
}

.custom-descriptions :deep(.el-descriptions__label) {
  background-color: #fafafa;
  font-weight: 500;
  color: #606266;
  width: 120px;
}
.custom-descriptions :deep(.el-descriptions__content) {
  background-color: #fff;
  color: #606266;
}

.audit-timeline-col {
  border-right: 1px solid #ebeef5;
  padding-right: 20px;
}

.audit-detail-col {
  padding-left: 20px;
}

.detail-content {
  padding-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-placeholder {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
  color: #909399;
}

.action-divider {
  font-weight: bold;
  margin-bottom: 8px;
  color: #303133;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 4px;
}

.empty-attachment {
  padding: 10px 0;
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

.template-description-box {
  padding: 16px;
  background-color: #f9fafc;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  min-height: 100px;
  margin-bottom: 16px;
}

.existing-files-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}
.existing-file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}
.file-name {
  flex: 1;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-size {
  font-size: 12px;
  color: #909399;
}
</style>
