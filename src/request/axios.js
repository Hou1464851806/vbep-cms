import axios from 'axios'
import router from '@/router'
import { ElMessageBox } from 'element-plus'

const AUTH_TOKEN_STORAGE_KEY = 'auth_token'
const isBrowser = typeof window !== 'undefined'

export const readAuthToken = () => {
  if (!isBrowser) return null
  try {
    return window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
  } catch (error) {
    console.error('Failed to read auth token', error)
    return null
  }
}

export const setAuthToken = (token) => {
  if (!isBrowser) return
  try {
    if (token) {
      window.localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, token)
    } else {
      window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
    }
  } catch (error) {
    console.error('Failed to persist auth token', error)
  }
}

export const clearAuthToken = () => {
  if (!isBrowser) return
  try {
    window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
  } catch (error) {
    console.error('Failed to clear auth token', error)
  }
}

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  headers: {
    Accept: 'application/json',
    'Content-Type': 'application/json',
  },
  timeout: 15000,
  withCredentials: true,
})

apiClient.interceptors.request.use(
  (config) => {
    const token = readAuthToken()
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

let isAuthAlerting = false

const resolveAuthRedirectTarget = () => {
  const currentRoute = router.currentRoute.value
  const currentPath = currentRoute?.path || ''
  const queryRedirect = currentRoute?.query?.redirect

  if (currentPath === '/login') {
    return typeof queryRedirect === 'string' && queryRedirect ? queryRedirect : null
  }

  return currentRoute?.fullPath || '/home'
}

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const skipAuthRedirect = error?.config?.skipAuthRedirect

    if (error?.response) {
      if (skipAuthRedirect) {
        return Promise.reject(error)
      }
      if (error.response.status === 401) {
        const redirectTarget = resolveAuthRedirectTarget()
        clearAuthToken()
        if (!isAuthAlerting) {
          isAuthAlerting = true
          ElMessageBox.alert('登录状态已过期，请重新登录', '提示', {
            confirmButtonText: '重新登录',
            type: 'warning',
            showClose: false,
            callback: () => {
              isAuthAlerting = false
              if (redirectTarget) {
                router.replace({
                  path: '/login',
                  query: { redirect: redirectTarget }
                })
                return
              }
              router.replace({ path: '/login' })
            }
          })
        }
      }
      console.error('API request failed:', error.response.status, error.response.data)
    }
    return Promise.reject(error)
  },
)

export default apiClient
