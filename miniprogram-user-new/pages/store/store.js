var api = require('../../utils/api')
function pad(n) { return n < 10 ? '0' + n : '' + n }
function slotStyle(past, now, active, range) {
  if (active) return { bg: '#d4af37', cs: '#1a1a2e', fw: 'bold', bd: 'none' }
  if (range) return { bg: 'rgba(212,175,55,0.15)', cs: '#d4af37', fw: 'normal', bd: 'none' }
  if (now) return { bg: '#26263a', cs: '#8a8a9a', fw: 'normal', bd: 'none' }
  if (past) return { bg: '#1e1e2c', cs: '#4a4a5a', fw: 'normal', bd: 'none' }
  return { bg: '#26263a', cs: '#aaa', fw: 'normal', bd: 'none' }
}

Page({
  data: { store: {}, types: [], slots: [], startIdx: -1, endIdx: -1, dayOffset: 0, rangeText: '', showConfirm: false, confirmRoom: '', confirmTime: '', confirmPrice: 0, confirmPriceFen: 0, originalPrice: 0, confirmHasDiscount: false, confirmHours: 0, confirmMin: 0, confirmRid: '', prorateNote: '', priceMap: {}, loadingAvailability: false, availabilityError: '', balance: 0, balanceYuan: '0.00', submitting: false, isMember: false, memberLevelName: '', memberDiscountBp: 0, memberDiscountLabel: '', memberSavings: '', typePriceMap: {} },
  onLoad: function (opts) {
    this.storeId = opts.id
    this._availabilitySeq = 0
    var self = this
    api.getBalance(this.storeId).then(function (r) { var d = r.data || r; self.setData({ balance: Number(d.balance) || 0, balanceYuan: ((Number(d.balance) || 0) / 100).toFixed(2) }) }).catch(function () {})
    this.setData({ storeId: opts.id, dayOffset: 0 })
    api.getStore(opts.id).then(function (r) { var s = r.data || r; if (s && s.cover) s.cover = String(s.cover).replace('http://127.0.0.1:8080', 'http://localhost:8080'); self.setData({ store: s }) }).catch(function () { wx.showToast({ title: '门店信息加载失败', icon: 'none' }) })
    api.getStorePrice(opts.id).then(function (r) {
      var d = r.data || r
      if (d && d.prices) self.setData({ priceMap: d.prices }, function () { self.decorateTypes() })
      else self.decorateTypes()
    }).catch(function () {})
    if (getApp().globalData.token) {
      api.getMyMembership(opts.id).then(function (r) {
        var m = r.data || r
        var bp = Number(m && m.discount_bp) || 0
        var label = ''
        if (m && m.is_member && bp > 0) label = (m.level_name || '会员') + ' ' + (bp / 1000) + ' 折'
        self.setData({
          isMember: !!(m && m.is_member),
          memberLevelName: (m && m.level_name) || '',
          memberDiscountBp: bp,
          memberDiscountLabel: label
        }, function () { self.decorateTypes(); self.loadTypePrices() })
      }).catch(function () {})
    }
    this.buildSlots()
  },
  decorateTypes: function () {
    var self = this
    var priceMap = this.data.priceMap || {}
    var typePriceMap = this.data.typePriceMap || {}
    var isMember = !!this.data.isMember
    var bp = Number(this.data.memberDiscountBp) || 0
    var useMemberPrice = isMember && bp > 0 && bp < 10000
    var types = (this.data.types || []).map(function (type) {
      var serverPrice = typePriceMap[type.id]
      var original, finalPrice, label = ''
      if (serverPrice && Number(serverPrice.original_price) > 0) {
        original = Number(serverPrice.original_price)
        finalPrice = Number(serverPrice.member_price)
        if (!(finalPrice > 0)) finalPrice = original
        var serverBp = Number(serverPrice.discount_bp) || bp
        var serverLevel = serverPrice.level_name || self.data.memberLevelName
        if (serverBp > 0 && serverBp < 10000) label = (serverLevel || '会员') + ' ' + (serverBp / 1000) + ' 折'
      } else {
        var priceInfo = priceMap[String(type.id)] || priceMap['0'] || {}
        original = Number(priceInfo.unit_price)
        if (!(original > 0)) original = 5000
        finalPrice = original
        if (useMemberPrice) {
          finalPrice = Math.floor(original * bp / 10000)
          if (finalPrice < 1) finalPrice = 1
          label = (self.data.memberLevelName || '会员') + ' ' + (bp / 1000) + ' 折'
        }
      }
      return Object.assign({}, type, {
        originalPriceText: (original / 100).toFixed(2),
        priceText: (finalPrice / 100).toFixed(2),
        memberPrice: finalPrice < original,
        discountLabel: label || self.data.memberDiscountLabel
      })
    })
    this.setData({ types: types })
  },
  loadTypePrices: function () {
    var self = this
    var types = this.data.types || []
    if (!getApp().globalData.token || !this.storeId || !types.length) return
    var seq = this._availabilitySeq
    var requests = types.map(function (type) {
      return api.getMyPrice(self.storeId, type.id).then(function (r) {
        return { id: type.id, price: r.data || r }
      }).catch(function () { return null })
    })
    Promise.all(requests).then(function (results) {
      if (seq !== self._availabilitySeq) return
      var map = {}
      results.forEach(function (item) {
        if (item && item.price) map[item.id] = item.price
      })
      self.setData({ typePriceMap: map }, function () { self.decorateTypes() })
    })
  },
  buildSlots: function () {
    var h = new Date().getHours(), s = [], off = this.data.dayOffset || 0
    for (var i = 0; i < 24; i++) { var past = off === 0 && i < h, now = off === 0 && i === h; s.push(Object.assign({ label: pad(i) + ':00', past: past, now: now }, slotStyle(past, now, false, false))) }
    if (off === 0) s.push({ label: '明', isNext: true, cs: '#d4af37', bg: '#26263a', fw: 'bold', bd: '1px solid #d4af37' })
    else s.push({ label: '今', isPrev: true, cs: '#d4af37', bg: '#26263a', fw: 'bold', bd: '1px solid #d4af37' })
    this._availabilitySeq++
    this.setData({ slots: s, startIdx: -1, endIdx: -1, rangeText: '', types: [], loadingAvailability: false, availabilityError: '' })
  },
  nextDay: function () { if ((this.data.dayOffset || 0) < 1) { this.setData({ dayOffset: 1 }); this.buildSlots() } },
  prevDay: function () { if ((this.data.dayOffset || 0) > 0) { this.setData({ dayOffset: 0 }); this.buildSlots() } },
  tapSlot: function (e) {
    var idx = e.currentTarget.dataset.idx, slots = this.data.slots, startIdx = this.data.startIdx, endIdx = this.data.endIdx
    if (slots[idx].isNext) return this.nextDay()
    if (slots[idx].isPrev) return this.prevDay()
    if (startIdx >= 0 && (idx === startIdx || idx === endIdx)) return this.buildSlots()
    if (startIdx < 0) {
      if (slots[idx].past) return
      var first = slots.map(function (sl, i) { return Object.assign({}, sl, slotStyle(sl.past, sl.now, i === idx, false)) })
      this.setData({ slots: first, startIdx: idx, rangeText: this.data.dayOffset ? '明天 ' + slots[idx].label + ' ～ 请选择结束时间' : slots[idx].label + ' ～ 请选择结束时间' })
      return
    }
    var wrap = idx < startIdx
    var updated = slots.map(function (sl, i) { return Object.assign({}, sl, slotStyle(sl.past, sl.now, i === startIdx || i === idx, wrap ? (i > startIdx || i < idx) : (i > startIdx && i < idx))) })
    this.setData({ slots: updated, endIdx: idx })
    var duration = wrap ? 24 - startIdx + idx : idx - startIdx
    this.setData({ rangeText: (this.data.dayOffset ? '明天 ' : '') + slots[startIdx].label + ' ～ ' + (wrap ? '次日 ' : '') + slots[idx].label + ' · ' + duration + '小时' })
    this.loadAvailable(startIdx, idx)
  },
  loadAvailable: function (startIdx, endIdx) {
    var seq = ++this._availabilitySeq, base = new Date(new Date().getFullYear(), new Date().getMonth(), new Date().getDate()).getTime(), off = this.data.dayOffset || 0
    var sh = parseInt(this.data.slots[startIdx].label, 10), eh = parseInt(this.data.slots[endIdx].label, 10), wrap = endIdx < startIdx
    var start = base + off * 86400000 + sh * 3600000, end = base + (off + (wrap ? 1 : 0)) * 86400000 + eh * 3600000
    var self = this
    this.setData({ loadingAvailability: true, availabilityError: '', types: [] })
    api.getSlotResources(this.storeId, start, end).then(function (r) {
      if (seq !== self._availabilitySeq) return
      var d = r.data || r, groups = Array.isArray(d) ? d : (d.groups || d.list || []), map = {}
      groups.forEach(function (g) {
        var type = g.type || {}, id = g.resource_type_id || type.id || g.type_id, items = g.items || g.resources || []
        if (!id) return
        if (!map[id]) map[id] = { id: id, name: g.type_name || type.name || '房间', resources: [] }
        items.forEach(function (item) { map[id].resources.push({ id: item.resource_id || item.id, name: item.name, resource_type_id: id }) })
      })
      self.setData({ types: Object.keys(map).map(function (id) { return map[id] }).filter(function (x) { return x.resources.length }), loadingAvailability: false }, function () { self.decorateTypes(); self.loadTypePrices() })
    }).catch(function () { if (seq === self._availabilitySeq) self.setData({ loadingAvailability: false, availabilityError: '房间信息加载失败，请重试' }) })
  },
  retryAvailability: function () { if (this.data.startIdx >= 0 && this.data.endIdx >= 0) this.loadAvailable(this.data.startIdx, this.data.endIdx) },
  goAvailability: function () { wx.navigateTo({ url: '/pages/availability/availability?id=' + this.storeId }) },
  book: function (e) {
    var rid = e.currentTarget.dataset.rid, a = this.data.startIdx, b = this.data.endIdx
    if (a < 0 || b < 0) { wx.showToast({ title: '请选择开始和结束时间', icon: 'none' }); return }
    var hours = b > a ? b - a : 24 - a + b, typeId = '', name = '', self = this
    this.data.types.forEach(function (t) { t.resources.forEach(function (r) { if (String(r.id) === String(rid)) { typeId = r.resource_type_id; name = r.name } }) })
    var unit = Number(this.data.priceMap[String(typeId)] && this.data.priceMap[String(typeId)].unit_price) || 5000
    var full = hours * unit, mins = 0
    if (this.data.dayOffset === 0 && this.data.slots[a].now) {
      mins = new Date().getMinutes()
      // 只对「当前这一小时」按剩余分钟计费，后续整点按整小时计费
      full = Math.max(1, Math.round(unit * (60 - mins) / 60)) + (hours - 1) * unit
    }
    var price = Math.round(full)
    var isMember = this.data.isMember, bp = Number(this.data.memberDiscountBp) || 0, finalPrice = price
    var originalPrice = ((hours * unit) / 100).toFixed(2), memberSavings = ''
    if (isMember && bp > 0 && bp < 10000) {
      finalPrice = Math.floor(price * bp / 10000)
      if (finalPrice < 1) finalPrice = 1
      originalPrice = (price / 100).toFixed(2)
      if (finalPrice < price) memberSavings = ((price - finalPrice) / 100).toFixed(2)
    }
    var crossesMidnight = b < a
    var confirmEnd = (crossesMidnight ? '次日 ' : '') + this.data.slots[b].label
    var confirmHasDiscount = Number(originalPrice) > finalPrice / 100
    this.setData({ showConfirm: true, confirmRid: rid, confirmRoom: name, confirmTime: this.data.slots[a].label + ' ～ ' + confirmEnd, confirmPrice: (finalPrice / 100).toFixed(2), confirmPriceFen: finalPrice, originalPrice: originalPrice, confirmHasDiscount: confirmHasDiscount, confirmHours: Math.round((hours - mins / 60) * 10) / 10, confirmMin: hours * 60 - mins, memberSavings: memberSavings, prorateNote: mins ? '已过' + mins + '分钟，按剩余' + (hours * 60 - mins) + '分钟计费' : '' })
  },
  cancelBook: function () { this.setData({ showConfirm: false, submitting: false }) }, noop: function () {},
  doBook: function () {
    if (this.data.submitting) return
    var self = this, a = this.data.startIdx, b = this.data.endIdx, slots = this.data.slots, wrap = b < a, price = Number(this.data.confirmPriceFen) || 0
    if (this.data.balance < price) { wx.showToast({ title: '余额不足，请先充值', icon: 'none' }); return }
    var base = new Date(new Date().getFullYear(), new Date().getMonth(), new Date().getDate()).getTime(), off = this.data.dayOffset || 0
    var start = base + off * 86400000 + parseInt(slots[a].label, 10) * 3600000, end = base + (off + (wrap ? 1 : 0)) * 86400000 + parseInt(slots[b].label, 10) * 3600000
    if (off === 0 && slots[a].now) start = Math.max(start, Date.now())
    this.setData({ submitting: true })
    api.createOrder({ resource_id: parseInt(this.data.confirmRid, 10), store_id: parseInt(this.storeId, 10), start_time: start, end_time: end, pay_method: 'balance' }).then(function () { self.setData({ showConfirm: false, submitting: false }); wx.showToast({ title: '预约成功', icon: 'success' }); setTimeout(function () { wx.switchTab({ url: '/pages/order/order' }) }, 900) }).catch(function (e) {
      self.setData({ submitting: false })
      if (e && e.code === 20005) {
        wx.showModal({
          title: '需要绑定手机号',
          content: '请先授权绑定手机号',
          confirmText: '去绑定',
          cancelText: '稍后',
          success: function (r) { if (r.confirm) wx.switchTab({ url: '/pages/mine/mine' }) }
        })
        return
      }
      wx.showToast({ title: e.message || '预约失败，请稍后重试', icon: 'none' })
    })
  },
  onShareAppMessage: function () {
    var name = (this.data.store && this.data.store.name) || '门店'
    return { title: name + ' · 金之莱无人空间', path: '/pages/store/store?id=' + this.storeId }
  }
})