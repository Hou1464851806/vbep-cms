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
import TableHeaderFilter from "@/components/TableHeaderFilter.vue"
import IdCell from "@/components/IdCell.vue"

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

const statusFilterOptions = [
  { label: '正常', value: 'normal' },
  { label: '撤销', value: 'revoked' },
]

const {
  loading: isDataLoading,
  data: licenseList,
  pagination,
  filterForm,
  sortState,
  isFilterActive,
  loadData: fetchLicenses,
  handleSortChange,
  handleFilterChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/licenses/', { params }),
  {
    normalize: normalizeLicense,
    dataKey: 'licenses',
    defaultSort: { prop: 'createdAt', order: 'descending' },
    filterSchema: { id_prefix: '', dataset_name: '', applicant: '', application_id: '', status: '', date: [], expire_date: [] },
    pageSize: 10,
    syncUrl: true,
  }
)

onMounted(() => {
  fetchLicenses()
})

// Dialogs
const revokeDialog = reactive({
  visible: false,
  loading: false,
  licenseId: '',
  reason: '',
})

const handleSourceClick = (row) => {
  ElMessageBox.confirm(
    `您即将离开当前页，前往查看"申请${formatShortId(row.applicationId)}"的详情。确定吗？`,
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
</script>

<template>
  <div>
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>许可管理</span>
          <div class="header-actions">
            <el-button
              v-if="isFilterActive"
              class="reset-filter-btn"
              plain
              size="small"
              @click="resetFilter"
            >重置筛选</el-button>
            <el-button :icon="Refresh" circle @click="fetchLicenses" :loading="isDataLoading" title="刷新列表" />
          </div>
        </div>
      </template>

      <div v-loading="isDataLoading">
        <el-table
          :data="licenseList"
          style="width: 100%"
        >
          <!-- 许可 ID -->
          <el-table-column label="许可ID" width="110">
            <template #header>
              <table-header-filter
                label="许可ID"
                type="id-search"
                v-model="filterForm.id_prefix"
                @change="handleFilterChange"
              />
            </template>
            <template #default="{ row }">
              <id-cell :id="row.id" />
            </template>
          </el-table-column>

          <!-- 所属数据集 -->
          <el-table-column label="所属数据集" min-width="180">
            <template #header>
              <table-header-filter
                label="所属数据集"
                type="input"
                v-model="filterForm.dataset_name"
                @change="handleFilterChange"
                sort-prop="datasetTitle"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">
              <span class="table-name-text">{{ row.datasetTitle }}</span>
            </template>
          </el-table-column>

          <!-- 获得者（管理员可见） -->
          <el-table-column v-if="isAdmin()" label="获得者" min-width="110">
            <template #header>
              <table-header-filter
                label="获得者"
                type="input"
                v-model="filterForm.applicant"
                @change="handleFilterChange"
                sort-prop="applicantName"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">{{ row.applicantName }}</template>
          </el-table-column>

          <!-- 许可来源（申请 ID）-->
          <el-table-column label="来源申请" min-width="140">
            <template #header>
              <table-header-filter
                label="来源申请"
                type="id-search"
                v-model="filterForm.application_id"
                @change="handleFilterChange"
              />
            </template>
            <template #default="{ row }">
              <el-link type="primary" @click="handleSourceClick(row)" style="display: inline-flex; align-items: center; gap: 4px;">
                申请{{ formatShortId(row.applicationId) }}
                <el-icon><TopRight /></el-icon>
              </el-link>
            </template>
          </el-table-column>

          <!-- 许可授权时间 -->
          <el-table-column label="许可授权时间" width="160">
            <template #header>
              <table-header-filter
                label="许可授权时间"
                type="daterange"
                v-model="filterForm.date"
                @change="handleFilterChange"
                sort-prop="createdAt"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">
              <span class="table-time-text">{{ row.createdAt }}</span>
            </template>
          </el-table-column>

          <!-- 预计过期时间 -->
          <el-table-column label="预计过期时间" width="160">
            <template #header>
              <table-header-filter
                label="预计过期时间"
                type="daterange"
                v-model="filterForm.expire_date"
                @change="handleFilterChange"
                sort-prop="expireAt"
                :sort-state="sortState"
                @sort="handleSortChange"
              />
            </template>
            <template #default="{ row }">
              <span class="table-time-text">{{ row.expireAt }}</span>
            </template>
          </el-table-column>

          <!-- 状态 -->
          <el-table-column label="状态" width="110">
            <template #header>
              <table-header-filter
                label="状态"
                type="select"
                v-model="filterForm.status"
                :options="statusFilterOptions"
                @change="handleFilterChange"
              />
            </template>
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

          <!-- 操作（管理员） -->
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
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

