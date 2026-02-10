<script setup>
import { computed } from 'vue';
import MarkdownIt from 'markdown-it';
import DOMPurify from 'dompurify';
import 'github-markdown-css/github-markdown-light.css';

// 定义 Props
const props = defineProps({
  // 核心数据：Markdown 文本
  modelValue: {
    type: String,
    default: '',
    required: true
  },
  // 是否允许在 markdown 中写 HTML 标签 (配合 dompurify 使用)
  html: {
    type: Boolean,
    default: false
  },
  // 是否将换行符 '\n' 转换为 <br>
  breaks: {
    type: Boolean,
    default: true
  },
  // 是否自动识别 URL 文本并转换为链接
  linkify: {
    type: Boolean,
    default: true
  }
});

// 初始化 markdown-it 实例
const md = new MarkdownIt({
  html: props.html,
  breaks: props.breaks,
  linkify: props.linkify,
  typographer: true // 启用排版优化（如智能引号）
});

// 链接优化：强制在新标签页打开，并增加安全属性
const defaultLinkRender = md.renderer.rules.link_open || function(tokens, idx, options, env, self) {
  return self.renderToken(tokens, idx, options);
};

md.renderer.rules.link_open = function (tokens, idx, options, env, self) {
  // 增加 target="_blank"
  tokens[idx].attrSet('target', '_blank');
  // 增加 rel="noopener noreferrer" 防止安全漏洞
  tokens[idx].attrSet('rel', 'noopener noreferrer');

  return defaultLinkRender(tokens, idx, options, env, self);
};

// 计算属性：当 props.modelValue 变化时，自动重新解析和清洗
const sanitizedHtml = computed(() => {
  if (!props.modelValue) return '';

  // 第一步：Markdown -> HTML 字符串
  const rawHtml = md.render(props.modelValue);

  // 第二步：清洗 HTML (防御 XSS 攻击)
  return DOMPurify.sanitize(rawHtml);
});

defineExpose({
  sanitizedHtml
});
</script>

<template>
  <div class="markdown-wrapper">
    <!--
      class="markdown-body" 是 github-markdown-css 必须的类名
      v-html 用于注入清洗后的 HTML
    -->
    <div
      class="markdown-body"
      v-html="sanitizedHtml"
    ></div>
  </div>
</template>

<style scoped>
/* 外层容器：确保整体布局安全 */
.markdown-wrapper {
  width: 100%;
  overflow-x: auto; /* 防止超宽内容（如表格）撑破布局 */
  word-wrap: break-word; /* 强制长单词换行 */
}

/*
  样式微调
  使用 :deep() 穿透 scoped 作用域，去修改 v-html 内部的样式
*/
.markdown-body {
  background-color: transparent !important; /* 移除默认背景色，适应父容器 */
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 16px;
  line-height: 1.6;
  padding: 10px 0;
  box-sizing: border-box;
}

/* 图片自适应：防止大图溢出 */
.markdown-body :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
}

/* 链接样式微调 */
.markdown-body :deep(a) {
  color: #0969da;
  text-decoration: none;
}
.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

/* 代码块背景微调 */
.markdown-body :deep(pre) {
  background-color: #f6f8fa;
  border-radius: 6px;
  padding: 16px;
}
</style>
