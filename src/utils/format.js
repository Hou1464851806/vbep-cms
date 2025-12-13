// src/utils/format.js

/**
 * 截取ID前8位
 * @param {string|number} id
 * @returns {string}
 */
export const formatShortId = (id) => {
  if (!id) return '-'
  return String(id).substring(0, 8)
}

/**
 * 将数据库日期格式化为更易读的形式
 * @param {string} datetime
 * @returns {string}
 */
export const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

/**
 * 将文件大小格式化为更易读的形式
 * @param {string} datetime
 * @returns {string}
 */
export const formatFileSize = (value) => {
  const size = Number(value) || 0
  if (size <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let idx = 0
  let current = size
  while (current >= 1024 && idx < units.length - 1) {
    current /= 1024
    idx += 1
  }
  const formatted = idx === 0 ? current.toFixed(0) : current.toFixed(2)
  return `${formatted} ${units[idx]}`
}

/**
 * 将字符串（支持中英文）转换为高分辨率的透明背景PNG图片。
 * @param {string} text - 需要转换的字符串。
 * @param {object} [options] - 可选的配置项。
 * @param {number} [options.scale=3] - 缩放因子，用于提高分辨率。3 表示约 300 DPI。
 * @param {number} [options.fontSize=100] - 基础字体大小（缩放前）。
 * @param {string} [options.fontFamily] - 字体栈。
 * @param {string} [options.textColor='rgba(0, 0, 0, 0.4)'] - 字体颜色。
 * @param {number} [options.padding=10] - 基础内边距（缩放前）。
 * @returns {Promise<ArrayBuffer>} 返回包含PNG图片数据的ArrayBuffer。
 */
export async function textToTransparentImage(text, options = {}) {
  // 1. 设置默认值，引入 scale
  const {
    scale = 3, // <--- 关键新增：缩放因子
    fontSize = 12,
    fontFamily = 'sans-serif',
    textColor = 'rgba(0, 0, 0, 1)',
    padding = 10, // 调小了默认padding，因为最终效果会乘以scale
  } = options

  // 2. 创建离屏Canvas
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    throw new Error('无法获取 Canvas 2D 上下文')
  }

  // 3. 按比例放大所有尺寸
  const scaledFontSize = fontSize * scale
  const scaledPadding = padding * scale
  const font = `${scaledFontSize}px ${fontFamily}`

  // 4. 测量放大后的文本尺寸
  ctx.font = font
  const textMetrics = ctx.measureText(text)
  const textHeight = textMetrics.actualBoundingBoxAscent + textMetrics.actualBoundingBoxDescent

  const canvasWidth = textMetrics.width + scaledPadding * 2
  const canvasHeight = textHeight + scaledPadding * 2

  canvas.width = canvasWidth
  canvas.height = canvasHeight

  // 5. 在放大的 Canvas 上绘制
  ctx.font = font
  ctx.fillStyle = textColor
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(text, canvas.width / 2, canvas.height / 2)

  // 6. 导出高分辨率图片
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error('无法从Canvas创建Blob'))
        return
      }
      resolve(blob.arrayBuffer())
    }, 'image/png')
  })
}
