const api = require('../../utils/api')
Page({
  data: { hours: [], rows: [] },
  goBack() { wx.navigateBack() },
  onLoad(opts) {
    var storeId = opts.id
    console.log('availability storeId:', storeId)
    var now = new Date()
    var today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
    var currentHour = now.getHours()

    // 当前小时往后24小时
    var hours = []
    for (var i = 0; i < 24; i++) {
      var h = (currentHour + i) % 24
      hours.push((h < 10 ? '0' : '') + h)
    }

    Promise.all([api.getResources(storeId), api.getStoreOccupied(storeId)]).then(function(results) {
      console.log('availability raw:', JSON.stringify(results).substring(0, 500))
      var resData = results[0].data || []
      var orders = results[1].data || []
      if (!Array.isArray(orders)) orders = []

      var rooms = []
      resData.forEach(function(g) {
        (g.resources || []).forEach(function(r) {
          if (r.status !== 3) rooms.push(r)
        })
      })

      var rows = rooms.map(function(room) {
        var cells = []
        for (var i = 0; i < 24; i++) {
          var h = currentHour + i
          var dayOff = Math.floor(h / 24)
          var hour = h % 24
          var st = today + dayOff * 86400000 + hour * 3600000
          var et = st + 3600000
          var conflict = orders.some(function(o) {
            if (String(o.resource_id) !== String(room.id)) return false
            if ([0, 1, 2].indexOf(o.status) < 0) return false
            return o.start_time < et && o.end_time > st
          })
          cells.push(conflict ? 1 : 0)
        }
        return { id: room.id, name: room.name, cells: cells }
      })

      console.log('availability: rooms=', rooms.length, 'orders=', orders.length)
      this.setData({ hours: hours, rows: rows })
    }.bind(this)).catch(function(e) {
      console.log('availability error:', e)
      wx.showToast({ title: '加载失败', icon: 'none' })
    })
  }
})
