function parseDeviceConfig(raw) {
  if (!raw) return {}
  try { return JSON.parse(raw) || {} } catch (e) { return {} }
}

function buildDeviceButtons(dev) {
  var name = dev.name || '设备'
  var config = parseDeviceConfig(dev.config_json)
  if (config.cmd_on) {
    if (config.is_toggle === false || !config.cmd_off) return [{ label: name, cmd: config.cmd_on }]
    return [{ label: name + '开', cmd: config.cmd_on }, { label: name + '关', cmd: config.cmd_off }]
  }
  var buttons = {
    door_lock: [{ label: '开门', cmd: 'door_open' }, { label: '关门', cmd: 'door_close' }],
    light: [{ label: name + '开', cmd: 'light_on' }, { label: name + '关', cmd: 'light_off' }],
    ac: [{ label: name + '开', cmd: 'ac_on' }, { label: name + '关', cmd: 'ac_off' }],
    exhaust: [{ label: name + '开', cmd: 'exhaust_on' }, { label: name + '关', cmd: 'exhaust_off' }],
    machine: [{ label: name + '开', cmd: 'machine_on' }, { label: name + '关', cmd: 'machine_off' }]
  }
  if (dev.type === 'combo') {
    return [
      { label: '开门', cmd: 'door_open' }, { label: '关门', cmd: 'door_close' },
      { label: '灯开', cmd: 'light_on' }, { label: '灯关', cmd: 'light_off' },
      { label: '空调开', cmd: 'ac_on' }, { label: '空调关', cmd: 'ac_off' },
      { label: '排风开', cmd: 'exhaust_on' }, { label: '排风关', cmd: 'exhaust_off' }
    ]
  }
  return buttons[dev.type] || []
}

module.exports = { buildDeviceButtons: buildDeviceButtons, parseDeviceConfig: parseDeviceConfig }
