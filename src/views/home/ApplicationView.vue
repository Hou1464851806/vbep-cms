<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject, watch, computed } from 'vue'
import { genFileId } from 'element-plus'
import MarkdownRenderer from "@/components/MarkdownRenderer.vue"
import {
  View,
  Download,
  Upload,
  Delete,
  Refresh,
  Timer,
  Edit,
  Plus,
  Back,
  Close,
  Warning,
  CircleCheck,
  TopRight,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize, formatShortId } from '@/utils/format.js'
import { APPLICATION_FILE_LIMIT_MB } from '@/utils/constrains.js'
import {useRoute, useRouter} from "vue-router";
import ListPagination from "@/components/ListPagination.vue";

import {useTableFilter} from "@/hooks/useTable.js"
import TableHeaderFilter from "@/components/TableHeaderFilter.vue"
import IdCell from "@/components/IdCell.vue"

const route = useRoute()
const router = useRouter()

const fetchDatasetListAPI = () => apiClient.get('/api/v1/datasets/')
const fetchApplicationListAPI = (page, size) => apiClient.get('/api/v1/applications/', { params: { page, size } })
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const auditApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/audit`, data)
const resubmitApplicationAPI = (id, data, config = {}) =>
  apiClient.post(`/api/v1/applications/${id}/resubmit`, data, config)
const fetchAuditDetailAPI = (appId, auditId) => apiClient.get(`/api/v1/applications/${appId}/audits/${auditId}`)
const fetchApplicationFilesAPI = (id) => apiClient.get(`/api/v1/applications/${id}/files`)
const downloadApplicationFileAPI = (applicationId, fileId) =>
  apiClient.get(`/api/v1/applications/${applicationId}/files/${fileId}`, { responseType: 'blob' })
const deleteApplicationAPI = (id) => apiClient.delete(`/api/v1/applications/${id}`)
const confirmApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/submit`, data)
const createApplicationAPI = (data) => apiClient.post('/api/v1/applications/', data)
const updateApplicationDraftAPI = (id, data) => apiClient.put(`/api/v1/applications/${id}`, data)
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

// // 分页相关
// const pagination = reactive({
//   currentPage: 1,
//   pageSize: 10,
//   total: 0
// })

const isAdmin = inject('isAdmin')
// const applicationList = ref([])
// const isLoading = ref(true)
const datasetList = ref([])
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

// 状态：审批对话框
const auditDialog = reactive({
  visible: false,
  loading: false,
  applicationId: '',
  action: '',
  result: '',
  licenseDuration: 3,
})

// 状态：草稿编辑对话框
const draftDialog = reactive({
  visible: false,
  isSubmitting: false,
  data: null,
  mode: 'create', // 'create' | 'resubmit'
  affiliation: '',
  position: '',
  mandatory: { list: [] },
  identity: { list: [] },
  optional: { list: [] },
})

// Upload Refs for handleExceed
const mandatoryUploadRef = ref()
const identityUploadRef = ref()

// (Removed licenseStatus and approvedDate logic)

// 状态：新增申请流程
const newAppDialog = reactive({
  visible: false,
  loading: false,
  selectedDatasetId: '',
  description: '',
  files: [],
})

const activeNames = ref(['1', '2'])

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
    affiliation: item.affiliation || '',
    position: item.position || '',
    applicant: {
      id: item.applicant?.id,
      nickname: item.applicant?.nickname,
      email: item.applicant?.email,
    },
    createdAt: formatDateTime(item.createdAt),
    auditedAt: formatDateTime(item.updatedAt),
    expireAt: item.expireAt,
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
      detailDialog.attachments.currentApplicationId = applicationId
      return normalized // 返回文件列表供调用者使用
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

// const fetchApplications = async () => {
//   isLoading.value = true
//   try {
//     const res = await fetchApplicationListAPI(pagination.currentPage, pagination.pageSize)
//     if (res.status === 200) {
//       applicationList.value = res.data.applications.map((item) => normalizeApplication(item))
//       pagination.total = res.data.total || 0
//     } else {
//       ElMessage.error(res.data?.message || '申请列表加载失败')
//     }
//   } catch (error) {
//     console.error('Failed to load applications', error)
//     ElMessage.error('申请列表加载失败')
//   } finally {
//     isLoading.value = false
//   }
// }

