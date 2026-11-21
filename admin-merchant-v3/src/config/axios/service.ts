import axios, { AxiosError, AxiosInstance, AxiosResponse } from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { config } from '@/config/axios/config'
import { getAccessToken, removeToken } from '@/utils/auth'
import { deleteUserCache } from '@/hooks/web/useCache'

export const isRelogin = { show: false }

const { base_url, request_timeout } = config

const UNAUTHORIZED_CODES = [10006, 20003, 20004]

const service: AxiosInstance = axios.create({
  baseURL: base_url,
  timeout: request_timeout,
  withCredentials: false
})

// request 拦截器：附加 JWT
service.interceptors.request.use(
  (config) => {
    const token = getAccessToken()
    if (token) {
      config.headers.Authorization = 'Bearer ' + token
    }
    // 商家端：当前操作门店
    const storeId = localStorage.getItem('merchantSid')
    if (storeId) {
      config.headers['X-Store-ID'] = storeId
    }
    return config
  },
  (error: AxiosError) => Promise.reject(error)
)

const handleUnauthorized = () => {
  removeToken()
  deleteUserCache()
  localStorage.removeItem('merchantSid')
  localStorage.removeItem('merchantStoreName')
  if (!window.location.pathname.includes('login')) {
    window.location.href = '/login'
  }
}

// response 拦截器：把 Go 的 { code, message, data } 统一成框架期望的 { code, data, msg }
service.interceptors.response.use(
  (response: AxiosResponse<any>) => {
    if (
      response.config.responseType === 'blob' ||
      response.config.responseType === 'arraybuffer'
    ) {
      return response
    }

    const body = response.data
    // 非标准响应（如文件、纯文本）直接返回
    if (!body || typeof body !== 'object' || !('code' in body)) {
      return body
    }

    if (body.code !== 0) {
      const message = body.message || '请求失败'
      if (body.code === 10009) {
        // 登录验证码挑战由登录页处理，不弹全局错误
      } else if (UNAUTHORIZED_CODES.includes(body.code)) {
        handleUnauthorized()
      } else {
        ElNotification.error({ title: message })
      }
      const error = new Error(message) as Error & { code?: number; data?: any }
      error.code = body.code
      error.data = body.data
      return Promise.reject(error)
    }

    return { code: 0, data: body.data, msg: body.message || 'ok' }
  },
  (error: AxiosError<any>) => {
    const status = error.response?.status
    const body = error.response?.data
    const code = body?.code
    const message = body?.message || error.message || '网络异常，请稍后重试'

    if (code === 10009) {
      // 登录验证码挑战由登录页处理，不弹全局错误
    } else if (status === 401) {
      handleUnauthorized()
    } else {
      ElMessage.error(message)
    }
    const rejected = error as AxiosError & { code?: number; data?: any }
    rejected.code = code
    rejected.data = body?.data
    return Promise.reject(rejected)
  }
)

export { service }
