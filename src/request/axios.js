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
let refreshTokenPromise = null

const resolveAuthRedirectTarget = () => {
  const currentRoute = router.currentRoute.value
  const currentPath = currentRoute?.path || ''
  const queryRedirect = currentRoute?.query?.redirect

  if (currentPath === '/login') {
    return typeof queryRedirect === 'string' && queryRedirect ? queryRedirect : null
  }

  return currentRoute?.fullPath || '/home'
}

const redirectToLogin = () => {
  const redirectTarget = resolveAuthRedirectTarget()
  clearAuthToken()
  if (isAuthAlerting) {
    return
  }
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
          query: { redirect: redirectTarget },
        })
        return
      }
      router.replace({ path: '/login' })
    },
  })
}

const requestAccessTokenRefresh = async () => {
  if (!refreshTokenPromise) {
    refreshTokenPromise = apiClient
      .post(
        '/api/v1/users/refresh',
        {},
        {
          skipAuthRedirect: true,
          skipAuthRefresh: true,
        },
      )
      .then((response) => {
        const token = response.data?.token
        if (!token) {
          throw new Error('Refresh response missing access token')
        }
        setAuthToken(token)
        return token
      })
      .finally(() => {
        refreshTokenPromise = null
      })
  }
  return refreshTokenPromise
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const skipAuthRedirect = error?.config?.skipAuthRedirect
    const skipAuthRefresh = error?.config?.skipAuthRefresh
    const originalRequest = error?.config

    if (error?.response) {
      if (error.response.status === 401 && !skipAuthRefresh && originalRequest && !originalRequest._retry) {
        originalRequest._retry = true
        try {
          const token = await requestAccessTokenRefresh()
          originalRequest.headers = originalRequest.headers || {}
          originalRequest.headers.Authorization = `Bearer ${token}`
          return apiClient(originalRequest)
        } catch (refreshError) {
          if (!skipAuthRedirect) {
            redirectToLogin()
          }
          return Promise.reject(refreshError)
        }
      }

      if (!skipAuthRedirect && error.response.status === 401) {
        redirectToLogin()
      }
      console.error('API request failed:', error.response.status, error.response.data)
    }
    return Promise.reject(error)
  },
)

export default apiClient