// const handlePageChange = () => {
//   fetchApplications()
// }

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

const goToLicense = (applicationId) => {
  ElMessageBox.confirm(
    '您即将离开当前页，前往查看对应的许可记录。确定吗？',
    '提示',
    {
      confirmButtonText: '前往',
      cancelButtonText: '取消',
      type: 'info',
    }
  ).then(() => {
    router.push({ path: '/home/licenses', query: { application_id: applicationId } })
  }).catch(() => {})
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

const auditLabel = computed(() => {
  const map = {
    return: '退回理由',
    reject: '拒绝理由',
    approve: '审批附言',
  }
  return map[auditDialog.action] || '审批意见'
})

const openAuditDialog = (row) => {
  if (!row?.id) return
  auditDialog.applicationId = row.id
  auditDialog.action = ''
  auditDialog.result = ''
  auditDialog.licenseDuration = 3
  auditDialog.visible = true
}

const handleAuditSubmit = async () => {
  if (!auditDialog.action) {
    ElMessage.warning('请选择审批操作')
    return
  }
  // 仅当操作不是批准时，才强制要求填写理由
  if (auditDialog.action !== 'approve' && !auditDialog.result.trim()) {
    ElMessage.warning('请填写' + auditLabel.value)
    return
  }

  const actionMap = {
    approve: { label: '批准', apiAction: 'approved', confirm: '您即将批准该申请，批准后用户将有权访问该数据集下的所有文件' },
    reject: { label: '拒绝', apiAction: 'rejected', confirm: '您即将拒绝该申请，拒绝后用户将无法修改并重新提交申请' },
    return: { label: '退回', apiAction: 'returned', confirm: '您即将退回该申请，退回后用户可以修改并重新提交申请' }
  }

  const actionInfo = actionMap[auditDialog.action]
  if (!actionInfo) return

  try {
    await ElMessageBox.confirm(actionInfo.confirm, `确认${actionInfo.label}`, {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: auditDialog.action === 'reject' ? 'error' : 'warning',
    })
  } catch {
    return
  }

  auditDialog.loading = true
  try {
    const res = await auditApplicationAPI(auditDialog.applicationId, {
      action: actionInfo.apiAction,
      result: auditDialog.result,
      duration: auditDialog.licenseDuration,
    })
    if (res.status === 200 && res.data.application) {
      updateApplicationEntry(res.data.application)
      ElMessage.success(`申请已${actionInfo.label}`)
      auditDialog.visible = false
      detailDialog.visible = false
    } else {
      ElMessage.error(res.data?.message || '操作失败')
    }
  } catch (error) {
    console.error('Failed to process application', error)
    ElMessage.error('操作失败')
  } finally {
    auditDialog.loading = false
  }
}

// --- 草稿与重填管理逻辑 ---

// 打开草稿/重填窗口的通用方法
const openEditDialog = async (row, mode = 'create') => {
  draftDialog.data = row
  draftDialog.mode = mode
  draftDialog.affiliation = row.affiliation || ''
  draftDialog.position = row.position || ''
  draftDialog.mandatory.list = []
  draftDialog.identity.list = []
  draftDialog.optional.list = []
  draftDialog.visible = true

  // 加载已有文件 (无论是重填还是继续编辑草稿)
  if (mode === 'resubmit' || mode === 'create') {
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
    } catch (error) {
      console.error('Failed to load existing files:', error)
      // 仅在重填模式下报错，草稿模式下如果没文件可能是正常的（或者新建的）
      if (mode === 'resubmit') ElMessage.error('加载已有文件失败')
    } finally {
      draftDialog.isSubmitting = false
    }
  }
}

