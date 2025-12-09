<!-- src/views/home/MainLayout.vue -->
<script setup>
import { ref, reactive, onMounted, provide } from 'vue'
import { useRoute } from 'vue-router'
import { Tickets, Memo, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import apiClient from '../../request/axios.js'

const fetchUserInfoAPI = (userId) => apiClient.get(`/api/v1/users/${userId}`)

const userInfo = reactive({ nickname: '', email: '', role: 'user' })
const isUserLoading = ref(true)
const route = useRoute() // 获取当前路由信息，用于菜单高亮

onMounted(async () => {
  isUserLoading.value = true
  try {
    const storedUserId = localStorage.getItem('user_id')
    if (!storedUserId) {
      ElMessage.error('无法获取用户信息，请重新登录')
      return
    }
    const res = await fetchUserInfoAPI(storedUserId)
    if (res && res.data?.message === 'success' && res.data.data) {
      const { nickname, email, role } = res.data.data
      Object.assign(userInfo, {
        nickname,
        email,
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

// 使用 Provide/Inject 将用户信息和管理员身份检查函数传递给所有子组件
// 这样子组件就不需要重复获取用户信息了，非常高效！
provide('userInfo', userInfo)
provide('isAdmin', () => userInfo.role === 'admin')
</script>

<template>
  <el-container class="main-layout-container">
    <!-- 左侧导航栏 -->
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
      <!-- 顶部 Header -->
      <el-header>
        <div class="header-content">
          <span class="page-title">{{
            route.name === 'dataset-list' ? '数据集列表' : '申请列表'
          }}</span>
          <el-skeleton :rows="1" animated :loading="isUserLoading" style="width: 250px">
            <template #default>
              <div class="user-info">
                <el-icon><User /></el-icon> {{ userInfo.nickname }}
                <el-divider direction="vertical" />
                <el-tag :type="userInfo.role === 'admin' ? 'success' : 'info'" size="small">{{
                  userInfo.role === 'admin' ? '管理员' : '用户'
                }}</el-tag>
              </div>
            </template>
          </el-skeleton>
        </div>
      </el-header>

      <!-- 主内容区 -->
      <el-main>
        <router-view />
        <!-- 子页面会在这里渲染 -->
      </el-main>
    </el-container>
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
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.el-main {
  background-color: #f0f2f5;
}
</style>
