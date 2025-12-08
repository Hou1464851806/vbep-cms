<!-- src/views/dataset/DatasetList.vue -->
<script setup>
import { ref, onMounted, inject, reactive } from 'vue';
import { Plus, Edit, Delete, Document, Upload, Download, Promotion } from '@element-plus/icons-vue';
import { ElMessage, ElMessageBox } from 'element-plus';

// --- 模拟 API 函数 ---
const fetchDatasetListAPI = () =>
  new Promise((resolve) =>
    setTimeout(
      () =>
        resolve({
          code: 200,
          data: [
            { id: 'ds-001', name: '2023年第一季度销售数据', description: '包含华北、华南、华东三大区的销售记录、客户反馈及退货情况的汇总数据。此数据集包含敏感财务信息，仅供决策层和数据分析部门主管参阅。', creator: '张三', fileCount: 15, createdAt: '2023-04-01', updatedAt: '2023-04-05' },
            { id: 'ds-002', name: '用户行为日志分析数据集', description: '记录了2023年3月App端所有用户的点击、浏览、停留时长等关键行为，用于用户画像分析。数据已经过脱敏处理，不包含用户个人身份信息。', creator: '李四', fileCount: 128, createdAt: '2023-03-15', updatedAt: '2023-04-10' },
          ],
        }),
      500,
    ),
  );

const fetchFilesForDatasetAPI = (datasetId) =>
  new Promise((resolve) =>
    setTimeout(
      () =>
        resolve({
          code: 200,
          data: [
            { id: 'file-01', name: '华北区销售额.xlsx', size: '2.3 MB', uploadedAt: '2023-04-01' },
            { id: 'file-02', name: '华南区客户画像.csv', size: '5.1 MB', uploadedAt: '2023-04-01' },
          ],
        }),
      400,
    ),
  );

const submitApplicationAPI = (applicationData) => {
  console.log('正在提交申请数据:', applicationData);
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({ code: 200, message: '申请提交成功，请等待管理员审批！' });
    }, 1000);
  });
};

// --- 状态定义 ---
const isAdmin = inject('isAdmin');
const datasetList = ref([]);
const isDataLoading = ref(true);

// 状态1: 新增/编辑数据集对话框
const datasetFormDialog = reactive({ visible: false, title: '', form: { id: null, name: '', description: '' } });

// 状态2: 文件管理对话框
const fileDialog = reactive({ visible: false, currentDataset: null, list: [], isLoading: false });

// 状态3: 申请使用对话框
const applicationFormRef = ref(null);
const uploadRef = ref(null);
const applicationDialog = reactive({
  visible: false,
  isSubmitting: false,
  dataset: null,
  form: { reason: '' },
  rules: { reason: [{ required: true, message: '申请理由不能为空', trigger: 'blur' }] },
});


// --- 生命周期钩子 ---
onMounted(async () => {
  isDataLoading.value = true;
  try {
    const res = await fetchDatasetListAPI();
    if (res && res.code === 200) datasetList.value = res.data;
  } catch (e) {
    ElMessage.error('数据集加载失败');
  } finally {
    isDataLoading.value = false;
  }
});

// --- 数据集 CRUD 事件处理 ---
const handleAdd = () => { datasetFormDialog.title = '新增数据集'; datasetFormDialog.form = { id: null, name: '', description: '' }; datasetFormDialog.visible = true; };
const handleEdit = (row) => { datasetFormDialog.title = '编辑数据集'; datasetFormDialog.form = { ...row }; datasetFormDialog.visible = true; };
const handleSaveDataset = () => { const form = datasetFormDialog.form; if (!form.name.trim()) { ElMessage.warning('数据集名称不能为空'); return; } if (form.id) { const index = datasetList.value.findIndex(item => item.id === form.id); if (index !== -1) datasetList.value[index] = { ...datasetList.value[index], ...form, updatedAt: new Date().toLocaleDateString() }; } else { datasetList.value.unshift({ ...form, id: `ds-new-${Date.now()}`, creator: '我', fileCount: 0, createdAt: new Date().toLocaleDateString(), updatedAt: new Date().toLocaleDateString() }); } ElMessage.success(`${datasetFormDialog.title}成功！`); datasetFormDialog.visible = false; };
const handleDelete = (row) => { ElMessageBox.confirm(`确定要删除数据集 "${row.name}" 吗？`, '警告', { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }).then(() => { datasetList.value = datasetList.value.filter((item) => item.id !== row.id); ElMessage.success(`数据集 "${row.name}" 已删除`); }).catch(() => { ElMessage.info('已取消删除'); }); };

