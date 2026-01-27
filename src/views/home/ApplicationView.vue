<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject, watch, computed } from 'vue'
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
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize, formatShortId } from '@/utils/format.js'
import { APPLICATION_FILE_LIMIT_MB } from '@/utils/constrains.js'

const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const auditApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/audit`, data)
const resubmitApplicationAPI = (id, data, config = {}) =>
  apiClient.post(`/api/v1/applications/${id}/resubmit`, data, config)
const fetchAuditDetailAPI = (appId, auditId) => apiClient.get(`/api/v1/applications/${appId}/audits/${auditId}`)
const fetchApplicationFilesAPI = (id) => apiClient.get(`/api/v1/applications/${id}/files`)
const downloadApplicationFileAPI = (applicationId, fileId) =>
  apiClient.get(`/api/v1/applications/${applicationId}/files/${fileId}`, { responseType: 'blob' })

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
    return window.localStorage.getItem('userID') || ''
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

const activeNames = ref(['2'])

const statusMetaMap = {
  pending: { label: '等待审批', type: 'warning' },
  approved: { label: '已批准', type: 'success' },
  rejected: { label: '已拒绝', type: 'danger' },
  returned: { label: '已退回', type: 'warning' },
}

const normalizeApplication = (item = {}) => ({
  id: item.id,
  status: item.status,
  statusMeta: statusMetaMap[item.status] || { label: '未知', type: 'info' },
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
})

const auditActionMap = {
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
    createdAt: formatDateTime(item.createdAt),
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

const resetResubmitState = () => {
  detailDialog.resubmit.loading = false
}

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

  if (canResubmitApplication.value) {
    return
  }

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

const handleResubmitApplication = async () => {
  const current = detailDialog.data
  if (!current?.id) return

  const { keptFileIds: keptFileIDs, newFiles } = getAttachmentDiff()
  const formData = new FormData()
  formData.append('keptFileIDs', JSON.stringify(keptFileIDs))
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
    if (res.status === 200 && res.data.application) {
      const appData = res.data.application
      updateApplicationEntry(appData)

      // 重新提交后刷新时间轴
      const normalizedAudits = (appData.audits || []).map(item => normalizeAudit(item))
      detailDialog.timeline = normalizedAudits

      // 选中新的（最新的）提交记录（最后一条）
      if (detailDialog.timeline.length > 0) {
        handleAuditClick(detailDialog.timeline[detailDialog.timeline.length - 1])
      }

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
        <el-table-column label="申请的数据集">
          <template #default="scope">
            {{ scope.row.dataset.title || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="初次申请时间" width="180" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.statusMeta.type">{{ scope.row.statusMeta.label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link :icon="View" @click="handleViewDetail(scope.row)">
              查看详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-skeleton>
  </el-card>

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
              <el-descriptions-item label="修改次数">
                {{ detailDialog.data.version > 1 ? detailDialog.data.version - 1 : 0 }}
              </el-descriptions-item>
              <el-descriptions-item label="申请数据集">
                {{ detailDialog.data.dataset.title }}
                <el-tooltip :content="detailDialog.data.dataset.id" placement="top">
                  <span class="info-sub-text">
                    (ID: {{ formatShortId(detailDialog.data.dataset.id) }})
                  </span>
                </el-tooltip>
              </el-descriptions-item>
              <el-descriptions-item label="申请人">
                {{ detailDialog.data.applicant.nickname }}
                <span v-if="detailDialog.data.applicant.email" class="info-sub-text"
                  >({{ detailDialog.data.applicant.email }})</span
                >
              </el-descriptions-item>
              <el-descriptions-item label="初次申请时间">
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
                  <div v-if="canViewAttachments && !canResubmitApplication" class="attachment-section">
                    <div class="attachment-header">
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
                          <el-table-column label="大小" width="100">
                            <template #default="scope">{{ formatFileSize(scope.row.size) }}</template>
                          </el-table-column>
                          <el-table-column label="操作" width="80" align="center">
                            <template #default="scope">
                              <el-button
                                text
                                size="small"
                                :icon="Download"
                                @click="handleDownloadAttachment(scope.row)"
                              >
                                下载
                              </el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                      </template>
                    </el-skeleton>
                  </div>

                  <!-- 重新提交区域 -->
                  <div v-if="canResubmitApplication" class="resubmit-area">
                    <el-alert
                      type="warning"
                      :closable="false"
                      show-icon
                      title="申请已被退回，请修改后重新提交"
                    />
                    <div class="resubmit-content">
                      <div class="resubmit-block">
                        <div class="resubmit-block-title">申请附件</div>
                        <el-skeleton :rows="3" animated :loading="detailDialog.attachments.isLoading">
                          <template #default>
                            <el-empty
                              v-if="!detailDialog.attachments.list.length"
                              description="暂无附件"
                              :image-size="60"
                              class="empty-attachment"
                            />
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
                                min-width="150"
                                show-overflow-tooltip
                              />
                              <el-table-column label="状态" width="80">
                                <template #default="scope">
                                  <el-tag size="small" :type="getAttachmentStatusMeta(scope.row).type">
                                    {{ getAttachmentStatusMeta(scope.row).label }}
                                  </el-tag>
                                </template>
                              </el-table-column>
                              <el-table-column label="操作" width="120" align="center">
                                <template #default="scope">
                                  <el-space :size="4">
                                    <el-button
                                      text
                                      size="small"
                                      :icon="Download"
                                      @click="handleDownloadAttachment(scope.row)"
                                    />
                                    <el-button
                                      text
                                      :type="scope.row.isRemoved ? 'primary' : 'danger'"
                                      size="small"
                                      :icon="Delete"
                                      @click="handleRemoveAttachmentLocal(scope.row)"
                                    />
                                  </el-space>
                                </template>
                              </el-table-column>
                            </el-table>
                          </template>
                        </el-skeleton>
                        <div class="attachment-upload-area">
                          <el-upload
                            ref="resubmitUploadRef"
                            action=""
                            :show-file-list="false"
                            :auto-upload="false"
                            :disabled="
                              detailDialog.attachments.isLoading || detailDialog.resubmit.loading
                            "
                            :on-change="handleAttachmentFileQueued"
                          >
                            <el-button
                              type="primary"
                              text
                              :icon="Upload"
                              :loading="detailDialog.resubmit.loading"
                              >上传补充材料</el-button
                            >
                          </el-upload>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- 审批意见展示 -->
                  <div v-if="detailDialog.timeline.find(i => i.id === detailDialog.audit.selectedId)?.result" class="audit-result-display">
                    <div class="result-label">审批意见：</div>
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
          :loading="detailDialog.resubmit.loading"
          @click="handleResubmitApplication"
          >重新提交审核</el-button
        >
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.table-sub-text,
.info-sub-text {
  color: #909399;
  font-size: 12px;
}
.reason-block {
  white-space: pre-wrap;
  line-height: 1.6;
}
.attachment-section {
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
.dialog-footer {
  display: flex;
  gap: 12px;
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

.audit-result-display {
  padding: 12px;
  background-color: #f4f4f5;
  border-radius: 4px;
}

.result-label {
  font-weight: bold;
  margin-bottom: 4px;
  color: #303133;
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
</style>
