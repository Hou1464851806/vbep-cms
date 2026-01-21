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
          <span>申请模版设置</span>
          <el-tooltip placement="top">
            <template #content>
              这里的内容是在用户提交申请时的指示。<br />
              用户会按照“模版描述”中的要求填写“模版文件”。
            </template>
            <el-icon class="help-icon"><QuestionFilled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <div class="info-list">
        <!-- App File Setting -->
        <div class="info-row">
          <div class="row-label">模版文件</div>
          <div class="row-content">
            <span v-if="!editState.template_file">{{ getPublicFileNames(settings.template_file) || '未设置' }}</span>
            <el-select v-else v-model="tempSettings.template_file" multiple collapse-tags collapse-tags-tooltip size="small" placeholder="请选择文件" clearable filterable style="width: 100%">
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
              <el-button v-if="!editState.template_file" type="primary" link :icon="Edit" @click="startEdit('template_file')">
                修改
              </el-button>
              <template v-else>
                <el-button type="success" link :icon="Check" @click="saveField('template_file', '模版文件')">确定</el-button>
                <el-button type="info" link :icon="Close" @click="cancelEdit('template_file')">取消</el-button>
              </template>
            </template>
          </div>
        </div>

        <!-- Template Description Setting -->
        <div class="info-row">
          <div class="row-label">模版描述</div>
          <div class="row-content">
            <span>{{ formatDescription(settings.template_description) }}</span>
          </div>
          <div class="row-action">
            <template v-if="canEdit">
              <el-button type="primary" link :icon="Edit" @click="openDescriptionDialog">
                修改
              </el-button>
            </template>
          </div>
        </div>
      </div>
    </el-card>
  </div>

  <!-- Description Edit Dialog -->
  <el-dialog
    v-model="descriptionDialogVisible"
    title="修改模版描述"
    width="500px"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <el-input
      v-model="tempSettings.template_description"
      type="textarea"
      :rows="6"
      placeholder="请输入模版描述"
    />
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="descriptionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveDescription">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, onMounted, reactive, inject, computed } from 'vue';
import axios from '@/request/axios';
import { ElMessage } from 'element-plus';
import '@/assets/styles/universal.css';
import { Edit, Check, Close, QuestionFilled } from '@element-plus/icons-vue';

const userInfo = inject('userInfo');
const canEdit = computed(() => userInfo?.roleId === 2);
const isSimpleAdmin = computed(() => userInfo?.roleId === 1);

const settings = ref({
  template_file: '',
  public_email: '',
  template_description: '',
});
const publicFiles = ref([]);

// State for inline editing
const editState = reactive({
  template_file: false,
  public_email: false,
});

// Temporary storage for editing values
const tempSettings = reactive({
  template_file: [],
  public_email: '',
  template_description: '',
});
const descriptionDialogVisible = ref(false);

const fetchSettings = async () => {
  try {
    const response = await axios.get('/api/v1/settings');
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

const getPublicFileNames = (idsStr) => {
  if (!idsStr) return '';
  const ids = idsStr.split(',');
  const names = ids.map(id => {
    const file = publicFiles.value.find(f => getFileId(f) === id);
    return file ? file.name : '未知文件';
  });
  return names.join(', ');
};

const formatDescription = (text) => {
  if (!text) return '未设置';
  const str = String(text);
  if (str.length <= 10) return str;
  return `"${str.slice(0, 10)}"和剩余${str.length - 10}字`;
};

onMounted(async () => {
  await fetchPublicFiles();
  await fetchSettings();
});

// Start editing a field
const startEdit = async (field) => {
  
  // Refresh the file list when starting to edit the template file
  if (field === 'template_file') {
    await fetchPublicFiles();
    // Convert comma-separated string to array for multiple select
    const val = settings.value[field];
    tempSettings[field] = val ? val.split(',') : [];
  } else {
    // Copy current value to temp storage
    tempSettings[field] = settings.value[field] || '';
  }

  // Set editing state to true
  editState[field] = true;
};

// Cancel editing a field
const cancelEdit = (field) => {
  editState[field] = false;
};

// Save a single field
const saveField = async (field, labelName) => {
  const newValue = tempSettings[field];
  const oldValue = settings.value[field] || '';

  // Diff check: if value hasn't changed, just exit edit mode
  if (field === 'template_file') {
    // Compare array vs string
    const newStr = Array.isArray(newValue) ? newValue.join(',') : '';
    if (newStr === oldValue) {
      cancelEdit(field);
      return;
    }
  } else {
    if (newValue === oldValue) {
      cancelEdit(field);
      return;
    }
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
    const response = await axios.patch('/api/v1/settings', payload);
    if (response.data.message === 'success') {
      ElMessage.success(`${labelName}已更新`);
      // Update local state
      if (field === 'template_file' && Array.isArray(newValue)) {
        settings.value[field] = newValue.join(',');
      } else {
        settings.value[field] = newValue;
      }
      // Exit edit mode
      cancelEdit(field);
    } else {
      ElMessage.error(response.data.message || '保存失败');
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存失败');
  }
};

const openDescriptionDialog = () => {
  tempSettings.template_description = settings.value.template_description || '';
  descriptionDialogVisible.value = true;
};

const saveDescription = async () => {
  const newValue = tempSettings.template_description || '';
  
  try {
    const payload = { template_description: newValue };
    const response = await axios.patch('/api/v1/settings', payload);
    if (response.data.message === 'success') {
      ElMessage.success('模版描述已更新');
      settings.value.template_description = newValue;
      descriptionDialogVisible.value = false;
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
  align-items: start;
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

.card-header {
  display: flex;
  align-items: center;
  justify-content: flex-start;
}

.help-icon {
  margin-left: 6px;
  color: #909399;
  cursor: help;
}
</style>
