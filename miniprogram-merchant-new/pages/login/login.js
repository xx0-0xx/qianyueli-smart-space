var api = require('../../utils/api')

Page({
  data: { inviteCode: '', loading: false, agreed: false },

  onLoad: function () {
    var app = getApp()
    var token = wx.getStorageSync('token')
    var storeId = wx.getStorageSync('merchantSid')

    // 已有 token + 已绑定门店 → 直接进后台
    if (token && storeId) {
      app.globalData.token = token
      app.globalData.storeId = storeId
      wx.switchTab({ url: '/pages/dashboard/dashboard' })
      return
    }
    // 仅记住邀请码占位，不再自动登录（邀请码需配合 wx.login 换 token）
    var savedCode = wx.getStorageSync('merchantInviteCode')
    if (savedCode) this.setData({ inviteCode: savedCode })
  },

  onInput: function (e) {
    // 邀请码统一大写
    this.setData({ inviteCode: (e.detail.value || '').toUpperCase() })
  },

  toggleAgree: function () {
    this.setData({ agreed: !this.data.agreed })
  },

  openAgreement: function () {
    wx.navigateTo({ url: '/pages/agreement/agreement' })
  },

  openPrivacy: function () {
    wx.navigateTo({ url: '/pages/privacy/privacy' })
  },

  // 重新登录（设置页/退出后调用）
  logout: function () {
    wx.removeStorageSync('token')
    wx.removeStorageSync('merchantSid')
    getApp().globalData.token = ''
    getApp().globalData.storeId = ''
    this.setData({ inviteCode: '' })
  },

  enter: function () {
    var self = this
    var code = (this.data.inviteCode || '').trim()
    if (!code) { wx.showToast({ title: '请输入邀请码', icon: 'none' }); return }
    if (!this.data.agreed) { wx.showToast({ title: '请先阅读并同意协议', icon: 'none' }); return }
    if (this.data.loading) return
    this.setData({ loading: true })

    // 邀请码直接登录（商家端为独立小程序，不走微信 code2session）
    api.inviteLogin(code).then(function (r) {
      var token = r && r.access_token
      if (!token) {
        self.setData({ loading: false })
        wx.showToast({ title: '登录失败', icon: 'none' })
        return
      }
      getApp().globalData.token = token
      wx.setStorageSync('token', token)
      // 门店ID
      var sid = (r.user && r.user.store_id) || r.store_id || (r.store && r.store.id)
      if (sid) {
        sid = parseInt(sid)
        getApp().globalData.storeId = sid
        wx.setStorageSync('merchantSid', sid)
      }
      wx.setStorageSync('merchantInviteCode', code)
      self.setData({ loading: false })
      wx.showToast({ title: '进入成功', icon: 'success' })
      setTimeout(function () {
        wx.switchTab({ url: '/pages/dashboard/dashboard' })
      }, 600)
    }).catch(function (e) {
      self.setData({ loading: false })
      var msg = (e && (e.message || e.errMsg)) || (typeof e === 'string' ? e : JSON.stringify(e))
      wx.showModal({ title: '登录失败', content: String(msg).substring(0, 200), showCancel: false })
    })
  }
})
