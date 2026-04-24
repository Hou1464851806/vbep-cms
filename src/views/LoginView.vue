<script setup>
import { Lock, User } from '@element-plus/icons-vue'
import { reactive, ref } from 'vue'

import logoSrc from '@/assets/img/logo.png'
import qrCodeSrc from '@/assets/img/QRcode.jpg'

import thirdPartyQrSrc from '@/assets/img/qr-code.svg'
import thirdPartyChatSrc from '@/assets/img/wechat.png'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import apiClient, { setAuthToken } from '@/request/axios.js'
import '@/assets/styles/login.css'
import sha256 from "crypto-js/sha256.js"

const route = useRoute()
const router = useRouter()

const saveUserId = (userId) => {
  if (userId === undefined || userId === null) return
  try {
    localStorage.setItem('userID', String(userId))
  } catch (error) {
    console.error('Failed to persist user id', error)
  }
}

const loginFormRef = ref()
const loginForm = reactive({
  userInfo: '',
  password: '',
})
const isLoading = ref(false)
const showQrLogin = false
const showOtherLoginMethods = false

const loginRules = reactive({
  userInfo: [{ required: true, message: '请输入邮箱', trigger: 'submit' }],
  password: [{ required: true, message: '请输入密码', trigger: 'submit' }],
})

const submitLoginFrom = async () => {
  if (!loginFormRef.value) return // 防御性编程，确保form实例存在
  if (isLoading.value) return // 防止重复提交
  try {
    // 步骤 A: 对整个表单进行校验
    await loginFormRef.value.validate()
    isLoading.value = true
    const passwordHash = sha256(loginForm.password).toString()
    // 步骤 B: 准备并发送 API 请求
    const requestData = {
      userInfo: loginForm.userInfo,
      password: passwordHash,
    }
    const response = await apiClient.post('/api/v1/users/login', requestData, {
      skipAuthRedirect: true,
      skipAuthRefresh: true,
    })
    // 步骤 C: 登录成功处理
    console.log('登录成功，Token已写入本地存储', response.data)
    const token = response.data?.token
    if (token) {
      setAuthToken(token)
    } else {
      console.warn('登录响应中未找到 token')
    }
    saveUserId(response.data?.userID)
    ElMessage({ message: '登录成功！', type: 'success', duration: 1000 })

    const redirectPath = route.query.redirect || '/home'
    // 跳转到主页或用户仪表盘
    setTimeout(() => {
      router.replace(redirectPath)
    }, 1000)
  } catch (error) {
    // 步骤 D: 统一错误处理
    if (error && error.response) {
      // API 返回的错误 (例如：密码错误, 用户不存在)
      console.error('登录失败:', error.response.data)
      ElMessage.error(error.response.data.message || '用户名或密码错误')
    } else if (error instanceof Error) {
      // 网络错误或代码错误
      ElMessage.error('网络连接失败，请稍后重试')
    } else {
      // 表单校验未通过的错误
      console.log('表单校验失败:', error)
      ElMessage.error('请检查输入')
    }
  } finally {
    // 步骤 E: 无论成功或失败，都结束加载状态
    isLoading.value = false
  }
}
</script>

<template>
  <div class="main-container">
    <div class="login-card-wrapper">
      <div class="register-button-wrapper">
        <RouterLink to="/register" class="register-button">注册</RouterLink>
      </div>
      <div class="sso-banner">
        <div class="sso-logo">
          <img :src="logoSrc" alt="Logo" class="sso-logo-img" />
        </div>
      </div>
      <div class="sso-header">
        <span class="heading">欢迎使用VBEP数据平台</span>
      </div>
      <div class="login-container">
        <div v-if="showQrLogin" class="login-left-QR-code">
          <img :src="qrCodeSrc" alt="QR Code" class="qr-code-img" />
          <p class="qr-code-text">使用微信扫一扫登录</p>
          <p class="qr-code-subtext">“中央民族大学<br />中国兴边富民战略研究院”</p>
          <div class="qr-code-line"></div>
        </div>
        <div class="login-panel" :class="{ 'no-qr': !showQrLogin }">
          <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" class="login-form">
            <el-form-item class="login-group" prop="userInfo">
              <el-input
                class="login-info"
                size="large"
                v-model="loginForm.userInfo"
                placeholder="邮箱"
              >
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item class="login-group" prop="password">
              <el-input
                class="login-info"
                size="large"
                v-model="loginForm.password"
                placeholder="密码"
                type="password"
                show-password
              >
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item class="login-group login-button-item">
              <el-button class="login-btn" type="primary" @click="submitLoginFrom"
                >立刻登录</el-button
              >
            </el-form-item>
          </el-form>
          <div class="login-group footer-links">
            <RouterLink to="/reset" class="forgot-password">忘记密码？</RouterLink>
          </div>
          <template v-if="showOtherLoginMethods">
            <div class="login-group other-methods">
              <span class="line"></span>
              <span class="text">其他方式</span>
              <span class="line"></span>
            </div>
            <div class="third-login">
              <img :src="thirdPartyQrSrc" alt="Third Party QR" class="third-login-icon" />
              <img :src="thirdPartyChatSrc" alt="Third Party Chat" class="third-login-icon" />
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Login Container - 左右布局的核心区域 */
.login-container {
  gap: 60px; /* 二维码和表单之间的间距 */
}
/* QR Code Display Area */
.login-left-QR-code {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  width: 260px; /* 固定宽度 */
  padding-right: 30px; /* 与右侧表单的间距 */
}
.qr-code-img {
  width: 180px; /* 根据图片调整二维码大小 */
  height: 180px;
  background-color: #f0f0f0; /* 占位背景 */
  border: 1px solid #eee;
  border-radius: 8px;
  margin-bottom: 20px;
}
.qr-code-text {
  font-size: 14px;
  color: #181818;
  margin-bottom: 8px;
  text-align: center;
}
.qr-code-subtext {
  font-size: 12px;
  color: #181818;
  text-align: center;
  line-height: 1.5;
}
.qr-code-line {
  position: absolute;
  right: 0;
  top: 0;
  height: 75%; /* 与内容等高 */
  width: 1px;
  background-color: #eee;
}
/* Login Form Panel */
.login-panel {
  flex-grow: 0;
  width: 260px;
  display: flex;
  flex-direction: column;
  padding-left: 30px; /* 补偿 qr-code-line */
}
.login-panel.no-qr {
  padding-left: 0;
}

/* 找回账号密码 */
.footer-links {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 40px;
}
.forgot-password {
  font-size: 14px;
  color: #181818;
  text-decoration: none;
}
.forgot-password:hover {
  color: #4292c3;
  text-decoration: underline;
}
/* 其他方式分隔线 */
.other-methods {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  margin-bottom: 30px;
}
.other-methods .line {
  flex-grow: 1;
  height: 1px;
  background-color: #eee;
  max-width: 100px;
}
.other-methods .text {
  font-size: 14px;
  color: #181818;
  white-space: nowrap;
}
/* 第三方登录图标 */
.third-login {
  display: flex;
  justify-content: center;
  gap: 30px; /* 图标间距 */
}
.third-login-icon {
  width: 30px; /* 图标大小 */
  height: 30px;
  cursor: pointer;
}

@media (max-width: 768px) {
  .login-container {
    gap: 0;
  }
  .login-left-QR-code {
    display: none;
  }
  .login-panel {
    width: 100%;
    max-width: 360px;
    padding-left: 0;
  }
  .footer-links {
    margin-bottom: 20px;
  }
}
</style>
