<template>
  <ContentWrap title="设备管理" :body-style="{ padding: '0 18px 18px' }">
    <template #header>
      <div class="toolbar">
        <el-select v-model="typeFilter" class="type-filter" placeholder="全部房型" clearable>
          <el-option v-for="type in resourceTypes" :key="type.id" :label="type.name" :value="type.id" />
        </el-select>
        <el-input
          v-model="keyword"
          class="search-input"
          :prefix-icon="Search"
          clearable
          placeholder="搜索设备名 / SN"
        />
        <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
        <el-button @click="openNetworkGuide">配网说明</el-button>
        <el-button type="primary" :icon="Plus" @click="openAdd">添加设备</el-button>
      </div>
    </template>

    <div class="summary">
      共 {{ devices.length }} 台设备，其中未绑定 {{ unboundCount }} 台
    </div>

    <el-card v-if="uniqueVoiceDevices.length" class="voice-card" shadow="never">
      <template #header>
        <div class="voice-card__head">
          <span>语音设备（只读）</span>
          <el-button link type="primary" @click="goVoiceConfig">去语音播报配置</el-button>
        </div>
      </template>
      <el-table :data="uniqueVoiceDevices" size="small">
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="device_sn" label="SN" min-width="180" />
        <el-table-column prop="room_name" label="所在房间" min-width="140" />
        <el-table-column label="类型" width="100">
          <template #default>云喇叭</template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button
              link
              type="danger"
              :loading="deletingVoiceId === Number(row.id)"
              @click="handleDeleteVoice(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <div v-loading="loading" class="group-list">
      <section v-for="group in visibleGroups" :key="group.key" class="type-group">
        <div class="type-group__head">
          <div class="type-group__name">
            {{ group.name }}
            <el-tag v-if="group.unbound" size="small" type="warning" effect="light">
              待分配
            </el-tag>
          </div>
          <div class="type-group__meta">
            {{ group.rooms.length }} 间 / {{ groupDeviceCount(group) }} 台设备
          </div>
        </div>

        <div class="room-list">
          <div v-for="room in group.rooms" :key="room.key" class="room-card">
            <div class="room-head" @click="toggleRoom(room.key)">
              <div class="room-head__left">
                <el-icon class="room-chevron" :class="{ 'is-expanded': isExpanded(room.key) }">
                  <ArrowDown />
                </el-icon>
                <span class="room-name">{{ room.name }}</span>
                <el-tag size="small" effect="plain">{{ room.devices.length }} 台</el-tag>
              </div>
              <span class="room-toggle-text">
                {{ isExpanded(room.key) ? '收起' : '展开' }}
              </span>
            </div>

            <div v-show="isExpanded(room.key)" class="room-body">
              <el-empty
                v-if="!room.devices.length"
                :description="room.unbound ? '暂无未绑定设备' : '该房间未绑定设备'"
                :image-size="52"
              />
              <div v-else class="device-list">
                <div v-for="dev in room.devices" :key="dev.id" class="device-card">
                  <div class="device-main">
                    <div class="device-info">
                      <span class="device-status-dot" :class="{ online: isOnline(dev.status) }"></span>
                      <span class="device-name">{{ dev.name }}</span>
                      <el-tag size="small" effect="plain">{{ dev.typeLabel }}</el-tag>
                      <el-tag
                        size="small"
                        effect="light"
                        :type="isOnline(dev.status) ? 'success' : 'info'"
                      >
                        {{ isOnline(dev.status) ? '在线' : '离线' }}
                      </el-tag>
                      <span class="device-sn">{{ dev.device_sn }}</span>
                      <el-tag v-if="dev.protocol === 'json'" size="small" type="warning" effect="plain">
                        JSON
                      </el-tag>
                    </div>
                    <div class="device-ops">
                      <el-button v-if="room.unbound" link type="warning" @click="openAssign(dev)">
                        分配到房间
                      </el-button>
                      <el-button link type="primary" @click="openRename(dev)">改名</el-button>
                      <el-button
                        link
                        type="danger"
                        :loading="deletingId === Number(dev.id)"
                        :disabled="deletingId !== null"
                        @click="handleDelete(dev)"
                      >
                        删除
                      </el-button>
                    </div>
                  </div>

                  <div v-if="dev.buttons.length" class="command-row">
                    <el-button
                      v-for="(btn, index) in dev.buttons"
                      :key="btn.cmd"
                      size="small"
                      plain
                      :type="index === 0 ? 'primary' : 'default'"
                      :loading="isSending(dev.id, btn.cmd)"
                      @click="sendCommand(dev, btn.cmd)"
                    >
                      {{ btn.label }}
                    </el-button>
                  </div>
                  <div v-else class="command-empty">未配置控制命令</div>

                  <div class="switch-row">
                    <div class="switch-item">
                      <span class="switch-label">用户端控制</span>
                      <el-switch
                        :model-value="dev.userControlEnabled"
                        :loading="isSwitchBusy(dev.id)"
                        @change="(val) => handleUserControl(dev, val)"
                      />
                    </div>
                    <div class="switch-item auto-close">
                      <span class="switch-label">预约结束自动断电</span>
                      <el-switch
                        :model-value="dev.autoClose"
                        :loading="isSwitchBusy(dev.id)"
                        @change="(val) => handleAutoCloseToggle(dev, val)"
                      />
                      <template v-if="dev.autoClose">
                        <el-input-number
                          :model-value="dev.autoCloseDelay"
                          :min="0"
                          :max="1440"
                          :step="1"
                          size="small"
                          controls-position="right"
                          class="delay-input"
                          @change="(val) => handleAutoCloseDelay(dev, val)"
                        />
                        <span class="switch-unit">分钟后断电</span>
                      </template>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <el-empty v-if="!visibleGroups.length && !loading" :description="emptyText" :image-size="90" />
    </div>

    <el-dialog v-model="addVisible" title="添加设备" width="560px" append-to-body style="max-width: 92vw">
      <el-form label-width="92px" @submit.prevent>
        <el-form-item label="设备 SN" required>
          <el-input v-model="addForm.deviceSn" maxlength="64" placeholder="设备序列号" />
        </el-form-item>
        <el-form-item label="产品模板" required>
          <el-select
            v-model="addForm.templateId"
            placeholder="请选择产品模板"
            style="width: 100%"
            @change="handleTemplateChange"
          >
            <el-option
              v-for="tpl in templates"
              :key="tpl.id"
              :label="templateLabel(tpl)"
              :value="Number(tpl.id)"
            />
          </el-select>
          <div v-if="addForm.templateId" class="help">设备名称由模板决定，添加后可在设备卡片里重命名。</div>
        </el-form-item>
        <el-form-item label="目标房间" required>
          <el-select v-model="addForm.roomId" placeholder="请选择目标房间" style="width: 100%">
            <el-option
              v-for="room in resources"
              :key="room.id"
              :label="roomLabel(room)"
              :value="Number(room.id)"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="handleAdd">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="renameVisible" title="修改设备名称" width="420px" append-to-body style="max-width: 92vw">
      <el-form label-width="80px" @submit.prevent>
        <el-form-item label="设备名称" required>
          <el-input v-model="renameName" maxlength="30" show-word-limit placeholder="请输入新名称" @keyup.enter="handleRename" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameVisible = false">取消</el-button>
        <el-button type="primary" :loading="renaming" @click="handleRename">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="assignVisible" :title="assignTarget ? `分配设备 - ${assignTarget.name}` : '分配设备'" width="420px" append-to-body style="max-width: 92vw">
      <el-form label-width="80px" @submit.prevent>
        <el-form-item label="目标房间" required>
          <el-select v-model="assignRoomId" placeholder="请选择房间" style="width: 100%">
            <el-option
              v-for="room in resources"
              :key="room.id"
              :label="roomLabel(room)"
              :value="Number(room.id)"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="dialog-tip">分配后会从原房间移除该设备。</div>
      <template #footer>
        <el-button @click="assignVisible = false">取消</el-button>
        <el-button type="primary" :loading="assigning" :disabled="!assignRoomId" @click="handleAssign">
          确认分配
        </el-button>
      </template>
    </el-dialog>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  controlDevice,
  createDevice,
  deleteDevice,
  getDevices,
  getProductTemplates,
  getResources,
  getResourceTypes,
  getVoiceDevices,
  linkDevices,
  updateDevice
} from '@/api/merchant'
import { parseJson } from '@/utils/merchant'
import { buildDeviceButtons, deviceTypeLabel } from '@/utils/deviceCommands'
import type { DeviceButton } from '@/utils/deviceCommands'

