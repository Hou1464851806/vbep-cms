<script setup>
import { ref, reactive, onMounted, inject, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Warning,
  Refresh,
  Remove,
  TopRight,
  QuestionFilled,
} from '@element-plus/icons-vue'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatShortId } from '@/utils/format.js'
import ListPagination from "@/components/ListPagination.vue"
import { useTableFilter } from "@/hooks/useTable.js"
import TableToolsBar from "@/components/TableToolsBar.vue"

const router = useRouter()
const route = useRoute()
const isAdmin = inject('isAdmin')

// Normalize data
const normalizeLicense = (item) => {
  let statusText = item.status === 'normal' ? '正常' : '撤销'
  let statusType = item.status === 'normal' ? 'success' : 'danger'

  return {
    id: item.id,
    datasetTitle: item.dataset?.title || '未知数据集',
    applicantName: item.applicant?.nickname || '未知用户',
    applicationId: item.applicationId,
    createdAt: formatDateTime(item.createdAt),
    expireAt: item.expireAt ? formatDateTime(item.expireAt) : '永久',
    status: item.status,
    statusText,
    statusType,
    revokeReason: item.revokeReason || '无',
    raw: item,
  }
}

// Filter configuration
const filterConfig = [
  {
    key: 'keyword',
    type: 'input',
    label: '搜索',
    placeholder: '搜索数据集或用户...',
  },
  {
    key: 'status',
    type: 'select',
    label: '状态',
    options: [
      { label: '全部', value: '' },
      { label: '正常', value: 'normal' },
      { label: '撤销', value: 'revoked' },
    ],
  },
]

const {
  loading: isDataLoading,
  data: licenseList,
  pagination,
  filterForm,
  loadData: fetchLicenses,
  handleSortChange,
  handleFilterChange,
  handleFilterInputChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/licenses/', { params }),
  {
    normalize: normalizeLicense,
    dataKey: 'licenses',
    defaultSort: { prop: 'createdAt', order: 'descending' },
    filterSchema: { keyword: '', status: '', application_id: '' },
    pageSize: 10,
  }
)

onMounted(() => {
  if (route.query.keyword) {
    filterForm.keyword = route.query.keyword
  }
  if (route.query.status) {
    filterForm.status = route.query.status
  }
  // application_id is a special param for exact match from ApplicationView
  if (route.query.application_id) {
    filterForm.application_id = route.query.application_id
  }
  fetchLicenses()
})

// Dialogs
const revokeDialog = reactive({
  visible: false,
  loading: false,
  licenseId: '',
  reason: '',
})

const reasonDialog = reactive({
  visible: false,
  reason: '',
})

const handleSourceClick = (row) => {
  ElMessageBox.confirm(
    `您即将离开当前页，前往查看“申请${formatShortId(row.applicationId)}”的详情。确定吗？`,
    '提示',
    {
      confirmButtonText: '前往',
      cancelButtonText: '取消',
      type: 'info',
    }
  ).then(() => {
    router.push({ path: '/home/applications', query: { id: row.applicationId } })
  }).catch(() => {})
}

const handleRevokeClick = (row) => {
  revokeDialog.licenseId = row.id
  revokeDialog.reason = ''
  revokeDialog.visible = true
}

const submitRevoke = async () => {
  if (!revokeDialog.reason.trim()) {
    ElMessage.warning('请填写撤回理由')
    return
  }

  revokeDialog.loading = true
  try {
    const res = await apiClient.put(`/api/v1/licenses/${revokeDialog.licenseId}/revoke`, {
      reason: revokeDialog.reason
    })
    if (res.status === 200) {
      ElMessage.success('许可已撤回')
      revokeDialog.visible = false
      fetchLicenses()
    } else {
      ElMessage.error(res.data?.message || '操作失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '撤销许可失败')
  } finally {
    revokeDialog.loading = false
  }
}

// (View Reason logic removed as we use tooltip)

</script>

<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>许可管理</span>
          <div class="header-actions">
            <el-button :icon="Refresh" circle @click="fetchLicenses" :loading="isDataLoading" title="刷新列表" />
          </div>
        </div>
      </template>

      <div v-loading="isDataLoading">
        <table-tools-bar
          :model-value="filterForm"
          @update:model-value="(newVal) => Object.assign(filterForm, newVal)"
          :filters="filterConfig"
          @search="handleFilterChange"
          @input-change="handleFilterInputChange"
          @reset="resetFilter"
        />

        <el-table
          :data="licenseList"
          style="width: 100%"
          @sort-change="handleSortChange"
          :default-sort="{ prop: 'createdAt', order: 'descending' }"
        >
          <el-table-column label="内部ID" width="100">
            <template #default="{ row }">
              <el-tooltip effect="dark" :content="row.id" placement="top" :show-after="500">
                <span class="monospace-id">{{ formatShortId(row.id) }}</span>
              </el-tooltip>
            </template>
          </el-table-column>

          <el-table-column prop="datasetTitle" label="所属数据集" min-width="180" sortable="custom">
            <template #default="{ row }">
              <span style="font-weight: 500">{{ row.datasetTitle }}</span>
            </template>
          </el-table-column>

          <el-table-column v-if="isAdmin()" prop="applicantName" label="获得者" min-width="120" sortable="custom" />

          <el-table-column label="许可来源" min-width="130">
            <template #default="{ row }">
              <el-link type="primary" @click="handleSourceClick(row)" style="display: inline-flex; align-items: center; gap: 4px;">
                申请{{ formatShortId(row.applicationId) }}
                <el-icon><TopRight /></el-icon>
              </el-link>
            </template>
          </el-table-column>

          <el-table-column prop="createdAt" label="许可授权时间" width="160" sortable="custom" />
          <el-table-column prop="expireAt" label="预计过期时间" width="160" sortable="custom" />

          <el-table-column label="状态" width="120" align="center">
            <template #default="{ row }">
              <div style="display: inline-flex; align-items: center; gap: 4px;">
                <el-tag :type="row.statusType">{{ row.statusText }}</el-tag>
                <el-tooltip v-if="row.status === 'revoked'" effect="dark" :content="row.revokeReason" placement="top">
                  <el-icon style="cursor: help; color: var(--text-color-secondary, #909399); font-size: 16px;">
                    <QuestionFilled />
                  </el-icon>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>

          <el-table-column v-if="isAdmin()" label="操作" width="120" fixed="right" align="center">
            <template #default="scope">
              <el-button 
                type="danger" 
                link 
                :icon="Remove" 
                :disabled="scope.row.status === 'revoked'"
                @click="handleRevokeClick(scope.row)"
              >
                撤回
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <list-pagination
          v-model="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          @change="handlePageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="revokeDialog.visible" title="撤回许可" width="400px">
      <div style="margin-bottom: 15px; display: flex; align-items: center; color: #E6A23C;">
        <el-icon style="margin-right: 5px;"><Warning /></el-icon>
        <span>撤回后，用户将立即失去对该数据集的所有访问权限。</span>
      </div>
      <el-input
        v-model="revokeDialog.reason"
        type="textarea"
        :rows="4"
        placeholder="请输入撤回理由..."
      />
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="revokeDialog.visible = false">取消</el-button>
          <el-button type="danger" :loading="revokeDialog.loading" @click="submitRevoke">确认撤回</el-button>
        </span>
      </template>
    </el-dialog>

  </div>
</template>

<style scoped>
.monospace-id {
  font-family: var(--font-family-mono, monospace);
  color: var(--text-color-secondary, #909399);
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
