<script setup>
import { Lock, CircleCheckFilled, CircleCheck } from '@element-plus/icons-vue'
import { reactive, ref, onMounted, watch } from 'vue'
import axios from 'axios'
import logoSrc from '@/assets/img/logo.png'
import { ElMessage } from 'element-plus'
import { useRouter, useRoute } from 'vue-router'
import '@/assets/styles/login.css'
import sha256 from "crypto-js/sha256.js"

const router = useRouter()
const route = useRoute()
const formRef = ref()
const isLoading = ref(false)
const isSuccess = ref(false)

const form = reactive({
  password: '',
  confirmPassword: '',
  token: '',
})

// === 密码强度检测逻辑 (从注册页移植) ===
const passwordStatus = reactive({
  length: false,
  upper: false,
  lower: false,
  number: false,
  special: false,
  count: 0
})

const checkPassword = (val) => {
  const v = val || ''
  const length = v.length >= 6 && v.length <= 20
  const upper = /[A-Z]/.test(v)
  const lower = /[a-z]/.test(v)
  const number = /\d/.test(v)
  const special = /[~!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(v)
  let count = 0
  if (upper) count++
  if (lower) count++
  if (number) count++
  if (special) count++
  return { length, upper, lower, number, special, count }
}

// 监听密码变化实时更新状态
watch(() => form.password, (val) => {
  Object.assign(passwordStatus, checkPassword(val))
})

// 密码校验规则
const validatePassword = (rule, value, callback) => {
  const { length, count } = checkPassword(value)
  if (!length) {
    return callback(new Error('密码长度不符合要求'))
  }
  if (count < 3) {
    return callback(new Error('密码需包含大写、小写、数字、特殊字符至少3类'))
  }
  callback()
}

// 确认密码校验规则
const validatePass2 = (rule, value, callback) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== form.password) {
    callback(new Error('两次输入密码不一致!'))
  } else {
    callback()
  }
}

// 表单规则
const rules = reactive({
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' }, // 修改为 blur 体验更好，或者 keep 'change'
    { validator: validatePassword, trigger: ['blur', 'change'] },
  ],
  confirmPassword: [
    { required: true, validator: validatePass2, trigger: ['blur', 'change'] },
  ],
})

// === 生命周期与业务逻辑 ===

onMounted(() => {
  const token = route.query.token
  if (!token) {
    ElMessage.error('无效的重置链接，缺少 Token')
  } else {
    form.token = token
  }
})

