<script setup>
import { Lock, User, Message, CircleCheckFilled, Refresh, CircleCheck } from '@element-plus/icons-vue'
import { reactive, ref, onBeforeUnmount, watch } from 'vue'
import axios from 'axios'
import logoSrc from '@/assets/img/logo.png'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import '@/assets/styles/login.css'
import sha256 from 'crypto-js/sha256.js'
import MarkdownRenderer from "@/components/MarkdownRenderer.vue"

const router = useRouter()

const registerFormRef = ref()
const registerForm = reactive({
  nickname: '',
  email: '',
  password: '',
  otp: '',
  agreement: false,
})
const isRegisterSuccess = ref(false)

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

watch(() => registerForm.password, (val) => {
  Object.assign(passwordStatus, checkPassword(val))
})

const validatePassword = (rule, value, callback) => {
  const { length, count } = checkPassword(value)
  if (!length) {
    return callback(new Error('密码长度不符合要求'))
  }
  if (count < 3) {
    return callback(new Error('密码未包含大写字母、小写字母、数字、特殊字符中的至少3项'))
  }
  callback()
}

const registerFormRules = reactive({
  nickname: [{ required: true, message: '真实姓名不能为空', trigger: 'submit' }],
  email: [
    { required: true, message: '邮箱不能为空', trigger: 'submit' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'submit' },
  ],
  password: [
    { required: true, message: '密码不能为空', trigger: 'submit' },
    { validator: validatePassword, trigger: 'submit' },
  ],
  otp: [{ required: true, message: '验证码不能为空', trigger: 'submit' }],
  agreement: [
    { validator: (rule, value, cb) => value ? cb() : cb(new Error('请同意用户协议')), trigger: 'submit' }
  ]
})

const isLoading = ref(false)

const submitRegisterForm = async () => {
  if (!registerFormRef.value) return
  if (isLoading.value) return
  try {
    await registerFormRef.value.validate()
    console.log('表单校验通过，发送请求')
    isLoading.value = true
    const passwordHash = sha256(registerForm.password).toString()
    const requestData = {
      email: registerForm.email,
      nickname: registerForm.nickname,
      password: passwordHash,
      otp: registerForm.otp,
    }
    await axios.post('/api/v1/users/', requestData)
    isLoading.value = false
    isRegisterSuccess.value = true
    registerFormRef.value.resetFields()
    setTimeout(() => {
      router.replace({ name: 'login' })
    }, 2000)
  } catch (e) {
    if (e && e.response) {
      console.error('API请求失败:', e.response.data)
      ElMessage.error(e.response.data.message || '注册失败，请稍后重试')
    } else if (e instanceof Error) {
      console.error('网络或代码错误:', e.message)
      ElMessage.error('网络连接失败，请检查您的网络')
    } else {
      console.log('表单验证失败:', e)
      ElMessage.error('请检查您填写的信息是否正确')
    }
  } finally {
    isLoading.value = false
  }
}

const sendOtpLoading = ref(false)
const countdown = ref(0)
let timer = null

const startCountdown = () => {
  countdown.value = 60
  timer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(timer)
      timer = null
    }
  }, 1000)
}

const captchaDialogVisible = ref(false)
const captchaData = reactive({
  id: '',
  image: '',
  answer: ''
})
const otpTarget = ref('')

const fetchCaptcha = async () => {
  try {
    const res = await axios.get('/api/v1/users/captcha')
    captchaData.id = res.data.captchaID
    captchaData.image = res.data.question
    captchaData.answer = ''
  } catch (e) {
    ElMessage.error('获取验证码失败')
  }
}

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})

const handleSendOTP = async () => {
  if (!registerFormRef.value) return

  // 单独校验邮箱字段
  registerFormRef.value.validateField('email', async (isValid) => {
    if (isValid) {
      otpTarget.value = registerForm.email
      await fetchCaptcha()
      captchaDialogVisible.value = true
    } else {
      ElMessage.warning('请输入有效的邮箱地址')
    }
  })
}

