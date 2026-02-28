<script setup>
import { Lock, User, Message, CircleCheckFilled, Refresh, CircleCheck } from '@element-plus/icons-vue'
import { reactive, ref, onBeforeUnmount, watch } from 'vue'
import axios from 'axios'
import logoSrc from '@/assets/img/logo.png'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import '@/assets/styles/login.css'

const router = useRouter()

const recoverFormRef = ref()
const recoverForm = reactive({
  email: '',
  otp: '',
})

const recoverFormRules = reactive({
  email: [
    { required: true, message: '邮箱不能为空', trigger: 'submit' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'submit' },
  ],
  otp: [{ required: true, message: '验证码不能为空', trigger: 'submit' }],
})

const isLoading = ref(false)

const submitRecoverForm = async () => {
  if (!recoverFormRef.value) return
  if (isLoading.value) return
  try {
    await recoverFormRef.value.validate()
    console.log('表单校验通过，发送请求')
    isLoading.value = true
    const requestData = {
      email: recoverForm.email,
      otp: recoverForm.otp,
    }
    await axios.post('/api/v1/users/', requestData)
    isLoading.value = false
    recoverFormRef.value.resetFields()
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
  if (!recoverFormRef.value) return

  // 单独校验邮箱字段
  recoverFormRef.value.validateField('email', async (isValid) => {
    if (isValid) {
      otpTarget.value = recoverForm.email
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
        <span class="heading">忘记密码</span>
      </div>
      <div class="login-container">
        <div class="login-content">
          <el-form
            ref="recoverFormRef"
            class="login-form"
            :model="recoverForm"
            :rules="recoverFormRules"
            label-width="80px"
            label-position="right"
            status-icon
          >

            <el-form-item class="login-group" label="邮箱" prop="email">
              <el-input
                class="login-info"
                size="large"
                v-model="recoverForm.email"
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
                v-model="recoverForm.otp"
                placeholder="请输入6位验证码"
              >
                <template #append>
                  <el-button :loading="sendOtpLoading" @click="handleSendOTP" :disabled="countdown > 0">
                    {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
                  </el-button>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group login-button-item">
              <el-button class="login-btn" type="primary" @click="submitRecoverForm"
              >发送重置邮件</el-button
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