defineOptions({ name: 'MerchantDevices' })

interface ResourceItem {
  id: number
  name: string
  resource_type_id?: number
  resource_type?: { id?: number; name?: string }
  device_ids?: string | number[] | null
  [key: string]: any
}

interface DeviceItem {
  id: number
  device_sn?: string
  name?: string
  type?: string
  status?: any
  config_json?: string
  protocol?: string
  user_control_enabled?: boolean
  [key: string]: any
}

interface DeviceView extends DeviceItem {
  buttons: DeviceButton[]
  typeLabel: string
  autoClose: boolean
  autoCloseDelay: number
  userControlEnabled: boolean
}

interface RoomView {
  key: string
  id: number
  name: string
  unbound: boolean
  devices: DeviceView[]
}

interface TypeGroup {
  key: string
  typeId: number
  name: string
  unbound: boolean
  rooms: RoomView[]
}

const merchantStore = useMerchantStore()
const router = useRouter()

const resourceTypes = ref<any[]>([])
const resources = ref<ResourceItem[]>([])
const devices = ref<DeviceItem[]>([])
const voiceDevices = ref<DeviceItem[]>([])
const templates = ref<any[]>([])
const loading = ref(false)

const uniqueVoiceDevices = computed(() => {
  const seen = new Set<string>()
  const out: DeviceItem[] = []
  voiceDevices.value.forEach((device) => {
    const key = device.device_sn || String(device.id)
    if (seen.has(key)) return
    seen.add(key)
    out.push(device)
  })
  return out
})

