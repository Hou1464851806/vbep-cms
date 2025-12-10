import axios from 'axios'

const AUTH_TOKEN_STORAGE_KEY = 'auth_token'
const isBrowser = typeof window !== 'undefined'

const readAuthToken = () => {
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

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response) {
      console.error('API request failed:', error.response.status, error.response.data)
    }
    return Promise.reject(error)
  },
)

export default apiClient
