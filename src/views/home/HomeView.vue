<script setup>
import { ref, reactive, onMounted, provide, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  Tickets,
  Memo,
  User,
  ArrowDown,
  Edit,
  SwitchButton,
  Key,
  Check,
  Close,
  Odometer,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 引入 axios 实例和清除 token 的方法
import apiClient, { clearAuthToken } from '@/request/axios.js'

// --- API 定义 ---
const fetchUserInfoAPI = (userId) => apiClient.get(`/api/v1/users/${userId}`)
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)
const logoutAPI = () => apiClient.get('/api/v1/users/logout')
const deleteUserAPI = (userId) => apiClient.delete(`/api/v1/users/${userId}`)

// --- 基础状态 ---
const router = useRouter()
const route = useRoute()
const userInfo = reactive({
  id: '',
  nickname: '',
  email: '',
  phone: '',
  // role: 'user',
  roleId: 0, // 0 - 'user', 1 - 'admin', 2 - 'super_admin'
})
const isUserLoading = ref(true)

// --- 初始化逻辑 ---
onMounted(async () => {
  isUserLoading.value = true
  try {
    const storedUserId = localStorage.getItem('user_id')
    if (!storedUserId) {
      ElMessage.error('无法获取用户信息，请重新登录')
      router.push('/login')
      return
    }
    const res = await fetchUserInfoAPI(storedUserId)
    if (res && res.data?.message === 'success' && res.data.data) {
      const { id, nickname, email, phone, role } = res.data.data
      const roleNum = Number(role)
      // let roleStr = 'user'
      // if (roleNum === 1) roleStr = 'admin'
      // if (roleNum === 2) roleStr = 'super_admin'

      Object.assign(userInfo, {
        id,
        nickname,
        email,
        phone,
        // role: roleStr,
        roleId: roleNum,
      })
    } else {
      ElMessage.error('用户信息异常')
    }
  } catch (error) {
    console.log(error)
    ElMessage.error('无法获取用户信息')
  } finally {
    isUserLoading.value = false
  }
})

provide('userInfo', userInfo)
// provide('isAdmin', () => ['admin', 'super_admin'].includes(userInfo.role))
provide('isAdmin', () => userInfo.roleId > 0)

// --- 登出逻辑 ---
const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗?', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    try {
      await logoutAPI()
    } catch (e) {
      console.warn('后端登出接口调用失败', e)
    }

    clearAuthToken()
    localStorage.removeItem('user_id')
    ElMessage.success('已安全退出')
    router.push('/login')
  } catch (e) {
    // Cancelled
    console.log(e)
  }
}

// --- 菜单指令 ---
const handleCommand = (command) => {
  if (command === 'logout') handleLogout()
  else if (command === 'profile') openProfileDialog()
  else if (command === 'password') openPasswordDialog()
}

// ================= 个人信息逻辑 (行内编辑) =================
const profileDialogVisible = ref(false)

// 用于控制每一行的编辑状态
const editState = reactive({
  nickname: false,
  email: false,
  phone: false,
})

// 临时存储正在编辑的值
const tempProfile = reactive({
  nickname: '',
  email: '',
  phone: '',
})

const openProfileDialog = () => {
  // 重置所有编辑状态
  editState.nickname = false
  editState.email = false
  editState.phone = false
  profileDialogVisible.value = true
}

// 开始编辑某一行
const startEdit = (field) => {
  tempProfile[field] = userInfo[field] // 复制当前值到临时变量
  editState[field] = true
}

// 取消编辑
const cancelEdit = (field) => {
  editState[field] = false
}

// 保存单个字段
const saveField = async (field, labelName) => {
  const newValue = tempProfile[field]
  const oldValue = userInfo[field]

  // 1. 简单的空值校验 (手机号除外，允许为空)
  if (field !== 'phone' && !newValue) {
    ElMessage.warning(`${labelName}不能为空`)
    return
  }
  // 邮箱格式校验
  if (field === 'email') {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(newValue)) {
      ElMessage.warning('请输入有效的邮箱地址')
      return
    }
  }

  // 2. Diff 检测：如果内容没变，直接切回查看模式
  // 注意处理 null/undefined 和空字符串的比较
  const safeNew = newValue || ''
  const safeOld = oldValue || ''

  if (safeNew === safeOld) {
    editState[field] = false
    return
  }

  // 3. 发送请求
  try {
    const payload = { [field]: newValue }
    const res = await updateUserAPI(userInfo.id, payload)

    if (res.status === 200) {
      ElMessage.success(`${labelName}已更新`)
      // 更新本地视图
      userInfo[field] = newValue
      // 退出编辑模式
      editState[field] = false
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || '更新失败'
    ElMessage.error(errorMsg)
  }
}