const goVoiceConfig = () => router.push('/voice-config')

const typeFilter = ref<number | ''>('')
const keyword = ref('')
const expandedKeys = ref<string[]>([])
let expandedInitialized = false

const sendingKeys = ref<string[]>([])
const switchBusyIds = ref<number[]>([])
const deletingId = ref<number | null>(null)
const deletingVoiceId = ref<number | null>(null)

const addVisible = ref(false)
const adding = ref(false)
const addForm = reactive({
  deviceSn: '',
  templateId: '' as number | '',
  type: 'other',
  protocol: 'ascii',
  name: '',
  roomId: '' as number | ''
})

const renameVisible = ref(false)
const renaming = ref(false)
const renameTarget = ref<DeviceView | null>(null)
const renameName = ref('')

const assignVisible = ref(false)
const assigning = ref(false)
const assignTarget = ref<DeviceView | null>(null)
const assignRoomId = ref<number | ''>('')

let loadSeq = 0
let statusTimer: number | undefined

const toList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const roomDeviceIds = (room?: ResourceItem | null): number[] => {
  const raw = room?.device_ids
  const parsed = Array.isArray(raw) ? raw : parseJson<any[]>(raw, [])
  if (!Array.isArray(parsed)) return []
  return parsed.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0)
}

const isOnline = (status: any) => Number(status) === 1 || status === 'online'

const decorate = (device: DeviceItem): DeviceView => {
  const cfg = parseJson<Record<string, any>>(device.config_json, {})
  const delay = Number(cfg.auto_close_delay_min)
  return {
    ...device,
    id: Number(device.id),
    name: device.name || device.device_sn || `设备 #${device.id}`,
    buttons: buildDeviceButtons(device),
    typeLabel: deviceTypeLabel(device.type || ''),
    autoClose: !!cfg.auto_close,
    autoCloseDelay: Number.isFinite(delay) && delay > 0 ? delay : 0,
    userControlEnabled: !!device.user_control_enabled
  }
}

