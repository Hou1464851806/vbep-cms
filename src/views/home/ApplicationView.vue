<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject, watch, computed } from 'vue'
import {
  Check,
  Close,
  View,
  Download,
  RefreshLeft,
  Memo,
  Clock,
  Upload,
  Delete,
  Refresh,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize } from '@/utils/format.js'

const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const approveApplicationAPI = (id, data) =>
  apiClient.post(`/api/v1/applications/${id}/approve`, data)
const rejectApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/reject`, data)
const returnApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/return`, data)
const resubmitApplicationAPI = (id, data, config = {}) =>
  apiClient.post(`/api/v1/applications/${id}/resubmit`, data, config)
const fetchApplicationAuditsAPI = (id) => apiClient.get(`/api/v1/applications/${id}/audits`)
const fetchApplicationFilesAPI = (id) => apiClient.get(`/api/v1/applications/${id}/files`)
const downloadApplicationFileAPI = (applicationId, fileId) =>
  apiClient.get(`/api/v1/applications/${applicationId}/files/${fileId}`, { responseType: 'blob' })

const APPLICATION_FILE_LIMIT_MB = 20
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

const readStoredUserId = () => {
  if (typeof window === 'undefined') return ''
  try {
    return window.localStorage.getItem('user_id') || ''
  } catch {
    return ''
  }
}

const currentUserId = ref(readStoredUserId())

