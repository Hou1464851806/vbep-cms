<script setup>
import { ref, onMounted, inject } from 'vue'
import {
  Document,
  Upload,
  Download,
  Delete,
  Refresh,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize } from '@/utils/format.js'

// --- API 函数 ---
const fetchPublicFilesAPI = () => apiClient.get('/api/v1/files/public')

const uploadPublicFileAPI = (file) => {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/api/v1/files/public', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

const deletePublicFileAPI = (fileId) => apiClient.delete(`/api/v1/files/public/${fileId}`)

const downloadPublicFileAPI = (fileId) =>
  apiClient.get(`/api/v1/files/public/${fileId}/download`, { responseType: 'blob' })

// --- 状态定义 ---
const isAdmin = inject('isAdmin')
const fileList = ref([])
const isLoading = ref(false)
const isUploading = ref(false)

// --- 方法 ---
const loadFiles = async () => {
  isLoading.value = true
  try {
    const res = await fetchPublicFilesAPI()
    if (res.data?.message === 'success') {
      fileList.value = res.data.files || []
    } else {
      ElMessage.error(res.data?.message || '加载文件列表失败')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('加载文件列表失败')
  } finally {
    isLoading.value = false
  }
}

const handleUpload = async (options) => {
  const { file } = options
  
  // 简单的大小检查，例如 500MB
  const isLt500M = file.size / 1024 / 1024 < 500
  if (!isLt500M) {
    ElMessage.error('文件大小不能超过 500MB')
    return
  }

  isUploading.value = true
  try {
    const res = await uploadPublicFileAPI(file)
    if (res.data?.message === 'success') {
      ElMessage.success(`文件 "${file.name}" 上传成功`)
      loadFiles()
    } else {
      ElMessage.error(res.data?.message || '上传失败')
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('上传失败')
  } finally {
    isUploading.value = false
  }
}

const handleDownload = async (row) => {
  try {
    const res = await downloadPublicFileAPI(row.fileID)
    if (!res || !res.data) {
      ElMessage.error('下载失败')
      return
    }
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', row.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error(error)
    ElMessage.error('下载出错')
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定要删除文件 "${row.name}" 吗？`, '警告', {
    confirmButtonText: '确定删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(async () => {
      try {
        const res = await deletePublicFileAPI(row.fileID)
        if (res.data?.message === 'success') {
          ElMessage.success('删除成功')
          loadFiles()
        } else {
          ElMessage.error(res.data?.message || '删除失败')
        }
      } catch (error) {
        console.error(error)
        ElMessage.error('删除失败')
      }
    })
    .catch(() => {})
}

onMounted(() => {
  loadFiles()
})
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>下载中心</span>
        <div class="header-actions">
          <el-button :icon="Refresh" circle @click="loadFiles" :loading="isLoading" title="刷新列表" />
          <el-upload
            v-if="isAdmin()"
            action="#"
            :show-file-list="false"
            :http-request="handleUpload"
            :disabled="isUploading"
            class="upload-btn-wrapper"
          >
            <el-button type="primary" :icon="Upload" :loading="isUploading">上传文件</el-button>
          </el-upload>
        </div>
      </div>
    </template>

    <el-table :data="fileList" stripe style="width: 100%" v-loading="isLoading">
      <el-table-column label="文件名" min-width="300">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon class="file-icon"><Document /></el-icon>
            <span class="file-name-text" :title="row.name">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="大小" width="120">
        <template #default="{ row }">
          {{ formatFileSize(row.size) }}
        </template>
      </el-table-column>

      <el-table-column label="上传时间" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.createdAt) }}
        </template>
      </el-table-column>

      <el-table-column label="操作" width="180" align="center" fixed="right">
        <template #default="{ row }">
          <el-button
            type="primary"
            link
            :icon="Download"
            @click="handleDownload(row)"
          >
            下载
          </el-button>
          <el-button
            v-if="isAdmin()"
            type="danger"
            link
            :icon="Delete"
            @click="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.upload-btn-wrapper {
  display: inline-flex;
}
.file-name-cell {
  display: flex;
  align-items: center;
}
.file-icon {
  margin-right: 8px;
  font-size: 16px;
  color: #909399;
}
.file-name-text {
  font-weight: 500;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>