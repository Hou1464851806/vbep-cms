<script setup>
import { ref, onMounted, inject, reactive, computed } from 'vue'
import { Edit, Check, Close, Refresh, Document, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize, formatShortId } from '@/utils/format.js'
import ListPagination from '@/components/ListPagination.vue'
import { useTableFilter } from '@/hooks/useTable.js'
import TableHeaderFilter from '@/components/TableHeaderFilter.vue'
import IdCell from '@/components/IdCell.vue'

// --- API 定义 ---
const fetchUserListAPI = (page, size) => apiClient.get('/api/v1/users', { params: { page, size } })
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)
const fetchUserFilesAPI = (userId, page, size) =>
  apiClient.get(`/api/v1/users/${userId}/files`, { params: { page, size } })
const downloadFileAPI = (fileID) =>
  apiClient.get(`/api/v1/files/${fileID}/download`, { responseType: 'blob' })

// --- 状态定义 ---
const currentUser = inject('userInfo')
const editMap = reactive({})

// 权限常量映射
const ROLE_MAP = {
  0: { label: '普通用户', type: 'info',    value: 0 },
  1: { label: '管理员',   type: 'success', value: 1 },
  2: { label: '超级管理员', type: 'warning', value: 2 },
}

const OWNER_TYPE_MAP = {
  datasets:         { label: '数据集文件',   type: 'primary' },
  applications:     { label: '申请附件',     type: 'success' },
  public_downloads: { label: '公共下载文件', type: 'warning' },
}

const isSuperAdmin = computed(() => currentUser && currentUser.roleId === 2)
const isSimpleAdmin = computed(() => currentUser && currentUser.roleId === 1)

const getRoleTag     = (v) => ROLE_MAP[v]       || { label: '未知', type: 'danger' }
const getOwnerTypeTag = (v) => OWNER_TYPE_MAP[v] || { label: v || '未知', type: 'info' }

// --- 筛选选项 ---
const roleFilterOptions = [
  { label: '普通用户',   value: 0 },
  { label: '管理员',     value: 1 },
  { label: '超级管理员', value: 2 },
]

const normalizeUser = (item) => ({
  userID:    item.userID,
  nickname:  item.nickname,
  email:     item.email || '',
  role:      item.role,
  createdAt: formatDateTime(item.createdAt),
})

const {
  loading:  isLoading,
  data:     userList,
  pagination,
  filterForm,
  sortState,
  isFilterActive,
  loadData: fetchUsers,
  handleSortChange,
  handleFilterChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/users', { params }),
  {
    normalize:    normalizeUser,
    dataKey:      'data',
    defaultSort:  { prop: 'createdAt', order: 'descending' },
    filterSchema: { role: '', keyword: '', date: [], id_prefix: '' },
    pageSize:     10,
  }
)

onMounted(fetchUsers)

// --- 编辑逻辑 ---
const isEditing = (id) => !!editMap[id]

const handleEdit = (row) => {
  editMap[row.userID] = { ...row }
}

const handleCancel = (id) => {
  delete editMap[id]
}

