const api = require('../../utils/api')

const voiceList = [
  { code: '00001', text: '开机成功' },
  { code: '00002', text: '您好！欢迎光临' },
  { code: '00003', text: '本次时间已结束' },
  { code: '00004', text: '谢谢光临！请带好随身物品' },
  { code: '00005', text: '欢迎光临！已成功开机' },
  { code: '00006', text: '重要提示！本场所禁止黄赌毒等违法行为' },
  { code: '00007', text: '续时成功！' },
  { code: '00008', text: '恭喜你续时成功！' },
  { code: '00009', text: '距结束还有30分钟，请及时续费' },
  { code: '00010', text: '距结束还有10分钟，请及时续费' },
  { code: '00011', text: '距结束仅剩5分钟，请及时续费' },
  { code: '00012', text: '本次时间已结束，欢迎下次光临' },
  { code: '00013', text: '包间距结束还有10分钟' },
  { code: '00014', text: '包间距结束还有5分钟' },
  { code: '00015', text: '包间时间已结束，系统将自动断电' },
  { code: '00016', text: '品茶时间还有10分钟结束' },
  { code: '00017', text: '包间时间还有10分钟即将结束' },
  { code: '00018', text: '包间还有5分钟即将结束' },
  { code: '00019', text: '预定时间马上结束' },
  { code: '00020', text: '欢迎光临，你的包间已通电' },
  { code: '00021', text: '包间还有10分钟即将结束' },
  { code: '00022', text: '免费使用时间即将结束' },
  { code: '00023', text: '免费使用时间即将结束，请到前台办理' },
  { code: '00024', text: '包间时间已经结束' },
  { code: '00025', text: '上班时间到了' },
  { code: '00026', text: '下班时间到了' },
  { code: '00027', text: '中午到了' },
  { code: '00028', text: '大家好！中午到了' },
  { code: '00029', text: '大家好！上班时间到了' }
]

Page({
  data: {
    devices: [],
    loading: true,
    saving: false,
    showList: false,
    voiceList: voiceList
  },

  goBack() {
    wx.navigateBack()
  },

  onShow() {
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const [voiceRes, deviceRes] = await Promise.all([
        api.getVoiceConfig(),
        api.getMerchantDevices()
      ])
      const list = (voiceRes && voiceRes.devices) || []
      const deviceList = deviceRes || []
      const nameMap = {}
      // 优先使用语音配置返回的展示名（集成套装会返回产品模板名，而不是“门锁”）
      list.forEach(function (item) {
        if (item.device_sn && item.name) {
          nameMap[item.device_sn] = item.name
        }
      })
      deviceList.forEach(function (d) {
        if (d.device_sn && !nameMap[d.device_sn]) {
          nameMap[d.device_sn] = d.name || d.device_sn
        }
      })
      const devices = list.map(function (item) {
        return {
          device_sn: item.device_sn,
          name: item.name || nameMap[item.device_sn] || item.device_sn,
          enabled: !!item.enabled,
          on_start: item.on_start || '',
          on_end: item.on_end || '',
          reminders: (item.reminders || []).map(function (r) {
            return {
              minutes_before_end: Number(r.minutes_before_end) || 30,
              command: r.command || ''
            }
          })
        }
      })
      this.setData({ devices: devices, loading: false })
    } catch (e) {
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  toggleEnabled(e) {
    const index = e.currentTarget.dataset.index
    this.setData({ ['devices[' + index + '].enabled']: e.detail.value })
  },

  onStartInput(e) {
    const index = e.currentTarget.dataset.index
    this.setData({ ['devices[' + index + '].on_start']: e.detail.value })
  },

  onEndInput(e) {
    const index = e.currentTarget.dataset.index
    this.setData({ ['devices[' + index + '].on_end']: e.detail.value })
  },

  onReminderMinutes(e) {
    const { di, ri } = e.currentTarget.dataset
    this.setData({
      ['devices[' + di + '].reminders[' + ri + '].minutes_before_end']:
        parseInt(e.detail.value, 10) || 0
    })
  },

  onReminderCommand(e) {
    const { di, ri } = e.currentTarget.dataset
    this.setData({
      ['devices[' + di + '].reminders[' + ri + '].command']: e.detail.value
    })
  },

  addReminder(e) {
    const index = e.currentTarget.dataset.index
    const device = this.data.devices[index]
    if (!device || device.reminders.length >= 10) return
    const reminders = device.reminders.concat([{ minutes_before_end: 30, command: '' }])
    this.setData({ ['devices[' + index + '].reminders']: reminders })
  },

  removeReminder(e) {
    const { di, ri } = e.currentTarget.dataset
    const reminders = this.data.devices[di].reminders.slice()
    reminders.splice(ri, 1)
    this.setData({ ['devices[' + di + '].reminders']: reminders })
  },

  toggleList() {
    this.setData({ showList: !this.data.showList })
  },

  async save() {
    if (this.data.saving) return
    const devices = this.data.devices
    for (let i = 0; i < devices.length; i++) {
      const d = devices[i]
      if (d.enabled && (!d.on_start.trim() || !d.on_end.trim())) {
        wx.showToast({ title: '请填写' + d.name + '的开始和结束语音码', icon: 'none' })
        return
      }
      const minutes = (d.reminders || []).map(function (r) {
        return Number(r.minutes_before_end)
      })
      if (
        minutes.some(function (v) {
          return !Number.isInteger(v) || v <= 0
        }) ||
        new Set(minutes).size !== minutes.length
      ) {
        wx.showToast({ title: d.name + '的提醒分钟必须为正整数且不能重复', icon: 'none' })
        return
      }
    }

    this.setData({ saving: true })
    try {
      await api.saveVoiceConfig(devices)
      wx.showToast({ title: '已保存' })
      this.load()
    } catch (e) {
      wx.showToast({ title: '保存失败', icon: 'none' })
    } finally {
      this.setData({ saving: false })
    }
  }
})