const boundDeviceIds = computed(() => {
  const set = new Set<number>()
  resources.value.forEach((room) => roomDeviceIds(room).forEach((id) => set.add(id)))
  return set
})

const unboundCount = computed(
  () => devices.value.filter((dev) => !boundDeviceIds.value.has(Number(dev.id))).length
)

const allGroups = computed<TypeGroup[]>(() => {
  const deviceMap = new Map<number, DeviceItem>()
  devices.value.forEach((dev) => deviceMap.set(Number(dev.id), dev))

  const groupMap = new Map<string, TypeGroup>()
  resources.value.forEach((room) => {
    const typeId = Number(room.resource_type_id ?? room.resource_type?.id ?? 0)
    const typeName =
      room.resource_type?.name ||
      resourceTypes.value.find((type) => Number(type.id) === typeId)?.name ||
      '未分类'
    const key = `type-${typeId}`
    let group = groupMap.get(key)
    if (!group) {
      group = { key, typeId, name: typeName, unbound: false, rooms: [] }
      groupMap.set(key, group)
    }
    const roomDevices = roomDeviceIds(room)
      .map((id) => deviceMap.get(id))
      .filter((dev): dev is DeviceItem => !!dev)
      .map(decorate)
    group.rooms.push({
      key: `room-${room.id}`,
      id: Number(room.id),
      name: room.name || `房间 #${room.id}`,
      unbound: false,
      devices: roomDevices
    })
  })

  const groups = Array.from(groupMap.values())
  const unboundDevices = devices.value
    .filter((dev) => !boundDeviceIds.value.has(Number(dev.id)))
    .map(decorate)
  if (unboundDevices.length) {
    groups.push({
      key: 'type-unbound',
      typeId: 0,
      name: '未绑定设备',
      unbound: true,
      rooms: [
        {
          key: 'room-unbound',
          id: 0,
          name: '未分配设备',
          unbound: true,
          devices: unboundDevices
        }
      ]
    })
  }
  return groups
})

const matchDevice = (dev: DeviceView, search: string) => {
  if (!search) return true
  return `${dev.name || ''} ${dev.device_sn || ''}`.toLowerCase().includes(search)
}

const visibleGroups = computed<TypeGroup[]>(() => {
  const search = keyword.value.trim().toLowerCase()
  const result: TypeGroup[] = []
  allGroups.value.forEach((group) => {
    if (group.unbound) {
      if (typeFilter.value !== '') return
      const unboundRoom = group.rooms[0]
      if (!unboundRoom) return
      const roomDevices = unboundRoom.devices.filter((dev) => matchDevice(dev, search))
      if (!roomDevices.length) return
      result.push({ ...group, rooms: [{ ...unboundRoom, devices: roomDevices }] })
      return
    }
    if (typeFilter.value !== '' && Number(group.typeId) !== Number(typeFilter.value)) return
    const rooms = group.rooms
      .map((room) => ({ ...room, devices: room.devices.filter((dev) => matchDevice(dev, search)) }))
      .filter((room) => !search || room.devices.length)
    if (!rooms.length) return
    result.push({ ...group, rooms })
  })
  return result
})

const emptyText = computed(() =>
  typeFilter.value !== '' || keyword.value.trim()
    ? '没有符合筛选条件的设备'
    : '暂无设备，点击右上角添加设备'
)

const groupDeviceCount = (group: TypeGroup) =>
  group.rooms.reduce((total, room) => total + room.devices.length, 0)

const templateLabel = (tpl: any) =>
  `${tpl.name || `模板 #${tpl.id}`}（${tpl.devices?.length || 0} 路）`

const roomLabel = (room: ResourceItem) => {
  const typeName =
    room.resource_type?.name ||
    resourceTypes.value.find((type) => Number(type.id) === Number(room.resource_type_id))?.name ||
    ''
  return typeName ? `${room.name}（${typeName}）` : room.name
}

const isExpanded = (key: string) => expandedKeys.value.includes(key)

const toggleRoom = (key: string) => {
  const index = expandedKeys.value.indexOf(key)
  if (index >= 0) expandedKeys.value.splice(index, 1)
  else expandedKeys.value.push(key)
}

