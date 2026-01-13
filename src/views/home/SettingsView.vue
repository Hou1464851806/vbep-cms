<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>通用设置</span>
      </div>
    </template>

    <div class="settings-list">
      <!-- App File Setting -->
      <div class="setting-row">
        <div class="row-label">文件名 (app_file)</div>
        <div class="row-content">
          <span v-if="!editState.app_file">{{ settings.app_file || '未设置' }}</span>
          <el-input v-else v-model="tempSettings.app_file" size="small" />
        </div>
        <div class="row-action">
          <el-button v-if="!editState.app_file" type="primary" link :icon="Edit" @click="startEdit('app_file')">
            修改
          </el-button>
          <template v-else>
            <el-button type="success" link :icon="Check" @click="saveField('app_file', '文件名')">确定</el-button>
            <el-button type="info" link :icon="Close" @click="cancelEdit('app_file')">取消</el-button>
          </template>
        </div>
      </div>

      <!-- Public Email Setting -->
      <div class="setting-row">
        <div class="row-label">邮箱地址 (public_email)</div>
        <div class="row-content">
          <span v-if="!editState.public_email">{{ settings.public_email || '未设置' }}</span>
          <el-input v-else v-model="tempSettings.public_email" size="small" />
        </div>
        <div class="row-action">
          <el-button v-if="!editState.public_email" type="primary" link :icon="Edit" @click="startEdit('public_email')">
            修改
          </el-button>
          <template v-else>
            <el-button type="success" link :icon="Check" @click="saveField('public_email', '邮箱地址')">确定</el-button>
            <el-button type="info" link :icon="Close" @click="cancelEdit('public_email')">取消</el-button>
          </template>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue';
import axios from '@/request/axios';
import { ElMessage } from 'element-plus';
import { Edit, Check, Close } from '@element-plus/icons-vue';

const settings = ref({
  app_file: '',
  public_email: '',
});

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

onMounted(() => {
  fetchSettings();
});

// Start editing a field
const startEdit = (field) => {
  // Copy current value to temp storage
  tempSettings[field] = settings.value[field];
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
  const oldValue = settings.value[field];

  // Diff check: if value hasn't changed, just exit edit mode
  if (newValue === oldValue) {
    cancelEdit(field);
    return;
  }

  // Basic validation (can be expanded)
  if (!newValue && field === 'app_file') {
    ElMessage.warning(`${labelName}不能为空`);
    return;
  }
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
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 18px;
  font-weight: bold;
}
.settings-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
  padding: 0 10px;
}
.setting-row {
  display: flex;
  align-items: center;
  border-bottom: 1px solid #f0f0f0;
  padding-bottom: 15px;
}
.setting-row:last-child {
  border-bottom: none;
}
.row-label {
  width: 180px; /* Increased width for longer labels */
  font-weight: bold;
  color: #606266;
  flex-shrink: 0; /* Prevent label from shrinking */
}
.row-content {
  flex: 1;
  padding: 0 15px;
  color: #303133;
  display: flex;
  align-items: center;
  min-height: 32px; /* Use min-height to prevent layout shift */
}
.row-action {
  width: 110px;
  display: flex;
  justify-content: flex-end;
  gap: 5px;
}
</style>
