<script setup>
import { ref, reactive, onMounted, provide, computed, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  Tickets,
  Memo,
  User,
  ArrowDown,
  Edit,
  SwitchButton,
  Key,
  Setting,
  Download,
  Refresh,
  DataLine,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 引入 axios 实例和清除 token 的方法
import apiClient, { clearAuthToken } from '@/request/axios.js'
import '@/assets/styles/universal.css'
import sha256 from "crypto-js/sha256.js";

// --- API 定义 ---
const fetchUserInfoAPI = (userId) => apiClient.get(`/api/v1/users/${userId}`)
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)
const logoutAPI = () => apiClient.get('/api/v1/users/logout')
const deleteUserAPI = (userId) => apiClient.delete(`/api/v1/users/${userId}`)

// --- 基础状态 ---
const router = useRouter()
const route = useRoute()
const userInfo = reactive({
  userID: '',
  nickname: '',
  email: '',
  // phone: '',
  // role: 'user',
  roleId: 0, // 0 - 'user', 1 - 'admin', 2 - 'super_admin'
})
const isUserLoading = ref(true)

// --- 初始化逻辑 ---
onMounted(async () => {
  isUserLoading.value = true
  try {
    const storedUserId = localStorage.getItem('userID')
    if (!storedUserId) {
      ElMessage.error('无法获取用户信息，请重新登录')
      router.push('/login')
      return
    }
    const res = await fetchUserInfoAPI(storedUserId)
    if (res && res.data?.data) {
      //const { userID, nickname, email, phone, role } = res.data.data
      const { userID, nickname, email, role } = res.data.data
      const roleNum = Number(role)
      // let roleStr = 'user'
      // if (roleNum === 1) roleStr = 'admin'
      // if (roleNum === 2) roleStr = 'super_admin'

      Object.assign(userInfo, {
        userID,
        nickname,
        email,
        // phone,
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
    localStorage.removeItem('userID')
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

const openProfileDialog = () => {
  profileDialogVisible.value = true
}

// --- 新增：独立弹窗修改邮箱/手机号逻辑 ---
const editDialog = reactive({
  visible: false,
  field: '',
  label: '',
  value: '',
  otp: '',
  loading: false,
  needsOtp: true
})

const openEditDialog = (field, label, needsOtp = true) => {
  editDialog.field = field
  editDialog.label = label
  editDialog.value = '' // 默认不填充旧值
  editDialog.otp = ''
  editDialog.needsOtp = needsOtp
  editDialog.visible = true
}

const submitEditDialog = async () => {
  const { field, value, otp, label, needsOtp } = editDialog

  if (!value) {
    ElMessage.warning(`请输入新${label}`)
    return
  }
  if (needsOtp && !otp) {
    ElMessage.warning('请输入验证码')
    return
  }

  // 格式校验
  if (field === 'email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    ElMessage.warning('请输入有效的邮箱地址')
    return
  }
  // if (field === 'phone' && !/^1[3-9]\d{9}$/.test(value)) {
  //   ElMessage.warning('请输入有效的手机号')
  //   return
  // }

  editDialog.loading = true
  try {
    const payload = { [field]: value }
    if (needsOtp) {
      payload.otp = otp
    }
    const res = await updateUserAPI(userInfo.userID, payload)
    if (res.status === 200) {
      ElMessage.success(`${label}已更新`)
      userInfo[field] = value
      editDialog.visible = false
    }
  } catch (error) {
    const errorMsg = error.response?.data?.message || '更新失败'
    ElMessage.error(errorMsg)
  } finally {
    editDialog.loading = false
  }
}

const captchaDialogVisible = ref(false)
const captchaData = reactive({
  id: '',
  image: '',
  answer: ''
})
const otpTarget = ref('')
const sendOtpLoading = ref(false)
const otpCountdown = ref(0)
let otpTimer = null

const startOtpCountdown = () => {
  otpCountdown.value = 60
  otpTimer = setInterval(() => {
    otpCountdown.value--
    if (otpCountdown.value <= 0) {
      clearInterval(otpTimer)
      otpTimer = null
    }
  }, 1000)
}

onBeforeUnmount(() => {
  if (otpTimer) clearInterval(otpTimer)
})

const fetchCaptcha = async () => {
  try {
    const res = await apiClient.get('/api/v1/users/captcha')
    captchaData.id = res.data.captchaID
    captchaData.image = res.data.question
    captchaData.answer = ''
  } catch (e) {
    ElMessage.error('获取验证码失败')
  }
}

const handleSendOTP = async (target) => {
  if (!target) {
    ElMessage.warning('请先输入目标邮箱')
    return
  }
  // 简单格式校验
  const isEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(target)
  // const isPhone = /^1[3-9]\d{9}$/.test(target)

  // if (!isEmail && !isPhone) {
  if (!isEmail) {
    ElMessage.warning('请输入正确的邮箱')
    return
  }

  otpTarget.value = target
  await fetchCaptcha()
  captchaDialogVisible.value = true
}

const confirmSendOTP = async () => {
  if (!captchaData.answer) {
    ElMessage.warning('请输入验证码')
    return
  }
  sendOtpLoading.value = true
  try {
    await apiClient.post('/api/v1/users/otp', {
      target: otpTarget.value,
      captchaID: captchaData.id,
      captchaAnswer: captchaData.answer
    })
    ElMessage.success('验证码已发送')
    captchaDialogVisible.value = false
    startOtpCountdown()
  } catch (e) {
    console.error(e)
    ElMessage.error(e.response?.data?.message || '发送失败')
    fetchCaptcha()
  } finally {
    sendOtpLoading.value = false
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
        const res = await deleteUserAPI(userInfo.userID)

        // 3. 处理响应
        if (res.status === 200) {
          ElMessage.success('账户已注销')
          // 清除本地信息并跳转
          clearAuthToken()
          localStorage.removeItem('userID')
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
    { required: true, message: '请输入新密码', trigger: 'submit' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'submit' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'submit' },
    { validator: validatePass2, trigger: 'submit' },
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
        const passwordHash = sha256(passwordForm.password).toString()
        const payload = { password: passwordHash }
        const res = await updateUserAPI(userInfo.userID, payload)

        if (res.status === 200) {
          ElMessage.success('密码修改成功，请重新登录')
          passwordDialogVisible.value = false
          clearAuthToken()
          localStorage.removeItem('userID')
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
  if (route.path.includes('/downloads')) {
    return roleId > 1 ? '公开下载管理' : '下载中心'
  }
  if (route.path.includes('/statistics')) {
    return '数据统计'
  }
  if (route.path.includes('/users')) {
    return '用户管理'
  }
  if (route.path.includes('/settings')) {
    return '全局设置'
  }

  return '系统面板' // 默认回退
})
</script>

<template>
  <el-container class="main-layout-container">
    <el-aside width="200px" class="sidebar-container">
      <div class="logo-area">VBEP 数据平台</div>
      <div class="sidebar-flex-wrapper">
        <el-menu :default-active="route.path" class="el-menu-vertical-demo menu-top" router>
          <el-menu-item-group>
            <template #title>
              <div class="group-title-wrapper">
                <span class="separator-start"></span>
                <span class="group-text">业务管理</span>
                <span class="separator-end"></span>
              </div>
            </template>
            <el-menu-item index="/home/datasets">
              <el-icon><Tickets /></el-icon>
              <span>{{ userInfo.roleId > 0 ? '数据集管理' : '数据集浏览' }}</span>
            </el-menu-item>

            <el-menu-item index="/home/applications">
              <el-icon><Memo /></el-icon>
              <span>{{ userInfo.roleId > 0 ? '审批管理' : '我的申请' }}</span>
            </el-menu-item>

            <el-menu-item index="/home/downloads">
              <el-icon><Download /></el-icon>
              <span>{{ userInfo.roleId > 1 ? '公开下载管理' : '下载中心' }}</span>
            </el-menu-item>
          </el-menu-item-group>
        </el-menu>

        <el-menu
          v-if="userInfo.roleId > 0"
          :default-active="route.path"
          class="el-menu-vertical-demo menu-bottom"
          router
        >
          <el-menu-item-group>
            <template #title>
              <div class="group-title-wrapper">
                <span class="separator-start"></span>
                <span class="group-text">系统管理</span>
                <span class="separator-end"></span>
              </div>
            </template>
            <el-menu-item index="/home/statistics">
              <el-icon><DataLine /></el-icon>
              <span>数据统计</span>
            </el-menu-item>

            <el-menu-item index="/home/users">
              <el-icon><User /></el-icon>
              <span>用户管理</span>
            </el-menu-item>

            <el-menu-item index="/home/settings">
              <el-icon><Setting /></el-icon>
              <span>全局设置</span>
            </el-menu-item>
          </el-menu-item-group>
        </el-menu>
      </div>
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
      <div class="info-list">
        <div class="info-row">
          <div class="row-label">姓名</div>
          <div class="row-content">
            <span>{{ userInfo.nickname }}</span>
          </div>
          <div class="row-action">
            <el-button
              type="primary"
              link
              :icon="Edit"
              @click="openEditDialog('nickname', '姓名', false)"
            >
              修改
            </el-button>
          </div>
        </div>

        <div class="info-row">
          <div class="row-label">邮箱</div>
          <div class="row-content">
            <span>{{ userInfo.email || '未设置' }}</span>
          </div>
          <div class="row-action">
            <el-button
              type="primary"
              link
              :icon="Edit"
              @click="openEditDialog('email', '邮箱')"
            >
              修改
            </el-button>
          </div>
        </div>

<!--        <div class="info-row">-->
<!--          <div class="row-label">手机号</div>-->
<!--          <div class="row-content">-->
<!--            <span>{{ userInfo.phone || '未设置' }}</span>-->
<!--          </div>-->
<!--          <div class="row-action">-->
<!--            <el-button-->
<!--              type="primary"-->
<!--              link-->
<!--              :icon="Edit"-->
<!--              @click="openEditDialog('phone', '手机号')"-->
<!--            >-->
<!--              修改-->
<!--            </el-button>-->
<!--          </div>-->
<!--        </div>-->

        <el-divider border-style="dashed" />

        <div class="info-row danger-row">
          <div class="row-label danger-label">账户控制</div>
          <div class="row-content">
            <span class="sub-text">注销后无法恢复，请谨慎操作</span>
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

    <el-dialog v-model="editDialog.visible" :title="'修改' + editDialog.label" width="400px" append-to-body>
      <el-form label-width="80px" @submit.prevent>
        <el-form-item :label="'新' + editDialog.label">
          <el-input v-model="editDialog.value" :placeholder="'请输入新' + editDialog.label" />
          <div v-if="editDialog.field === 'nickname'" class="dialog-tip">
            真实姓名需要与申请表和身份证明文件相符
          </div>
        </el-form-item>
        <el-form-item label="验证码" v-if="editDialog.needsOtp">
          <el-input v-model="editDialog.otp" placeholder="请输入验证码">
            <template #append>
              <el-button @click="handleSendOTP(editDialog.value)" :disabled="otpCountdown > 0">{{ otpCountdown > 0 ? `${otpCountdown}s` : '发送' }}</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="editDialog.loading" @click="submitEditDialog">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="captchaDialogVisible" title="安全验证" width="310px" append-to-body>
      <div style="display: flex; flex-direction: column; align-items: center; gap: 15px;">
        <div style="display: flex; gap: 10px; align-items: center;">
          <img :src="captchaData.image" style="height: 50px; border: 1px solid #eee; border-radius: 4px;" />
          <el-button :icon="Refresh" circle @click="fetchCaptcha" />
        </div>
        <el-input v-model="captchaData.answer" placeholder="请输入图片字符(不分大小写)" @keyup.enter="confirmSendOTP" />
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="captchaDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="sendOtpLoading" @click="confirmSendOTP">确定</el-button>
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
.sidebar-container {
  display: flex;
  flex-direction: column;
  border-right: 1px solid #e6e6e6;
  background-color: #fff;
}
.sidebar-flex-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.menu-top {
  flex: 1;
  overflow-y: auto;
}
.el-menu {
  border-right: none;
}
.group-title-wrapper {
  display: flex;
  align-items: center;
  font-size: 12px;
}
.separator-start {
  width: 12px;
  height: 1px;
  background-color: #dcdfe6;
  margin-right: 8px;
}
.separator-end {
  flex: 1;
  height: 1px;
  background-color: #dcdfe6;
  margin-left: 8px;
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

.info-row {
  display: flex;
  align-items: center;
  min-height: 32px;
}
.info-row.is-editing {
  align-items: flex-start;
}
.row-label {
  width: 60px;
  flex-shrink: 0;
  line-height: 32px;
}
.row-content {
  flex: 1;
  margin-right: 5px;
}
.row-action {
  min-width: 60px;
  text-align: right;
  flex-shrink: 0;
  white-space: nowrap;
  margin-left: 5px;
}
.edit-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}
.edit-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-start;
  margin-top: 5px;
}

/* Slide Fade Animation */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}
.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(1, 0.5, 0.8, 1);
}
.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(-10px);
  opacity: 0;
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

.dialog-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}
</style>
