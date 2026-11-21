var api = require('../../utils/api')
var STORE_KEY = 'selected_store_id'

Page({
  data: { amount: '', presets: [50, 100, 200, 500], submitted: false, amountYuan: '0' },

  onAmount: function (e) {
    this.setData({ amount: e.detail.value, submitted: false })
  },

  pickPreset: function (e) {
    this.setData({ amount: String(e.currentTarget.dataset.v), submitted: false })
  },

  submit: function () {
    var self = this
    var amount = parseFloat(this.data.amount)
    if (!amount || amount < 1) {
      wx.showToast({ title: '请输入有效金额', icon: 'none' })
      return
    }
    var storeId = wx.getStorageSync(STORE_KEY)
    if (!storeId) {
      wx.showToast({ title: '请先选择门店', icon: 'none' })
      return
    }

    api.getUserInfo().then(function (r) {
      var u = (r && r.data) || r || {}
      if (!u.phone) {
        self.promptBind()
        return
      }
      self.doSubmit(storeId, amount)
    }).catch(function (e) {
      wx.showToast({ title: (e && e.message) || '获取用户信息失败', icon: 'none' })
    })
  },

  doSubmit: function (storeId, amount) {
    var self = this
    wx.showLoading({ title: '提交中...' })
    api.createRecharge({ store_id: parseInt(storeId), amount: Math.round(amount * 100) }).then(function () {
      wx.hideLoading()
      self.setData({ submitted: true, amountYuan: amount.toFixed(2) })
    }).catch(function (e) {
      wx.hideLoading()
      if (e && e.code === 20005) {
        self.promptBind()
        return
      }
      wx.showToast({ title: (e && e.message) || '提交失败', icon: 'none' })
    })
  },

  promptBind: function () {
    wx.showModal({
      title: '请先绑定手机号',
      content: '充值前需要先绑定手机号，是否去绑定？',
      confirmText: '去绑定',
      success: function (res) {
        if (res.confirm) {
          wx.switchTab({ url: '/pages/mine/mine' })
        }
      }
    })
  }
})
