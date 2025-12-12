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