const isAdmin = inject('isAdmin')
const applicationList = ref([])
const isLoading = ref(true)
const resubmitUploadRef = ref(null)
const detailDialog = reactive({
  visible: false,
  loading: false,
  data: null,
  audit: {
    result: '',
    loading: false,
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
})

const auditDialog = reactive({
  visible: false,
  loading: false,
  list: [],
  title: '',
  currentApplicationId: '',
  currentStatus: '',
})

const statusMetaMap = {
  pending: { label: '待审批', type: 'warning' },
  approved: { label: '已批准', type: 'success' },
  rejected: { label: '已驳回', type: 'danger' },
  returned: { label: '待修改', type: 'warning' },
}

const getStatusMeta = (status) => statusMetaMap[status] || { label: '未知', type: 'info' }
const getActionButtonLabel = (row) => {
  if (isAdmin()) {
    return row.status === 'pending' ? '审核' : '结果详情'
  }
  if (row.status === 'pending') return '申请详情'
  if (row.status === 'returned' && row.applicant?.id === currentUserId.value) {
    return '重新编辑'
  }
  return '结果详情'
}

const normalizeApplication = (item = {}) => ({
  id: item.id,
  status: item.status,
  statusMeta: getStatusMeta(item.status),
  dataset: {
    id: item.dataset?.id,
    title: item.dataset?.title,
  },
  applicant: {
    id: item.applicant?.id,
    nickname: item.applicant?.nickname,
    email: item.applicant?.email,
  },
  auditor: item.auditor
    ? {
        id: item.auditor.id,
        nickname: item.auditor.nickname,
        email: item.auditor.email,
      }
    : null,
  auditorResult: item.auditorResult || '',
  createdAt: formatDateTime(item.createdAt),
  auditedAt: formatDateTime(item.auditedAt),
})

const auditActionMeta = {
  submit: { label: '申请提交', color: '#409EFF', operatorLabel: '提交人', resultLabel: '备注' },
  approve: {
    label: '管理员审批',
    color: '#409EFF',
    operatorLabel: '审批人',
    actionDetail: '通过',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  reject: {
    label: '管理员审批',
    color: '#409EFF',
    operatorLabel: '审批人',
    actionDetail: '驳回',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  return: {
    label: '管理员审批',
    color: '#409EFF',
    operatorLabel: '审批人',
    actionDetail: '退回',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  resubmit: { label: '发起者修改', color: '#409EFF', operatorLabel: '提交人', resultLabel: '备注' },
}

const placeholderTimelineMeta = {
  admin: {
    id: 'placeholder-admin',
    label: '管理员审批',
    color: '#C0C4CC',
    placeholderText: '等待管理员审批',
  },
  applicant: {
    id: 'placeholder-applicant',
    label: '发起者修改',
    color: '#C0C4CC',
    placeholderText: '等待发起者修改后重新提交',
  },
}

const finalTimelineMeta = {
  approved: {
    id: 'final-approved',
    label: '审批流程结束',
    color: '#67C23A',
    placeholderText: '审批已通过，流程结束',
    icon: Check,
  },
  rejected: {
    id: 'final-rejected',
    label: '审批流程结束',
    color: '#F56C6C',
    placeholderText: '审批已驳回，流程结束',
    icon: Close,
  },
}

const normalizeAudit = (item = {}) => {
  const meta = auditActionMeta[item.action] || { label: '未知操作', color: '#909399' }
  return {
    id: item.id,
    action: item.action || 'unknown',
    label: meta.label,
    color: meta.color,
    operatorLabel: meta.operatorLabel || '操作人',
    resultLabel: meta.resultLabel || '备注',
    actionDetail: meta.actionDetail || '',
    showActionDetail: Boolean(meta.showActionDetail && meta.actionDetail),
    operator: item.operator
      ? {
          id: item.operator.id,
          nickname: item.operator.nickname,
          email: item.operator.email,
        }
      : null,
    result: item.result || '',
    time: formatDateTime(item.auditedAt || item.createdAt),
    isPlaceholder: false,
    isFinal: false,
    // icon: Aim,
    isHollow: true,
  }
}

const createPlaceholderTimelineItem = (key) => {
  const meta = placeholderTimelineMeta[key]
  if (!meta) return null
  return {
    id: meta.id,
    action: key,
    label: meta.label,
    color: meta.color,
    operatorLabel: '',
    resultLabel: '',
    actionDetail: '',
    showActionDetail: false,
    operator: null,
    result: '',
    time: '',
    isPlaceholder: true,
    isFinal: false,
    placeholderText: meta.placeholderText,
    icon: Clock,
    isHollow: false,
  }
}

const createFinalTimelineItem = (status) => {
  const meta = finalTimelineMeta[status]
  if (!meta) return null
  return {
    id: meta.id,
    action: 'final',
    label: meta.label,
    color: meta.color,
    operatorLabel: '',
    resultLabel: '',
    actionDetail: '',
    showActionDetail: false,
    operator: null,
    result: '',
    time: '',
    isPlaceholder: false,
    isFinal: true,
    placeholderText: meta.placeholderText,
    icon: meta.icon || Check,
    isHollow: false,
  }
}

const buildAuditTimeline = (list = [], status = '') => {
  const timeline = [...list]
  if (status === 'pending') {
    const placeholder = createPlaceholderTimelineItem('admin')
    if (placeholder) timeline.push(placeholder)
  } else if (status === 'returned') {
    const placeholder = createPlaceholderTimelineItem('applicant')
    if (placeholder) timeline.push(placeholder)
  } else if (status === 'approved' || status === 'rejected') {
    const finalItem = createFinalTimelineItem(status)
    if (finalItem) timeline.push(finalItem)
  }
  return timeline
}

const normalizeApplicationFiles = (files = []) =>
  files.map((item) => ({
    id: item.file_id || item.fileId || item.fileID || item.FileId || item.FileID,
    name: item.name || item.Name || '未命名文件',
    size: Number(item.size ?? item.Size ?? 0),
    createdAt: formatDateTime(item.createdAt || item.CreatedAt || item.Time?.CreatedAt),
  }))

const toEditableAttachment = (item = {}) => ({
  id: item.id || '',
  uid: item.id || `local-${Date.now()}-${Math.random().toString(16).slice(2)}`,
  name: item.name || '未命名文件',
  size: Number(item.size ?? 0),
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
  if (resubmitUploadRef.value) {
    resubmitUploadRef.value.clearFiles?.()
  }
}

const resetAuditDialog = () => {
  auditDialog.loading = false
  auditDialog.list = []
  auditDialog.title = ''
  auditDialog.currentApplicationId = ''
  auditDialog.currentStatus = ''
}

const resetResubmitState = () => {
  detailDialog.resubmit.loading = false
}

const loadApplicationFiles = async (applicationId) => {
  if (!applicationId) return
  detailDialog.attachments.isLoading = true
  try {
    const res = await fetchApplicationFilesAPI(applicationId)
    if (res?.data?.message === 'success') {
      const normalized = normalizeApplicationFiles(res.data.files)
      detailDialog.attachments.originalList = normalized.map((item) => ({ ...item }))
      detailDialog.attachments.list = normalized.map((item) => toEditableAttachment(item))
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
    ElMessage.error(`文件 "${rawFile.name}" 超过 ${APPLICATION_FILE_LIMIT_MB}MB 限制，无法上传`)
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
    if (res.data?.message === 'success') {
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
  detailDialog.audit.result = ''
  resetAttachmentState()
  try {
    const res = await fetchApplicationDetailAPI(row.id)
    if (res.data?.message === 'success') {
      detailDialog.data = normalizeApplication(res.data.application)
    } else {
      ElMessage.error(res.data?.message || '获取申请详情失败')
    }
  } catch (error) {
    console.error('Failed to load application detail', error)
    ElMessage.error('获取申请详情失败')
  } finally {
    detailDialog.loading = false
  }
}

const handleViewAuditLog = async (row) => {
  if (!row?.id) return
  auditDialog.visible = true
  auditDialog.loading = true
  auditDialog.list = []
  auditDialog.currentApplicationId = row.id
  auditDialog.currentStatus = row.status || ''
  const dialogSuffix = row.dataset?.title ? ` - ${row.dataset.title}` : ` - ${row.id}`
  auditDialog.title = `审批日志${dialogSuffix}`
  try {
    const res = await fetchApplicationAuditsAPI(row.id)
    if (res.data?.message === 'success') {
      const normalizedAudits = (res.data.audits || []).map((item) => normalizeAudit(item))
      auditDialog.list = buildAuditTimeline(normalizedAudits, auditDialog.currentStatus)
    } else {
      ElMessage.error(res.data?.message || '审批日志加载失败')
    }
  } catch (error) {
    console.error('Failed to load application audits', error)
    ElMessage.error('审批日志加载失败')
  } finally {
    auditDialog.loading = false
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

const auditActionAPIMap = {
  approve: approveApplicationAPI,
  reject: rejectApplicationAPI,
  return: returnApplicationAPI,
}

const auditActionSuccessMessage = {
  approve: '申请已批准',
  reject: '申请已驳回',
  return: '申请已退回',
}

const handleDialogAudit = async (action) => {
  const current = detailDialog.data
  if (!current?.id) return
  const result = detailDialog.audit.result.trim()
  if (!result) {
    ElMessage.warning('请填写审核意见')
    return
  }

  const api = auditActionAPIMap[action]
  if (!api) return

  detailDialog.audit.loading = true
  try {
    const res = await api(current.id, { result })
    if (res.data?.message === 'success' && res.data.application) {
      const normalized = normalizeApplication(res.data.application)
      updateApplicationEntry(res.data.application)
      detailDialog.audit.result = normalized.auditorResult
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

const handleResubmitApplication = async () => {
  const current = detailDialog.data
  if (!current?.id) return

  const { keptFileIds, newFiles } = getAttachmentDiff()
  const formData = new FormData()
  formData.append('keptFileIds', JSON.stringify(keptFileIds))
  newFiles.forEach((item) => {
    if (item.file) {
      formData.append('files', item.file, item.name)
    }
  })

  detailDialog.resubmit.loading = true
  try {
    const res = await resubmitApplicationAPI(current.id, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    if (res.data?.message === 'success' && res.data.application) {
      updateApplicationEntry(res.data.application)
      await loadApplicationFiles(current.id)
      ElMessage.success('申请已重新提交')
    } else {
      ElMessage.error(res.data?.message || '重新提交失败')
    }
  } catch (error) {
    console.error('Failed to resubmit application', error)
    ElMessage.error('重新提交失败')
  } finally {
    detailDialog.resubmit.loading = false
  }
}

watch(
  () => detailDialog.data,
  (value) => {
    detailDialog.audit.result = value?.auditorResult || ''
    if (!value) {
      resetAttachmentState()
      resetResubmitState()
      return
    }
    if (canViewApplicationAttachments(value)) {
      const shouldReload =
        detailDialog.attachments.currentApplicationId !== value.id ||
        detailDialog.attachments.list.length === 0
      if (shouldReload && value.id) {
        loadApplicationFiles(value.id)
      }
    } else {
      resetAttachmentState()
    }
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

watch(
  () => auditDialog.visible,
  (visible) => {
    if (!visible) {
      resetAuditDialog()
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
        </div>
      </div>
    </template>

    <el-skeleton :rows="5" animated :loading="isLoading">
      <el-table :data="applicationList" style="width: 100%">
        <el-table-column label="申请人">
          <template #default="scope">
            <div>{{ scope.row.applicant.nickname }}</div>
            <small v-if="scope.row.applicant.email" class="table-sub-text">{{
              scope.row.applicant.email
            }}</small>
          </template>
        </el-table-column>
        <el-table-column label="申请的数据集">
          <template #default="scope">
            {{ scope.row.dataset.title || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="申请日期" width="180" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.statusMeta.type">{{ scope.row.statusMeta.label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="scope">
            <el-space direction="vertical" :size="0" alignment="start">
              <el-button type="primary" text :icon="View" @click="handleViewDetail(scope.row)">
                {{ getActionButtonLabel(scope.row) }}
              </el-button>
              <el-button type="primary" text :icon="Memo" @click="handleViewAuditLog(scope.row)">
                审批日志
              </el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>
    </el-skeleton>
  </el-card>

  <el-dialog v-model="auditDialog.visible" :title="auditDialog.title || '审批日志'" width="600px">
    <el-skeleton :rows="4" animated :loading="auditDialog.loading">
      <template #default>
        <el-empty v-if="!auditDialog.list.length" description="暂无审批记录" />
        <el-timeline v-else class="audit-log-steps">
          <el-timeline-item
            v-for="item in auditDialog.list"
            :key="item.id"
            :timestamp="item.time"
            :color="item.color"
            :icon="item.icon"
            :hollow="item.isHollow"
          >
            <div class="audit-step">
              <div class="audit-step-label">{{ item.label }}</div>
              <el-card
                class="audit-step-card"
                shadow="never"
                :body-style="{ padding: '12px 12px' }"
              >
                <template v-if="item.isPlaceholder || item.isFinal">
                  <div class="audit-step-placeholder">
                    {{ item.placeholderText || '等待处理' }}
                  </div>
                </template>
                <template v-else>
                  <div class="audit-step-operator">
                    <template v-if="item.operator">
                      {{ item.operatorLabel }}：<span class="audit-step-bold-text">{{
                        item.operator.nickname
                      }}</span>
                      <span v-if="item.operator.email" class="table-sub-text">
                        ({{ item.operator.email }})</span
                      >
                    </template>
                    <template v-else
                      >{{ item.operatorLabel }}：<span class="audit-step-bold-text"
                        >系统</span
                      ></template
                    >
                  </div>
                  <div v-if="item.showActionDetail" class="audit-step-action">
                    审批操作：<span class="audit-step-bold-text">{{ item.actionDetail }}</span>
                  </div>
                  <div v-if="item.result" class="audit-step-result">
                    {{ item.resultLabel }}：<span class="audit-step-bold-text">{{
                      item.result
                    }}</span>
                  </div>
                </template>
              </el-card>
            </div>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-skeleton>
  </el-dialog>

  <el-dialog v-model="detailDialog.visible" title="申请详情" width="600px">
    <el-skeleton :rows="5" animated :loading="detailDialog.loading">
      <template #default>
        <el-descriptions border :column="1" v-if="detailDialog.data">
          <el-descriptions-item label="申请数据集">
            {{ detailDialog.data.dataset.title }} (ID: {{ detailDialog.data.dataset.id }})
          </el-descriptions-item>
          <el-descriptions-item label="申请人">
            {{ detailDialog.data.applicant.nickname }}
            <span v-if="detailDialog.data.applicant.email" class="table-sub-text"
              >({{ detailDialog.data.applicant.email }})</span
            >
          </el-descriptions-item>
          <el-descriptions-item label="申请状态">
            <el-tag :type="detailDialog.data.statusMeta.type">{{
              detailDialog.data.statusMeta.label
            }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="申请时间">
            {{ detailDialog.data.createdAt }}
          </el-descriptions-item>
          <el-descriptions-item label="审核时间">
            {{ detailDialog.data.auditedAt || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="审核人">
            <template v-if="detailDialog.data.auditor">
              {{ detailDialog.data.auditor.nickname }}
              <span v-if="detailDialog.data.auditor.email" class="table-sub-text"
                >({{ detailDialog.data.auditor.email }})</span
              >
            </template>
            <span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item label="审核意见">
            <div v-if="detailDialog.data.auditorResult">
              {{ detailDialog.data.auditorResult }}
            </div>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
        <div v-if="canViewAttachments && !canResubmitApplication" class="attachment-section">
          <div class="attachment-header">
            <span>申请附件</span>
          </div>
          <el-skeleton :rows="3" animated :loading="detailDialog.attachments.isLoading">
            <template #default>
              <el-empty v-if="!detailDialog.attachments.list.length" description="暂无附件" />
              <el-table
                v-else
                :data="detailDialog.attachments.list"
                size="small"
                class="attachment-table"
              >
                <el-table-column prop="name" label="文件名" min-width="200" show-overflow-tooltip />
                <el-table-column label="大小" width="120">
                  <template #default="scope">{{ formatFileSize(scope.row.size) }}</template>
                </el-table-column>
                <el-table-column prop="createdAt" label="上传时间" width="180" />
                <el-table-column label="操作" width="200">
                  <template #default="scope">
                    <el-space>
                      <el-button
                        text
                        size="small"
                        :icon="Download"
                        @click="handleDownloadAttachment(scope.row)"
                      >
                        下载
                      </el-button>
                    </el-space>
                  </template>
                </el-table-column>
              </el-table>
            </template>
          </el-skeleton>
        </div>
        <div v-if="canResubmitApplication" class="resubmit-area">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="申请已被退回，请根据意见修改信息后重新提交"
          />
          <div class="resubmit-content">
            <div class="resubmit-block">
              <div class="resubmit-block-title">申请附件</div>
              <el-skeleton :rows="3" animated :loading="detailDialog.attachments.isLoading">
                <template #default>
                  <el-empty v-if="!detailDialog.attachments.list.length" description="暂无附件" />
                  <el-table
                    v-else
                    :data="detailDialog.attachments.list"
                    size="small"
                    class="attachment-table"
                    :row-class-name="getAttachmentRowClass"
                  >
                    <el-table-column
                      prop="name"
                      label="文件名"
                      min-width="200"
                      show-overflow-tooltip
                    />
                    <el-table-column label="大小" width="120">
                      <template #default="scope">{{ formatFileSize(scope.row.size) }}</template>
                    </el-table-column>
                    <el-table-column prop="createdAt" label="更新时间" width="180" />
                    <el-table-column label="状态" width="120">
                      <template #default="scope">
                        <el-tag :type="getAttachmentStatusMeta(scope.row).type">
                          {{ getAttachmentStatusMeta(scope.row).label }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="220">
                      <template #default="scope">
                        <el-space>
                          <el-button
                            text
                            size="small"
                            :icon="Download"
                            @click="handleDownloadAttachment(scope.row)"
                          >
                            下载
                          </el-button>
                          <el-button
                            text
                            :type="scope.row.isRemoved ? 'primary' : 'danger'"
                            size="small"
                            :icon="Delete"
                            @click="handleRemoveAttachmentLocal(scope.row)"
                          >
                            <template v-if="scope.row.id">
                              {{ scope.row.isRemoved ? '撤销移除' : '移除' }}
                            </template>
                            <template v-else>撤销</template>
                          </el-button>
                        </el-space>
                      </template>
                    </el-table-column>
                  </el-table>
                </template>
              </el-skeleton>
              <div class="attachment-upload-area">
                <div class="attachment-upload-header">
                  <span>可重新上传补充材料</span>
                  <small>单个文件大小不超过 {{ APPLICATION_FILE_LIMIT_MB }}MB</small>
                </div>
                <el-upload
                  ref="resubmitUploadRef"
                  action=""
                  :show-file-list="false"
                  :auto-upload="false"
                  :disabled="detailDialog.attachments.isLoading || detailDialog.resubmit.loading"
                  :on-change="handleAttachmentFileQueued"
                >
                  <el-button
                    type="primary"
                    text
                    :icon="Upload"
                    :loading="detailDialog.resubmit.loading"
                    >选择文件</el-button
                  >
                </el-upload>
                <div class="attachment-upload-tip">
                  支持常见文档、图片及压缩包格式，变更仅在重新提交后生效。
                </div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="isAdmin() && detailDialog.data?.status === 'pending'" class="audit-action-area">
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
              >退回</el-button
            >
            <el-button
              type="danger"
              :loading="detailDialog.audit.loading"
              :icon="Close"
              @click="handleDialogAudit('reject')"
              >驳回</el-button
            >
            <el-button
              type="success"
              :loading="detailDialog.audit.loading"
              :icon="Check"
              @click="handleDialogAudit('approve')"
              >批准</el-button
            >
          </div>
        </div>
      </template>
    </el-skeleton>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="detailDialog.visible = false">返回</el-button>
        <el-button
          v-if="canResubmitApplication"
          type="primary"
          :loading="detailDialog.resubmit.loading"
          @click="handleResubmitApplication"
          >重新提交审核</el-button
        >
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.table-sub-text {
  color: #909399;
  font-size: 12px;
}
.reason-block {
  white-space: pre-wrap;
  line-height: 1.6;
}
.attachment-section {
  margin-top: 20px;
  padding: 16px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background-color: #f9fafc;
}
.attachment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
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
  margin-top: 16px;
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
  margin-top: 20px;
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
}
.audit-step {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.6;
}
.audit-step-card {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background-color: #f8fafc;
}
.audit-step-bold-text {
  color: #000000;
}
.audit-step-label {
  font-weight: 600;
  color: #303133;
  font-size: 15px;
}
.audit-step-operator {
  color: #606266;
}
.audit-step-action {
  color: #606266;
}
.audit-step-result {
  color: #606266;
}
.audit-step-placeholder {
  color: #606266;
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
