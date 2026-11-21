var api = require('../../utils/api')
var commandMap = require('../../utils/device_commands')

Page({
  data: { devices: [], loading: true, error: '', sendingId: '', expired: false },
  onLoad: function (opts) {
    this.roomId = opts.rid
    this.storeId = opts.sid || getApp().globalData.storeId
    this.endTime = Number(opts.end || 0)
    if (this.endTime && Date.now() >= this.endTime) {
      this.setData({ loading: false, expired: true })
      return
    }
    this.load()
  },
  onShow: function () {
    this.startExpiryTimer()
  },
  onHide: function () {
    this.clearExpiryTimer()
  },
  onUnload: function () {
    this.clearExpiryTimer()
  },
  startExpiryTimer: function () {
    var self = this
    this.clearExpiryTimer()
    if (!this.endTime) return
    this.checkExpired()
    this._expiryTimer = setInterval(function () { self.checkExpired() }, 5000)
  },
  clearExpiryTimer: function () {
    if (this._expiryTimer) {
      clearInterval(this._expiryTimer)
      this._expiryTimer = null
    }
  },
  checkExpired: function () {
    if (!this.endTime || this.data.expired) return
    if (Date.now() >= this.endTime) {
      this.clearExpiryTimer()
      this.setData({ expired: true, devices: [], loading: false, error: '' })
      wx.showToast({ title: '预约已结束，设备控制已关闭', icon: 'none' })
    }
  },
  load: function () {
    if (this.data.expired) return
    var self = this
    this.setData({ loading: true, error: '', devices: [] })
    api.getResources(this.storeId).then(function (r) {
      var data = r.data || []
      var flat = data.reduce(function (arr, g) { return arr.concat(g.resources || []) }, [])
      var room = flat.find(function (item) { return String(item.id) === String(self.roomId) })
      var ids
      try { ids = room ? JSON.parse(room.device_ids || '[]').filter(Boolean).map(Number) : [] } catch (e) { ids = [] }
      if (!ids.length) {
        self.setData({ loading: false, devices: [] })
        return
      }
      self._roomHasDevices = true
      return Promise.all(ids.map(function (id) {
        return api.getDeviceStatus(id).then(function (r) { return { id: id, result: r } }).catch(function () { return null })
      }))
        .then(function (pairs) {
          if (self.data.expired) return
          var devices = pairs.filter(Boolean).map(function (p) {
            var device = p.result.data || p.result
            device.id = device.device_id || p.id
            device.buttons = device.user_control_enabled ? commandMap.buildDeviceButtons(device) : []
            return device
          }).filter(function (device) { return device.user_control_enabled && device.buttons.length > 0 })
          self.setData({ loading: false, devices: devices, noUserControlledDevices: devices.length === 0 && self._roomHasDevices })
        })
    }).catch(function (e) {
      self.setData({ loading: false, error: (e && e.message) || '设备信息加载失败，请检查网络后重试' })
    })
  },
  retry: function () {
    if (this.data.expired) return
    this.load()
  },
  sendCmd: function (e) {
    if (this.data.expired) {
      wx.showToast({ title: '预约已结束，无法控制设备', icon: 'none' })
      return
    }
    if (this.data.sendingId) return
    var self = this, id = String(e.currentTarget.dataset.id), cmd = e.currentTarget.dataset.cmd
    this.setData({ sendingId: id })
    api.controlDevice(id, cmd).then(function () {
      wx.showToast({ title: '指令已发送' })
    }).catch(function (err) {
      wx.showToast({ title: (err && err.message) || '指令发送失败', icon: 'none' })
    }).finally(function () { self.setData({ sendingId: '' }) })
  }
})