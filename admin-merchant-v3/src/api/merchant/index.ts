import request from '@/config/axios'

/** 公网 API 地址：音频等浏览器/设备直接访问的 URL */
export const API_ORIGIN = import.meta.env.VITE_BASE_URL || 'http://localhost:8080'
export const API_BASE = API_ORIGIN + '/api/v1'

export interface PageResult<T = any> {
  list: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface MerchantLoginResult {
  access_token: string
  user?: {
    id: number
    nickname: string
    role: string
    store_id?: number
  }
  store_id?: number
  store?: { id: number; name?: string }
}

// ────────── 登录 / 门店 ──────────
export const inviteLogin = (inviteCode: string) =>
  request.post<MerchantLoginResult>({
    url: '/merchant/invite-login',
    data: { invite_code: inviteCode }
  })

export interface CaptchaPayload {
  captcha_id?: string
  captcha_answer?: string
}

/** Web/桌面端登录：失败次数过多时要求验证码 */
export const inviteLoginSecure = async (
  inviteCode: string,
  captcha?: CaptchaPayload
): Promise<MerchantLoginResult> => {
  try {
    return await request.post<MerchantLoginResult>({
      url: '/merchant/invite-login/web',
      data: {
        invite_code: inviteCode,
        ...(captcha?.captcha_id
          ? { captcha_id: captcha.captcha_id, captcha_answer: captcha.captcha_answer || '' }
          : {})
      }
    })
  } catch (e: any) {
    // 后端旧版本尚未提供 /web 路由时，回退到原邀请码登录，保证桌面端可用。
    if (e?.response?.status === 404) {
      return inviteLogin(inviteCode)
    }
    throw e
  }
}

export const getMerchantStores = () => request.get<any[]>({ url: '/merchant/stores' })

export const bindMerchantStore = (bindCode: string) =>
  request.post({
    url: '/merchant/stores/bind',
    data: { bind_code: bindCode }
  })

// ────────── 房型 / 房间 ──────────
export const getResourceTypes = () => request.get<any[]>({ url: '/merchant/resource-types' })

export const createResourceType = (data: Record<string, any>) =>
  request.post({ url: '/merchant/resource-types', data })

export const deleteResourceType = (id: number) =>
  request.delete({ url: `/merchant/resource-types/${id}` })

export const getResources = () => request.get<any[]>({ url: '/merchant/resources' })

export const createResource = (data: Record<string, any>) =>
  request.post({ url: '/merchant/resources', data })

export const deleteResource = (id: number) => request.delete({ url: `/merchant/resources/${id}` })

export const linkDevices = (resourceId: number, deviceIds: number[]) =>
  request.put({
    url: `/merchant/resources/${resourceId}/devices`,
    data: { device_ids: JSON.stringify(deviceIds) }
  })

// ────────── 设备 ──────────
export const getDevices = () => request.get<any[]>({ url: '/merchant/devices' })

export const getProductTemplates = () => request.get<any[]>({ url: '/merchant/product-templates' })

export const createDevice = (data: Record<string, any>) =>
  request.post({ url: '/merchant/devices', data })

export const updateDevice = (id: number, data: Record<string, any>) =>
  request.put({ url: `/merchant/devices/${id}`, data })

export const deleteDevice = (id: number) => request.delete({ url: `/merchant/devices/${id}` })

export const controlDevice = (id: number, cmd: string) =>
  request.post({ url: `/devices/${id}/control`, data: { cmd } })

// ────────── 订单 / 退款 ──────────
export const getOrders = (params: Record<string, any> = {}) =>
  request.get<PageResult>({ url: '/merchant/orders', params })

export const getAllOrders = async () => {
  let page = 1
  const size = 100
  let all: any[] = []
  while (true) {
    const res = await getOrders({ page, size })
    const list = Array.isArray(res) ? res : res.list || []
    all = all.concat(list)
    const totalPages = Number(res.total_pages || 1)
    if (page >= totalPages || !list.length) break
    page++
  }
  return all
}

export const requestRefund = (orderId: number) =>
  request.post({ url: `/orders/${orderId}/refund-request` })

export const getRefundRequests = () => request.get<any[]>({ url: '/merchant/refund-requests' })

export const approveRefund = (id: number) =>
  request.post({ url: `/merchant/refund-requests/${id}/approve` })

export const rejectRefund = (id: number) =>
  request.post({ url: `/merchant/refund-requests/${id}/reject` })

export const merchantCancelOrder = (id: number) =>
  request.post({ url: `/merchant/orders/${id}/cancel` })

export const merchantDirectRefund = (id: number) =>
  request.post({ url: `/merchant/orders/${id}/refund` })

// ────────── 定价 ──────────
export const getPriceRules = () => request.get<any[]>({ url: '/merchant/config/price-rules' })

export const createPriceRule = (data: Record<string, any>) =>
  request.post({ url: '/merchant/config/price-rules', data })

export const replacePriceRules = (rules: any[]) =>
  request.post({ url: '/merchant/config/price-rules/bulk', data: { rules } })

// ────────── 券码 ──────────
export const getCoupons = () => request.get<any[]>({ url: '/merchant/coupons' })

export const createCoupon = (data: Record<string, any>) =>
  request.post<{ count?: number }>({ url: '/merchant/coupons', data })

export const deleteCoupon = (id: number) => request.delete({ url: `/merchant/coupons/${id}` })

// ────────── 充值 ──────────
export const getRecharges = (status: string | number) =>
  request.get<any[]>({ url: '/merchant/recharges', params: { status } })

export const confirmRecharge = (id: number) =>
  request.post({ url: `/merchant/recharges/${id}/confirm`, data: {} })

export const cancelRecharge = (id: number) =>
  request.post({ url: `/merchant/recharges/${id}/cancel`, data: {} })

// ────────── 门店设置 ──────────
export const getStoreConfig = () =>
  request.get<Record<string, any>>({ url: '/merchant/store-config' })

export const updateStoreConfig = (data: Record<string, any>) =>
  request.put({ url: '/merchant/store-config', data })

// ────────── 成品控制器语音播报 ──────────
export interface VoiceReminder {
  minutes_before_end: number
  command: string
}

export interface VoiceDevice {
  device_sn: string
  name?: string
  enabled: boolean
  on_start: string
  on_end: string
  reminders: VoiceReminder[]
}

export const getVoiceConfig = () =>
  request.get<{ devices: VoiceDevice[] }>({ url: '/merchant/voice-config' })

export const getVoiceDevices = () =>
  request.get<any[]>({ url: '/merchant/voice-devices' })

export const saveVoiceConfig = (devices: VoiceDevice[]) =>
  request.put({ url: '/merchant/voice-config', data: { devices } })

// ────────── 会员 ──────────
export interface MemberLevel {
  id: number
  name: string
  discount_bp: number
  sort: number
  status: number
  member_count?: number
}

export interface StoreCustomer {
  id: number
  nickname: string
  avatar: string
  phone: string
  order_count: number
  total_spent: number
  last_order_at: number
  level_id?: number
  level_name?: string
  level_status?: number
  expires_at?: number | null
  member_status?: number
  remark?: string
}

export const getMemberLevels = () => request.get<MemberLevel[]>({ url: '/merchant/member-levels' })

export const createMemberLevel = (data: Partial<MemberLevel>) =>
  request.post({ url: '/merchant/member-levels', data })

export const updateMemberLevel = (id: number, data: Partial<MemberLevel>) =>
  request.put({ url: `/merchant/member-levels/${id}`, data })

export const deleteMemberLevel = (id: number) =>
  request.delete({ url: `/merchant/member-levels/${id}` })

export const getCustomers = (params: Record<string, any> = {}) =>
  request.get<PageResult<StoreCustomer>>({ url: '/merchant/customers', params })

export interface CustomerOrder {
  order_no: string
  resource_name: string
  status: number
  total_amount: number
  start_time: number
  end_time: number
  created_at: string
}

export interface CustomerRecharge {
  id: number
  amount: number
  status: number
  created_at: string
}

export interface CustomerDetail {
  id: number
  nickname: string
  avatar: string
  phone: string
  level_id: number
  level_name: string
  level_status?: number
  expires_at?: number | null
  member_status: number
  remark: string
  orders: CustomerOrder[]
  recharges: CustomerRecharge[]
}

export const getCustomer = (userId: number) =>
  request.get<CustomerDetail>({ url: `/merchant/customers/${userId}` })

export const setCustomerMembership = (
  userId: number,
  data: { level_id: number; expires_at?: number | null; remark?: string }
) => request.put({ url: `/merchant/customers/${userId}/membership`, data })

export const removeCustomerMembership = (userId: number) =>
  request.delete({ url: `/merchant/customers/${userId}/membership` })
