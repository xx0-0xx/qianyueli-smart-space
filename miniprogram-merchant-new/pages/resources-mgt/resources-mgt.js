const api = require('../../utils/api')
Page({
  data: { types: [] },
  goBack() { wx.navigateBack() },
  onShow() { this.load() },
  async load() {
    try {
      var types = await api.getResourceTypes()
      var typeList = types || []
      var r = await api.getResources()
      var data = r || []
      var typeMap = {}
      typeList.forEach(function(t) {
        typeMap[t.id] = { id: t.id, name: t.name, rooms: [], _input: '' }
      })
      data.forEach(function(room) {
        var tid = room.resource_type_id || (room.resource_type && room.resource_type.id)
        if (typeMap[tid]) typeMap[tid].rooms.push(room)
      })
      var types2 = Object.keys(typeMap).map(function(k) { return typeMap[k] })
      this.setData({ types: types2 })
    } catch(e) {}
  },
  onRoomInput(e) {
    var tid = e.currentTarget.dataset.tid
    var types = this.data.types
    types.forEach(function(t) { if (String(t.id)===String(tid)) t._input = e.detail.value })
    this.setData({ types: types })
  },
  async addRoom(e) {
    var tid = e.currentTarget.dataset.tid
    var t = this.data.types.find(function(t) { return String(t.id) === String(tid) })
    var n = t ? t._input : ''
    if (!n) return
    try {
      await api.createResource({ name: n, resource_type_id: parseInt(tid) })
      if (t) t._input = ''
      this.load()
    } catch(e) { wx.showToast({ title: "失败", icon: "none" }) }
  },
  delRoom(e) {
    var self = this
    var id = e.currentTarget.dataset.id
    wx.showModal({
      title: "确认删除",
      content: "删除后不可恢复",
      success: function(res) {
        if (!res.confirm) return
        api.deleteResource(id).then(function() { self.load() })
          .catch(function() { wx.showToast({ title: "失败", icon: "none" }) })
      }
    })
  },
  bindDevice(e) {
    var rid = e.currentTarget.dataset.rid
    var self = this
    wx.showActionSheet({
      itemList: ["扫码绑定", "手动输入SN"],
      success: function(res) {
        if (res.tapIndex === 0) self.scanBind(rid)
        else self.inputBind(rid)
      }
    })
  },
  scanBind(rid) {
    var self = this
    wx.scanCode({ success: function(res) { self.doBind(rid, res.result.trim()) } })
  },
  inputBind(rid) {
    var self = this
    wx.showModal({
      title: "输入设备SN",
      editable: true,
      placeholderText: "ESP32序列号",
      success: function(res) {
        if (res.confirm && res.content) self.doBind(rid, res.content.trim())
      }
    })
  },
  doBind(rid, sn) {
    var self = this
    wx.showActionSheet({
      itemList: ["门锁", "灯光", "插座/机器", "空调", "排风", "其他"],
      success: function(res) {
        var types = ['door_lock', 'light', 'machine', 'ac', 'exhaust', 'other']
        var names = ["门锁", "灯光", "插座", "空调", "排风", "设备"]
        self.showNameInput(rid, sn, types[res.tapIndex], names[res.tapIndex])
      }
    })
  },
  showNameInput(rid, sn, type, defaultName) {
    var self = this
    wx.showModal({
      title: "设备名称",
      editable: true, placeholderText: defaultName,
      content: '',
      success: function(res) {
        if (res.confirm) {
          var name = res.content || defaultName
          self.createAndLink(rid, sn, name, type)
        }
      }
    })
  },
  async createAndLink(rid, sn, name, type) {
    wx.showLoading({ title: "绑定中..." })
    try {
      await api.createDevice({
        device_sn: sn,
        name: name,
        type: type,
        store_id: parseInt(getApp().globalData.storeId) || 100,
        room_id: parseInt(rid) || 0
      })
      wx.hideLoading()
      wx.showToast({ title: "绑定成功" })
      this.load()
    } catch(e) {
      wx.hideLoading()
      wx.showToast({ title: "绑定失败", icon: "none" })
    }
  }
})