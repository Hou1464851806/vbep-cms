<script setup>
import { ref, onMounted, inject, reactive, computed } from 'vue'
import {Edit, Check, Close, Refresh, Document, Download} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 复用现有的 axios 实例
import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import {formatDateTime, formatFileSize, formatShortId} from '@/utils/format.js'
import ListPagination from "@/components/ListPagination.vue";
import { useTableFilter } from "@/hooks/useTable.js"
import TableToolsBar from "@/components/TableToolsBar.vue"

// --- API 定义 ---
// 获取所有用户列表 (假设后端支持该接口)
const fetchUserListAPI = (page,size) => apiClient.get('/api/v1/users', { params: { page, size } })
// 更新用户信息 (复用 HomeView 中的逻辑)
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)

const fetchUserFilesAPI = (userId, page, size) => apiClient.get(`/api/v1/users/${userId}/files`, { params: { page, size } })

// 获取通用文件下载
const downloadFileAPI = (fileID) => apiClient.get(`/api/v1/files/${fileID}/download`, { responseType: 'blob' })

// --- 状态定义 ---
const currentUser = inject('userInfo')
// const userList = ref([])
// const isLoading = ref(true)

// // 分页相关
// const pagination = reactive({
//   currentPage: 1,
//   pageSize: 10,
//   total: 0
// })

// 编辑状态管理：key为userId, value为正在编辑的副本数据
const editMap = reactive({})

// 权限常量映射
const ROLE_MAP = {
  0: { label: '普通用户', type: 'info', value: 0 },
  1: { label: '管理员', type: 'success', value: 1 },
  2: { label: '超级管理员', type: 'warning', value: 2 },
}

const OWNER_TYPE_MAP = {
  datasets: { label: '数据集文件', type: 'primary' },
  applications: { label: '申请附件', type: 'success' },
  public_downloads: { label: '公共下载文件', type: 'warning' }
}

// --- 计算属性 ---

// 动态判断当前登录用户是否为超级管理员 (Role === 2)
// 注意：HomeView 可能会把 role 格式化为 'admin'/'user' 字符串，
// 所以我们通过 ID 在完整的 userList 中查找自己的原始 role 值来判断最准确
const isSuperAdmin = computed(() => currentUser && currentUser.roleId === 2)
const isSimpleAdmin = computed(() => currentUser && currentUser.roleId === 1)

// --- 数据格式化 ---
const getRoleTag = (roleVal) => {
  return ROLE_MAP[roleVal] || { label: '未知', type: 'danger' }
}

const getOwnerTypeTag = (typeVal) => OWNER_TYPE_MAP[typeVal] || { label: typeVal || '未知', type: 'info' }

// // --- 数据加载 ---
// const fetchUsers = async () => {
//   isLoading.value = true
//   try {
//     const res = await fetchUserListAPI(pagination.currentPage, pagination.pageSize)
//     if (res && Array.isArray(res.data?.data)) {
//       // 直接赋值，后端 GORM 已经过滤了软删除用户，这里不需要再前端 filter
//       userList.value = res.data.data
//       pagination.total = res.data.total || 0
//     } else {
//       ElMessage.error(res?.data?.message || '用户列表加载失败')
//     }
//   } catch (error) {
//     console.error(error)
//     ElMessage.error('无法获取用户列表')
//   } finally {
//     isLoading.value = false
//   }
// }

const normalizeUser = (item) => ({
  userID: item.userID,
  nickname: item.nickname,
  email: item.email || '',
  role: item.role,
  createdAt: formatDateTime(item.createdAt),
})

// --- 筛选配置 ---
const roleFilterOptions = [
  { label: '普通用户', value: 0 },
  { label: '管理员', value: 1 },
  { label: '超级管理员', value: 2 },
]
const filterConfig = [
  {
    key: 'role',
    type: 'select',
    label: '权限组',
    placeholder: '全部角色',
    options: roleFilterOptions,
  },
  {
    key: 'keyword',
    type: 'input',
    label: '姓名/邮箱',
    placeholder: '搜索姓名或邮箱...',
  },
  {
    key: 'date',
    type: 'daterange',
    label: '注册时间',
  },
]
const {
  loading: isLoading,
  data: userList,
  pagination,
  filterForm,
  loadData: fetchUsers,
  handleSortChange,
  handleFilterChange,
  handleFilterInputChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/users', { params }),
  {
    normalize: normalizeUser,
    dataKey: 'data',
    defaultSort: { prop: 'createdAt', order: 'descending' },
    filterSchema: { role: '', keyword: '', date: [] },
    pageSize: 10,
  }
)

onMounted(fetchUsers)

// const handlePageChange = () => {
//   fetchUsers()
// }

// --- 编辑逻辑 ---

// 判断某一行是否处于编辑模式
const isEditing = (id) => !!editMap[id] // id here refers to userID

// 进入编辑模式
const handleEdit = (row) => {
  // 深拷贝当前行数据到 editMap，作为临时修改区
  editMap[row.userID] = { ...row }
}

// 取消编辑
const handleCancel = (id) => { // id is userID
  delete editMap[id]
}

