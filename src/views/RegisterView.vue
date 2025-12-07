<script setup>
import { Lock, User } from '@element-plus/icons-vue'
import { reactive, ref } from 'vue'
import logoSrc from '../assets/img/logo.png'

const registerFormRef = ref()
const registerForm = reactive({
  userInfo: '',
  password: '',
})
const isFormValid = ref(false)

const validatePassword = (rule, value, callback) => {
  // if (!value) {
  //   return callback(new Error('密码不能为空'));
  // }
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
  if (/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(value)) {
    complexity++
  }

  if (complexity < 3) {
    return callback(new Error('密码未包含大写字母、小写字母、数字、特殊字符中的至少3项'))
  }

  callback()
}

const registerFormRules = reactive({
  userInfo: [
    { required: true, message: '邮箱不能为空', trigger: 'change' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'change' },
  ],
  password: [
    { required: true, message: '密码不能为空', trigger: 'change' },
    { validator: validatePassword, trigger: 'change' },
  ],
})

const submitRegisterForm = async (formEl) => {
  if (!formEl) return
  await registerFormRef.value.validate((valid) => {
    if (valid) {
      console.log('验证通过', registerForm)
    } else {
      console.log('验证失败')
      isFormValid.value = false
    }
  })
}
</script>
<template>
  <body class="main-container">
    <div class="login-card-wrapper">
      <div class="register-button-wrapper">
        <a href="/login" class="register-button">登录</a>
      </div>
      <div class="sso-banner">
        <div class="sso-logo">
          <img :src="logoSrc" alt="Logo" class="sso-logo-img" />
        </div>
      </div>
      <div class="sso-header">
        <span class="heading">VBEP注册</span>
      </div>
      <div class="login-container">
        <div class="login-panel">
          <!-- 对齐label -->
          <el-form
            ref="registerFormRef"
            class="login-form"
            :model="registerForm"
            :rules="registerFormRules"
            label-width="auto"
          >
            <el-form-item class="login-group" label="邮箱名" prop="userInfo">
              <el-input
                class="login-info"
                size="large"
                v-model="registerForm.userInfo"
                placeholder="请输入邮箱"
              >
                <template #prefix>
                  <el-icon><User /></el-icon>
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
              <p class="forgot-password">密码长度应该为6到20个字符，且需包含大写字母、小写字母、数字、特殊字符中的至少3项</p>
            </div>
            <el-form-item class="login-group login-button-item">
              <el-button
                class="login-btn"
                type="primary"
                @click="submitRegisterForm(registerFormRef)"
                >立刻注册</el-button
              >
            </el-form-item>
          </el-form>
        </div>
      </div>
    </div>
  </body>
</template>
<style>
/* 全局背景和居中 */
.main-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  margin: 0;
  background-color: #ffffff;
  overflow: hidden; /* 防止注册按钮溢出时的滚动条 */
}
/* 整个白色卡片的容器，为了更好地控制注册按钮的定位 */
.login-card-wrapper {
  background-color: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  width: 800px; /* 根据图片估算宽度 */
  /* height: 500px; 可以设置固定高度或让内容撑开 */
  padding-bottom: 30px; /* 留出底部空间 */
  position: relative; /* For sso-banner content positioning */
  box-sizing: border-box;
}
/* 注册按钮及对角线背景 */
.register-button-wrapper {
  position: absolute;
  top: 0;
  right: 0;
  width: 120px; /* 按钮区域的宽度 */
  height: 120px; /* 按钮区域的高度 */
  overflow: hidden;
}
.register-button {
  position: absolute;
  top: 15px; /* 根据图片调整文字位置 */
  right: 15px; /* 根据图片调整文字位置 */
  font-size: 14px;
  text-decoration: none;
  color: #181818;
}
.register-button:hover {
  color: #4292c3;
  text-decoration: underline;
}
/* SSO Banner - 包含 Logo */
.sso-banner {
  padding: 30px 40px 0; /* 根据图片调整内边距 */
  display: flex; /* 让 logo 更好地定位 */
  align-items: center;
  margin-bottom: 20px; /* 与标题的间距 */
}
.sso-logo img {
  height: 80px; /* 根据图片调整 logo 大小 */
}
/* SSO Header - 欢迎语 */
.sso-header {
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
  margin-bottom: 40px; /* 与登录区域的间距 */
}
.sso-header .heading {
  font-size: 24px;
  font-weight: bold;
  color: #181818;
}
/* Login Container - 左右布局的核心区域 */
.login-container {
  display: flex;
  justify-content: center; /* 确保内部内容居中，如果卡片区域小 */
  padding: 0 40px; /* 左右内边距 */
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
  flex-grow: 1; /* 占据剩余空间 */
  width: 300px; /* 估算宽度，flex-grow会使其适应 */
  display: flex;
  flex-direction: column;
  padding-left: 30px; /* 补偿 qr-code-line */
}
.login-form {
  width: 100%;
}

/* 悬停时的样式调整：边框变亮，阴影更明显 */
:deep(.login-info.el-input .el-input__wrapper:hover) {
  border-color: #a4cef8; /* 悬停时边框变浅蓝 */
}
/* 聚焦时的样式调整：使用 Element Plus 的主色强调聚焦，阴影更深 */
:deep(.login-info.el-input.is-focused .el-input__wrapper) {
  border-color: #409eff; /* 聚焦时边框变为 Element Plus 的主蓝色 */
}
/* 针对输入框内部的文字区域 */
:deep(.login-info.el-input .el-input__inner) {
  color: #333; /* 文本颜色，保持深色可见性 */
  padding-left: 10px; /* 图标与文本的间距 */
}
/* 针对 placeholder 文字的颜色：比图片中更深，更清晰 */
:deep(.login-info.el-input .el-input__inner::placeholder) {
  color: #808080; /* 中灰色，比 #999 更显眼，但又不是纯黑 */
}
/* 针对前缀图标的样式：颜色与 placeholder 保持一致，更清晰 */
:deep(.login-info.el-input .el-input__prefix) {
  color: #808080; /* 图标颜色，与占位符颜色一致 */
  font-size: 18px; /* 图标大小 */
  margin-right: 8px; /* 图标与文本的间距 */
  display: flex; /* 确保图标垂直居中 */
  align-items: center; /* 确保图标垂直居中 */
}

:deep(.login-info.el-input .el-input__suffix) {
  color: #808080; /* 图标颜色，与占位符颜色一致 */
  font-size: 18px; /* 图标大小 */
  display: flex; /* 确保图标垂直居中 */
  align-items: center; /* 确保图标垂直居中 */
}

.login-button-item {
  margin-top: 30px; /* 登录按钮与输入框的额外间距 */
}

.login-btn {
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
  background-color: #d1181f;
}

.footer-links {
  display: flex;
  justify-content: flex-end;
}
.forgot-password {
  font-size: 12px;
  color: #f32b2b;
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
</style>
