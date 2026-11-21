function buildDeviceButtons(device) {
  var name = device.name || '设备'
  if (device.cmd_on) {
    if (device.is_toggle === false || !device.cmd_off) return [{ label: name, cmd: device.cmd_on }]
    return [{ label: name + '开', cmd: device.cmd_on }, { label: name + '关', cmd: device.cmd_off }]
  }
  var defaults = {
    door_lock: [{ label: '开门', cmd: 'door_open' }, { label: '关门', cmd: 'door_close' }],
    light: [{ label: name + '开', cmd: 'light_on' }, { label: name + '关', cmd: 'light_off' }],
    ac: [{ label: name + '开', cmd: 'ac_on' }, { label: name + '关', cmd: 'ac_off' }],
    exhaust: [{ label: name + '开', cmd: 'exhaust_on' }, { label: name + '关', cmd: 'exhaust_off' }],
    machine: [{ label: name + '开', cmd: 'machine_on' }, { label: name + '关', cmd: 'machine_off' }],
    combo: [
      { label: '开门', cmd: 'door_open' }, { label: '关门', cmd: 'door_close' },
      { label: '灯开', cmd: 'light_on' }, { label: '灯关', cmd: 'light_off' },
      { label: '空调开', cmd: 'ac_on' }, { label: '空调关', cmd: 'ac_off' },
      { label: '排风开', cmd: 'exhaust_on' }, { label: '排风关', cmd: 'exhaust_off' }
    ]
  }
  return defaults[device.type] || []
}

module.exports = { buildDeviceButtons: buildDeviceButtons }
