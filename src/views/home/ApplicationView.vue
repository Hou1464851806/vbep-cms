<!-- src/views/application/ApplicationList.vue -->
<script setup>
import { ref, onMounted, inject } from 'vue'
import { Check, Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// --- 模拟 API & 状态 ---
const fetchApplicationListAPI = () =>
  new Promise((res) =>
    setTimeout(
      () =>
        res({
          code: 200,
          data: [
            {
              id: 'app-01',
              applicant: '普通用户A',
              datasetName: '用户行为日志分析',
              status: 'pending',
              applyDate: '2023-05-10',
            },
            {
              id: 'app-02',
              applicant: '研发部小王',
              datasetName: '2023年第一季度销售数据',
              status: 'approved',
              applyDate: '2023-05-09',
            },
          ],
        }),
      300,
    ),
  )

const isAdmin = inject('isAdmin')
const applicationList = ref([])
const isLoading = ref(true)

onMounted(async () => {
  isLoading.value = true
  try {
    const res = await fetchApplicationListAPI()
    if (res.code === 200) applicationList.value = res.data
  } catch (e) {
    ElMessage.error('申请列表加载失败')
  } finally {
    isLoading.value = false
  }
})

// --- 事件处理 ---
const handleApprove = (row) => {
  row.status = 'approved'
  ElMessage.success(`已批准 ${row.applicant} 的申请`)
  // 在这里调用真实的API
}
const handleReject = (row) => {
  row.status = 'rejected'
  ElMessage.warning(`已驳回 ${row.applicant} 的申请`)
  // 在这里调用真实的API
}
</script>

<template>
  <el-card shadow="never">
    <template #header><span>数据集申请列表</span></template>

    <el-skeleton :rows="5" animated :loading="isLoading">
      <el-table :data="applicationList" style="width: 100%">
        <el-table-column prop="applicant" label="申请人" />
        <el-table-column prop="datasetName" label="申请的数据集" />
        <el-table-column prop="applyDate" label="申请日期" />
        <el-table-column label="状态">
          <template #default="scope">
            <el-tag v-if="scope.row.status === 'pending'" type="warning">待审批</el-tag>
            <el-tag v-else-if="scope.row.status === 'approved'" type="success">已批准</el-tag>
            <el-tag v-else type="danger">已驳回</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" v-if="isAdmin()">
          <template #default="scope">
            <template v-if="scope.row.status === 'pending'">
              <el-button size="small" type="success" :icon="Check" @click="handleApprove(scope.row)"
                >批准</el-button
              >
              <el-button size="small" type="danger" :icon="Close" @click="handleReject(scope.row)"
                >驳回</el-button
              >
            </template>
          </template>
        </el-table-column>
      </el-table>
    </el-skeleton>
  </el-card>
</template>
