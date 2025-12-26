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
  Aim,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import apiClient from '@/request/axios.js'
import { formatDateTime, formatFileSize } from '@/utils/format.js'

const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const approveApplicationAPI = (id, data) =>
  apiClient.post(`/api/v1/applications/${id}/approve`, data)
const rejectApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/reject`, data)
const returnApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/return`, data)
const resubmitApplicationAPI = (id, data) =>
  apiClient.post(`/api/v1/applications/${id}/resubmit`, data)
const fetchApplicationAuditsAPI = (id) => apiClient.get(`/api/v1/applications/${id}/audits`)
const fetchApplicationFilesAPI = (id) => apiClient.get(`/api/v1/applications/${id}/files`)
const downloadApplicationFileAPI = (applicationId, fileId) =>
  apiClient.get(`/api/v1/applications/${applicationId}/files/${fileId}`, { responseType: 'blob' })

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
    currentApplicationId: '',
  },
  resubmit: {
    reason: '',
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
  reason: item.reason,
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
    actionDetail: '审批通过',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  reject: {
    label: '管理员审批',
    color: '#409EFF',
    operatorLabel: '审批人',
    actionDetail: '审批驳回',
    resultLabel: '审批理由',
    showActionDetail: true,
  },
  return: {
    label: '管理员审批',
    color: '#409EFF',
    operatorLabel: '审批人',
    actionDetail: '退回修改',
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
  detailDialog.attachments.currentApplicationId = ''
}

const resetAuditDialog = () => {
  auditDialog.loading = false
  auditDialog.list = []
  auditDialog.title = ''
  auditDialog.currentApplicationId = ''
  auditDialog.currentStatus = ''
}

const resetResubmitState = () => {
  detailDialog.resubmit.reason = ''
  detailDialog.resubmit.loading = false
}

const loadApplicationFiles = async (applicationId) => {
  if (!applicationId) return
  detailDialog.attachments.isLoading = true
  detailDialog.attachments.list = []
  try {
    const res = await fetchApplicationFilesAPI(applicationId)
    if (res?.data?.message === 'success') {
      detailDialog.attachments.list = normalizeApplicationFiles(res.data.files)
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

const handleDownloadAttachment = async (file) => {
  const applicationId = detailDialog.attachments.currentApplicationId
  if (!applicationId) {
    ElMessage.error('未找到对应的申请')
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
  detailDialog.resubmit.reason = ''
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
    ElMessage.warning('请填写审核结果')
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
  const reason = detailDialog.resubmit.reason.trim()
  if (!reason) {
    ElMessage.warning('请填写修改后的申请理由')
    return
  }
  detailDialog.resubmit.loading = true
  try {
    const res = await resubmitApplicationAPI(current.id, { reason })
    if (res.data?.message === 'success' && res.data.application) {
      updateApplicationEntry(res.data.application)
      detailDialog.resubmit.reason = reason
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
    detailDialog.resubmit.reason = value?.reason || ''
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
      <span>数据集申请列表</span>
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
              <template v-if="item.isPlaceholder || item.isFinal">
                <div class="audit-step-placeholder">
                  {{ item.placeholderText || '等待处理' }}
                </div>
              </template>
              <template v-else>
                <div class="audit-step-operator">
                  <template v-if="item.operator">
                    {{ item.operatorLabel }}：{{ item.operator.nickname }}
                    <span v-if="item.operator.email" class="table-sub-text"
                      >({{ item.operator.email }})</span
                    >
                  </template>
                  <template v-else>{{ item.operatorLabel }}：系统</template>
                </div>
                <div v-if="item.showActionDetail" class="audit-step-action">
                  操作类型：{{ item.actionDetail }}
                </div>
                <div v-if="item.result" class="audit-step-result">
                  {{ item.resultLabel }}：{{ item.result }}
                </div>
              </template>
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
          <el-descriptions-item label="申请理由">
            <div class="reason-block">{{ detailDialog.data.reason }}</div>
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
          <el-descriptions-item label="审核结果">
            <div v-if="detailDialog.data.auditorResult">
              {{ detailDialog.data.auditorResult }}
            </div>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
        <div v-if="canViewAttachments" class="attachment-section">
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
        <div v-if="canResubmitApplication" class="resubmit-area">
          <el-alert
            type="warning"
            :closable="false"
            show-icon
            title="申请已被退回，请根据意见修改信息后重新提交"
          />
          <el-input
            v-model="detailDialog.resubmit.reason"
            type="textarea"
            :rows="4"
            maxlength="200"
            show-word-limit
            placeholder="请填写修改后的申请理由"
          />
          <div class="resubmit-buttons">
            <el-button
              type="primary"
              :loading="detailDialog.resubmit.loading"
              @click="handleResubmitApplication"
              >重新提交审核</el-button
            >
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
      <el-button @click="detailDialog.visible = false">关闭</el-button>
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
.attachment-table {
  margin-top: 8px;
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
.resubmit-buttons {
  display: flex;
  justify-content: flex-end;
}
.audit-log-steps {
  max-height: 420px;
  padding-right: 8px;
  overflow-y: auto;
}
.audit-step {
  display: flex;
  flex-direction: column;
  gap: 4px;
  line-height: 1.6;
}
.audit-step-label {
  font-weight: 600;
  color: #303133;
}
.audit-step-operator {
  color: #909399;
  font-size: 13px;
}
.audit-step-action {
  color: #606266;
}
.audit-step-result {
  color: #606266;
}
.audit-step-placeholder {
  color: #909399;
  font-size: 13px;
}
</style>