const syncExpanded = () => {
  const keys = resources.value.map((room) => `room-${room.id}`)
  if (unboundCount.value > 0) keys.push('room-unbound')
  if (!expandedInitialized) {
    expandedKeys.value = []
    expandedInitialized = true
    return
  }
  const valid = new Set(keys)
  expandedKeys.value = expandedKeys.value.filter((key) => valid.has(key))
}

const loadAll = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const [resourcesRes, devicesRes, typesRes, templatesRes, voiceDevicesRes] = await Promise.all([
      getResources(),
      getDevices(),
      getResourceTypes(),
      getProductTemplates(),
      getVoiceDevices().catch(() => [])
    ])
    if (seq !== loadSeq) return
    resources.value = toList(resourcesRes) as ResourceItem[]
    devices.value = toList(devicesRes) as DeviceItem[]
    resourceTypes.value = toList(typesRes)
    templates.value = toList(templatesRes)
    voiceDevices.value = toList(voiceDevicesRes) as DeviceItem[]
    syncExpanded()
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const refreshDeviceStatus = async () => {
  try {
    const res = await getDevices()
    devices.value = toList(res) as DeviceItem[]
  } catch {
    // 请求层已统一提示
  }
}

const isSending = (id: number, cmd: string) =>
  sendingKeys.value.includes(`${Number(id)}:${cmd}`)

const sendCommand = async (dev: DeviceView, cmd: string) => {
  const id = Number(dev.id)
  const key = `${id}:${cmd}`
  if (sendingKeys.value.includes(key)) return
  sendingKeys.value = [...sendingKeys.value, key]
  try {
    await controlDevice(id, cmd)
    ElMessage.success('指令已发送')
  } finally {
    sendingKeys.value = sendingKeys.value.filter((item) => item !== key)
  }
}

const isSwitchBusy = (id: number) => switchBusyIds.value.includes(Number(id))

const setSwitchBusy = (id: number, busy: boolean) => {
  const target = Number(id)
  if (busy) {
    if (!switchBusyIds.value.includes(target)) {
      switchBusyIds.value = [...switchBusyIds.value, target]
    }
    return
  }
  switchBusyIds.value = switchBusyIds.value.filter((item) => item !== target)
}

const handleUserControl = async (dev: DeviceView, value: any) => {
  const id = Number(dev.id)
  if (isSwitchBusy(id)) return
  const enabled = !!value
  setSwitchBusy(id, true)
  try {
    await updateDevice(id, { user_control_enabled: enabled })
    ElMessage.success(enabled ? '用户端控制已开启' : '用户端控制已关闭')
    await loadAll()
  } finally {
    setSwitchBusy(id, false)
  }
}

const handleAutoCloseToggle = async (dev: DeviceView, value: any) => {
  const id = Number(dev.id)
  if (isSwitchBusy(id)) return
  const enabled = !!value
  const cfg = parseJson<Record<string, any>>(dev.config_json, {})
  cfg.auto_close = enabled
  if (cfg.auto_close_delay_min === undefined) cfg.auto_close_delay_min = dev.autoCloseDelay || 0
  setSwitchBusy(id, true)
  try {
    await updateDevice(id, { config_json: JSON.stringify(cfg) })
    ElMessage.success(enabled ? '已开启预约结束自动断电' : '已关闭预约结束自动断电')
    await loadAll()
  } finally {
    setSwitchBusy(id, false)
  }
}

const handleAutoCloseDelay = async (dev: DeviceView, value: any) => {
  const id = Number(dev.id)
  if (isSwitchBusy(id)) return
  const raw = Number(value)
  const delay = Number.isFinite(raw) ? Math.min(1440, Math.max(0, Math.round(raw))) : 0
  const cfg = parseJson<Record<string, any>>(dev.config_json, {})
  cfg.auto_close = true
  cfg.auto_close_delay_min = delay
  setSwitchBusy(id, true)
  try {
    await updateDevice(id, { config_json: JSON.stringify(cfg) })
    ElMessage.success('自动断电延迟已保存')
    await loadAll()
  } finally {
    setSwitchBusy(id, false)
  }
}

