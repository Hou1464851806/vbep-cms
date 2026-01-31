<script setup>
import { ref, onMounted, getCurrentInstance, onBeforeUnmount, nextTick, inject, computed } from 'vue'
import {
  User,
  Files,
  Document,
  Check,
  CaretTop,
  CaretBottom,
  Minus,
  Refresh,
  DataLine,
} from '@element-plus/icons-vue'
import apiClient from '@/request/axios.js'
import { ElMessage } from 'element-plus'
import { formatFileSize, formatDateTime } from '@/utils/format.js'

const { proxy } = getCurrentInstance()
const echarts = proxy.$echarts

const userInfo = inject('userInfo')
const isSuperAdmin = computed(() => userInfo?.roleId === 2)

const storageChartRef = ref(null)
const submissionChartRef = ref(null)
const userChartRef = ref(null)
let storageChart = null
let submissionChart = null
let userChart = null

const loading = ref(false)
const refreshing = ref(false)
const stats = ref({
  user: {
    totalUsers: 0,
    commonUsers: 0,
    adminUsers: 0,
    superAdminUsers: 0
  },
  dataset: {
    totalDatasets: 0,
  },
  application: {
    totalApplications: 0,
    approvedApplications: 0,
    pendingApplications: 0,
    rejectedApplications: 0,
    returnedApplications: 0,
  },
  audit: {
    submissions: 0,
    auditApproved: 0,
    auditRejected: 0,
    auditReturned: 0,
    auditPending: 0,
  },
  file: {
    files: 0,
    datasetFiles: 0,
    publicFiles: 0,
    applicationFiles: 0,
  },
  storage: {
    totalFileStorage: 0,
    publicFileStorage: 0,
    datasetFileStorage: 0,
    applicationFileStorage: 0
  },
  userGrowth: 0,
  approvalRate: 0,
  history: []
})

const fetchStats = async () => {
  loading.value = true
  try {
    const res = await apiClient.get('/api/v1/statistics')
    if (res.data && res.data.data) {
      const list = res.data.data
      if (!Array.isArray(list) || list.length === 0) return

      const latest = list[0]
      // 前端计算 userGrowth
      let growth = 0
      if (list.length >= 2) {
        // list[0] 是最新，list[1] 是上一次
        growth = list[0].user.commonUsers - list[1].user.commonUsers
      }
      // 历史数据用于图表 (需要反转为时间正序)
      const history = [...list].reverse()
      stats.value = { ...latest, userGrowth: growth, history: history }
      if (stats.value.application.totalApplications > 0) {
        stats.value.approvalRate = stats.value.application.approvedApplications / stats.value.application.totalApplications
      } else {
        stats.value.approvalRate = 0
      }
      nextTick(() => {
        initCharts()
      })
    }
  } catch (error) {
    if (error.response && error.response.status === 404) {
      // 暂无数据，保持默认值
      return
    }
    console.error(error)
    ElMessage.error('获取统计数据失败')
  } finally {
    loading.value = false
  }
}

const handleRefresh = async () => {
  refreshing.value = true
  try {
    await apiClient.get('/api/v1/statistics/refresh')
    ElMessage.success('统计数据已刷新')
    await fetchStats()
  } catch (error) {
    console.error(error)
    ElMessage.error(error.response?.data?.message || '刷新失败')
  } finally {
    refreshing.value = false
  }
}

