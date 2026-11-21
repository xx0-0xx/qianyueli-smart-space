var api = require('../../utils/api')

function checkPhone() {
  api.getUserInfo().then(function (res) {
    var user = res.data || res
    getApp().globalData.user = user
    if (!user.phone) {
      wx.showModal({
        title: '绑定手机号',
        content: '绑定手机号后可用于预约和会员服务',
        confirmText: '去绑定',
        cancelText: '稍后',
        success: function (r) {
          if (r.confirm) wx.switchTab({ url: '/pages/mine/mine' })
        }
      })
    }
  }).catch(function () {})
}

function enterApp() {
  wx.switchTab({
    url: '/pages/index/index',
    success: function () { checkPhone() }
  })
}

Page({
  data: { showConsent: false },

  onLoad: function () {
    var agreed = wx.getStorageSync('userPrivacyAgreed')
    if (!agreed) { this.setData({ showConsent: true }); return }
    this.doLogin()
  },

  agree: function () {
    wx.setStorageSync('userPrivacyAgreed', true)
    this.setData({ showConsent: false })
    this.doLogin()
  },

  disagree: function () {
    wx.showToast({ title: '需同意后方可使用', icon: 'none' })
  },

  openAgreement: function () {
    wx.navigateTo({ url: '/pages/agreement/agreement' })
  },

  openPrivacy: function () {
    wx.navigateTo({ url: '/pages/privacy/privacy' })
  },

  doLogin: function () {
    var t = wx.getStorageSync('token')
    if (t) { getApp().globalData.token = t; enterApp(); return }
    wx.login({
      success: function (r) {
        if (!r.code) { wx.showToast({ title: '登录失败', icon: 'none' }); return }
        api.wechatLogin(r.code).then(function (u) {
          var token = (u.data || u).access_token
          if (token) { getApp().saveToken(token); enterApp() }
          else { wx.showToast({ title: '获取凭证失败', icon: 'none' }) }
        }).catch(function (e) {
          wx.showToast({ title: e.message || '登录失败', icon: 'none' })
        })
      },
      fail: function () { wx.showToast({ title: '微信登录失败', icon: 'none' }) }
    })
  }
})
