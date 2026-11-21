const test = require('node:test')
const assert = require('node:assert/strict')
const { buildDeviceButtons } = require('./device_commands.js')

test('uses configured relay commands for a user-controllable machine', () => {
  assert.deepEqual(buildDeviceButtons({
    name: '插座1', type: 'machine', user_control_enabled: true,
    cmd_on: 'relay1_on', cmd_off: 'relay1_off', is_toggle: true,
  }), [
    { label: '插座1开', cmd: 'relay1_on' },
    { label: '插座1关', cmd: 'relay1_off' },
  ])
})

test('uses known defaults for a door lock', () => {
  assert.deepEqual(buildDeviceButtons({ name: '包间门', type: 'door_lock', user_control_enabled: true }), [
    { label: '开门', cmd: 'door_open' },
    { label: '关门', cmd: 'door_close' },
  ])
})

test('does not create controls for a device without a known command mapping', () => {
  assert.deepEqual(buildDeviceButtons({ name: '未知设备', type: 'other', user_control_enabled: true }), [])
})