const performResubmit = async (applicationId, keptFileIDs, newFiles) => {
  const formData = new FormData()
  formData.append('keptFileIDs', JSON.stringify(keptFileIDs))
  formData.append('affiliation', draftDialog.affiliation)
  formData.append('position', draftDialog.position)
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

// --- File Upload Handlers ---

const handleBeforeUpload = (rawFile, type) => {
  const fileName = rawFile.name.toLowerCase()

  // Size check
  if (rawFile.size > APPLICATION_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${formatFileSize(APPLICATION_FILE_LIMIT_BYTES)} 限制`)
    return false
  }

  // Type check
  if (type === 'form') {
    if (!fileName.endsWith('.pdf')) {
      ElMessage.error('申请表仅支持 PDF 格式')
      return false
    }
  } else if (type === 'identity') {
    if (!IDENTITY_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
      ElMessage.error(`文件格式不支持："${rawFile.name}"，请上传 PDF/Word/图片 格式`)
      return false
    }
  } else if (type === 'general') {
    if (DENIED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
      ElMessage.error(`文件格式被拒绝："${rawFile.name}"`)
      return false
    }
    if (!ALLOWED_EXTENSIONS.some((ext) => fileName.endsWith(ext))) {
      ElMessage.error(`文件格式不受支持："${rawFile.name}"`)
      return false
    }
  }
  return true
}

const handleUploadRequest = async (options, type) => {
  const { file, onProgress, onSuccess, onError } = options
  const applicationId = draftDialog.data.id

  try {
    const res = await uploadApplicationFileAPI(applicationId, file, {
      type: type,
      onUploadProgress: (e) => {
        if (e.total > 0) {
          e.percent = Math.round((e.loaded / e.total) * 100)
        }
        onProgress(e)
      },
    })

    if (res.status === 200 && res.data.file) {
      onSuccess(res.data)

      // 手动回填 ID，确保提交时的校验和 ID 收集能获取到新上传文件的 ID
      const updateListWithId = (list) => {
        const item = list.find((f) => f.uid === file.uid)
        if (item) {
          item.id = res.data.file.fileID
        }
      }
      if (type === 'form') updateListWithId(draftDialog.mandatory.list)
      else if (type === 'identity') updateListWithId(draftDialog.identity.list)
      else if (type === 'general') updateListWithId(draftDialog.optional.list)
    } else {
      onError(new Error(res.data?.message || '上传失败'))
    }
  } catch (err) {
    onError(err)
  }
}

const handleRemoveFile = async (uploadFile) => {
  // 判断是否需要物理删除：
  // 1. 如果是新上传的文件 (有 response)，删除。
  // 2. 如果是草稿模式下的已有文件，删除。
  // 3. 如果是重填模式下的已有文件 (历史文件)，仅从列表移除，不物理删除。
  const isNewUpload = !!uploadFile.response
  const isDraft = draftDialog.mode === 'create'

  if (isNewUpload || isDraft) {
    const fileId = uploadFile.id || uploadFile.response?.file?.fileID
    if (fileId) {
      try {
        await apiClient.delete(`/api/v1/applications/${draftDialog.data.id}/files/${fileId}`)
      } catch (e) {
        console.error('Failed to delete file', e)
        // 不阻断 UI 移除
      }
    }
  }
  return true
}

const handleExceedMandatory = (files) => {
  const upload = mandatoryUploadRef.value
  // 手动触发移除逻辑以清理后端（如果需要）
  if (draftDialog.mandatory.list.length > 0) {
    handleRemoveFile(draftDialog.mandatory.list[0])
  }
  upload.clearFiles()
  const file = files[0]
  file.uid = genFileId()
  upload.handleStart(file)
}

const handleExceedIdentity = (files) => {
  const upload = identityUploadRef.value
  if (draftDialog.identity.list.length > 0) {
    handleRemoveFile(draftDialog.identity.list[0])
  }
  upload.clearFiles()
  const file = files[0]
  file.uid = genFileId()
  upload.handleStart(file)
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

const handleSaveDraft = async () => {
  if (!draftDialog.data?.id) return

  draftDialog.isSubmitting = true
  try {
    await updateApplicationDraftAPI(draftDialog.data.id, {
      affiliation: draftDialog.affiliation,
      position: draftDialog.position,
    })
    ElMessage.success('已暂存')
    draftDialog.visible = false
    await fetchApplications()
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '暂存失败')
  } finally {
    draftDialog.isSubmitting = false
  }
}

const handleSubmitDraft = async () => {
  // 校验逻辑：检查列表中是否有文件（无论是新上传的还是已有的）
  // 此时文件应该已经上传，item 应该有 id (旧文件) 或 response.file.fileID (新文件)
  const getFileId = (item) => item.id || item.response?.file?.fileID

  const hasMandatory = draftDialog.mandatory.list.some(getFileId)
  const hasIdentity = draftDialog.identity.list.some(getFileId)
  const hasOptional = draftDialog.optional.list.some(getFileId)

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
      // 收集所有当前列表中的文件 ID (包括旧的和新上传的)
      const allFiles = [
        ...draftDialog.mandatory.list,
        ...draftDialog.identity.list,
        ...draftDialog.optional.list,
      ]
      const keptFileIDs = allFiles.map(getFileId).filter((id) => !!id)

      // 调用 resubmit 接口，只传 keptFileIDs，不传 files (因为都已上传)
      const formData = new FormData()
      formData.append('keptFileIDs', JSON.stringify(keptFileIDs))
      formData.append('affiliation', draftDialog.affiliation)
      formData.append('position', draftDialog.position)
      await resubmitApplicationAPI(applicationId, formData)

      ElMessage.success('申请已重新提交')
    } else {
      // 草稿模式：文件已在选择时上传，直接确认提交
      await confirmApplicationAPI(applicationId, {
        affiliation: draftDialog.affiliation,
        position: draftDialog.position,
      })
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


const statusFilterOptions = [
  { label: '待提交', value: 'draft' },
  { label: '待审批', value: 'pending' },
  { label: '已批准', value: 'approved' },
  { label: '已拒绝', value: 'rejected' },
  { label: '已退回', value: 'returned' },
]
const {
  loading: isLoading,
  data: applicationList,
  pagination,
  filterForm,
  sortState,
  isFilterActive,
  loadData: fetchApplications,
  handleSortChange,
  handleFilterChange,
  handleFilterInputChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/applications/', { params }),
  {
    normalize: normalizeApplication,
    dataKey: 'applications',
    defaultSort: { prop: 'auditedAt', order: 'descending' },
    filterSchema: { id_prefix: '', applicant: '', dataset_name: '', status: '', date: [] },
    pageSize: 10,
    syncUrl: true,
  }
)

watch(
  () => detailDialog.data,
  (value) => {
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

onMounted(async () => {
  // 保存 ?id= 参数，避免 syncUrl 替换 URL 后丢失
  const pendingDetailId = route.query.id
  await fetchApplications()
  if (pendingDetailId) {
    handleViewDetail({ id: pendingDetailId })
  }
})
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>申请列表</span>
        <div class="header-actions">
          <el-button
            v-if="isFilterActive"
            class="reset-filter-btn"
            plain
            size="small"
            @click="resetFilter"
          >重置筛选</el-button>
          <el-button :icon="Refresh" circle @click="fetchApplications" :loading="isLoading" title="刷新列表" />
          <el-button v-if="!isAdmin()" type="primary" :icon="Plus" @click="handleNewApplication">新增申请</el-button>
        </div>
      </div>
    </template>

<!--    <el-skeleton :rows="5" animated :loading="isLoading">-->
    <div v-loading="isLoading">
      <el-table
        :data="applicationList"
        style="width: 100%"
      >
        <el-table-column label="申请ID" width="110">
          <template #header>
            <table-header-filter
              label="申请ID"
              type="id-search"
              v-model="filterForm.id_prefix"
              @change="handleFilterChange"
            />
          </template>
          <template #default="{ row }">
            <id-cell :id="row.id" />
          </template>
        </el-table-column>

        <el-table-column label="申请人" width="140">
          <template #header>
            <table-header-filter
              label="申请人"
              type="input"
              v-model="filterForm.applicant"
              @change="handleFilterChange"
              sort-prop="applicantName"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="scope">
            <div>{{ scope.row.applicant.nickname }}</div>
          </template>
        </el-table-column>

        <el-table-column label="目标数据集" min-width="150" show-overflow-tooltip>
          <template #header>
            <table-header-filter
              label="目标数据集"
              type="input"
              v-model="filterForm.dataset_name"
              @change="handleFilterChange"
              sort-prop="datasetTitle"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="scope">
            <span class="table-name-text">{{ scope.row.dataset.title || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="最后更新时间" width="180">
          <template #header>
            <table-header-filter
              label="最后更新时间"
              type="daterange"
              v-model="filterForm.date"
              @change="handleFilterChange"
              sort-prop="auditedAt"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">
            <span class="table-time-text">{{ row.auditedAt }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="130">
          <template #header>
            <table-header-filter
              label="状态"
              type="select"
              v-model="filterForm.status"
              :options="statusFilterOptions"
              @change="handleFilterChange"
            />
          </template>
          <template #default="scope">
            <el-tag :type="scope.row.statusMeta.type">{{ scope.row.statusMeta.label }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="详细信息" width="100" fixed="right" align="center">
          <template #default="scope">
            <el-button
              v-if="scope.row.status !== 'draft'"
              type="primary"
              link
              :icon="View"
              @click="handleViewDetail(scope.row)"
            >
              查看
            </el-button>
            <el-tooltip v-else content="草稿状态不适用" placement="top">
              <div style="display: inline-block;">
                <el-button
                  type="info"
                  link
                  :icon="View"
                  disabled
                >
                  查看
                </el-button>
              </div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="scope">
            <template v-if="scope.row.status === 'draft'">
              <el-button v-if="!isAdmin()" type="primary" link :icon="Edit" @click="handleContinueDraft(scope.row)">
                修改
              </el-button>
              <el-tooltip content="删除草稿" placement="top">
                <el-button type="danger" link :icon="Delete" @click="handleDeleteDraft(scope.row)" />
              </el-tooltip>
            </template>
            <template v-else>
              <el-button
                v-if="isAdmin() && scope.row.status === 'pending'"
                type="primary"
                link
                :icon="Edit"
                @click="openAuditDialog(scope.row)"
              >
                审批
              </el-button>
              <el-tooltip
                v-else-if="isAdmin()"
                :content="(scope.row.status === 'draft' || scope.row.status === 'returned') ? '未提交的申请不可审批' : '已审批完成'"
                placement="top"
              >
                <div style="display: inline-block;">
                  <el-button
                    type="info"
                    link
                    :icon="Edit"
                    disabled
                  >
                    审批
                  </el-button>
                </div>
              </el-tooltip>
              <template v-else>
                <el-button
                  v-if="scope.row.status === 'returned'"
                  type="primary"
                  link
                  :icon="Edit"
                  @click="handleEditReturned(scope.row)"
                >
                  修改
                </el-button>
                <el-tooltip v-else :content="'当前状态无法修改'" placement="top">
                  <div style="display: inline-block;">
                    <el-button
                      type="info"
                      link
                      :icon="Edit"
                      disabled
                    >
                      修改
                    </el-button>
                  </div>
                </el-tooltip>
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
<!--    </el-skeleton>-->
    </div>
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
        <div class="upload-label required-label">1. 申请表 (必填)</div>
        <div class="upload-desc">请上传填写并签署后的申请模板文件，仅支持 PDF 格式，限传 1 个（每个不超过 {{ formatFileSize(APPLICATION_FILE_LIMIT_BYTES) }}）。</div>
        <el-upload
          class="upload-demo"
          ref="mandatoryUploadRef"
          action="#"
          :auto-upload="true"
          accept=".pdf"
          :limit="1"
          :on-exceed="handleExceedMandatory"
          :http-request="(options) => handleUploadRequest(options, 'form')"
          :before-upload="(file) => handleBeforeUpload(file, 'form')"
          :on-remove="handleRemoveFile"
          v-model:file-list="draftDialog.mandatory.list"
        >
          <el-button :icon="Upload">{{ draftDialog.mandatory.list.length > 0 ? '替换文件' : '选择文件' }}</el-button>
        </el-upload>
      </div>

      <el-divider />

      <div class="upload-section">
        <div class="upload-label required-label">2. 身份信息 (必填)</div>

        <div class="sub-input-group">
          <div class="input-guide-text">请再次确认您所代表的单位和职务：</div>
          <div style="display: flex; gap: 16px;">
            <el-input v-model="draftDialog.affiliation" placeholder="单位全称" style="flex: 2" />
            <el-input v-model="draftDialog.position" placeholder="职务" style="flex: 1" />
          </div>
        </div>

        <div class="sub-input-group">
          <div class="input-guide-text">请上传身份证明文件：</div>
          <div class="upload-desc">请上传学生证或工作证等有效身份证明文件，支持 PDF、Word、图片格式，限传 1 个（每个不超过 {{ formatFileSize(APPLICATION_FILE_LIMIT_BYTES) }}）。</div>
          <el-upload
            class="upload-demo"
            ref="identityUploadRef"
            action="#"
            :auto-upload="true"
            :accept="IDENTITY_EXTENSIONS.join(',')"
            :limit="1"
            :on-exceed="handleExceedIdentity"
            :http-request="(options) => handleUploadRequest(options, 'identity')"
            :before-upload="(file) => handleBeforeUpload(file, 'identity')"
            :on-remove="handleRemoveFile"
            v-model:file-list="draftDialog.identity.list"
          >
            <el-button :icon="Upload">{{ draftDialog.identity.list.length > 0 ? '替换文件' : '选择文件' }}</el-button>
          </el-upload>
        </div>
      </div>

      <el-divider />

      <div class="upload-section">
        <div class="upload-label">3. 补充材料 (可选)</div>
        <div class="upload-desc">如有其他证明材料或情况说明，可在此上传。支持大部分常见格式，不限数量。（每个不超过 {{ formatFileSize(APPLICATION_FILE_LIMIT_BYTES) }}）。</div>
        <el-upload
          class="upload-demo"
          action="#"
          :auto-upload="true"
          multiple
          :http-request="(options) => handleUploadRequest(options, 'general')"
          :before-upload="(file) => handleBeforeUpload(file, 'general')"
          :on-remove="handleRemoveFile"
          v-model:file-list="draftDialog.optional.list"
        >
          <el-button :icon="Upload">选择文件</el-button>
        </el-upload>
      </div>
    </div>
    <template #footer>
      <el-button @click="handleSaveDraft" :loading="draftDialog.isSubmitting">暂存</el-button>
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
              <el-descriptions-item label="审批状态">
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
              <el-descriptions-item label="创建时间">
                {{ detailDialog.data.createdAt }}
              </el-descriptions-item>
              <el-descriptions-item label="申请人">
                {{ detailDialog.data.applicant.nickname }}
                <span v-if="detailDialog.data.applicant.email" class="sub-text">
                  ({{ detailDialog.data.applicant.email }})
                </span>
              </el-descriptions-item>
              <el-descriptions-item label="最后修改时间">
                {{ detailDialog.data.auditedAt || '-' }}
              </el-descriptions-item>
              <el-descriptions-item label="单位/职务">
                {{ detailDialog.data.affiliation || '-' }}
                <span v-if="detailDialog.data.position"> / {{ detailDialog.data.position }}</span>
              </el-descriptions-item>
              <el-descriptions-item v-if="detailDialog.data.status === 'approved'" label="许可状态">
                <el-link type="primary" @click="goToLicense(detailDialog.data.id)" style="display: inline-flex; align-items: center; gap: 4px;">
                  查看许可 <el-icon><TopRight /></el-icon>
                </el-link>
              </el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>

          <el-collapse-item title="审批时间线" name="2">
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
                            <div style="display: flex; align-items: center; gap: 8px;">
                              <el-icon><Timer /></el-icon>
                              <span>等待管理员审批</span>
                            </div>
                            <el-button
                              v-if="isAdmin()"
                              type="primary"
                              link
                              :icon="Edit"
                              @click.stop="openAuditDialog(detailDialog.data)"
                            >
                              立即审批
                            </el-button>
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

                </div>
              </el-col>
            </el-row>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-skeleton>
    <template #footer>
      <div class="dialog-footer">
        <el-button
          v-if="canResubmitApplication"
          type="primary"
          @click="handleEditReturned(detailDialog.data)"
          >修改并重新提交</el-button
        >
      </div>
    </template>
  </el-dialog>

  <!-- 审批对话框 -->
  <el-dialog v-model="auditDialog.visible" title="审批申请" width="500px" :close-on-click-modal="false">
    <el-form label-position="top">
      <el-form-item label="审批操作" required>
        <el-select v-model="auditDialog.action" placeholder="请选择审批操作" style="width: 100%">
          <el-option label="退回修改" value="return">
            <div style="display: flex; align-items: center;">
              <el-icon style="color: var(--el-color-warning); margin-right: 8px"><Back /></el-icon>
              <span>退回修改</span>
            </div>
          </el-option>
          <el-option label="拒绝申请" value="reject">
            <div style="display: flex; align-items: center;">
              <el-icon style="color: var(--el-color-danger); margin-right: 8px"><Close /></el-icon>
              <span>拒绝申请</span>
            </div>
          </el-option>
          <el-option label="批准申请" value="approve">
            <div style="display: flex; align-items: center;">
              <el-icon style="color: var(--el-color-success); margin-right: 8px"><Check /></el-icon>
              <span>批准申请</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item v-if="auditDialog.action === 'approve'" label="许可期限" required>
        <el-select v-model="auditDialog.licenseDuration" placeholder="请选择许可期限" style="width: 100%">
          <el-option label="1 个月" :value="1" />
          <el-option label="2 个月" :value="2" />
          <el-option label="3 个月" :value="3" />
          <el-option label="6 个月" :value="6" />
          <el-option label="12 个月" :value="12" />
          <el-option label="24 个月" :value="24" />
        </el-select>
      </el-form-item>
      <el-form-item :label="auditLabel" :required="!!auditDialog.action && auditDialog.action !== 'approve'">
        <el-input
          v-if="!auditDialog.action"
          disabled
          type="textarea"
          :rows="4"
          placeholder="请先选择审批操作..."
        />
        <el-input
          v-else
          v-model="auditDialog.result"
          type="textarea"
          :rows="4"
          :placeholder="`请输入${auditLabel}...`"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="auditDialog.visible = false">取消</el-button>
      <el-button type="primary" @click="handleAuditSubmit" :loading="auditDialog.loading">确认</el-button>
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
  justify-content: space-between;
  width: 100%;
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-size {
  font-size: 12px;
  color: #909399;
}
.license-status-wrapper {
  padding: 10px 0;
}
.license-status-box {
  width: 90%;
  margin: 0 auto;
  padding: 16px 24px;
  border-radius: 8px;
  border: 1px solid;
  display: flex;
  align-items: center;
}
.license-left {
  flex: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-right: 1px solid rgba(0, 0, 0, 0.05);
  padding-right: 20px;
  text-align: center;
}
.license-right {
  flex: 3;
  padding-left: 20px;
}
.license-icon {
  font-size: 48px;
  margin-bottom: 8px;
}
.license-title {
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 0;
}
.license-content {
  font-size: 14px;
  line-height: 1.6;
  color: #606266;
}
.license-content p {
  margin: 0 0 8px 0;
}
.license-content p strong {
  font-weight: bold;
}
.license-content p:last-child {
  margin-bottom: 0;
}
.license-status-box.is-valid {
  background-color: #f0f9eb;
  border-color: #b3e19d;
}
.license-status-box.is-valid .license-title {
  color: #67c23a;
}
.license-status-box.is-valid .license-icon {
  color: #b3e19d;
}
.license-status-box.is-revoked {
  background-color: #fef0f0;
  border-color: #fab6b6;
}
.license-status-box.is-revoked .license-title {
  color: #f56c6c;
}
.license-status-box.is-revoked .license-icon {
  color: #fab6b6;
}
.license-status-box.is-unavailable {
  background-color: #f4f4f5;
  border-color: #c8c9cc;
}
.license-status-box.is-unavailable .license-title {
  color: #909399;
}
.license-status-box.is-unavailable .license-icon {
  color: #c8c9cc;
}
.sub-input-group {
  margin-bottom: 20px;
}
.input-guide-text {
  font-size: 14px;
  color: #606266;
  margin-bottom: 8px;
}
</style>
