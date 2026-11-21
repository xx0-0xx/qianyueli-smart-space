const api = require('../../utils/api')
function fmtTime(ms){
  if(!ms)return ''
  var d=new Date(ms)
  var p=function(n){return n<10?'0'+n:''+n}
  return (d.getMonth()+1)+'月'+d.getDate()+'日 '+p(d.getHours())+':'+p(d.getMinutes())
}
function fmtDate(ms){
  var d=new Date(ms)
  var p=function(n){return n<10?'0'+n:''+n}
  return d.getFullYear()+'-'+p(d.getMonth()+1)+'-'+p(d.getDate())
}
function maskPhone(phone){
  if(!phone)return ''
  return String(phone).replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2')
}
Page({
  data: { user: {}, displayPhone: '', balance: 0, balanceYuan: '0.00', recharges: [], balanceLogs: [], membership: {}, loading: false, error: '', needStore: false, showBalanceLogs: false, showRecharges: false },
  onShow() {
    if (getApp().globalData.token) {
      this.load()
    }
  },
  async load() {
    this.setData({ loading: true, error: '' })
    const storeId = wx.getStorageSync('selected_store_id')
    const tasks = [api.getUserInfo(), api.getRecharges()]
    if (storeId) tasks.push(api.getBalance(storeId))
    let membershipIdx = -1
    if (storeId && getApp().globalData.token) {
      membershipIdx = tasks.length
      tasks.push(api.getMyMembership(storeId))
    }
    const results = await Promise.allSettled(tasks)
    const updates = { loading: false, needStore: !storeId, membership: {} }
    if (results[0].status === 'fulfilled') {
      const u = results[0].value.data || results[0].value
      updates.user = u
      updates.displayPhone = maskPhone(u.phone)
    }
    else updates.error = '账户资料暂时无法加载'
    if (results[1].status === 'fulfilled') {
      const value = results[1].value.data || results[1].value
      const list = Array.isArray(value) ? value : (value.list || [])
      updates.recharges = list.map(item => Object.assign({}, item, {
        amountYuan: (Number(item.amount || 0) / 100).toFixed(2),
        timeText: fmtTime(item.created_at)
      }))
    }
    if (storeId && results[2]) {
      if (results[2].status === 'fulfilled') {
        const d = results[2].value.data || results[2].value
        updates.balance = Number(d.balance) || 0
        updates.balanceYuan = (updates.balance / 100).toFixed(2)
        updates.balanceLogs = (d.logs || []).map(item => Object.assign({}, item, {
          amountYuan: (Number(item.amount || 0) / 100).toFixed(2),
          timeText: fmtTime(item.created_at)
        }))
      } else updates.error = updates.error || '余额信息暂时无法加载'
    } else {
      updates.balance = 0
      updates.balanceYuan = '0.00'
      updates.balanceLogs = []
    }
    if (membershipIdx >= 0 && results[membershipIdx] && results[membershipIdx].status === 'fulfilled') {
      const m = results[membershipIdx].value.data || results[membershipIdx].value
      if (m && m.is_member) {
        m.expires_text = m.expires_at ? '有效期至 ' + fmtDate(m.expires_at) : '永久有效'
        updates.membership = m
      }
    }
    this.setData(updates)
  },
  onGetPhoneNumber(e) {
    if (!e.detail.code) {
      wx.showToast({ title: '未授权手机号', icon: 'none' })
      return
    }
    const self = this
    wx.showLoading({ title: '绑定中...' })
    api.bindWechatPhone(e.detail.code).then((r) => {
      const d = r.data || r
      wx.hideLoading()
      self.setData({ 'user.phone': d.phone, displayPhone: maskPhone(d.phone) })
      wx.showToast({ title: '手机号已绑定' })
      self.load().then(function() {
        setTimeout(function() { self.editNickname() }, 500)
      })
    }).catch((err) => {
      wx.hideLoading()
      wx.showToast({ title: (err && err.message) || '手机号绑定失败', icon: 'none' })
    })
  },
  goRecharge() {
    wx.navigateTo({ url: '/pages/recharge/recharge' })
  },
  goVoucherRecharge() {
    wx.navigateTo({ url: '/pages/voucher-recharge/voucher-recharge' })
  },
  toggleBalanceLogs() {
    this.setData({ showBalanceLogs: !this.data.showBalanceLogs })
  },
  toggleRecharges() {
    this.setData({ showRecharges: !this.data.showRecharges })
  },
  editNickname() {
    var self = this
    var cur = this.data.user.nickname || ''
    wx.showModal({
      title: '设置昵称',
      editable: true,
      placeholderText: '请输入昵称（1-20字）',
      content: cur,
      success: function(res) {
        if (!res.confirm) return
        var nick = (res.content || '').trim()
        if (!nick) { wx.showToast({ title: '昵称不能为空', icon: 'none' }); return }
        if (nick.length > 20) { wx.showToast({ title: '昵称最多20字', icon: 'none' }); return }
        wx.showLoading({ title: '保存中...' })
        api.updateUserInfo({ nickname: nick }).then(function() {
          wx.hideLoading()
          wx.showToast({ title: '已保存' })
          self.load()
        }).catch(function(e) {
          wx.hideLoading()
          wx.showToast({ title: (e && e.message) || '保存失败', icon: 'none' })
        })
      }
    })
  },
  openAgreement() { wx.navigateTo({ url: '/pages/agreement/agreement' }) },
  openPrivacy() { wx.navigateTo({ url: '/pages/privacy/privacy' }) },
  onShareAppMessage() { return { title: '金之莱无人空间', path: '/pages/index/index' } }
})