// 注销账户模块
const handleDeleteAccount = () => {
  // 1. 二次确认弹窗
  ElMessageBox.confirm(
    '注销后，您的所有数据将被永久删除且无法恢复。确定要继续吗？',
    '危险操作警告',
    {
      confirmButtonText: '确定注销',
      cancelButtonText: '取消',
      type: 'error', // 使用 error 类型显示红色图标
      confirmButtonClass: 'el-button--danger', // 确认按钮变红
      showClose: false,
    },
  )
    .then(async () => {
      try {
        // 2. 调用后端接口
        const res = await deleteUserAPI(userInfo.id)

        // 3. 处理响应
        if (res.status === 200) {
          ElMessage.success('账户已注销')
          // 清除本地信息并跳转
          clearAuthToken()
          localStorage.removeItem('user_id')
          profileDialogVisible.value = false // 关闭弹窗
          router.push('/login')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || '注销失败'
        ElMessage.error(errorMsg)
      }
    })
    .catch(() => {
      // 用户点击取消，不做任何操作
      ElMessage.info('已取消操作')
    })
}

// ================= 修改密码逻辑 =================
const passwordDialogVisible = ref(false)
const passwordFormRef = ref(null)
const passwordForm = reactive({
  password: '',
  confirmPassword: '',
})

const validatePass2 = (rule, value, callback) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== passwordForm.password) {
    callback(new Error('两次输入密码不一致!'))
  } else {
    callback()
  }
}

const passwordRules = {
  // 加上 required: true 会自动显示红星
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' }, // 这里加了 required
    { validator: validatePass2, trigger: 'blur' },
  ],
}

const openPasswordDialog = () => {
  passwordForm.password = ''
  passwordForm.confirmPassword = ''
  passwordDialogVisible.value = true
}

const submitPassword = async () => {
  if (!passwordFormRef.value) return
  await passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        const payload = { password: passwordForm.password }
        const res = await updateUserAPI(userInfo.id, payload)

        if (res.status === 200) {
          ElMessage.success('密码修改成功，请重新登录')
          passwordDialogVisible.value = false
          clearAuthToken()
          localStorage.removeItem('user_id')
          router.push('/login')
        }
      } catch (error) {
        const errorMsg = error.response?.data?.message || '密码修改失败'
        ElMessage.error(errorMsg)
      }
    }
  })
}

const roleLabel = computed(() => {
  const map = {
    0: '用户',
    1: '管理员',
    2: '超级管理员',
  }
  return map[userInfo.roleId] || '未知'
})

const roleTagType = computed(() => {
  const map = {
    0: 'info',
    1: 'success',
    2: 'warning', // 超管用醒目的颜色
  }
  return map[userInfo.roleId] || 'info'
})

const currentPageTitle = computed(() => {
  const roleId = userInfo.roleId

  if (route.path.includes('/datasets')) {
    return roleId > 0 ? '数据集管理' : '数据集浏览'
  }
  if (route.path.includes('/applications')) {
    return roleId > 0 ? '审批管理' : '我的申请'
  }
  if (route.path.includes('/users')) {
    return '用户管理'
  }
  if (route.path.includes('/statistic')) {
    return '统计信息'
  }

  return '系统面板' // 默认回退
})
</script>

