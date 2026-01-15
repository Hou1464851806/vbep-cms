<template>
  <div class="settings-container">
    <el-alert
      v-if="isSimpleAdmin"
      title="只有超级管理员可以修改本页的设置项"
      type="warning"
      show-icon
      class="full-width-alert"
    />
    <!-- Global Settings Card -->
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>通用设置</span>
        </div>
      </template>
      <div class="info-list">
        <!-- Public Email Setting -->
        <div class="info-row">
          <div class="row-label">公邮地址</div>
          <div class="row-content">
            <span v-if="!editState.public_email">{{ settings.public_email || '未设置' }}</span>
            <el-input v-else v-model="tempSettings.public_email" size="small" />
          </div>
          <div class="row-action">
            <template v-if="canEdit">
              <el-button v-if="!editState.public_email" type="primary" link :icon="Edit" @click="startEdit('public_email')">
                修改
              </el-button>
              <template v-else>
                <el-button type="success" link :icon="Check" @click="saveField('public_email', '公邮地址')">确定</el-button>
                <el-button type="info" link :icon="Close" @click="cancelEdit('public_email')">取消</el-button>
              </template>
            </template>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Template Settings Card -->
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>模版设置</span>
        </div>
      </template>
      <div class="info-list">
        <!-- App File Setting -->
        <div class="info-row">
          <div class="row-label">模版文件</div>
          <div class="row-content">
            <span v-if="!editState.app_file">{{ getPublicFileName(settings.app_file) || '未设置' }}</span>
            <el-select v-else v-model="tempSettings.app_file" size="small" placeholder="请选择文件" clearable filterable style="width: 100%">
              <el-option
                v-for="file in publicFiles"
                :key="getFileId(file)"
                :label="file.name"
                :value="getFileId(file)"
              />
            </el-select>
          </div>
          <div class="row-action">
            <template v-if="canEdit">
              <el-button v-if="!editState.app_file" type="primary" link :icon="Edit" @click="startEdit('app_file')">
                修改
              </el-button>
              <template v-else>
                <el-button type="success" link :icon="Check" @click="saveField('app_file', '模版文件')">确定</el-button>
                <el-button type="info" link :icon="Close" @click="cancelEdit('app_file')">取消</el-button>
              </template>
            </template>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, inject, computed } from 'vue';
import axios from '@/request/axios';
import { ElMessage } from 'element-plus';
import '@/assets/styles/universal.css';
import { Edit, Check, Close } from '@element-plus/icons-vue';

const userInfo = inject('userInfo');
const canEdit = computed(() => userInfo?.roleId === 2);
const isSimpleAdmin = computed(() => userInfo?.roleId === 1);

const settings = ref({
  app_file: '',
  public_email: '',
});
const publicFiles = ref([]);

// State for inline editing
const editState = reactive({
  app_file: false,
  public_email: false,
});

// Temporary storage for editing values
const tempSettings = reactive({
  app_file: '',
  public_email: '',
});

const fetchSettings = async () => {
  try {
    const response = await axios.get('/api/v1/setting');
    if (response.data.message === 'success') {
      settings.value = response.data.data;
    } else {
      ElMessage.error(response.data.message || '获取设置失败');
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '获取设置失败');
  }
};

const fetchPublicFiles = async () => {
  try {
    const response = await axios.get('/api/v1/files/public');
    if (response.data.message === 'success') {
      publicFiles.value = response.data.files || [];
    }
  } catch (error) {
    console.error('Failed to fetch public files:', error);
  }
};

const getFileId = (file) => {
  // 兼容各种可能的 ID 字段名
  return file.id || file.fileID || file.fileId || file.FileID || file.FileId || file.ID || file.fileid;
};

const getPublicFileName = (id) => {
  if (!id) return '';
  const file = publicFiles.value.find(f => getFileId(f) === id);
  return file ? file.name : '';
};

onMounted(async () => {
  await fetchPublicFiles();
  await fetchSettings();
});

// Start editing a field
const startEdit = async (field) => {
  
  // Refresh the file list when starting to edit the template file
  if (field === 'app_file') {
    await fetchPublicFiles();
  }

  // Copy current value to temp storage
  tempSettings[field] = settings.value[field] || '';
  // Set editing state to true
  editState[field] = true;
};

// Cancel editing a field
const cancelEdit = (field) => {
  editState[field] = false;
};

// Save a single field
const saveField = async (field, labelName) => {
  const newValue = tempSettings[field] || '';
  const oldValue = settings.value[field];

  // Diff check: if value hasn't changed, just exit edit mode
  if (newValue === oldValue) {
    cancelEdit(field);
    ElMessage.info('设置未发生变化');
    return;
  }

  // Basic validation (can be expanded)
  if (field === 'public_email' && newValue) { // Validate only if not empty
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(newValue)) {
      ElMessage.warning('请输入有效的邮箱地址');
      return;
    }
  }

  try {
    const payload = { [field]: newValue };
    const response = await axios.patch('/api/v1/setting', payload);
    if (response.data.message === 'success') {
      ElMessage.success(`${labelName}已更新`);
      // Update local state
      settings.value[field] = newValue;
      // Exit edit mode
      cancelEdit(field);
    } else {
      ElMessage.error(response.data.message || '保存失败');
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存失败');
  }
};
</script>

<style scoped>
.settings-container {
  display: grid;
  grid-template-columns: 1fr;
  gap: 20px;
}

@media (min-width: 1100px) {
  .settings-container {
    grid-template-columns: 1fr 1fr;
  }
}

.row-label {
  width: 150px;
}

.full-width-alert {
  grid-column: 1 / -1;
}
</style>
