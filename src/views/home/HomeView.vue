<script setup>
import { ref, reactive, onMounted, provide } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Tickets, Memo, User, ArrowDown, Edit, SwitchButton, Key } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 引入 axios 实例和清除 token 的方法
import apiClient, { clearAuthToken } from '../../request/axios.js'

// --- API 定义 ---
const fetchUserInfoAPI = (userId) => apiClient.get(`/api/v1/users/${userId}`)
// PATCH 接口：后端根据传入字段进行部分更新
const updateUserAPI = (userId, data) => apiClient.patch(`/api/v1/users/${userId}`, data)
const logoutAPI = () => apiClient.get('/api/v1/users/logout')

// --- 基础状态 ---
const router = useRouter()
const route = useRoute()
const userInfo = reactive({
  id: '',
  nickname: '',
  email: '',
  phone: '', 
  role: 'user'
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
      Object.assign(userInfo, {
        id,
        nickname,
        email,
        phone, 
        role: Number(role) === 1 ? 'admin' : 'user',
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
provide('isAdmin', () => userInfo.role === 'admin')

// --- 登出逻辑 ---
const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗?', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    
    // 尝试调用后端登出
    try {
      await logoutAPI()
    } catch (e) {
      console.warn('后端登出接口调用失败，继续执行前端清除', e)
    }

    // 清除前端 Token 和 UserID
    clearAuthToken()
    localStorage.removeItem('user_id')

    ElMessage.success('已安全退出')
    router.push('/login')
  } catch (e) {
    // 用户取消或报错
    console.log(e)
  }
}

// --- 下拉菜单指令处理 ---
const handleCommand = (command) => {
  if (command === 'logout') {
    handleLogout()
  } else if (command === 'profile') {
    openProfileDialog()
  } else if (command === 'password') {
    openPasswordDialog()
  }
}

// ================= 修改个人信息逻辑 =================
const profileDialogVisible = ref(false)
const profileFormRef = ref(null)
const profileForm = reactive({
  nickname: '',
  email: '',
  phone: ''
})

const profileRules = {
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ]
}

const openProfileDialog = () => {
  // 回显当前数据到表单
  profileForm.nickname = userInfo.nickname
  profileForm.email = userInfo.email
  profileForm.phone = userInfo.phone
  profileDialogVisible.value = true
}

const submitProfile = async () => {
  if (!profileFormRef.value) return
  await profileFormRef.value.validate(async (valid) => {
    if (valid) {
      // 1. 差异比对：只发送修改过的字段
      // 使用严格全等 !== 比较缓存的 userInfo 和表单 profileForm
      const payload = {}
      
      if (profileForm.nickname !== userInfo.nickname) {
        payload.nickname = profileForm.nickname
      }
      if (profileForm.email !== userInfo.email) {
        payload.email = profileForm.email
      }
      // 考虑到 phone 可能为 null 或 undefined，处理一下空字符串的情况
      const oldPhone = userInfo.phone || ''
      const newPhone = profileForm.phone || ''
      if (newPhone !== oldPhone) {
        payload.phone = newPhone
      }

      // 2. 如果没有任何字段被修改，直接提示并返回
      if (Object.keys(payload).length === 0) {
        ElMessage.info('未检测到内容修改')
        return
      }

      try {
        // 3. 发送请求
        const res = await updateUserAPI(userInfo.id, payload)
        
        // 4. 成功处理 (HTTP 200 OK)
        if (res.status === 200) {
          ElMessage.success('个人信息修改成功')
          // 更新本地视图，保持一致性
          Object.assign(userInfo, payload)
          // 关闭弹窗
          profileDialogVisible.value = false
        }
      } catch (error) {
        // 5. 失败处理 (非 200，被 Axios 拦截器捕获)
        // 显示后端返回的 message
        const errorMsg = error.response?.data?.message || '修改失败'
        ElMessage.error(errorMsg)
        // 失败时不关闭弹窗，允许用户修正
      }
    }
  })
}

// ================= 修改密码逻辑 =================
const passwordDialogVisible = ref(false)
const passwordFormRef = ref(null)
const passwordForm = reactive({
  password: '',
  confirmPassword: ''
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
  password: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }],
  confirmPassword: [{ validator: validatePass2, trigger: 'blur' }]
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
        const payload = {
          password: passwordForm.password
        }
        
        const res = await updateUserAPI(userInfo.id, payload)
        
        // 成功处理
        if (res.status === 200) {
          ElMessage.success('密码修改成功，请重新登录')
          passwordDialogVisible.value = false
          
          // 强制登出
          clearAuthToken()
          localStorage.removeItem('user_id')
          router.push('/login')
        }
      } catch (error) {
        // 失败处理：显示后端 message
        const errorMsg = error.response?.data?.message || '密码修改失败'
        ElMessage.error(errorMsg)
        // 失败时不关闭弹窗
      }
    }
  })
}
</script>

<template>
  <el-container class="main-layout-container">
    <el-aside width="200px">
      <div class="logo-area">VBEP 数据平台</div>
      <el-menu :default-active="route.path" class="el-menu-vertical-demo" router>
        <el-menu-item index="/home/datasets">
          <el-icon><Tickets /></el-icon>
          <span>数据集管理</span>
        </el-menu-item>
        <el-menu-item index="/home/applications">
          <el-icon><Memo /></el-icon>
          <span>审批管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header>
        <div class="header-content">
          <span class="page-title">{{
            route.name === 'dataset-list' ? '数据集列表' : '申请列表'
          }}</span>
          
          <el-skeleton :rows="1" animated :loading="isUserLoading" style="width: 250px">
            <template #default>
              <el-dropdown trigger="click" @command="handleCommand">
                <div class="user-info-trigger">
                  <el-icon><User /></el-icon>
                  <span class="nickname">{{ userInfo.nickname }}</span>
                  <el-icon class="el-icon--right"><arrow-down /></el-icon>
                  
                  <el-divider direction="vertical" />
                  <el-tag :type="userInfo.role === 'admin' ? 'success' : 'info'" size="small">
                    {{ userInfo.role === 'admin' ? '管理员' : '用户' }}
                  </el-tag>
                </div>

                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="profile" :icon="Edit">修改个人信息</el-dropdown-item>
                    <el-dropdown-item command="password" :icon="Key">修改密码</el-dropdown-item>
                    <el-dropdown-item command="logout" divided :icon="SwitchButton">退出登录</el-dropdown-item>
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

    <el-dialog v-model="profileDialogVisible" title="修改个人信息" width="400px">
      <el-form :model="profileForm" :rules="profileRules" ref="profileFormRef" label-width="80px">
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="profileForm.nickname" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="profileForm.email" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="profileForm.phone" placeholder="选填" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="profileDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitProfile">保存</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordDialogVisible" title="修改密码" width="400px">
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="100px">
        <el-form-item label="新密码" prop="password">
          <el-input v-model="passwordForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
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
</style>