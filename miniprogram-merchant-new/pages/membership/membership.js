var api = require('../../utils/api')

function pad(n) {
  return n < 10 ? '0' + n : '' + n
}

function fmtDate(ms) {
  if (!ms) return ''
  var d = new Date(Number(ms))
  if (isNaN(d.getTime())) return ''
  return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate())
}

function todayStr() {
  return fmtDate(Date.now())
}

function endOfDay(dateStr) {
  var parts = String(dateStr || '').split('-')
  if (parts.length !== 3) return null
  var y = Number(parts[0])
  var m = Number(parts[1])
  var d = Number(parts[2])
  if (!y || !m || !d) return null
  var t = new Date(y, m - 1, d, 23, 59, 59, 999).getTime()
  return isNaN(t) ? null : t
}

function money(cents) {
  return (Number(cents || 0) / 100).toFixed(2)
}

function decorateCustomer(c) {
  var levelId = Number(c.level_id || 0)
  var isMember = Number(c.member_status) === 1 && levelId > 0
  var expiresAt = c.expires_at ? Number(c.expires_at) : 0
  return {
    id: c.id,
    nickname: c.nickname || '用户',
    phone: c.phone || '',
    avatar: c.avatar || '',
    order_count: Number(c.order_count || 0),
    totalSpent: money(c.total_spent),
    levelId: levelId,
    levelName: c.level_name || '',
    levelStatus: Number(c.level_status || 0),
    isMember: isMember,
    expires_at: expiresAt,
    expiryText: isMember ? (expiresAt ? fmtDate(expiresAt) : '长期有效') : '',
    expired: !!(isMember && expiresAt && expiresAt < Date.now()),
    remark: c.remark || ''
  }
}

