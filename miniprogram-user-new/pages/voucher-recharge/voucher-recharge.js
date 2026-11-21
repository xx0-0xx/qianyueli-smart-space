const api = require('../../utils/api')

Page({
  data: {
    storeId: '',
    platforms: [],
    platform: '',
    code: '',
    prepare: null,
    loading: false,
    submitting: false,
    result: null,
    error: ''
  },

  onShow() {
    const storeId = wx.getStorageSync('selected_store_id')
    if (!storeId) {
      this.setData({ error: '请先在首页选择门店' })
      return
    }
    this.setData({ storeId: storeId, error: '' })
    this.loadConfig(storeId)
  },

  loadConfig(storeId) {
    const self = this
    api.getElysVoucherConfig(storeId).then(function (r) {
      const d = r.data || r
      const platforms = (d.platforms || []).filter(function (p) { return p.authorized })
      self.setData({
        platforms: platforms,
        platform: platforms.length ? platforms[0].platform : '',
        error: d.enabled ? '' : '该门店暂未开通团购券充值'
      })
    }).catch(function () {
      self.setData({ error: '团购券配置加载失败' })
    })
  },

  pickPlatform(e) {
    this.setData({ platform: e.currentTarget.dataset.value, prepare: null, result: null })
  },

  scanCode() {
    const self = this
    wx.scanCode({
      success(res) {
        self.setData({ code: String(res.result || '').trim(), prepare: null, result: null })
      }
    })
  },

  onCodeInput(e) {
    this.setData({ code: e.detail.value, prepare: null, result: null })
  },

  prepare() {
    const d = this.data
    if (!d.platform) return wx.showToast({ title: '请选择平台', icon: 'none' })
    if (!d.code) return wx.showToast({ title: '请扫码或输入券码', icon: 'none' })
    const self = this
    this.setData({ loading: true, error: '' })
    api.prepareElysVoucher({ store_id: Number(d.storeId), platform: d.platform, code: d.code }).then(function (r) {
      const p = r.data || r
      p.payAmountYuan = ((Number(p.pay_amount) || 0) / 100).toFixed(2)
      p.actualPayAmountYuan = ((Number(p.actual_pay_amount) || 0) / 100).toFixed(2)
      self.setData({ loading: false, prepare: p })
    }).catch(function (e) {
      self.setData({ loading: false })
      wx.showToast({ title: (e && e.message) || '验券失败', icon: 'none' })
    })
  },

  confirm() {
    const p = this.data.prepare
    if (!p) return
    const self = this
    this.setData({ submitting: true })
    api.consumeElysVoucher(p.prepare_token).then(function (r) {
      const d = r.data || r
      d.amountYuan = ((Number(d.amount) || 0) / 100).toFixed(2)
      d.newBalanceYuan = ((Number(d.new_balance) || 0) / 100).toFixed(2)
      self.setData({ submitting: false, prepare: null, result: d })
      wx.showModal({
        title: d.status === 'pending_credit' ? '核销成功' : '充值成功',
        content: d.status === 'pending_credit'
          ? '核销成功，余额处理中，请稍后查看'
          : '已充值 ¥' + d.amountYuan + '，当前余额 ¥' + d.newBalanceYuan,
        showCancel: false
      })
    }).catch(function (e) {
      self.setData({ submitting: false })
      wx.showToast({ title: (e && e.message) || '核销失败', icon: 'none' })
    })
  }
})
