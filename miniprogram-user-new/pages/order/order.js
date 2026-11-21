const api = require('../../utils/api')
const PAGE_SIZE = 20

Page({
  data: { orders: [], loading: false, loadingMore: false, error: '', page: 1, hasMore: false },
  onShow() {
    if (!getApp().globalData.token) return
    this.load(true)
  },
  onReachBottom() { if (this.data.hasMore) this.load(false) },
  onPullDownRefresh() {
    this.load(true).finally(function () { wx.stopPullDownRefresh() })
  },
  async load(reset) {
    if (this.data.loading || this.data.loadingMore) return
    const page = reset ? 1 : this.data.page + 1
    this.setData(reset ? { loading: true, error: '' } : { loadingMore: true, error: '' })
    try {
      const r = await api.getOrders(page, PAGE_SIZE)
      const d = r.data || r
      const items = (Array.isArray(d) ? d : (d.list || [])).map(o => {
        o.timeRange = this.formatRange(o.start_time, o.end_time)
        o.priceYuan = (Number(o.total_amount || 0) / 100).toFixed(2)
        o.original_amount = Number(o.original_amount || 0)
        o.member_discount = Number(o.member_discount || 0)
        o.originalAmountYuan = (o.original_amount / 100).toFixed(2)
        o.memberDiscountYuan = (o.member_discount / 100).toFixed(2)
        const now = Date.now()
        o.canControl = [1, 2].indexOf(o.status) >= 0 && now >= o.start_time - 900000 && now < o.end_time
        return o
      })
      const orders = reset ? items : this.data.orders.concat(items)
      const totalPages = Number(d.total_pages) || 1
      this.setData({
        orders: orders,
        page: page,
        hasMore: page < totalPages && items.length > 0,
        loading: false,
        loadingMore: false,
        error: ''
      })
    } catch (e) {
      this.setData({ loading: false, loadingMore: false, error: (e && e.message) || '订单加载失败，请检查网络后重试' })
    }
  },
  retry() { this.load(true) },
  formatRange(start, end) {
    const pad = n => n < 10 ? '0' + n : '' + n
    const s = new Date(start), e = new Date(end)
    const fmt = d => (d.getMonth() + 1) + '/' + d.getDate()
    const st = pad(s.getHours()) + ':' + pad(s.getMinutes())
    const et = pad(e.getHours()) + ':' + pad(e.getMinutes())
    return fmt(s) === fmt(e) ? fmt(s) + ' ' + st + ' ~ ' + et : fmt(s) + ' ' + st + ' ~ ' + fmt(e) + ' ' + et
  },
  refund(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: '申请退款',
      editable: true,
      placeholderText: '请填写退款理由（如：临时有事来不了）',
      success: async (r) => {
        if (!r.confirm) return
        const reason = (r.content || '').trim()
        if (!reason) { wx.showToast({ title: '请填写退款理由', icon: 'none' }); return }
        try {
          await api.refundOrder(id, reason)
          wx.showToast({ title: '退款申请已提交' })
          this.load(true)
        } catch (e) {
          wx.showToast({ title: (e && e.message) || '提交失败，请重试', icon: 'none' })
        }
      }
    })
  },
  goControl(e) {
    const { rid, sid, oid, end } = e.currentTarget.dataset
    wx.navigateTo({ url: '/pages/control/control?rid=' + rid + '&sid=' + sid + '&oid=' + oid + '&end=' + end })
  }
})