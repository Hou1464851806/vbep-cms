<template>
  <el-row :gutter="20">
    <el-col :span="24" v-if="isSimpleAdmin">
      <el-alert
        title="目前您仅可浏览设置项，只有超级管理员方可修改"
        type="warning"
        show-icon
        style="margin-bottom: 20px;"
      />
    </el-col>
    <!-- Global Settings Card -->
    <el-col :xs="24" :lg="12" class="responsive-mb">
      <el-card shadow="never" class="h-100">
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
            <span v-if="!editState.publicEmail">{{ settings.publicEmail || '未设置' }}</span>
            <el-input v-else v-model="tempSettings.publicEmail" size="small" />
          </div>
          <div class="row-action">
            <template v-if="canEdit">
              <el-button v-if="!editState.publicEmail" type="primary" link :icon="Edit" @click="startEdit('publicEmail')">
                修改
              </el-button>
              <template v-else>
                <el-button type="success" link :icon="Check" @click="saveField('publicEmail', '公邮地址')">确定</el-button>
                <el-button type="info" link :icon="Close" @click="cancelEdit('publicEmail')">取消</el-button>
              </template>
            </template>
          </div>
        </div>

        <!-- User Agreement Setting -->
        <div class="info-row">
          <div class="row-label">用户协议</div>
          <div class="row-content">
            <span>{{ formatDescription(settings.userAgreement) }}</span>
          </div>
          <div class="row-action">
            <el-button type="primary" link :icon="View" @click="openViewDialog('userAgreement')">查看</el-button>
            <template v-if="canEdit">
              <el-button type="primary" link :icon="Edit" @click="openLongTextEditDialog('userAgreement')">修改</el-button>
            </template>
          </div>
        </div>
      </div>
      </el-card>
    </el-col>

    <!-- Template Settings Card -->
    <el-col :xs="24" :lg="12">
      <el-card shadow="never" class="h-100">
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
            <span v-if="!editState.templateFile">{{ getPublicFileNames(settings.templateFile) || '未设置' }}</span>
            <el-select v-else v-model="tempSettings.templateFile" multiple collapse-tags collapse-tags-tooltip size="small" placeholder="请选择文件" clearable filterable style="width: 100%">
              <el-option
                v-for="file in publicFiles"
                :key="file.fileID"
                :label="file.name"
                :value="file.fileID"
              />
            </el-select>
          </div>
          <div class="row-action">
            <el-button v-if="!editState.templateFile" type="primary" link :icon="View" @click="openViewDialog('templateFile')">查看</el-button>
            <template v-if="canEdit">
              <el-button v-if="!editState.templateFile" type="primary" link :icon="Edit" @click="startEdit('templateFile')">
                修改
              </el-button>
              <template v-else>
                <el-button type="success" link :icon="Check" @click="saveField('templateFile', '模版文件')">确定</el-button>
                <el-button type="info" link :icon="Close" @click="cancelEdit('templateFile')">取消</el-button>
              </template>
            </template>
          </div>
        </div>

        <!-- Template Description Setting -->
        <div class="info-row">
          <div class="row-label">模版描述</div>
          <div class="row-content">
            <span>{{ formatDescription(settings.templateDescription) }}</span>
          </div>
          <div class="row-action">
            <el-button type="primary" link :icon="View" @click="openViewDialog('templateDescription')">查看</el-button>
            <template v-if="canEdit">
              <el-button type="primary" link :icon="Edit" @click="openLongTextEditDialog('templateDescription')">
                修改
              </el-button>
            </template>
          </div>
        </div>
      </div>
      </el-card>
    </el-col>
  </el-row>

  <!-- Long Text Edit Dialog (Description & Agreement) -->
  <el-dialog
    v-model="longTextEditDialogVisible"
    :title="longTextEditTitle"
    width="500px"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <el-input
      v-model="tempSettings[currentLongTextField]"
      type="textarea"
      :rows="6"
      placeholder="请输入模版描述"
    />
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="longTextEditDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveLongText">确定</el-button>
      </span>
    </template>
  </el-dialog>

  <!-- View Dialog -->
  <el-dialog v-model="viewDialog.visible" :title="viewDialog.title" width="500px">
    <div v-if="viewDialog.type === 'list'" style="max-height: 400px; overflow-y: auto;">
      <ul style="padding-left: 20px; margin: 0;">
        <li v-for="(item, index) in viewDialog.list" :key="index" style="margin-bottom: 5px;">{{ item }}</li>
      </ul>
    </div>
    <div v-else style="white-space: pre-wrap; max-height: 400px; overflow-y: auto; line-height: 1.6;">
      {{ viewDialog.content }}
    </div>
    <template #footer>
      <el-button @click="viewDialog.visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, onMounted, reactive, inject, computed } from 'vue';