<template>
  <el-container class="main-layout-container">
    <el-aside width="200px">
      <div class="logo-area">VBEP 数据平台</div>
      <el-menu :default-active="route.path" class="el-menu-vertical-demo" router>
        <el-menu-item index="/home/datasets">
          <el-icon><Tickets /></el-icon>
          <span>{{ userInfo.roleId > 0 ? '数据集管理' : '数据集浏览' }}</span>
        </el-menu-item>

        <el-menu-item index="/home/applications">
          <el-icon><Memo /></el-icon>
          <span>{{ userInfo.roleId > 0 ? '审批管理' : '我的申请' }}</span>
        </el-menu-item>

        <el-menu-item index="/home/users" v-if="userInfo.roleId > 0">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>

        <el-menu-item index="/home/statistic" v-if="userInfo.roleId > 0">
          <el-icon><Odometer /></el-icon>
          <span>统计信息</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header>
        <div class="header-content">
          <span class="page-title">{{ currentPageTitle }}</span>

          <el-skeleton :rows="1" animated :loading="isUserLoading" style="width: 250px">
            <template #default>
              <el-dropdown trigger="click" @command="handleCommand">
                <div class="user-info-trigger">
                  <el-icon><User /></el-icon>
                  <span class="nickname">{{ userInfo.nickname }}</span>
                  <el-icon class="el-icon--right"><arrow-down /></el-icon>

                  <el-divider direction="vertical" />

                  <el-tag :type="roleTagType" size="small" effect="dark">
                    {{ roleLabel }}
                  </el-tag>
                </div>

                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="profile" :icon="Edit">个人信息</el-dropdown-item>
                    <el-dropdown-item command="password" :icon="Key">修改密码</el-dropdown-item>
                    <el-dropdown-item command="logout" divided :icon="SwitchButton"
                      >退出登录</el-dropdown-item
                    >
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-skeleton>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>

    <el-dialog v-model="profileDialogVisible" title="个人信息" width="500px">
      <div class="profile-list">
        <div class="profile-row">
          <div class="row-label">姓名</div>
          <div class="row-content">
            <span v-if="!editState.nickname">{{ userInfo.nickname }}</span>
            <el-input v-else v-model="tempProfile.nickname" size="small" />
          </div>
          <div class="row-action">
            <el-button
              v-if="!editState.nickname"
              type="primary"
              link
              :icon="Edit"
              @click="startEdit('nickname')"
            >
              修改
            </el-button>
            <template v-else>
              <el-button type="success" link :icon="Check" @click="saveField('nickname', '姓名')"
                >确定</el-button
              >
              <el-button type="info" link :icon="Close" @click="cancelEdit('nickname')"
                >取消</el-button
              >
            </template>
          </div>
        </div>

        <div class="profile-row">
          <div class="row-label">邮箱</div>
          <div class="row-content">
            <span v-if="!editState.email">{{ userInfo.email || '未设置' }}</span>
            <el-input v-else v-model="tempProfile.email" size="small" />
          </div>
          <div class="row-action">
            <el-button
              v-if="!editState.email"
              type="primary"
              link
              :icon="Edit"
              @click="startEdit('email')"
            >
              修改
            </el-button>
            <template v-else>
              <el-button type="success" link :icon="Check" @click="saveField('email', '邮箱')"
                >确定</el-button
              >
              <el-button type="info" link :icon="Close" @click="cancelEdit('email')"
                >取消</el-button
              >
            </template>
          </div>
        </div>

        <div class="profile-row">
          <div class="row-label">手机号</div>
          <div class="row-content">
            <span v-if="!editState.phone">{{ userInfo.phone || '未设置' }}</span>
            <el-input v-else v-model="tempProfile.phone" size="small" />
          </div>
          <div class="row-action">
            <el-button
              v-if="!editState.phone"
              type="primary"
              link
              :icon="Edit"
              @click="startEdit('phone')"
            >
              修改
            </el-button>
            <template v-else>
              <el-button type="success" link :icon="Check" @click="saveField('phone', '手机号')"
                >确定</el-button
              >
              <el-button type="info" link :icon="Close" @click="cancelEdit('phone')"
                >取消</el-button
              >
            </template>
          </div>
        </div>

        <el-divider border-style="dashed" />

        <div class="profile-row danger-row">
          <div class="row-label danger-label">账户控制</div>
          <div class="row-content">
            <span class="danger-desc">注销后无法恢复，请谨慎操作</span>
          </div>
          <div class="row-action">
            <el-button type="danger" link size="small" @click="handleDeleteAccount">
              注销账户
            </el-button>
          </div>
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="profileDialogVisible = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordDialogVisible" title="修改密码" width="400px">
      <el-form
        :model="passwordForm"
        :rules="passwordRules"
        ref="passwordFormRef"
        label-width="100px"
        status-icon
      >
        <el-form-item label="新密码" prop="password">
          <el-input v-model="passwordForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>

      <div class="password-warning">
        <el-icon><Key /></el-icon> 注意：修改密码后，你将会被要求重新登录
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="passwordDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitPassword">确认修改</el-button>
        </span>
      </template>
    </el-dialog>
  </el-container>
</template>

<style scoped>
.main-layout-container {
  height: 100vh;
}
.logo-area {
  height: 60px;
  line-height: 60px;
  text-align: center;
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
  background-color: #fff;
}
.el-aside {
  border-right: 1px solid #e6e6e6;
  background-color: #fff;
}
.el-menu {
  border-right: none;
}
.el-header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.page-title {
  font-size: 18px;
  font-weight: 500;
}
.user-info-trigger {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 5px 10px;
  border-radius: 4px;
  transition: background-color 0.3s;
}
.user-info-trigger:hover {
  background-color: #f5f7fa;
}
.user-info-trigger .nickname {
  margin: 0 8px;
  font-weight: 500;
}
.el-main {
  background-color: #f0f2f5;
}

/* 个人信息 Dialog 样式 */
.profile-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
  padding: 0 10px;
}
.profile-row {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #f0f0f0;
  padding-bottom: 15px;
}
.profile-row:last-child {
  border-bottom: none;
}
.row-label {
  width: 60px;
  font-weight: bold;
  color: #606266;
}
.row-content {
  flex: 1;
  padding: 0 15px;
  color: #303133;
  display: flex;
  align-items: center;
  height: 32px; /* 固定高度防止Input切换时抖动 */
}
.row-action {
  width: 110px;
  display: flex;
  justify-content: flex-end;
  gap: 5px;
}

/* 密码 Dialog 提示样式 */
.password-warning {
  margin-top: 20px;
  padding: 8px 12px;
  background-color: #fef0f0;
  color: #f56c6c;
  font-size: 12px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* 危险区域样式 */
.danger-row {
  margin-top: 10px;
}
.danger-label {
  color: #f56c6c; /* Element Plus 的 Danger 红色 */
}
.danger-desc {
  font-size: 12px;
  color: #909399;
}
</style>
