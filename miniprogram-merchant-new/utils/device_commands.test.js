const test = require('node:test')
const assert = require('node:assert/strict')
const { buildDeviceButtons } = require('./device_commands.js')

test('uses configured command for custom machine device', () => {
  const buttons = buildDeviceButtons({
    name: '插座1', type: 'machine',
    config_json: JSON.stringify({ cmd_on: 'relay1_on', cmd_off: 'relay1_off', is_toggle: true }),
  })
  assert.deepEqual(buttons, [
    { label: '插座1开', cmd: 'relay1_on' },
    { label: '插座1关', cmd: 'relay1_off' },
  ])
})

test('falls back to known machine commands when no mapping exists', () => {
  assert.deepEqual(buildDeviceButtons({ name: '插座1', type: 'machine' }), [
    { label: '插座1开', cmd: 'machine_on' },
    { label: '插座1关', cmd: 'machine_off' },
  ])
})

test('shows one button for a pulse device', () => {
  assert.deepEqual(buildDeviceButtons({
    name: '门锁', type: 'machine', config_json: '{"cmd_on":"door_open","is_toggle":false}',
  }), [{ label: '门锁', cmd: 'door_open' }])
})