const openRename = (dev: DeviceView) => {
  renameTarget.value = dev
  renameName.value = dev.name || ''
  renameVisible.value = true
}

const handleRename = async () => {
  if (renaming.value || !renameTarget.value) return
  const name = renameName.value.trim()
  if (!name) {
    ElMessage.warning('请输入设备名称')
    return
  }
  renaming.value = true
  try {
    await updateDevice(Number(renameTarget.value.id), { name })
    ElMessage.success('设备名称已更新')
    renameVisible.value = false
    renameTarget.value = null
    await loadAll()
  } finally {
    renaming.value = false
  }
}

const handleDeleteVoice = async (dev: DeviceItem) => {
  if (deletingVoiceId.value !== null) return
  try {
    await ElMessageBox.confirm(
      `确认删除云喇叭「${dev.name || dev.device_sn}」？删除后需要重新添加才能恢复语音配置。`,
      '删除语音设备',
      {
        type: 'warning',
        confirmButtonText: '确认删除',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }
  deletingVoiceId.value = Number(dev.id)
  try {
    await deleteDevice(Number(dev.id))
    ElMessage.success('语音设备已删除')
    await loadAll()
  } finally {
    deletingVoiceId.value = null
  }
}

const handleDelete = async (dev: DeviceView) => {
  if (deletingId.value !== null) return
  try {
    await ElMessageBox.confirm(`确认删除设备「${dev.name}」？删除后不可恢复。`, '删除设备', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  deletingId.value = Number(dev.id)
  try {
    await deleteDevice(Number(dev.id))
    ElMessage.success('设备已删除')
    await loadAll()
  } finally {
    deletingId.value = null
  }
}

const openAdd = () => {
  addForm.deviceSn = ''
  addForm.templateId = ''
  addForm.type = 'other'
  addForm.protocol = 'ascii'
  addForm.name = ''
  addForm.roomId = ''
  addVisible.value = true
}

const handleTemplateChange = () => {
  const template = templates.value.find((tpl) => Number(tpl.id) === Number(addForm.templateId))
  if (!template || addForm.name.trim()) return
  addForm.name =
    template.devices?.length === 1 ? template.devices[0]?.name || template.name : template.name
}

const handleAdd = async () => {
  if (adding.value) return
  const sn = addForm.deviceSn.trim()
  if (!sn) {
    ElMessage.warning('请输入设备 SN')
    return
  }
  const templateId = Number(addForm.templateId) || 0
  if (!templateId) {
    ElMessage.warning('请选择产品模板')
    return
  }
  const roomId = Number(addForm.roomId) || 0
  if (!roomId) {
    ElMessage.warning('请选择目标房间')
    return
  }
  const storeId = Number(merchantStore.storeId) || 0
  if (!storeId) {
    ElMessage.warning('请先选择门店')
    return
  }
  if (!resources.value.some((room) => Number(room.id) === roomId)) {
    ElMessage.warning('目标房间不存在，请刷新后重试')
    return
  }

  adding.value = true
  try {
    const body: Record<string, any> = {
      device_sn: sn,
      name: sn,
      type: 'other',
      store_id: storeId,
      room_id: roomId,
      mqtt_topic: String(templateId)
    }

    const created = await createDevice(body)
    const createdList = Array.isArray(created) ? created : created ? [created] : []
    const newIds = createdList
      .map((item) => Number(item?.id))
      .filter((id) => Number.isFinite(id) && id > 0)

    if (newIds.length) {
      const room = resources.value.find((item) => Number(item.id) === roomId)
      const ids = Array.from(new Set([...roomDeviceIds(room), ...newIds]))
      await linkDevices(roomId, ids)
      ElMessage.success('设备已添加并绑定到房间')
    } else {
      ElMessage.success('设备已添加')
    }
    addVisible.value = false
    await loadAll()
  } finally {
    adding.value = false
  }
}

const openAssign = (dev: DeviceView) => {
  assignTarget.value = dev
  assignRoomId.value = ''
  assignVisible.value = true
}

const handleAssign = async () => {
  if (assigning.value || !assignTarget.value) return
  const roomId = Number(assignRoomId.value) || 0
  if (!roomId) {
    ElMessage.warning('请选择目标房间')
    return
  }
  if (!resources.value.some((room) => Number(room.id) === roomId)) {
    ElMessage.warning('目标房间不存在，请刷新后重试')
    return
  }
  const deviceId = Number(assignTarget.value.id)
  assigning.value = true
  try {
    const tasks: Promise<any>[] = []
    resources.value.forEach((room) => {
      const ids = roomDeviceIds(room)
      const hasDevice = ids.includes(deviceId)
      if (Number(room.id) === roomId) {
        if (!hasDevice) tasks.push(linkDevices(roomId, [...ids, deviceId]))
        return
      }
      if (hasDevice) tasks.push(linkDevices(Number(room.id), ids.filter((id) => id !== deviceId)))
    })
    await Promise.all(tasks)
    ElMessage.success('设备已分配到房间')
    assignVisible.value = false
    assignTarget.value = null
    await loadAll()
  } finally {
    assigning.value = false
  }
}

const openNetworkGuide = () => {
  ElMessageBox.alert(
    '有按钮的设备，长按第一个按钮至灯光快速闪烁，打开手机wifi找到ESP开头的网络连接，自动跳转到配网界面，输入门店的wifi名称和密码，即可完成配网',
    '配网说明',
    { confirmButtonText: '知道了' }
  )
}

watch(
  () => merchantStore.storeId,
  (id) => {
    typeFilter.value = ''
    keyword.value = ''
    expandedKeys.value = []
    expandedInitialized = false
    addVisible.value = false
    renameVisible.value = false
    assignVisible.value = false
    if (!id) {
      resourceTypes.value = []
      resources.value = []
      devices.value = []
      templates.value = []
      return
    }
    loadAll()
  }
)

onMounted(() => {
  loadAll()
  statusTimer = window.setInterval(refreshDeviceStatus, 15000)
})

onUnmounted(() => {
  if (statusTimer) window.clearInterval(statusTimer)
})
</script>

<style lang="scss" scoped>
.voice-card {
  margin-bottom: 16px;
}

.voice-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.toolbar {
  display: flex;
  flex-grow: 1;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.type-filter {
  width: 150px;
}

.search-input {
  width: 220px;
}

.summary {
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.group-list {
  min-height: 120px;
}

.type-group {
  margin-bottom: 16px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
}

.type-group__head {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
}

.type-group__name {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.type-group__meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.room-list {
  display: flex;
  flex-direction: column;
}

.room-card {
  border-top: 1px solid var(--el-border-color-lighter);
}

.room-card:first-child {
  border-top: none;
}

.room-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 11px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.room-head:hover {
  background: var(--el-fill-color-lighter);
}

.room-head__left {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.room-name {
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-chevron {
  color: var(--el-text-color-secondary);
  transition: transform 0.2s;
}

.room-chevron.is-expanded {
  transform: rotate(0deg);
}

.room-chevron:not(.is-expanded) {
  transform: rotate(-90deg);
}

.room-toggle-text {
  font-size: 12px;
  color: var(--el-color-primary);
}

.room-body {
  padding: 2px 16px 14px;
  background: var(--el-fill-color-lighter);
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.device-card {
  padding: 12px 14px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.device-main {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.device-info {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.device-status-dot {
  width: 7px;
  height: 7px;
  background: var(--el-text-color-placeholder);
  border-radius: 50%;
}

.device-status-dot.online {
  background: #22c55e;
}

.device-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.device-sn {
  font-family: Consolas, Monaco, monospace;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.device-ops {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

.command-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.command-empty {
  margin-top: 10px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.switch-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 24px;
  align-items: center;
  padding-top: 10px;
  margin-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}

.switch-item {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.switch-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.switch-unit {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.delay-input {
  width: 110px;
}

.dialog-tip,
.help {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

@media (width <= 768px) {
  .toolbar {
    justify-content: flex-start;
  }

  .type-filter,
  .search-input {
    width: 100%;
  }

  .device-main {
    align-items: flex-start;
  }

  .switch-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>