const initCharts = () => {
  // 存储空间饼图
  if (storageChartRef.value) {
    if (storageChart) storageChart.dispose()
    storageChart = echarts.init(storageChartRef.value)
    
    const totalStorage = stats.value.storage.datasetFileStorage + stats.value.storage.applicationFileStorage + stats.value.storage.publicFileStorage
    const storageData = totalStorage > 0 ? [
      { value: stats.value.storage.datasetFileStorage, name: '数据集附件', itemStyle: { color: '#409EFF' } },
      { value: stats.value.storage.applicationFileStorage, name: '申请附件', itemStyle: { color: '#E6A23C' } },
      { value: stats.value.storage.publicFileStorage, name: '公开下载文件', itemStyle: { color: '#67C23A' } }
    ] : [{ value: 1, name: '暂无数据', itemStyle: { color: '#EBEEF5' } }]

    storageChart.setOption({
      tooltip: {
        show: totalStorage > 0,
        trigger: 'item',
        formatter: function (params) {
          return `${params.name}: ${formatFileSize(params.value)} (${params.percent}%)`
        }
      },
      legend: { show: totalStorage > 0, top: '5%', left: 'center' },
      series: [{
        name: '存储占用',
        type: 'pie',
        radius: ['30%', '55%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
        label: { show: false, position: 'center' },
        emphasis: { label: { show: true, fontSize: 16, fontWeight: 'bold' } },
        labelLine: { show: false },
        data: storageData
      }]
    })
  }

  // 提交状态饼图
  if (submissionChartRef.value) {
    if (submissionChart) submissionChart.dispose()
    submissionChart = echarts.init(submissionChartRef.value)

    const totalAudits = stats.value.audit.submissions
    // 使用 Audit 维度的统计数据
    const submissionData = totalAudits > 0 ? [
      { value: stats.value.audit.auditApproved, name: '已通过', itemStyle: { color: '#67C23A' } },
      { value: stats.value.audit.auditRejected, name: '已拒绝', itemStyle: { color: '#F56C6C' } },
      { value: stats.value.audit.auditReturned, name: '被退回', itemStyle: { color: '#E6A23C' } },
      { value: stats.value.audit.auditPending, name: '待审核', itemStyle: { color: '#909399' } }
    ] : [{ value: 1, name: '暂无数据', itemStyle: { color: '#EBEEF5' } }]

    submissionChart.setOption({
      tooltip: { show: totalAudits > 0, trigger: 'item' },
      legend: { show: totalAudits > 0, top: '5%', left: 'center' },
      series: [{
        name: '申请状态',
        type: 'pie',
        radius: '50%',
        data: submissionData,
        emphasis: {
          itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
        }
      }]
    })
  }

  // 用户增长折线图
  if (userChartRef.value) {
    if (userChart) userChart.dispose()
    userChart = echarts.init(userChartRef.value)

    const history = stats.value.history || []
    const dates = history.map(item => {
      // 格式化日期，仅显示月-日
      const d = new Date(item.created_at)
      return `${d.getMonth() + 1}/${d.getDate()}`
    })
    const counts = history.map(item => item.user.commonUsers)

    userChart.setOption({
      tooltip: {
        trigger: 'axis',
        formatter: '{b}<br />用户数: {c}'
      },
      grid: {
        top: '10%',
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: dates,
        axisLine: { show: false },
        axisTick: { show: false }
      },
      yAxis: {
        type: 'value',
        splitLine: { lineStyle: { type: 'dashed' } }
      },
      series: [{
        data: counts,
        type: 'line',
        smooth: true,
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 158, 255, 0.5)' },
            { offset: 1, color: 'rgba(64, 158, 255, 0.01)' }
          ])
        },
        itemStyle: { color: '#409EFF' },
        lineStyle: { width: 3 }
      }]
    })
  }
}

const handleResize = () => {
  storageChart?.resize()
  submissionChart?.resize()
  userChart?.resize()
}

onMounted(() => {
  fetchStats()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  storageChart?.dispose()
  submissionChart?.dispose()
  userChart?.dispose()
})

const formatRate = (rate) => {
  return (rate * 100).toFixed(2) + '%'
}
</script>