import axios from '@/request/axios';
import { ElMessage } from 'element-plus';
import '@/assets/styles/universal.css';
import { Edit, Check, Close, QuestionFilled, View } from '@element-plus/icons-vue';

const userInfo = inject('userInfo');
const canEdit = computed(() => userInfo?.roleId === 2);
const isSimpleAdmin = computed(() => userInfo?.roleId === 1);

const settings = ref({
  template_file: '',
  public_email: '',
  template_description: '',
  userAgreement: '',
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
  userAgreement: '',
});
const longTextEditDialogVisible = ref(false);
const currentLongTextField = ref('');

const fetchSettings = async () => {
  try {
    const response = await axios.get('/api/v1/settings');
    if (response.status === 200) {
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
    if (response.status === 200) {
      publicFiles.value = response.data.files || [];
    }
  } catch (error) {
    console.error('Failed to fetch public files:', error);
  }
};

const getPublicFileNames = (idsStr) => {
  if (!idsStr) return '';
  const ids = idsStr.split(',');
  const names = ids.map(id => {
    const file = publicFiles.value.find(f => f.fileID === id);
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
  if (field === 'templateFile') {
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
  if (field === 'templateFile') {
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
  if (field === 'publicEmail' && newValue) { // Validate only if not empty
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(newValue)) {
      ElMessage.warning('请输入有效的邮箱地址');
      return;
    }
  }

  try {
    const payload = { [field]: newValue };
    const response = await axios.patch('/api/v1/settings', payload);
    if (response.status === 200) {
      ElMessage.success(`${labelName}已更新`);
      // Update local state
      if (field === 'templateFile' && Array.isArray(newValue)) {
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

const longTextEditTitle = computed(() => {
  return currentLongTextField.value === 'templateDescription' ? '修改模版描述' : '修改用户协议';
});

const openLongTextEditDialog = (field) => {
  currentLongTextField.value = field;
  tempSettings[field] = settings.value[field] || '';
  longTextEditDialogVisible.value = true;
};

const saveLongText = async () => {
  const field = currentLongTextField.value;
  const newValue = tempSettings[field] || '';
  
  try {
    const payload = { [field]: newValue };
    const response = await axios.patch('/api/v1/settings', payload);
    if (response.status === 200) {
      ElMessage.success('设置已更新');
      settings.value[field] = newValue;
      longTextEditDialogVisible.value = false;
    } else {
      ElMessage.error(response.data.message || '保存失败');
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || '保存失败');
  }
};

const viewDialog = reactive({
  visible: false,
  title: '',
  content: '',
  type: 'text', // 'text' or 'list'
  list: []
});

const openViewDialog = (field) => {
  if (field === 'templateFile') {
    viewDialog.title = '模版文件列表';
    viewDialog.type = 'list';
    const idsStr = settings.value.templateFile || '';
    if (!idsStr) {
      viewDialog.list = ['未设置'];
    } else {
      const ids = idsStr.split(',');
      viewDialog.list = ids.map(id => {
        const file = publicFiles.value.find(f => f.fileID === id);
        return file ? file.name : `未知文件 (${id})`;
      });
    }
  } else {
    viewDialog.type = 'text';
    if (field === 'templateDescription') {
      viewDialog.title = '模版描述';
      viewDialog.content = settings.value.templateDescription || '未设置';
    } else if (field === 'userAgreement') {
      viewDialog.title = '用户协议';
      viewDialog.content = settings.value.userAgreement || '未设置';
    }
  }
  viewDialog.visible = true;
};
</script>

<style scoped>
.row-label {
  width: 150px;
}

.h-100 {
  height: 100%;
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

@media (max-width: 1199px) {
  .responsive-mb {
    margin-bottom: 20px;
  }
}
</style>