// --- 文件管理事件处理 ---
const handleViewFiles = async (row) => { fileDialog.currentDataset = row; fileDialog.visible = true; fileDialog.isLoading = true; fileDialog.list = []; try { const res = await fetchFilesForDatasetAPI(row.id); if (res && res.code === 200) fileDialog.list = res.data; } catch (e) { ElMessage.error('文件列表加载失败'); } finally { fileDialog.isLoading = false; } };
const handleFileUploadSuccess = (response, file) => { ElMessage.success(`${file.name} 上传成功! (模拟)`); fileDialog.list.unshift({ id: `new-${Date.now()}`, name: file.name, size: `${(file.size / 1024 / 1024).toFixed(2)} MB`, uploadedAt: new Date().toLocaleDateString() }); if (fileDialog.currentDataset) { fileDialog.currentDataset.fileCount++; } };
const handleDownloadFile = (file) => { ElMessage.info(`开始下载文件: ${file.name} (模拟)`); const link = document.createElement('a'); link.style.display = 'none'; link.href = '#'; link.setAttribute('download', file.name); document.body.appendChild(link); link.click(); document.body.removeChild(link); };

// --- 申请功能事件处理 ---
const handleApply = (row) => { applicationDialog.dataset = row; applicationDialog.form.reason = ''; if (uploadRef.value) { uploadRef.value.clearFiles(); } applicationDialog.visible = true; };
const handleSubmitApplication = async () => { if (!applicationFormRef.value) return; await applicationFormRef.value.validate(async (valid) => { if (valid) { applicationDialog.isSubmitting = true; try { const uploadedFiles = uploadRef.value?.getFiles().map(f => ({ name: f.name, size: f.size })); const res = await submitApplicationAPI({ datasetId: applicationDialog.dataset.id, datasetName: applicationDialog.dataset.name, reason: applicationDialog.form.reason, attachments: uploadedFiles }); if (res.code === 200) { ElMessage.success(res.message); applicationDialog.visible = false; } else { ElMessage.error(res.message || '申请提交失败'); } } catch (error) { console.error('Application submission failed:', error); ElMessage.error('提交申请时发生错误'); } finally { applicationDialog.isSubmitting = false; } } else { ElMessage.warning('请填写必要的申请信息'); } }); };
</script>

