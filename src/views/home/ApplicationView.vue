<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject, watch, computed } from 'vue'
import { Check, Close, View, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import apiClient from '@/request/axios.js'
import { formatDateTime, formatFileSize } from '@/utils/format.js'

const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const approveApplicationAPI = (id, data) =>
  apiClient.post(`/api/v1/applications/${id}/approve`, data)
const rejectApplicationAPI = (id, data) => apiClient.post(`/api/v1/applications/${id}/reject`, data)
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
})

const statusMetaMap = {
  pending: { label: '待审批', type: 'warning' },
  approved: { label: '已批准', type: 'success' },
  disapproved: { label: '已驳回', type: 'danger' },
}

const getStatusMeta = (status) => statusMetaMap[status] || { label: '未知', type: 'info' }
const getActionButtonLabel = (row) => {
  if (isAdmin()) {
    return row.status === 'pending' ? '审核' : '结果详情'
  }
  return row.status === 'pending' ? '申请详情' : '结果详情'
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

const resetAttachmentState = () => {
  detailDialog.attachments.isLoading = false
  detailDialog.attachments.list = []
  detailDialog.attachments.currentApplicationId = ''
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

const handleDialogAudit = async (action) => {
  const current = detailDialog.data
  if (!current?.id) return
  const result = detailDialog.audit.result.trim()
  if (!result) {
    ElMessage.warning('请填写审核结果')
    return
  }
  detailDialog.audit.loading = true
  try {
    const api = action === 'approve' ? approveApplicationAPI : rejectApplicationAPI
    const res = await api(current.id, { result })
    if (res.data?.message === 'success' && res.data.application) {
      const normalized = normalizeApplication(res.data.application)
      updateApplicationEntry(res.data.application)
      detailDialog.audit.result = normalized.auditorResult
      ElMessage.success(action === 'approve' ? '申请已批准' : '申请已驳回')
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

watch(
  () => detailDialog.data,
  (value) => {
    detailDialog.audit.result = value?.auditorResult || ''
    if (!value) {
      resetAttachmentState()
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
            <el-button type="primary" text :icon="View" @click="handleViewDetail(scope.row)">
              {{ getActionButtonLabel(scope.row) }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-skeleton>
  </el-card>

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
</style>