Page({
  data: {
    sid: '',
    noStore: false,
    keyword: '',
    keywordInput: '',
    customers: [],
    page: 1,
    pageSize: 20,
    total: 0,
    totalPages: 0,
    hasMore: false,
    loading: false,
    loadingMore: false,
    error: '',
    levels: [],
    activeLevels: [],
    levelsLoaded: false,
    levelsError: '',
    showSheet: false,
    selectedCustomer: null,
    selectedLevelId: '',
    expiresDate: '',
    remark: '',
    saving: false,
    removingId: '',
    todayStr: todayStr()
  },
  _seq: 0,
  _levelsSeq: 0,
  _searchTimer: null,

  onShow() {
    this.setData({ todayStr: todayStr() })
    this.init()
  },
  onUnload() {
    if (this._searchTimer) clearTimeout(this._searchTimer)
  },
  onReachBottom() {
    this.loadMore()
  },
  init() {
    var sid = String(getApp().globalData.storeId || wx.getStorageSync('merchantSid') || '')
    if (!sid) {
      this._seq += 1
      this._levelsSeq += 1
      this.setData({
        sid: '', noStore: true, loading: false, loadingMore: false,
        customers: [], total: 0, totalPages: 0, hasMore: false,
        error: '', showSheet: false, levels: [], activeLevels: [],
        levelsLoaded: false, levelsError: '', saving: false, removingId: ''
      })
      return
    }
    this.setData({ sid: sid, noStore: false })
    this.loadLevels()
    this.reload()
  },
  loadLevels() {
    var self = this
    var sid = this.data.sid
    if (!sid) return
    var seq = ++this._levelsSeq
    this.setData({ levelsLoaded: false, levelsError: '' })
    api.getMemberLevels(sid).then(function(r) {
      if (seq !== self._levelsSeq) return
      var list = Array.isArray(r) ? r : ((r && r.list) || [])
      list = list.map(function(level) {
        var copy = Object.assign({}, level)
        copy.discountText = parseFloat((Number(level.discount_bp || 0) / 1000).toFixed(2)) + '折'
        return copy
      })
      var active = list.filter(function(level) { return Number(level.status) === 1 })
      var payload = { levels: list, activeLevels: active, levelsLoaded: true, levelsError: '' }
      var hasSelected = active.some(function(level) { return String(level.id) === String(self.data.selectedLevelId) })
      if (self.data.showSheet && !hasSelected && active.length > 0) {
        var customer = self.data.selectedCustomer || {}
        var currentMatch = active.find(function(level) { return String(level.id) === String(customer.levelId) })
        if (currentMatch) payload.selectedLevelId = currentMatch.id
        else if (!customer.isMember) payload.selectedLevelId = active[0].id
      }
      self.setData(payload)
    }).catch(function(e) {
      if (seq !== self._levelsSeq) return
      self.setData({ levels: [], activeLevels: [], levelsLoaded: true, levelsError: (e && e.message) || '会员等级加载失败' })
    })
  },
  retryLevels() {
    this.setData({ levels: [], activeLevels: [], levelsLoaded: false, levelsError: '', selectedLevelId: '' })
    this.loadLevels()
  },
  reload() {
    var self = this
    var sid = this.data.sid
    if (!sid) return
    var seq = ++this._seq
    this.setData({
      loading: true, loadingMore: false, error: '',
      customers: [], total: 0, page: 1, totalPages: 0, hasMore: false
    })
    api.getCustomers({ keyword: (this.data.keyword || '').trim(), page: 1, page_size: this.data.pageSize }, sid).then(function(r) {
      if (seq !== self._seq) return
      var d = r || {}
      var list = Array.isArray(d) ? d : (d.list || [])
      var page = Number(d.page || 1)
      var totalPages = Number(d.total_pages || 0)
      self.setData({
        customers: list.map(decorateCustomer),
        total: Number(d.total || list.length),
        page: page,
        totalPages: totalPages,
        hasMore: page < totalPages,
        loading: false,
        error: ''
      })
    }).catch(function(e) {
      if (seq !== self._seq) return
      self.setData({ loading: false, error: (e && e.message) || '客户列表加载失败' })
    })
  },
  loadMore() {
    if (!this.data.sid || this.data.loading || this.data.loadingMore || !this.data.hasMore) return
    var self = this
    var seq = this._seq
    var nextPage = this.data.page + 1
    this.setData({ loadingMore: true })
    api.getCustomers({ keyword: (this.data.keyword || '').trim(), page: nextPage, page_size: this.data.pageSize }, this.data.sid).then(function(r) {
      if (seq !== self._seq) return
      var d = r || {}
      var list = Array.isArray(d) ? d : (d.list || [])
      var page = Number(d.page || nextPage)
      var totalPages = Number(d.total_pages || 0)
      self.setData({
        customers: self.data.customers.concat(list.map(decorateCustomer)),
        total: Number(d.total || self.data.total),
        page: page,
        totalPages: totalPages,
        hasMore: page < totalPages,
        loadingMore: false
      })
    }).catch(function(e) {
      if (seq !== self._seq) return
      self.setData({ loadingMore: false })
      wx.showToast({ title: (e && e.message) || '加载更多失败', icon: 'none' })
    })
  },
  onKeywordInput(e) {
    var value = e.detail.value || ''
    var self = this
    this.setData({ keywordInput: value })
    if (this._searchTimer) clearTimeout(this._searchTimer)
    this._searchTimer = setTimeout(function() {
      if (self.data.keyword === value) return
      self.setData({ keyword: value })
      self.reload()
    }, 400)
  },
  onSearch() {
    if (this._searchTimer) {
      clearTimeout(this._searchTimer)
      this._searchTimer = null
    }
    this.setData({ keyword: this.data.keywordInput || '' })
    this.reload()
  },
  openMembership(e) {
    var id = e.currentTarget.dataset.id
    var customer = this.data.customers.find(function(c) { return String(c.id) === String(id) })
    if (!customer) return
    if (!String(customer.phone || '').trim()) {
      wx.showToast({ title: '请先让用户授权手机号', icon: 'none' })
      return
    }
    var active = this.data.activeLevels
    var selected = ''
    if (customer.isMember) {
      var matched = active.find(function(level) { return String(level.id) === String(customer.levelId) })
      if (matched) selected = matched.id
    }
    if (!selected && active.length > 0) selected = active[0].id
    this.setData({
      showSheet: true,
      selectedCustomer: customer,
      selectedLevelId: selected,
      expiresDate: customer.isMember && customer.expires_at > Date.now() ? fmtDate(customer.expires_at) : '',
      remark: customer.remark || '',
      saving: false
    })
  },
  pickLevel(e) {
    this.setData({ selectedLevelId: e.currentTarget.dataset.id })
  },
  onExpiresChange(e) {
    this.setData({ expiresDate: e.detail.value || '' })
  },
  clearExpires() {
    this.setData({ expiresDate: '' })
  },
  onRemarkInput(e) {
    this.setData({ remark: e.detail.value || '' })
  },
  closeSheet() {
    if (this.data.saving) return
    this.setData({ showSheet: false })
  },
  noop() {},
  saveMembership() {
    if (this.data.saving) return
    var customer = this.data.selectedCustomer
    if (!customer) return
    var levelId = parseInt(this.data.selectedLevelId, 10)
    if (!levelId) {
      wx.showToast({ title: '请选择会员等级', icon: 'none' })
      return
    }
    var expiresAt = null
    if (this.data.expiresDate) {
      expiresAt = endOfDay(this.data.expiresDate)
      if (!expiresAt || expiresAt <= Date.now()) {
        wx.showToast({ title: '到期日不能早于今天', icon: 'none' })
        return
      }
    }
    var self = this
    this.setData({ saving: true })
    api.setCustomerMembership(customer.id, {
      level_id: levelId,
      expires_at: expiresAt,
      remark: (this.data.remark || '').trim()
    }, this.data.sid).then(function() {
      wx.showToast({ title: '会员设置成功' })
      self.setData({ saving: false, showSheet: false })
      self.reload()
    }).catch(function(e) {
      self.setData({ saving: false })
      wx.showToast({ title: (e && e.message) || '设置失败', icon: 'none' })
    })
  },
  removeMembership(e) {
    if (this.data.removingId) return
    var id = e.currentTarget.dataset.id
    var customer = this.data.customers.find(function(c) { return String(c.id) === String(id) })
    var self = this
    this.setData({ removingId: id })
    wx.showModal({
      title: '取消会员',
      content: '确认取消' + (customer ? '「' + customer.nickname + '」' : '该客户') + '的会员？',
      confirmColor: '#e94560',
      success: function(res) {
        if (!res.confirm) {
          self.setData({ removingId: '' })
          return
        }
        wx.showLoading({ title: '处理中...' })
        api.removeCustomerMembership(id, self.data.sid).then(function() {
          wx.hideLoading()
          self.setData({ removingId: '' })
          wx.showToast({ title: '已取消会员' })
          self.reload()
        }).catch(function(err) {
          wx.hideLoading()
          self.setData({ removingId: '' })
          wx.showToast({ title: (err && err.message) || '操作失败', icon: 'none' })
        })
      },
      fail: function() {
        self.setData({ removingId: '' })
      }
    })
  },
  goDashboard() {
    wx.switchTab({ url: '/pages/dashboard/dashboard' })
  },
  goBack() {
    wx.navigateBack()
  }
})
