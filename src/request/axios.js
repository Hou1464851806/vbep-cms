import axios from 'axios'
import router from '@/router'
import { ElMessageBox } from 'element-plus'

const AUTH_TOKEN_STORAGE_KEY = 'auth_token'
const ACCESS_TOKEN_REFRESH_LEAD_MS = 60 * 1000
const isBrowser = typeof window !== 'undefined'

const decodeJwtPayload = (token) => {
  if (!token) return null
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const normalized = payload.padEnd(payload.length + ((4 - (payload.length % 4)) % 4), '=')
    return JSON.parse(window.atob(normalized))
  } catch (error) {
    console.error('Failed to decode auth token payload', error)
    return null
  }
}

const getTokenRemainingMs = (token) => {
  if (!isBrowser || !token) return 0
  const payload = decodeJwtPayload(token)
  const exp = Number(payload?.exp)
  if (!Number.isFinite(exp)) return 0
  return exp * 1000 - Date.now()
}

const getTokenRefreshLeadMs = (token) => {
  const payload = decodeJwtPayload(token)
  const exp = Number(payload?.exp)
  const iat = Number(payload?.iat)
  if (!Number.isFinite(exp) || !Number.isFinite(iat) || exp <= iat) {
    return ACCESS_TOKEN_REFRESH_LEAD_MS
  }
  const lifetimeMs = (exp - iat) * 1000
  return Math.min(ACCESS_TOKEN_REFRESH_LEAD_MS, Math.max(5000, Math.floor(lifetimeMs / 3)))
}

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

const handleRefreshFailure = (error, options = {}) => {
  const { allowRetry = false } = options
  const status = error?.response?.status
  if (status === 429 && allowRetry) {
    return false
  }
  if (status === 401) {
    redirectToLogin()
  }
  return true
}

const ensureFreshAccessToken = async (currentToken) => {
  if (!currentToken) return currentToken
  const remainingMs = getTokenRemainingMs(currentToken)
  if (remainingMs > getTokenRefreshLeadMs(currentToken)) {
    return currentToken
  }
  try {
    return await requestAccessTokenRefresh()
  } catch (error) {
    if (error?.response?.status === 429 && remainingMs > 0) {
      return currentToken
    }
    handleRefreshFailure(error)
    throw error
  }
}

apiClient.interceptors.request.use(
  async (config) => {
    let token = readAuthToken()
    if (token && !config.skipAuthRefresh) {
      try {
        token = await ensureFreshAccessToken(token)
      } catch (error) {
        if (!config.skipAuthRedirect) {
          handleRefreshFailure(error)
        }
        return Promise.reject(error)
      }
    }
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

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
            handleRefreshFailure(refreshError)
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