<template>
  <div>
    <!-- 主体：数据集列表 -->
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>数据集列表</span>
          <el-button v-if="isAdmin()" type="primary" :icon="Plus" @click="handleAdd">新增数据集</el-button>
        </div>
      </template>

      <el-skeleton :rows="6" animated :loading="isDataLoading">
        <el-table :data="datasetList" stripe style="width: 100%">
          <el-table-column v-if="isAdmin()" type="expand">
            <template #default="props">
              <div class="dataset-detail-description"><p>{{ props.row.description }}</p></div>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="数据集名称" min-width="200" />
          <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
          <el-table-column prop="fileCount" label="文件数" width="90" align="center"/>
          <el-table-column prop="creator" label="创建者" width="120" />
          <el-table-column prop="updatedAt" label="更新时间" width="160" />
          <el-table-column label="操作" width="280" fixed="right" align="center">
            <template #default="scope">
              <!-- 管理员操作 -->
              <template v-if="isAdmin()">
                <el-button size="small" :icon="Document" @click="handleViewFiles(scope.row)">管理文件</el-button>
                <el-button size="small" type="primary" :icon="Edit" @click="handleEdit(scope.row)">编辑</el-button>
                <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(scope.row)">删除</el-button>
              </template>
              <!-- 普通用户操作 -->
              <template v-else>
                <el-button type="primary" :icon="Promotion" @click="handleApply(scope.row)">申请使用</el-button>
              </template>
            </template>
          </el-table-column>
        </el-table>
      </el-skeleton>
    </el-card>

    <!-- ======== 所有对话框 ======== -->

    <!-- 对话框1：新增/编辑数据集 (管理员) -->
    <el-dialog v-model="datasetFormDialog.visible" :title="datasetFormDialog.title" width="50%">
      <el-form :model="datasetFormDialog.form" label-width="100px">
        <el-form-item label="数据集名称">
          <el-input v-model="datasetFormDialog.form.name" placeholder="请输入数据集名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="datasetFormDialog.form.description" type="textarea" :rows="5" placeholder="请输入详细的数据集描述信息..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="datasetFormDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveDataset">确定</el-button>
      </template>
    </el-dialog>

    <!-- 对话框2：文件列表 (管理员) -->
    <el-dialog v-model="fileDialog.visible" :title="`文件列表 - ${fileDialog.currentDataset?.name}`" width="70%" top="5vh">
      <div class="file-dialog-toolbar">
        <el-upload action="#" :show-file-list="false" :on-success="handleFileUploadSuccess" :data="{ datasetId: fileDialog.currentDataset?.id }" :http-request="() => {}">
          <el-button type="primary" :icon="Upload">上传文件</el-button>
        </el-upload>
      </div>
      <el-skeleton :rows="5" animated :loading="fileDialog.isLoading">
        <el-table :data="fileDialog.list" stripe height="450px">
          <el-table-column property="name" label="文件名">
            <template #default="scope">
              <el-icon style="vertical-align: middle; margin-right: 5px;"><Document /></el-icon>
              <span>{{ scope.row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column property="size" label="大小" width="120" />
          <el-table-column property="uploadedAt" label="上传时间" width="180" />
          <el-table-column label="操作" width="120" align="center">
            <template #default="scope">
              <el-button type="success" :icon="Download" size="small" @click="handleDownloadFile(scope.row)">下载</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-skeleton>
      <template #footer>
        <el-button @click="fileDialog.visible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 对话框3：申请使用 (普通用户) -->
    <el-dialog v-model="applicationDialog.visible" :title="`申请使用数据集 - ${applicationDialog.dataset?.name}`" width="600px" :close-on-click-modal="false">
      <el-form ref="applicationFormRef" :model="applicationDialog.form" :rules="applicationDialog.rules" label-position="top" v-loading="applicationDialog.isSubmitting" element-loading-text="正在提交申请...">
        <el-form-item label="申请理由" prop="reason">
          <el-input v-model="applicationDialog.form.reason" type="textarea" :rows="5" placeholder="请详细说明您申请此数据集的用途、目的以及期望的使用方式。" />
        </el-form-item>
        <el-form-item label="上传附件 (可选)">
          <el-upload ref="uploadRef" class="upload-demo" action="#" :auto-upload="false" :http-request="() => {}">
            <el-button type="primary" :icon="Upload">点击上传</el-button>
            <template #tip>
              <div class="el-upload__tip">可以上传项目计划书、需求文档等补充材料 (非必须)</div>
            </template>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="applicationDialog.visible = false" :disabled="applicationDialog.isSubmitting">取消</el-button>
        <el-button type="primary" @click="handleSubmitApplication" :loading="applicationDialog.isSubmitting">提交申请</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 18px;
}
.file-dialog-toolbar {
  margin-bottom: 20px;
}
.dataset-detail-description {
  padding: 10px 20px 10px 60px;
  background-color: #fafafa;
  border-radius: 4px;
}
.dataset-detail-description p {
  margin: 0;
  line-height: 1.7;
  color: #606266;
  white-space: pre-wrap;
}
.el-upload__tip {
  color: #909399;
  font-size: 12px;
  margin-left: 10px;
}
</style>
