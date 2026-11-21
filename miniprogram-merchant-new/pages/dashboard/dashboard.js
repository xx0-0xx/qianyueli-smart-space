const api = require('../../utils/api')
const deviceCommands = require('../../utils/device_commands')
function formatRemain(endTime) {
  var left = endTime - Date.now()
  if (left <= 0) return '已结束'
  var h = Math.floor(left / 3600000)
  var m = Math.floor((left % 3600000) / 60000)
  return h > 0 ? h + '时' + m + '分' : m + '分钟'
}
function filterRooms(rooms, filter, keyword) {
  var q = (keyword || '').trim().toLowerCase()
  return rooms.filter(function(room) {
    var matchStatus = filter === 'all' || (filter === 'active' ? room.active : !room.active)
    return matchStatus && (!q || room.name.toLowerCase().indexOf(q) >= 0 || (room.typeName || '').toLowerCase().indexOf(q) >= 0)
  })
}
Page({
  data: { revenue: '0.00', orderCount: 0, activeRooms: 0, totalRooms: 0, pendingOrders: 0, pendingRefunds: 0, rooms: [], filteredRooms: [], roomFilter: 'all', roomKeyword: '', expandedId: '', rechargeCount: 0, stores: [], currentStoreName: '', currentStoreId: '', showStoreSwitch: false, bindCode: '', showBind: false, roomTypes: [] },
  _loadSeq: 0,
  onShow() { this.loadStores(); this.load() },
  noop() {},
  loadStores() {
    var self = this
    var sid = getApp().globalData.storeId || wx.getStorageSync('merchantSid')
    api.getMerchantStores().then(function(list) {
      var arr = Array.isArray(list) ? list : (list && list.list) || []
      var cur = arr.find(function(s) { return String(s.id) === String(sid) })
      self.setData({ stores: arr, currentStoreName: cur ? cur.name : '选择门店', currentStoreId: cur ? cur.id : '' })
    }).catch(function() {})
  },
  showStoreSwitch() { this.setData({ showStoreSwitch: true }) },
  hideStoreSwitch() { this.setData({ showStoreSwitch: false, showBind: false, bindCode: '' }) },
  pickStore(e) {
    var sid = String(e.currentTarget.dataset.id)
    var store = this.data.stores.find(function(s) { return String(s.id) === sid })
    getApp().globalData.storeId = sid
    wx.setStorageSync('merchantSid', sid)
    this._loadSeq += 1
    this.setData({ showStoreSwitch: false, currentStoreName: store ? store.name : '', currentStoreId: sid, rooms: [], filteredRooms: [], revenue: '0.00', orderCount: 0, activeRooms: 0, totalRooms: 0, pendingOrders: 0, pendingRefunds: 0, rechargeCount: 0, expandedId: '', roomFilter: 'all', roomKeyword: '' })
    this.load()
  },
  showBind() { this.setData({ showBind: true }) },
  onBindCode(e) { this.setData({ bindCode: e.detail.value }) },
  doBind() {
    var self = this
    var code = (this.data.bindCode || '').trim()
    if (!code) { wx.showToast({ title: '请输入门店邀请码', icon: 'none' }); return }
    wx.showLoading({ title: '绑定中...' })
    api.bindMerchantStore(code).then(function() {
      wx.hideLoading()
      wx.showToast({ title: '已添加门店' })
      self.setData({ showBind: false, bindCode: '' })
      self.loadStores()
    }).catch(function(e) {
      wx.hideLoading()
      wx.showToast({ title: (e && e.message) || '绑定失败', icon: 'none' })
    })
  },
  async load() {
    var sid = String(getApp().globalData.storeId || wx.getStorageSync('merchantSid') || '')
    if (!sid) return
    var seq = ++this._loadSeq
    try {
      var results = await Promise.all([
        api.getResources(sid), api.getAllMerchantOrders(sid), api.getMerchantDevices(sid),
        api.getMerchantRecharges('0', sid), api.getRefundRequests(sid)
      ])
      if (seq !== this._loadSeq || sid !== String(getApp().globalData.storeId || wx.getStorageSync('merchantSid') || '')) return
      var resData = Array.isArray(results[0]) ? results[0] : (results[0].list || [])
      var orders = results[1] || []
      var devices = Array.isArray(results[2]) ? results[2] : (results[2].list || [])
      var recharges = Array.isArray(results[3]) ? results[3] : ((results[3] && results[3].list) || [])
      var refunds = Array.isArray(results[4]) ? results[4] : ((results[4] && results[4].list) || [])
      var now = Date.now()
      var dayStart = new Date(new Date().getFullYear(), new Date().getMonth(), new Date().getDate()).getTime()
      var dayEnd = dayStart + 86400000
      var todayOrders = orders.filter(function(o) { return o.start_time < dayEnd && o.end_time > dayStart })
      var revenue = todayOrders.reduce(function(sum, o) { return [1, 2, 3].indexOf(Number(o.status)) >= 0 ? sum + Number(o.total_amount || 0) : sum }, 0)
      var rooms = resData.map(function(r) {
        var activeOrder = orders.find(function(o) { return String(o.resource_id) === String(r.id) && [1, 2].indexOf(Number(o.status)) >= 0 && o.start_time <= now && o.end_time > now })
        var roomDevices = []
        try {
          var ids = JSON.parse(r.device_ids || '[]').map(function(x) { return Number(x) })
          roomDevices = devices.filter(function(d) { return ids.indexOf(Number(d.id)) >= 0 }).map(function(d) {
            return Object.assign({}, d, { controlButtons: deviceCommands.buildDeviceButtons(d) })
          })
        } catch(e) {}
        return { id:r.id, name:r.name, typeName:r.resource_type ? r.resource_type.name : '', active:!!activeOrder, orderId:activeOrder ? activeOrder.id : '', remainText:activeOrder ? formatRemain(activeOrder.end_time) : '', devices:roomDevices }
      })
      this.setData({
        revenue:(revenue / 100).toFixed(2), orderCount:todayOrders.length,
        activeRooms:rooms.filter(function(r) { return r.active }).length, totalRooms:rooms.length,
        pendingOrders:orders.filter(function(o) { return Number(o.status) === 0 }).length,
        pendingRefunds:refunds.filter(function(o) { return Number(o.status) === 0 }).length,
        rechargeCount:recharges.length, rooms:rooms,
        filteredRooms:filterRooms(rooms,this.data.roomFilter,this.data.roomKeyword),
        roomTypes:['全部'].concat(Array.from(new Set(rooms.map(function(r) { return r.typeName }).filter(Boolean))))
      })
    } catch(e) {
      if (seq === this._loadSeq) wx.showToast({ title:'看板加载失败', icon:'none' })
    }
  },
  filterRoomStatus(e) {
    var f = e.currentTarget.dataset.filter
    this.setData({ roomFilter:f, filteredRooms:filterRooms(this.data.rooms,f,this.data.roomKeyword) })
  },
  onRoomSearch(e) {
    var q = e.detail.value || ''
    this.setData({ roomKeyword:q, filteredRooms:filterRooms(this.data.rooms,this.data.roomFilter,q) })
  },
  goOrders() { getApp().globalData.ordersTab = 'orders'; wx.switchTab({ url:'/pages/orders/orders' }) },
  goRefunds() { getApp().globalData.ordersTab = 'refunds'; wx.switchTab({ url:'/pages/orders/orders' }) },
  toggleRoom(e) {
    var id = e.currentTarget.dataset.id
    this.setData({ expandedId:this.data.expandedId === id ? '' : id })
  },
  sendCmd(e) {
    var id = e.currentTarget.dataset.id
    var cmd = e.currentTarget.dataset.cmd
    api.controlDevice(id, cmd).then(function() {
      wx.showToast({ title:'指令已发送' })
    }).catch(function(err) {
      wx.showToast({ title:(err && err.message) || '设备控制失败', icon:'none' })
    })
  },
  goTypes() { wx.switchTab({ url:'/pages/resource-types/resource-types' }) },
  goResources() { wx.navigateTo({ url:'/pages/resources-mgt/resources-mgt' }) },
  goDevices() { wx.navigateTo({ url:'/pages/devices/devices' }) },
  goReport() { wx.navigateTo({ url:'/pages/report/report' }) },
  goTimeline() { wx.navigateTo({ url:'/pages/timeline/timeline' }) },
  goRecharges() { wx.navigateTo({ url:'/pages/recharges/recharges' }) },
  logout() {
    wx.removeStorageSync('token')
    getApp().globalData.token = ''
    wx.reLaunch({ url:'/pages/login/login' })
  }
})
