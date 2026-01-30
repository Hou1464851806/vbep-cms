<script setup>
import { ref, onMounted, inject, reactive, computed } from 'vue'
import { Edit, Check, Close, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 复用现有的 axios 实例
import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatShortId } from '@/utils/format.js'

// --- API 定义 ---
// 获取所有用户列表 (假设后端支持该接口)
const fetchUserListAPI = () => apiClient.get('/api/v1/users')
// 更新用户信息 (复用 HomeView 中的逻辑)
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)

// --- 状态定义 ---
const currentUser = inject('userInfo')
const userList = ref([])
const isLoading = ref(true)

// 编辑状态管理：key为userId, value为正在编辑的副本数据
const editMap = reactive({})

// 权限常量映射
const ROLE_MAP = {
  0: { label: '普通用户', type: 'info', value: 0 },
  1: { label: '管理员', type: 'success', value: 1 },
  2: { label: '超级管理员', type: 'warning', value: 2 },
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

// --- 数据加载 ---
const fetchUsers = async () => {
  isLoading.value = true
  try {
    const res = await fetchUserListAPI()
    if (res && Array.isArray(res.data?.data)) {
      // 直接赋值，后端 GORM 已经过滤了软删除用户，这里不需要再前端 filter
      userList.value = res.data.data
    } else {
      ElMessage.error(res?.data?.message || '用户列表加载失败')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('无法获取用户列表')
  } finally {
    isLoading.value = false
  }
}

onMounted(fetchUsers)

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
  if (draft.phone !== originalRow.phone) changes.phone = draft.phone

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

    <el-table :data="userList" style="width: 100%" v-loading="isLoading">
      <el-table-column label="内部ID" width="100">
        <template #default="{ row }">
          <el-tooltip effect="dark" :content="row.userID" placement="top" :show-after="500">
            <span class="monospace-id">
              {{ formatShortId(row.userID) }}
            </span>
          </el-tooltip>
        </template>
      </el-table-column>

      <el-table-column label="姓名" min-width="120">
        <template #default="{ row }">
          <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].nickname" size="small" />
          <span v-else>{{ row.nickname }}</span>
        </template>
      </el-table-column>

      <el-table-column label="邮箱" min-width="180">
        <template #default="{ row }">
          <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].email" size="small" />
          <span v-else>{{ row.email || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="手机号" width="140">
        <template #default="{ row }">
          <el-input v-if="isEditing(row.userID)" v-model="editMap[row.userID].phone" size="small" />
          <span v-else>{{ row.phone || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="注册时间" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.createdAt) }}
        </template>
      </el-table-column>

      <el-table-column label="权限组" width="140" align="center">
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
          </template>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
</style>
