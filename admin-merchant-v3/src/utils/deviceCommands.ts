import { parseJson } from '@/utils/merchant'

export interface DeviceButton {
  label: string
  cmd: string
}

/** 根据设备 config_json / type 生成控制按钮（与小程序逻辑一致） */
export const buildDeviceButtons = (dev: any): DeviceButton[] => {
  const name = dev?.name || '设备'
  const config = parseJson<Record<string, any>>(dev?.config_json, {})

  if (config.cmd_on) {
    if (config.is_toggle === false || !config.cmd_off) {
      return [{ label: name, cmd: config.cmd_on }]
    }
    return [
      { label: `${name}开`, cmd: config.cmd_on },
      { label: `${name}关`, cmd: config.cmd_off }
    ]
  }

  const type = dev?.type
  if (type === 'combo') {
    return [
      { label: '开门', cmd: 'door_open' },
      { label: '关门', cmd: 'door_close' },
      { label: '灯开', cmd: 'light_on' },
      { label: '灯关', cmd: 'light_off' },
      { label: '空调开', cmd: 'ac_on' },
      { label: '空调关', cmd: 'ac_off' },
      { label: '排风开', cmd: 'exhaust_on' },
      { label: '排风关', cmd: 'exhaust_off' }
    ]
  }

  const map: Record<string, DeviceButton[]> = {
    door_lock: [
      { label: '开门', cmd: 'door_open' },
      { label: '关门', cmd: 'door_close' }
    ],
    light: [
      { label: `${name}开`, cmd: 'light_on' },
      { label: `${name}关`, cmd: 'light_off' }
    ],
    ac: [
      { label: `${name}开`, cmd: 'ac_on' },
      { label: `${name}关`, cmd: 'ac_off' }
    ],
    exhaust: [
      { label: `${name}开`, cmd: 'exhaust_on' },
      { label: `${name}关`, cmd: 'exhaust_off' }
    ],
    machine: [
      { label: `${name}开`, cmd: 'machine_on' },
      { label: `${name}关`, cmd: 'machine_off' }
    ]
  }
  return map[type] || []
}

export const deviceTypeLabel = (type: string) =>
  (
    {
      door_lock: '门锁',
      light: '灯光',
      ac: '空调',
      exhaust: '排风',
      machine: '插座/机器',
      combo: '组合',
      speaker: '音响',
      other: '其他'
    } as Record<string, string>
  )[type] || type || '-'