// 保存修改
const handleSave = async (originalRow) => {
  const draft = editMap[originalRow.userID]
  if (!draft) return

  // 1. 简单校验
  if (!draft.nickname) {
    ElMessage.warning('昵称不能为空')
    return
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (draft.email && !emailRegex.test(draft.email)) {
    ElMessage.warning('请输入有效的邮箱地址')
    return
  }

  // 2. Diff Check (检查是否有实质性修改)
  const changes = {}
  if (draft.nickname !== originalRow.nickname) changes.nickname = draft.nickname
  if (draft.email !== originalRow.email) changes.email = draft.email
  // if (draft.phone !== originalRow.phone) changes.phone = draft.phone

  // 仅超级管理员可以修改权限组
  if (isSuperAdmin.value && Number(draft.role) !== Number(originalRow.role)) {
    changes.role = Number(draft.role)
  }

  // 如果没有变化，直接退出编辑模式，不发送请求
  if (Object.keys(changes).length === 0) {
    handleCancel(originalRow.userID)
    ElMessage.info('未检测到数据变更')
    return
  }

  // 3. 发送 API 请求
  try {
    const res = await updateUserAPI(originalRow.userID, changes)
    if (res.status === 200) {
      ElMessage.success('用户信息更新成功')
      // 更新本地视图数据
      Object.assign(originalRow, changes)
      // 退出编辑
      handleCancel(originalRow.userID)
    } else {
      ElMessage.error(res.data?.message || '更新失败')
    }
  } catch (error) {
    const msg = error.response?.data?.message || '更新异常'
    ElMessage.error(msg)
  }
}

// 查看用户文件列表关联逻辑
const filesDialogVisible = ref(false)
const currentSelectedUser = ref(null)
const userFiles = ref([])
const filesLoading = ref(false)
const filesPagination = reactive({ currentPage: 1, pageSize: 10, total: 0 })
const handleViewFiles = (user) => {
  currentSelectedUser.value = user
  filesDialogVisible.value = true
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
      userFiles.value = res.data.data || []
      filesPagination.total = res.data.total || 0
    }
  } catch (error) {
    ElMessage.error('无法获取该用户的文件列表')
    console.error(error)
  } finally {
    filesLoading.value = false
  }
}
const handleFilesPageChange = () => {
  loadUserFiles()
}

// --- 下载文件逻辑 ---
const isDownloading = ref(false)
const handleDownloadFile = async (row) => {
  isDownloading.value = true
  try {
    const res = await downloadFileAPI(row.id)
    if (!res || !res.data) {
      ElMessage.error('下载失败：未获取到文件数据')
      return
    }

    // Blob 转换与触发下载
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
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
          <el-button :icon="Refresh" circle @click="fetchUsers" :loading="isLoading" title="刷新列表" />
        </div>
      </div>
    </template>

    <div v-loading="isLoading">
      <!-- 筛选栏 -->
      <table-tools-bar
        :model-value="filterForm"
        @update:model-value="(newVal) => Object.assign(filterForm, newVal)"
        :filters="filterConfig"
        @search="handleFilterChange"
        @input-change="handleFilterInputChange"
        @reset="resetFilter"
      />
      <el-table
        :data="userList"
        style="width: 100%"
        @sort-change="handleSortChange"
        :default-sort="{ prop: 'createdAt', order: 'descending' }"
      >
        <el-table-column label="内部ID" width="100">
          <template #default="{ row }">
            <el-tooltip effect="dark" :content="row.userID" placement="top" :show-after="500">
              <span class="monospace-id">
                {{ formatShortId(row.userID) }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>

        <el-table-column label="姓名" min-width="120" prop="nickname" sortable="custom">
          <template #default="{ row }">
            <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].nickname" size="small" />
            <span v-else>{{ row.nickname }}</span>
          </template>
        </el-table-column>

        <el-table-column label="邮箱" min-width="180" prop="email" sortable="custom">
          <template #default="{ row }">
            <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].email" size="small" />
            <span v-else>{{ row.email || '-' }}</span>
          </template>
        </el-table-column>

  <!--      <el-table-column label="手机号" width="140">-->
  <!--        <template #default="{ row }">-->
  <!--          <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].phone" size="small" />-->
  <!--          <span v-else>{{ row.phone || '-' }}</span>-->
  <!--        </template>-->
  <!--      </el-table-column>-->

        <el-table-column label="注册时间" width="180" prop="createdAt" sortable="custom">
          <template #default="{ row }">
            {{ formatDateTime(row.createdAt) }}
          </template>
        </el-table-column>

        <el-table-column label="权限组" width="140" align="center" prop="role" sortable="custom">
          <template #default="{ row }">
            <el-select
              v-if="isEditing(row.userID) && isSuperAdmin"
              v-model="editMap[row.userID].role"
              size="small"
            >
              <el-option label="普通用户" :value="0" />
              <el-option label="管理员" :value="1" />
              <el-option label="超级管理员" :value="2" />
            </el-select>

            <el-tag v-else :type="getRoleTag(row.role).type">
              {{ getRoleTag(row.role).label }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column v-if="isSuperAdmin" label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <template v-if="isEditing(row.userID)">
              <el-button type="success" link :icon="Check" @click="handleSave(row)">确定</el-button>
              <el-button type="info" link :icon="Close" @click="handleCancel(row.userID)">取消</el-button>
            </template>

            <template v-else>
              <el-button v-if="isSuperAdmin" type="primary" link :icon="Edit" @click="handleEdit(row)">修改</el-button>
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
    <el-table
      :data="userFiles"
      v-loading="filesLoading"
      height="400"
      style="width: 100%"
    >
      <el-table-column prop="name" label="文件名" min-width="180" show-overflow-tooltip />

<!--      <el-table-column label="文件类型" width="120" prop="Type" align="center">-->
<!--        <template #default="{ row }">-->
<!--          <el-tag :type="row.Type ? 'info' : ''" size="small">{{ row.Type || '未分类' }}</el-tag>-->
<!--        </template>-->
<!--      </el-table-column>-->
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
        <template #default="{ row }">
          {{ formatFileSize(row.size) }}
        </template>
      </el-table-column>
      <el-table-column label="上传时间" width="160" align="center">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
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
    <!-- 文件列表的分页 -->
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
