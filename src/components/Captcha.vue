<script setup>
import { ref, watch, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true,
    default: () => ({ id: '', answer: '' })
  },
  size: {
    type: String,
    default: 'default'
  },
  placeholder: {
    type: String,
    default: '请输入图片字符(不区分大小写)'
  }
})

const emit = defineEmits(['update:modelValue', 'keyup-enter'])

const loading = ref(false)
const imageUrl = ref('')

// 获取验证码
const fetchCaptcha = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/users/captcha')
    // 更新父组件绑定的对象
    emit('update:modelValue', {
      id: res.data.captchaID,
      answer: '' // 刷新时清空输入
    })
    imageUrl.value = res.data.question
  } catch (e) {
    ElMessage.error('获取验证码失败')
  } finally {
    loading.value = false
  }
}

// 供父组件手动调用刷新
defineExpose({
  refresh: fetchCaptcha
})

// 按下回车
const handleEnter = (e) => {
  emit('keyup-enter', e)
}

onMounted(() => {
  fetchCaptcha()
})
</script>

<template>
  <div class="captcha-input-container">
    <el-input
      v-model="modelValue.answer"
      :size="size"
      :placeholder="placeholder"
      @keyup.enter="handleEnter"
    >
      <template #prefix v-if="$slots.prefix">
        <slot name="prefix"></slot>
      </template>
    </el-input>

    <div class="captcha-image-wrapper">
      <el-skeleton :loading="loading" animated style="height: 100%; width: 100px;">
        <template #default>
          <img
            v-if="imageUrl"
            :src="imageUrl"
            class="captcha-img"
            @click="fetchCaptcha"
            alt="验证码"
          />
        </template>
      </el-skeleton>
    </div>
  </div>
</template>

<style scoped>
.captcha-input-container {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 100%;
}

.captcha-image-wrapper {
  height: var(--el-component-size-large, 40px); /* 匹配 input 的高度 */
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.captcha-img {
  height: 100%;
  max-height: 40px;
  border: 1px solid #eee;
  border-radius: 4px;
  cursor: pointer; /* 点击图片刷新 */
  transition: opacity 0.2s;
}

.captcha-img:hover {
  opacity: 0.8;
}
</style>
