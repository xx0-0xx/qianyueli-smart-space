const config: {
  base_url: string
  result_code: number | string
  default_headers: string
  request_timeout: number
} = {
  /**
   * Go 后端 API 基础路径
   * 开发环境：VITE_BASE_URL 为空，走 Vite 代理
   * 生产环境：VITE_BASE_URL=http://localhost:8080
   */
  base_url: (import.meta.env.VITE_BASE_URL || '') + import.meta.env.VITE_API_URL,
  /**
   * 接口成功返回状态码（Go 后端 0 表示成功）
   */
  result_code: 0,

  /**
   * 接口请求超时时间
   */
  request_timeout: 30000,

  /**
   * 默认接口请求类型
   */
  default_headers: 'application/json'
}

export { config }
