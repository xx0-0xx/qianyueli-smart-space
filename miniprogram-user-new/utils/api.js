const BASE = 'http://localhost:8080/api/v1'

function unwrap(res) {
  const d = res.data
  // 分页格式 {list:[], total:...} → 直接返回 list
  if (d && d.list) return d.list
  // 普通格式 {token:..., user_id:...}
  return d
}

function request(method, url, data) {
  const app = getApp()
  return new Promise((resolve, reject) => {
    wx.request({
      url: BASE + url,
      method,
      data: method === 'GET' ? data : JSON.stringify(data),
      header: (function() {
        var t = app.globalData.token || wx.getStorageSync('token')
        return t ? { Authorization: 'Bearer ' + t, 'Content-Type': 'application/json' } : { 'Content-Type': 'application/json' }
      })(),
      success(res) {
        const body = res.data
        if (body.code === 0) resolve({ data: unwrap(body), raw: body })
        else if (body.code === 20003 || body.code === 20004 || body.code === 10006) {
          // token 过期，清除并跳登录
          getApp().globalData.token = ''
          wx.removeStorageSync('token')
          wx.reLaunch({ url: '/pages/login/login' })
          reject({ code: body.code, message: body.message })
        }
        else reject({ code: body.code, message: body.message })
      },
      fail(err) { reject({ code: -1, message: err.errMsg || '网络错误' }) }
    })
  })
}

module.exports = {
  login:        (code, role) => request('POST', '/login', { code, role }),
  getStores:    () => request('GET', '/stores'),
  getStore:     (id) => request('GET', '/stores/' + id),
  wechatLogin: (code) => request('POST', '/login/wechat', { code }),
  getResources: (storeId) => request('GET', '/stores/' + storeId + '/resources'),
  getSlotResources: (storeId, st, et) => request('GET', '/stores/' + storeId + '/slots/resources?start=' + st + '&end=' + et),
  createOrder:  (data) => request('POST', '/orders', data),
  getOrders:    (page, size) => request('GET', '/orders?page=' + (page || 1) + '&size=' + (size || 20)).then(res => ({ data: res.raw.data, raw: res.raw })),
  cancelOrder:  (id) => request('POST', '/orders/' + id + '/cancel'),
  refundOrder:  (id, reason) => request('POST', '/orders/' + id + '/refund-request', {reason: reason || ''}),
  mockPay:      (id) => request('POST', '/orders/' + id + '/mock-pay'),
  controlDevice: (id, cmd) => request('POST', '/devices/' + id + '/control', { cmd }),
  getDeviceStatus: (id) => request('GET', '/devices/' + id + '/status'),
  getUserInfo:  () => request('GET', '/user/info'),
  updateUserInfo: (data) => request('PUT', '/user/info', data),
  getStorePrice: (id) => request('GET', '/stores/' + id + '/price'),
  getStoreOccupied: (id) => request('GET', '/stores/' + id + '/occupied'),
  getCouponInfo: (code) => request('GET', '/coupons/'+code+'/info'),
  redeemCoupon: (code) => request('POST', '/coupons/redeem', {code:code}),
  getElysVoucherConfig: (storeId) => request('GET', '/user/elys-vouchers/config?store_id=' + storeId),
  prepareElysVoucher: (data) => request('POST', '/user/elys-vouchers/prepare', data),
  consumeElysVoucher: (prepareToken) => request('POST', '/user/elys-vouchers/consume', { prepare_token: prepareToken }),
  createRecharge: (data) => request('POST', '/recharges', data),
  getRecharges: () => request('GET', '/recharges'),
  getBalance: (storeId) => request('GET', '/balance?store_id=' + storeId),
  bindWechatPhone: (code) => request('POST', '/user/wechat-phone', { code }),
  getMyMembership: (storeId) => request('GET', '/stores/' + storeId + '/my-membership'),
  getMyPrice: (storeId, resourceTypeId) => request('GET', '/stores/' + storeId + '/my-price?resource_type_id=' + resourceTypeId),
}
