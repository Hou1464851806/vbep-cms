<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, reactive, onMounted, inject } from 'vue'
import { Check, Close, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import apiClient from '../../request/axios.js'

const fetchApplicationListAPI = () => apiClient.get('/api/v1/applications/')
const fetchApplicationDetailAPI = (id) => apiClient.get(`/api/v1/applications/${id}`)
const approveApplicationAPI = (id) => apiClient.post(`/api/v1/applications/${id}/approve`)
const rejectApplicationAPI = (id) => apiClient.post(`/api/v1/applications/${id}/reject`)

const isAdmin = inject('isAdmin')
const applicationList = ref([])
const isLoading = ref(true)
const actionLoadingId = ref(null)
const detailDialog = reactive({
  visible: false,
  loading: false,
  data: null,
})

const statusMetaMap = {
  pending: { label: '待审批', type: 'warning' },
  approved: { label: '已批准', type: 'success' },
  disapproved: { label: '已驳回', type: 'danger' },
}

const getStatusMeta = (status) => statusMetaMap[status] || { label: '未知', type: 'info' }
const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
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
  createdAt: formatDateTime(item.createdAt),
  auditedAt: formatDateTime(item.auditedAt),
})

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

const processApplicationAction = async (row, action) => {
  if (!row?.id) return
  actionLoadingId.value = row.id
  try {
    const api = action === 'approve' ? approveApplicationAPI : rejectApplicationAPI
    const res = await api(row.id)
    if (res.data?.message === 'success' && res.data.application) {
      updateApplicationEntry(res.data.application)
      ElMessage.success(action === 'approve' ? '申请已批准' : '申请已驳回')
    } else {
      ElMessage.error(res.data?.message || '操作失败')
    }
  } catch (error) {
    console.error('Failed to process application', error)
    ElMessage.error('操作失败')
  } finally {
    actionLoadingId.value = null
  }
}

const handleApprove = (row) => processApplicationAction(row, 'approve')
const handleReject = (row) => processApplicationAction(row, 'reject')

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
            <small class="table-sub-text">{{ scope.row.applicant.email }}</small>
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
        <el-table-column label="操作" width="220">
          <template #default="scope">
            <el-button type="primary" text :icon="View" @click="handleViewDetail(scope.row)"
              >查看详情</el-button
            >
            <template v-if="isAdmin() && scope.row.status === 'pending'">
              <el-button
                size="small"
                type="success"
                :icon="Check"
                :loading="actionLoadingId === scope.row.id"
                @click="handleApprove(scope.row)"
                >批准</el-button
              >
              <el-button
                size="small"
                type="danger"
                :icon="Close"
                :loading="actionLoadingId === scope.row.id"
                @click="handleReject(scope.row)"
                >驳回</el-button
              >
            </template>
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
            <span class="table-sub-text">({{ detailDialog.data.applicant.email }})</span>
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
              <span class="table-sub-text">({{ detailDialog.data.auditor.email }})</span>
            </template>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
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
</style>
