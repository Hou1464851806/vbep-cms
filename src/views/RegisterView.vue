<script setup>
import { Lock, User, Iphone, Message } from '@element-plus/icons-vue'
import { reactive, ref } from 'vue'
import axios from 'axios'
import logoSrc from '@/assets/img/logo.png'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const router = useRouter()

const registerFormRef = ref()
const registerForm = reactive({
  nickname: '',
  phone: '',
  email: '',
  password: '',
})

const validatePassword = (rule, value, callback) => {
  if (value.length < 6 || value.length > 20) {
    return callback(new Error('密码长度不符合要求'))
  }
  let complexity = 0
  // 小写字母
  if (/[a-z]/.test(value)) {
    complexity++
  }
  // 大写字母
  if (/[A-Z]/.test(value)) {
    complexity++
  }
  // 数字
  if (/\d/.test(value)) {
    complexity++
  }
  // 特殊字符
  if (/[~!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(value)) {
    complexity++
  }

  if (complexity < 3) {
    return callback(new Error('密码未包含大写字母、小写字母、数字、特殊字符中的至少3项'))
  }

  callback()
}

const registerFormRules = reactive({
  nickname: [
    { required: true, message: '真实姓名不能为空', trigger: 'blur' }
  ],
  phone: [
    { required: true, message: '手机号不能为空', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号格式', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '邮箱不能为空', trigger: 'change' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'change' },
  ],
  password: [
    { required: true, message: '密码不能为空', trigger: 'change' },
    { validator: validatePassword, trigger: 'change' },
  ],
})

const isLoading = ref(false)

const submitRegisterForm = async () => {
  if (!registerFormRef.value) return
  if (isLoading.value) return
  try {
    await registerFormRef.value.validate()
    console.log('表单校验通过，发送请求')
    isLoading.value = true
    const requestData = {
      email: registerForm.email,
      phone: registerForm.phone,
      nickname: registerForm.nickname,
      password: registerForm.password,
    }
    const response = await axios.post('/api/v1/users/', requestData)
    console.log('注册成功: ', response.data)
    ElMessage({
      message: '注册成功',
      type: 'success',
      duration: 2000,
    })
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
        <div class="login-panel">
          <el-form
            ref="registerFormRef"
            class="login-form"
            :model="registerForm"
            :rules="registerFormRules"
            label-width="auto"
            status-icon
          >
            <el-form-item class="login-group" label="真实姓名" prop="nickname">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.nickname"
                placeholder="请输入真实姓名"
              >
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="手机号" prop="phone">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.phone"
                placeholder="请输入手机号"
              >
                <template #prefix>
                  <el-icon><Iphone /></el-icon>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="邮箱名" prop="email">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.email"
                placeholder="请输入邮箱"
              >
                <template #prefix>
                  <el-icon><Message /></el-icon>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item class="login-group" label="密码" prop="password">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.password"
                placeholder="请输入密码"
                type="password"
                show-password
              >
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            
            <div class="footer-links">
              <p class="forgot-password">
                密码长度应该为6到20个字符，且需包含大写字母、小写字母、数字、特殊字符中的至少3项
              </p>
            </div>
            <el-form-item class="login-group login-button-item">
              <el-button class="login-btn" type="primary" @click="submitRegisterForm"
                >立刻注册</el-button
              >
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
.main-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  margin: 0;
  background-color: #ffffff;
  overflow: hidden; 
}
.login-card-wrapper {
  background-color: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  width: 800px; 
  padding-bottom: 30px; 
  position: relative; 
  box-sizing: border-box;
}
.register-button-wrapper {
  position: absolute;
  top: 0;
  right: 0;
  width: 120px; 
  height: 120px; 
  overflow: hidden;
}
.register-button {
  position: absolute;
  top: 15px; 
  right: 15px; 
  font-size: 14px;
  text-decoration: none;
  color: #181818;
}
.register-button:hover {
  color: #4292c3;
  text-decoration: underline;
}
.sso-banner {
  padding: 30px 40px 0; 
  display: flex; 
  align-items: center;
  margin-bottom: 20px; 
}
.sso-logo img {
  height: 80px; 
}
.sso-header {
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
  margin-bottom: 40px; 
}
.sso-header .heading {
  font-size: 24px;
  font-weight: bold;
  color: #181818;
}
.login-container {
  display: flex;
  justify-content: center; 
  padding: 0 40px; 
  gap: 60px; 
}

/* Form Styles */
.login-panel {
  flex-grow: 1; 
  width: 300px; 
  display: flex;
  flex-direction: column;
  padding-left: 30px; 
}
.login-form {
  width: 100%;
}

:deep(.login-info.el-input .el-input__wrapper:hover) {
  border-color: #a4cef8; 
}
:deep(.login-info.el-input.is-focused .el-input__wrapper) {
  border-color: #409eff; 
}
:deep(.login-info.el-input .el-input__inner) {
  color: #333; 
  padding-left: 10px; 
}
:deep(.login-info.el-input .el-input__inner::placeholder) {
  color: #808080; 
}
:deep(.login-info.el-input .el-input__prefix) {
  color: #808080; 
  font-size: 18px; 
  margin-right: 8px; 
  display: flex; 
  align-items: center; 
}
:deep(.login-info.el-input .el-input__suffix) {
  color: #808080; 
  font-size: 18px; 
  display: flex; 
  align-items: center; 
}

.login-button-item {
  margin-top: 30px; 
}
.login-btn {
  --el-color-primary: #ef1c23;
  width: 100%;
  height: 48px;
  border-radius: 8px;
  background: #ef1c23;
  color: white;
  font-size: 16px;
  font-weight: bold;
  border: none;
  cursor: pointer;
  transition: all 0.3s ease;
}
.login-btn:hover {
  --el-button-hover-bg-color: #d1181f;
}

.footer-links {
  display: flex;
  justify-content: flex-end;
}
.forgot-password {
  font-size: 12px;
  color: #f32b2b;
}
</style>