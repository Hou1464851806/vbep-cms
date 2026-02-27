<script setup>
import { ref, onMounted, inject } from 'vue'
import {
  Document,
  Upload,
  Download,
  Delete,
  Refresh,
  Picture,
  VideoPlay,
  Headset,
  Box,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize } from '@/utils/format.js'
import {DOWNLOAD_FILE_LIMIT_MB} from "@/utils/constrains.js";

const DOWNLOAD_FILE_LIMIT_BYTES = DOWNLOAD_FILE_LIMIT_MB * 1024 * 1024

// --- API 函数 ---
const fetchPublicFilesAPI = () => apiClient.get('/api/v1/files/public')

const uploadPublicFileAPI = (file) => {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/api/v1/files/public', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

const deletePublicFileAPI = (fileID) => apiClient.delete(`/api/v1/files/public/${fileID}`)

const downloadPublicFileAPI = (fileID) =>
  apiClient.get(`/api/v1/files/public/${fileID}/download`, { responseType: 'blob' })

// --- 状态定义 ---
const isAdmin = inject('isAdmin')
const fileList = ref([])
const isLoading = ref(false)
const isUploading = ref(false)

// --- 辅助函数 ---
const getFileIcon = (fileName) => {
  if (!fileName) return Document
  const ext = fileName.split('.').pop().toLowerCase()

  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'svg', 'webp']
  const videoExts = ['mp4', 'avi', 'mov', 'mkv', 'webm']
  const audioExts = ['mp3', 'wav', 'ogg', 'flac']
  const archiveExts = ['zip', 'rar', '7z', 'tar', 'gz']

  if (imageExts.includes(ext)) return Picture
  if (videoExts.includes(ext)) return VideoPlay
  if (audioExts.includes(ext)) return Headset
  if (archiveExts.includes(ext)) return Box

  return Document
}

// --- 方法 ---
const loadFiles = async () => {
  isLoading.value = true
  try {
    const res = await fetchPublicFilesAPI()
    if (res.status === 200) {
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

  if (file.size > DOWNLOAD_FILE_LIMIT_BYTES) {
    ElMessage.error(`文件 "${file.name}" 超过 ${formatFileSize(DOWNLOAD_FILE_LIMIT_BYTES)} 限制，无法上传`)
    return
  }

  isUploading.value = true
  try {
    const res = await uploadPublicFileAPI(file)
    if (res.status === 200) {
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
        if (res.status === 200) {
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
        <span v-if="isAdmin()">下载中心（上传文件大小不能超过 {{ formatFileSize(DOWNLOAD_FILE_LIMIT_BYTES) }}）</span>
        <span v-else>下载中心</span>
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

    <el-table :data="fileList" style="width: 100%" v-loading="isLoading">
      <el-table-column label="文件名" min-width="300">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon class="file-icon">
              <component :is="getFileIcon(row.name)" />
            </el-icon>
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
.upload-btn-wrapper {
  display: inline-flex;
}
</style>
