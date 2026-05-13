<script setup>
import {ref, onMounted, inject} from 'vue'
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
  Memo,
  DataAnalysis,
  Tickets,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import apiClient from '@/request/axios.js'
import '@/assets/styles/universal.css'
import { formatDateTime, formatFileSize } from '@/utils/format.js'
import {DOWNLOAD_FILE_LIMIT_MB} from "@/utils/constrains.js";
import ListPagination from "@/components/ListPagination.vue";
import { useTableFilter } from "@/hooks/useTable.js"
import TableHeaderFilter from "@/components/TableHeaderFilter.vue"

const DOWNLOAD_FILE_LIMIT_BYTES = DOWNLOAD_FILE_LIMIT_MB * 1024 * 1024

// --- API 函数 ---
const fetchPublicFilesAPI = (page, size) => apiClient.get('/api/v1/files/public', { params: { page, size } })

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
const isUploading = ref(false)

// --- 辅助函数 ---
const FILE_TYPE_MAP = [
  { exts: ['pdf'],                                           icon: Document,     cls: 'file-icon-pdf'     },
  { exts: ['doc', 'docx', 'wps', 'odt', 'rtf'],             icon: Memo,         cls: 'file-icon-word'    },
  { exts: ['xls', 'xlsx', 'et', 'ods', 'csv'],              icon: DataAnalysis, cls: 'file-icon-excel'   },
  { exts: ['ppt', 'pptx', 'dps', 'odp'],                    icon: Tickets,      cls: 'file-icon-ppt'     },
  { exts: ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'svg', 'webp'], icon: Picture,  cls: 'file-icon-image'   },
  { exts: ['mp4', 'avi', 'mov', 'mkv', 'webm'],             icon: VideoPlay,    cls: 'file-icon-video'   },
  { exts: ['mp3', 'wav', 'ogg', 'flac'],                    icon: Headset,      cls: 'file-icon-audio'   },
  { exts: ['zip', 'rar', '7z', 'tar', 'gz', 'tgz'],        icon: Box,          cls: 'file-icon-zip'     },
]

const getFileType = (fileName) => {
  if (!fileName) return { icon: Document, cls: 'file-icon-default' }
  const ext = fileName.split('.').pop().toLowerCase()
  return FILE_TYPE_MAP.find(t => t.exts.includes(ext)) || { icon: Document, cls: 'file-icon-default' }
}

const getFileIcon = (fileName) => getFileType(fileName).icon
const getFileIconClass = (fileName) => getFileType(fileName).cls

const normalizeFile = (item) => ({
  fileID: item.fileID || item.id,
  name: item.name || '未命名文件',
  size: Number(item.size ?? 0),
  createdAt: formatDateTime(item.createdAt),
})



const {
  loading: isLoading,
  data: fileList,
  pagination,
  filterForm,
  sortState,
  isFilterActive,
  loadData: loadFiles,
  handleSortChange,
  handleFilterChange,
  resetFilter,
  handlePageChange,
} = useTableFilter(
  (params) => apiClient.get('/api/v1/files/public', { params }),
  {
    normalize: normalizeFile,
    dataKey: 'files',
    defaultSort: { prop: 'createdAt', order: 'descending' },
    filterSchema: { name: '', date: [] },
    pageSize: 10,
    syncUrl: true,
  }
)

// --- 方法 ---

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
        <span>下载中心</span>
        <div class="header-actions">
          <el-button
            v-if="isFilterActive"
            class="reset-filter-btn"
            plain
            size="small"
            @click="resetFilter"
          >重置筛选</el-button>
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

    <div v-loading="isLoading">
      <el-table
        :data="fileList"
        style="width: 100%"
      >
        <el-table-column label="文件名" min-width="300" prop="name">
          <template #header>
            <table-header-filter
              label="文件名"
              type="input"
              v-model="filterForm.name"
              @change="handleFilterChange"
              sort-prop="name"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">
            <div class="file-name-cell">
              <el-icon class="file-icon" :class="getFileIconClass(row.name)"><component :is="getFileIcon(row.name)" /></el-icon>
              <span class="file-name-text table-name-text" :title="row.name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="大小" width="120">
          <template #header>
            <table-header-filter
              label="大小"
              sort-prop="size"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">{{ formatFileSize(row.size) }}</template>
        </el-table-column>

        <el-table-column label="上传时间" width="180">
          <template #header>
            <table-header-filter
              label="上传时间"
              type="daterange"
              v-model="filterForm.date"
              @change="handleFilterChange"
              sort-prop="createdAt"
              :sort-state="sortState"
              @sort="handleSortChange"
            />
          </template>
          <template #default="{ row }">
            <span class="table-time-text">{{ formatDateTime(row.createdAt) }}</span>
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
      <list-pagination
        v-model="pagination.currentPage"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="handlePageChange"
      />
    </div>
  </el-card>
</template>

<style scoped>
.upload-btn-wrapper {
  display: inline-flex;
}
</style>