<template>
  <div class="statistic-container" v-loading="loading">
    <div class="statistic-header">
      <span class="update-time" v-if="stats.created_at">最后更新: {{ formatDateTime(stats.created_at) }}</span>
      <div class="header-actions" v-if="isSuperAdmin">
        <el-button :icon="Refresh" circle @click="handleRefresh" :loading="refreshing" title="强制刷新" />
      </div>
    </div>
    <el-row :gutter="20">
      <!-- 基本信息 (原管理员统计 + 数据集数量) -->
      <el-col :xs="12" :sm="12" :lg="6" class="responsive-mb">
        <el-card shadow="hover" class="stat-card small-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><DataLine /></el-icon>基本信息</span>
            </div>
          </template>
          <div class="overview-content">
            <div class="overview-item">
              <div class="label">管理员(超级管理员)</div>
              <div class="val">{{ stats.user.adminUsers + stats.user.superAdminUsers }}({{ stats.user.superAdminUsers }})</div>
            </div>
            <div class="overview-divider"></div>
            <div class="overview-item">
              <div class="label">数据集</div>
              <div class="val">{{ stats.dataset.totalDatasets }}</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 申请概览 -->
      <el-col :xs="12" :sm="12" :lg="6" class="responsive-mb">
        <el-card shadow="hover" class="stat-card small-card">
              <template #header>
                <div class="card-header">
                  <span><el-icon><Document /></el-icon>申请</span>
                </div>
              </template>
              <div class="overview-content">
                <div class="overview-item">
                  <div class="label">总申请</div>
                  <div class="val">{{ stats.application.totalApplications }}</div>
                </div>
                <div class="overview-divider"></div>
                <div class="overview-item">
                  <div class="label">已通过</div>
                  <div class="val success-text">{{ stats.application.approvedApplications }}</div>
                </div>
                <div class="overview-divider"></div>
                <div class="overview-item">
                  <div class="label">通过率</div>
                  <div class="val">{{ formatRate(stats.approvalRate) }}</div>
                </div>
              </div>
        </el-card>
      </el-col>

      <!-- 用户统计 (普通用户 + 趋势) -->
      <el-col :xs="24" :lg="12" class="responsive-mb">
        <el-card shadow="hover" class="stat-card user-stat-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><User /></el-icon>用户增长趋势</span>
            </div>
          </template>
          <div class="user-stat-content">
            <div class="user-stat-info">
              <div class="card-value">{{ stats.user.commonUsers }}</div>
              <div class="trend-indicator">
                <span class="trend-label">较上期</span>
                <span v-if="stats.userGrowth > 0" class="trend-up">
                  <el-icon><CaretTop /></el-icon> {{ stats.userGrowth }}
                </span>
                <span v-else-if="stats.userGrowth < 0" class="trend-down">
                  <el-icon><CaretBottom /></el-icon> {{ Math.abs(stats.userGrowth) }}
                </span>
                <span v-else class="trend-flat">
                  <el-icon><Minus /></el-icon> 持平
                </span>
              </div>
            </div>
            <div class="mini-chart-container" ref="userChartRef"></div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 第二行：图表区域 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <!-- 数据集与存储合并卡片 -->
      <el-col :xs="24" :lg="12" class="responsive-mb">
        <el-card shadow="hover" class="stat-card chart-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Files /></el-icon> 存储</span>
            </div>
          </template>
          <div class="merged-card-content">
            <!-- 左侧：数字信息 -->
            <div class="merged-left">
              <div class="merged-item">
                <div class="merged-label">文件总数</div>
                <div class="merged-value">{{ stats.file.files }}</div>
              </div>
            </div>
            <!-- 右侧：饼图 -->
            <div class="merged-right">
              <div class="chart-container" ref="storageChartRef"></div>
              <div class="total-storage-label">总占用: {{ formatFileSize(stats.storage.totalFileStorage) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 审批分布 (原申请状态分布) -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="stat-card chart-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Check /></el-icon>申请提交情况</span>
            </div>
          </template>
          <div class="merged-card-content">
            <div class="merged-left">
              <div class="merged-item">
                <div class="merged-label">提交总数</div>
                <div class="merged-value">{{ stats.audit.submissions }}</div>
              </div>
            </div>
            <div class="merged-right">
              <div class="chart-container" ref="submissionChartRef"></div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.statistic-container {
  padding: 20px;
}
.statistic-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}
.update-time {
  font-size: 14px;
  color: #909399;
}
.header-actions {
  margin-left: auto;
}
.stat-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.card-header {
  display: flex;
  align-items: center;
  font-weight: bold;
  color: #606266;
}
.card-header .el-icon {
  margin-right: 8px;
}
.card-value {
  font-size: 36px;
  font-weight: bold;
  color: #409eff;
  text-align: center;
  margin: 10px 0;
}
.mini-value {
  font-size: 28px;
  margin: 5px 0;
}

.card-sub {
  display: flex;
  justify-content: space-around;
  font-size: 12px;
  color: #909399;
  margin-top: 10px;
}
.sub-item {
  margin: 0 5px;
}

/* User Card Specifics */
.user-stat-content {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 20px;
}
.user-stat-info {
  display: flex;
  flex-direction: column;
}
.trend-indicator {
  margin-top: 10px;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 5px;
}
.trend-label {
  color: #909399;
}
.trend-up {
  color: #F56C6C;
  display: flex;
  align-items: center;
}
.trend-down {
  color: #67C23A;
  display: flex;
  align-items: center;
}
.trend-flat {
  color: #909399;
  display: flex;
  align-items: center;
}
.mini-chart-container {
  flex: 1;
  height: 100%;
  margin-left: 20px;
}

/* Small Cards */

/* Overview Content */
.overview-content {
  display: flex;
  justify-content: space-around;
  align-items: center;
  height: 100%;
  padding: 20px 0;
}
.overview-item {
  text-align: center;
}
.overview-item .label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}
.overview-item .val {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
}
.overview-item .val.success-text {
  color: #67c23a;
}
.overview-item .val.warning-text {
  color: #E6A23C;
}
.overview-divider {
  width: 1px;
  height: 40px;
  background-color: #ebeef5;
}

/* Chart Cards */
.chart-card {
  height: 350px;
}
.chart-container {
  width: 100%;
  height: 230px;
}
.total-storage-label {
  text-align: center;
  font-size: 14px;
  color: #606266;
  margin-top: -15px;
  margin-bottom: 15px;
  font-weight: bold;
}

/* Merged Card Styles */
.merged-card-content {
  display: flex;
  height: 100%;
}
.merged-left {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  border-right: 1px solid #ebeef5;
}
.merged-right {
  flex: 3;
  position: relative;
}
.merged-item {
  text-align: center;
  margin: 15px 0;
}
.merged-label {
  font-size: 14px;
  color: #909399;
}
.merged-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}
.merged-divider {
  width: 60%;
  height: 1px;
  background-color: #ebeef5;
}

@media (max-width: 1199px) {
  .responsive-mb {
    margin-bottom: 20px;
  }
}
</style>