const submitReset = async () => {
  if (!formRef.value) return
  if (!form.token) {
    ElMessage.error('Token 丢失，请重新点击邮件链接')
    return
  }

  try {
    await formRef.value.validate()
    isLoading.value = true

    const passwordHash = sha256(form.password).toString()
    await axios.post('/api/v1/users/reset/confirm', {
      token: form.token,
      new_password: passwordHash,
    })

    isSuccess.value = true
    setTimeout(() => {
      router.push('/login')
    }, 3000)

  } catch (e) {
    if (e.response) {
      const status = e.response.status
      const msg = e.response.data?.message || '重置失败'
      ElMessage.error(msg)
      if (status === 400) {
        setTimeout(() => router.push('/reset/send'), 2000)
      }
    } else if (e.request) {
      console.error('无法连接到服务器')
      ElMessage.error('网络连接不可用，请检查您的网络设置')
    } else {
      console.warn('表单验证未通过或其他错误:', e)
    }
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="main-container">
    <div class="login-card-wrapper">
      <div class="register-button-wrapper">
        <RouterLink to="/login" class="register-button">登录</RouterLink>
      </div>

      <div class="sso-banner">
        <div class="sso-logo">
          <img :src="logoSrc" alt="Logo" class="sso-logo-img" />
        </div>
      </div>

      <!-- 成功状态 -->
      <div v-if="isSuccess" class="success-state">
        <el-icon class="success-icon" color="#67C23A"><CircleCheckFilled /></el-icon>
        <h2 class="success-title">密码重置成功!</h2>
        <p class="success-desc">您的密码已更新，3秒后将自动跳转至登录页...</p>
        <el-button type="primary" @click="router.push('/login')">立即跳转</el-button>
      </div>

      <!-- 表单状态 -->
      <div v-else>
        <div class="sso-header">
          <span class="heading">设置新密码</span>
        </div>

        <div class="login-container">
          <div class="login-content">
            <el-form
              ref="formRef"
              class="login-form"
              :model="form"
              :rules="rules"
              label-width="80px"
              label-position="right"
              status-icon
            >
              <!-- 新密码 (带 Popover 提示) -->
              <el-form-item class="login-group" label="新密码" prop="password">
                <el-popover
                  placement="right"
                  :width="270"
                  trigger="focus"
                  popper-class="password-popover"
                >
                  <template #reference>
                    <el-input
                      class="login-info"
                      size="large"
                      v-model="form.password"
                      placeholder="请输入新密码"
                      type="password"
                      show-password
                    >
                      <template #prefix>
                        <el-icon><Lock /></el-icon>
                      </template>
                    </el-input>
                  </template>

                  <!-- Popover 内容：密码强度指示器 -->
                  <div class="password-strength-popover">
                    <div class="strength-item" :class="{ active: passwordStatus.length }">
                      <el-icon><component :is="passwordStatus.length ? CircleCheckFilled : CircleCheck" /></el-icon>
                      <span>长度为 6-20 个字符</span>
                    </div>
                    <div class="strength-item" :class="{ active: passwordStatus.count >= 3 }">
                      <el-icon><component :is="passwordStatus.count >= 3 ? CircleCheckFilled : CircleCheck" /></el-icon>
                      <span>包含以下至少 3 种字符：</span>
                    </div>
                    <div class="strength-sub-items">
                      <span :class="{ active: passwordStatus.upper }">大写</span>
                      <span :class="{ active: passwordStatus.lower }">小写</span>
                      <span :class="{ active: passwordStatus.number }">数字</span>
                      <span :class="{ active: passwordStatus.special }">特殊</span>
                    </div>
                  </div>
                </el-popover>
              </el-form-item>

              <!-- 确认密码 -->
              <el-form-item class="login-group" label="确认密码" prop="confirmPassword">
                <el-input
                  class="login-info"
                  size="large"
                  v-model="form.confirmPassword"
                  placeholder="请再次输入密码"
                  type="password"
                  show-password
                >
                  <template #prefix>
                    <el-icon><Lock /></el-icon>
                  </template>
                </el-input>
              </el-form-item>

              <el-form-item class="login-group login-button-item">
                <el-button
                  class="login-btn"
                  type="primary"
                  :loading="isLoading"
                  @click="submitReset"
                >
                  确认修改
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-content {
  width: 380px;
  margin: 0 auto;
}

:deep(.el-form-item__label) {
  color: #181818;
  font-weight: 500;
}

.success-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 20px;
  text-align: center;
  animation: fadeIn 0.5s ease;
}

.success-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.success-title {
  font-size: 24px;
  color: #303133;
  margin: 0 0 10px 0;
  font-weight: 600;
}

.success-desc {
  color: #909399;
  margin-bottom: 30px;
  font-size: 14px;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>

<!-- 全局样式：用于 Popover -->
<style>
.password-popover .password-strength-popover {
  font-size: 12px;
  color: #606266;
}
.password-popover .strength-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  color: #909399;
  transition: color 0.3s;
}
.password-popover .strength-item.active {
  color: #67c23a;
}
.password-popover .strength-sub-items {
  padding-left: 24px;
  display: flex;
  gap: 10px;
  color: #909399;
}
.password-popover .strength-sub-items span {
  transition: all 0.3s;
}
.password-popover .strength-sub-items span.active {
  color: #67c23a;
  font-weight: bold;
}
</style>