const confirmSendOTP = async () => {
  if (!captchaData.answer) {
    ElMessage.warning('请输入验证码')
    return
  }
  try {
    sendOtpLoading.value = true
    await axios.post('/api/v1/users/otp', {
      target: otpTarget.value,
      captchaID: captchaData.id,
      captchaAnswer: captchaData.answer
    })
    ElMessage.success('验证码已发送')
    captchaDialogVisible.value = false
    startCountdown()
  } catch (e) {
    console.error(e)
    ElMessage.error(e.response?.data?.message || '发送失败')
    fetchCaptcha() // 失败刷新验证码
  } finally {
    sendOtpLoading.value = false
  }
}

const agreementDialogVisible = ref(false)
const agreementContent = ref('')

const showAgreement = async () => {
  try {
    const res = await axios.get('/api/v1/agreement')
    agreementContent.value = res.data.content || '暂无协议内容'
    agreementDialogVisible.value = true
  } catch (e) {
    ElMessage.error('获取用户协议失败')
  }
}
const confirmAgreement = () => {
  registerForm.agreement = true
  agreementDialogVisible.value = false
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
      <div class="sso-header">
        <span class="heading">VBEP数据平台注册</span>
      </div>
      <div class="login-container">
        <div class="login-content">
          <div v-if="isRegisterSuccess" class="success-state">
            <el-icon class="success-icon" color="#67C23A"><CircleCheckFilled /></el-icon>
            <h3 class="success-title">注册成功！</h3>
            <p class="success-desc">账号已创建，2秒后自动跳转登录...</p>
            <el-button type="primary" @click="router.replace({ name: 'login' })">
              立即登录
            </el-button>
          </div>

          <el-form
            v-else
            ref="registerFormRef"
            class="login-form"
            :model="registerForm"
            :rules="registerFormRules"
            label-width="80px"
            label-position="right"
            status-icon
          >
            <el-form-item class="login-group" label="真实姓名" prop="nickname">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.nickname"
                placeholder="请输入真实姓名"
              >
                <template #prefix
                  ><el-icon><User /></el-icon
                ></template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="邮箱" prop="email">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.email"
                placeholder="请输入邮箱"
              >
                <template #prefix
                  ><el-icon><Message /></el-icon
                ></template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="验证码" prop="otp">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.otp"
                placeholder="请输入6位验证码"
              >
                <template #append>
                  <el-button :loading="sendOtpLoading" @click="handleSendOTP" :disabled="countdown > 0">
                    {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
                  </el-button>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="密码" prop="password">
              <el-popover placement="right" :width="270" trigger="focus" popper-class="register-password-popover">
                <template #reference>
                  <el-input
                    class="login-info"
                    size="large"
                    v-model="registerForm.password"
                    placeholder="请输入密码"
                    type="password"
                    show-password
                  >
                    <template #prefix><el-icon><Lock /></el-icon></template>
                  </el-input>
                </template>
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

            <el-form-item prop="agreement">
              <el-checkbox v-model="registerForm.agreement">
                我已阅读并同意 <a href="#" @click.prevent="showAgreement">《用户协议》</a>
              </el-checkbox>
            </el-form-item>

            <el-form-item class="login-group login-button-item">
              <el-button class="login-btn" type="primary" @click="submitRegisterForm"
                >立刻注册</el-button
              >
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>

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

    <el-dialog v-model="agreementDialogVisible" title="用户协议" width="600px" append-to-body>
      <div style="max-height: 50vh; overflow-y: auto; padding: 10px;">
        <markdown-renderer :model-value="agreementContent" />
      </div>
      <template #footer>
        <el-button type="primary" @click="confirmAgreement">我已阅读并同意</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 保持内容宽度限制 */
.login-content {
  width: 380px; /* 稍微增加一点宽度以容纳 label + input */
  margin: 0 auto;
}

/* 调整 label 的样式，使其更对齐 */
:deep(.el-form-item__label) {
  color: #181818;
  font-weight: 500;
}

.password-hint {
  padding-left: 80px;
  margin-bottom: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  text-align: left;
}

.success-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
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
}
.success-desc {
  color: #909399;
  margin-bottom: 30px;
}
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>

<style>
.register-password-popover .password-strength-popover {
  font-size: 12px;
  color: #606266;
}
.register-password-popover .strength-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  color: #909399;
}
.register-password-popover .strength-item.active {
  color: #67c23a;
}
.register-password-popover .strength-sub-items {
  padding-left: 24px;
  display: flex;
  gap: 10px;
  color: #909399;
}
.register-password-popover .strength-sub-items span.active {
  color: #67c23a;
  font-weight: bold;
}
</style>
