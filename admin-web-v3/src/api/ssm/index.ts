import request from '@/config/axios'

/** 公网 API 地址：用于二维码、音频等浏览器/设备直接访问的 URL */
export const API_ORIGIN = import.meta.env.VITE_BASE_URL || 'http://localhost:8080'
export const API_BASE = API_ORIGIN + '/api/v1'

export interface LoginVO {
  username: string
  password: string
  captcha_id?: string
  captcha_answer?: string
}

export interface LoginResultVO {
  access_token: string
  user: {
    id: number
    nickname: string
    role: string
    store_id?: number
  }
}

export interface PageResult<T = any> {
  list: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// ────────── 登录 ──────────
export const login = async (data: LoginVO): Promise<LoginResultVO> => {
  try {
    return await request.post<LoginResultVO>({
      url: '/admin/login/web',
      data
    })
  } catch (e: any) {
    // 后端旧版本尚未提供 /web 路由时，回退到原登录接口。
    if (e?.response?.status === 404) {
      return request.post<LoginResultVO>({ url: '/admin/login', data })
    }
    throw e
  }
}

// ────────── 门店 ──────────
export const getStores = (params?: Record<string, any>) =>
  request.get<PageResult>({
    url: '/admin/stores',
    params: { size: 500, ...params }
  })

export const createStore = (data: Record<string, any>) =>
  request.post<{ id: number }>({ url: '/admin/stores', data })

export const updateStore = (id: number, data: Record<string, any>) =>
  request.put({ url: `/admin/stores/${id}`, data })

export const deleteStore = (id: number) =>
  request.delete({ url: `/admin/stores/${id}` })

export const forceDeleteStore = (id: number) =>
  request.delete({ url: `/admin/stores/${id}/force` })

export const regenerateInviteCode = (id: number) =>
  request.post<{ invite_code: string }>({
    url: `/admin/stores/${id}/invite-code`
  })

export const storeQRCodeUrl = (id: number, token: string) =>
  `${API_BASE}/admin/stores/${id}/qrcode?token=${encodeURIComponent(token)}`

// ────────── 余额组 ──────────
export const getBalanceGroups = () =>
  request.get<any[]>({ url: '/admin/balance-groups' })

export const createBalanceGroup = (data: { name: string }) =>
  request.post<{ id: number; name: string }>({
    url: '/admin/balance-groups',
    data
  })

export const setStoreBalanceGroup = (storeId: number, groupId: number) =>
  request.put({
    url: `/admin/stores/${storeId}/balance-group`,
    data: { group_id: groupId }
  })

// ────────── 房型 ──────────
export const getResourceTypes = (storeId?: number) =>
  request.get<any[]>({
    url: '/merchant/resource-types',
    params: storeId ? { store_id: storeId } : undefined
  })

export const createResourceType = (data: Record<string, any>) =>
  request.post({ url: '/merchant/resource-types', data })

export const updateResourceType = (id: number, data: Record<string, any>) =>
  request.put({ url: `/merchant/resource-types/${id}`, data })

export const deleteResourceType = (id: number) =>
  request.delete({ url: `/merchant/resource-types/${id}` })

// ────────── 房间 / 资源 ──────────
export const getResources = (storeId: number) =>
  request.get<any[]>({ url: `/stores/${storeId}/resources` })

export const createResource = (data: Record<string, any>) =>
  request.post({ url: '/merchant/resources', data })

export const updateResource = (id: number, data: Record<string, any>) =>
  request.put({ url: `/merchant/resources/${id}`, data })

export const linkDevices = (resourceId: number, deviceIds: number[]) =>
  request.put({
    url: `/merchant/resources/${resourceId}/devices`,
    data: { device_ids: JSON.stringify(deviceIds) }
  })

export const deleteResource = (id: number) =>
  request.delete({ url: `/merchant/resources/${id}` })

// ────────── 设备 ──────────
export const getDevices = (params?: Record<string, any>) =>
  request.get<PageResult>({
    url: '/admin/devices',
    params: { size: 500, ...params }
  })

export const createDevice = (data: Record<string, any>) =>
  request.post({ url: '/admin/devices', data })

export const updateDevice = (id: number, data: Record<string, any>) =>
  request.put({ url: `/admin/devices/${id}`, data })

export const deleteDevice = (id: number) =>
  request.delete({ url: `/admin/devices/${id}` })

export const probeDevice = (deviceSn: string) =>
  request.post<{ channels: number; raw?: string }>({
    url: '/admin/devices/probe',
    data: { device_sn: deviceSn }
  })

export const speakerTest = (deviceSn: string, url: string) =>
  request.post({
    url: `/admin/devices/${deviceSn}/speaker-test`,
    data: { url }
  })

export const controlDevice = (id: number, cmd: string) =>
  request.post({ url: `/devices/${id}/control`, data: { cmd } })

// ────────── 音频 ──────────
export const getAudioFiles = () =>
  request.get<any[]>({ url: '/admin/audio' })

export const uploadAudio = (formData: FormData) =>
  request.upload<{ code: number; data: any }>({
    url: '/admin/audio/upload',
    data: formData
  })

export const deleteAudio = (name: string) =>
  request.delete({ url: `/admin/audio/${encodeURIComponent(name)}` })

export const getAudioConfigs = () =>
  request.get<any[]>({ url: '/merchant/audio-configs' })

export const saveAudioConfig = (deviceSn: string, data: Record<string, any>) =>
  request.put({
    url: `/merchant/audio-config/${encodeURIComponent(deviceSn)}`,
    data
  })

// ────────── 语音播报 ──────────
export interface VoiceReminder {
  minutes_before_end: number
  command: string
}

export interface VoiceDevice {
  device_sn: string
  enabled: boolean
  on_start: string
  on_end: string
  reminders: VoiceReminder[]
}

export const getVoiceConfig = () =>
  request.get<{ devices?: VoiceDevice[] } & Record<string, any>>({
    url: '/admin/voice-config'
  })

export const saveVoiceConfig = (devices: VoiceDevice[]) =>
  request.put({ url: '/admin/voice-config', data: { devices } })

// ────────── 产品模板 ──────────
export const getProductTemplates = () =>
  request.get<any[]>({ url: '/admin/product-templates' })

export const createProductTemplate = (data: Record<string, any>) =>
  request.post({ url: '/admin/product-templates', data })

export const updateProductTemplate = (id: number, data: Record<string, any>) =>
  request.put({ url: `/admin/product-templates/${id}`, data })

export const moveProductTemplate = (id: number, direction: 'up' | 'down') =>
  request.post({ url: `/admin/product-templates/${id}/move`, params: { direction } })

export const deleteProductTemplate = (id: number) =>
  request.delete({ url: `/admin/product-templates/${id}` })

// ────────── 意向客户 ──────────
export const getLeads = (params: Record<string, any>) =>
  request.get<PageResult>({ url: '/admin/leads', params })

export const updateLead = (id: number, data: Record<string, any>) =>
  request.put({ url: `/admin/leads/${id}`, data })

// ────────── 统计 / 订单 ──────────
export const getOverview = () =>
  request.get<{
    total_stores: number
    total_users: number
    total_orders: number
    total_revenue: number
  }>({ url: '/admin/statistics/overview' })

export const getOrders = (params?: Record<string, any>) =>
  request.get<PageResult>({
    url: '/merchant/orders',
    params: { size: 500, ...params }
  })

// Elys bindings
export interface ElysBindingVO {
  id: string
  store_id: string
  store_name: string
  elys_shop_id: number
  elys_master_shop_id: number
  meituan_authorized: boolean
  douyin_authorized: boolean
  status: number
  last_error?: string
}

export const getElysBindings = () => request.get<ElysBindingVO[]>({ url: '/admin/elys/bindings' })
export const upsertElysBinding = (data: Record<string, any>) =>
  request.post({ url: '/admin/elys/bindings', data })
export const deleteElysBinding = (storeId: string) =>
  request.delete({ url: `/admin/elys/bindings/${storeId}` })
export const getElysAuthUrl = (storeId: string, platform: 'meituan' | 'douyin') =>
  request.post<{ url: string }>({ url: `/admin/elys/bindings/${storeId}/auth-url`, data: { platform } })
export const getElysVerifications = (params: Record<string, any>) =>
  request.get({ url: '/admin/elys/verifications', params })
export const retryElysCredit = (id: string) =>
  request.post({ url: `/admin/elys/verifications/${id}/retry-credit` })
