<script setup>
import { Lock, User, Message, CircleCheckFilled, Refresh, CircleCheck } from '@element-plus/icons-vue'
import { reactive, ref, onBeforeUnmount, watch } from 'vue'
import axios from 'axios'
import logoSrc from '@/assets/img/logo.png'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import '@/assets/styles/login.css'

const router = useRouter()

const resetFormRef = ref()
const resetForm = reactive({
  email: '',
})

const resetFormRules = reactive({
  email: [
    { required: true, message: '邮箱不能为空', trigger: 'submit' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'submit' },
  ],
})

const isLoading = ref(false)


const resetPasswordPage = '/reset/confirm'
const frontendUrl = `${window.location.origin}${resetPasswordPage}`

const submitRecoverForm = async () => {
  if (!resetFormRef.value) return
  if (isLoading.value) return
  try {
    await resetFormRef.value.validate()
    console.log('表单校验通过，发送请求')
    isLoading.value = true
    const requestData = {
      email: resetForm.email,
      frontend_url: frontendUrl,
    }
    await axios.post('/api/v1/users/reset/send', requestData)
    isLoading.value = false
    resetFormRef.value.resetFields()
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
            ref="resetFormRef"
            class="login-form"
            :model="resetForm"
            :rules="resetFormRules"
            label-width="80px"
            label-position="right"
            status-icon
          >

            <el-form-item class="login-group" label="邮箱" prop="email">
              <el-input
                class="login-info"
                size="large"
                v-model="resetForm.email"
                placeholder="请输入邮箱"
              >
                <template #prefix
                ><el-icon><Message /></el-icon
                ></template>
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