const handleSave = async (originalRow) => {
  const draft = editMap[originalRow.userID]
  if (!draft) return

  if (!draft.nickname) { ElMessage.warning('昵称不能为空'); return }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (draft.email && !emailRegex.test(draft.email)) {
    ElMessage.warning('请输入有效的邮箱地址'); return
  }

  const changes = {}
  if (draft.nickname !== originalRow.nickname) changes.nickname = draft.nickname
  if (draft.email    !== originalRow.email)    changes.email    = draft.email
  if (isSuperAdmin.value && Number(draft.role) !== Number(originalRow.role)) {
    changes.role = Number(draft.role)
  }

  if (Object.keys(changes).length === 0) {
    handleCancel(originalRow.userID)
    ElMessage.info('未检测到数据变更')
    return
  }

  try {
    const res = await updateUserAPI(originalRow.userID, changes)
    if (res.status === 200) {
      ElMessage.success('用户信息更新成功')
      Object.assign(originalRow, changes)
      handleCancel(originalRow.userID)
    } else {
      ElMessage.error(res.data?.message || '更新失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '更新异常')
  }
}

// --- 文件弹窗 ---
const filesDialogVisible    = ref(false)
const currentSelectedUser   = ref(null)
const userFiles             = ref([])
const filesLoading          = ref(false)
const filesPagination       = reactive({ currentPage: 1, pageSize: 10, total: 0 })

const handleViewFiles = (user) => {
  currentSelectedUser.value = user
  filesDialogVisible.value  = true
  filesPagination.currentPage = 1
  loadUserFiles()
}

const loadUserFiles = async () => {
  if (!currentSelectedUser.value) return
  filesLoading.value = true
  try {
    const res = await fetchUserFilesAPI(
      currentSelectedUser.value.userID,
      filesPagination.currentPage,
      filesPagination.pageSize
    )
    if (res && res.data) {
      userFiles.value          = res.data.data  || []
      filesPagination.total    = res.data.total  || 0
    }
  } catch (error) {
    ElMessage.error('无法获取该用户的文件列表')
    console.error(error)
  } finally {
    filesLoading.value = false
  }
}

const handleFilesPageChange = () => loadUserFiles()

// --- 下载 ---
const isDownloading = ref(false)
const handleDownloadFile = async (row) => {
  isDownloading.value = true
  try {
    const res = await downloadFileAPI(row.id)
    if (!res || !res.data) { ElMessage.error('下载失败：未获取到文件数据'); return }
    const url  = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href  = url
    link.setAttribute('download', row.name || '已下载文件')
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error(error)
    ElMessage.error('下载出错，可能是文件不存在或权限不足')
  } finally {
    isDownloading.value = false
  }
}
</script>

<template>
  <el-alert
    v-if="isSimpleAdmin"
    title="目前您仅可浏览用户信息，只有超级管理员方可修改"
    type="warning"
    show-icon
    style="margin-bottom: 20px"
  />

  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>用户列表</span>
        <div class="header-actions">
          <el-button
            v-if="isFilterActive"
            class="reset-filter-btn"
            plain
            size="small"
            @click="resetFilter"
          >重置筛选</el-button>
          <el-button :icon="Refresh" circle @click="fetchUsers" :loading="isLoading" title="刷新列表" />
        </div>
      </div>
    </template>

    <div v-loading="isLoading">
      <el-table
        :data="userList"
        style="width: 100%"
      >
        <!-- 用户 ID -->
        <el-table-column label="内部ID" width="110">
          <template #header>
            <table-header-filter
              label="内部ID"
              type="id-search"
              v-model="filterForm.id_prefix"
              @change="handleFilterChange"
            />
          </template>
          <template #default="{ row }">
            <id-cell :id="row.userID" />
          </template>
        </el-table-column>

        <!-- 姓名 -->
        <el-table-column label="姓名" min-width="120">
          <template #header>
            <table-header-filter
              label="姓名"
              type="input"
              v-model="filterForm.keyword"
              @change="handleFilterChange"
              sort-prop="nickname"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">
            <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].nickname" size="small" />
            <span v-else class="table-name-text">{{ row.nickname }}</span>
          </template>
        </el-table-column>

        <!-- 邮箱 -->
        <el-table-column label="邮箱" min-width="180">
          <template #header>
            <table-header-filter
              label="邮箱"
              sort-prop="email"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">
            <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].email" size="small" />
            <span v-else class="sub-text">{{ row.email || '-' }}</span>
          </template>
        </el-table-column>

        <!-- 注册时间 -->
        <el-table-column label="注册时间" width="180">
          <template #header>
            <table-header-filter
              label="注册时间"
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

        <!-- 权限组 -->
        <el-table-column label="权限组" width="140" align="center">
          <template #header>
            <table-header-filter
              label="权限组"
              type="select"
              v-model="filterForm.role"
              :options="roleFilterOptions"
              clear-label="全部角色"
              @change="handleFilterChange"
            />
          </template>
          <template #default="{ row }">
            <el-select
              v-if="isEditing(row.userID) && isSuperAdmin"
              v-model="editMap[row.userID].role"
              size="small"
            >
              <el-option label="普通用户"   :value="0" />
              <el-option label="管理员"     :value="1" />
              <el-option label="超级管理员" :value="2" />
            </el-select>
            <el-tag v-else :type="getRoleTag(row.role).type" disable-transitions>
              {{ getRoleTag(row.role).label }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- 操作 -->
        <el-table-column v-if="isSuperAdmin" label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <template v-if="isEditing(row.userID)">
              <el-button type="success" link :icon="Check"  @click="handleSave(row)">确定</el-button>
              <el-button type="info"    link :icon="Close"  @click="handleCancel(row.userID)">取消</el-button>
            </template>
            <template v-else>
              <el-button v-if="isSuperAdmin" type="primary" link :icon="Edit"     @click="handleEdit(row)">修改</el-button>
              <el-button v-if="isSuperAdmin" type="primary" link :icon="Document" @click="handleViewFiles(row)">文件</el-button>
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
    </div>
  </el-card>

  <!-- 用户文件列表弹窗 -->
  <el-dialog
    v-model="filesDialogVisible"
    :title="`文件列表 - ${currentSelectedUser?.nickname || ''}`"
    width="75%"
    destroy-on-close
  >
    <el-table :data="userFiles" v-loading="filesLoading" height="400" style="width: 100%">
      <el-table-column prop="name" label="文件名" min-width="180" show-overflow-tooltip />
      <el-table-column label="文件类型" width="160" align="center">
        <template #default="{ row }">
          <el-tag :type="getOwnerTypeTag(row.owner_type).type">
            {{ getOwnerTypeTag(row.owner_type).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="md5" label="MD5" width="120" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="monospace-id">{{ formatShortId(row.md5) || row.md5 }}</span>
        </template>
      </el-table-column>
      <el-table-column label="大小" width="100" align="right">
        <template #default="{ row }">{{ formatFileSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="上传时间" width="160" align="center">
        <template #default="{ row }">
          <span class="table-time-text">{{ formatDateTime(row.created_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" align="center" fixed="right">
        <template #default="{ row }">
          <el-button
            type="primary"
            link
            :icon="Download"
            :disabled="isDownloading"
            @click="handleDownloadFile(row)"
          >
            下载
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div style="margin-top: 15px; display: flex; justify-content: flex-end;">
      <el-pagination
        v-model:current-page="filesPagination.currentPage"
        v-model:page-size="filesPagination.pageSize"
        :total="filesPagination.total"
        layout="total, prev, pager, next"
        @current-change="handleFilesPageChange"
      />
    </div>
  </el-dialog>
</template>

<style scoped>
</style>
