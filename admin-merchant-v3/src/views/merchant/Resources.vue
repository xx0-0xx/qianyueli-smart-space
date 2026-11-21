<template>
  <ContentWrap title="房间列表" :body-style="{ padding: '0 18px 18px' }">
    <div class="toolbar">
      <div class="toolbar-left">
        <el-select v-model="typeFilter" class="type-filter" placeholder="全部房型">
          <el-option label="全部房型" value="" />
          <el-option
            v-for="type in resourceTypes"
            :key="type.id"
            :label="type.name"
            :value="type.id"
          />
        </el-select>
        <el-input
          v-model="keyword"
          class="search-input"
          :prefix-icon="Search"
          clearable
          placeholder="搜索房间名"
        />
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreateRoom">新增房间</el-button>
    </div>

    <el-table v-loading="loading" :data="visibleRooms" row-key="id" empty-text="暂无房间">
      <el-table-column label="房间名" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="room-name">{{ row.name || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="房型" width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.typeName }}</template>
      </el-table-column>
      <el-table-column label="已绑定设备" min-width="280">
        <template #default="{ row }">
          <div v-if="row.deviceList.length" class="device-tags">
            <el-tag
              v-for="device in row.deviceList"
              :key="device.id"
              type="info"
              effect="plain"
              size="small"
            >
              {{ device.label }}
            </el-tag>
          </div>
          <span v-else class="muted">未绑定</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button link type="primary" @click="openBind(row)">绑定设备</el-button>
            <el-button
              link
              type="danger"
              :loading="deletingId === row.id"
              :disabled="deletingId !== null"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </div>
        </template>
      </el-table-column>

      <template #empty>
        <el-empty :description="emptyText" :image-size="90" />
      </template>
    </el-table>

    <el-dialog
      v-model="createVisible"
      title="新增房间"
      width="440px"
      append-to-body
      style="max-width: 92vw"
    >
      <el-form label-width="80px" @submit.prevent>
        <el-form-item label="房型" required>
          <el-select v-model="newRoomTypeId" placeholder="请选择房型" style="width: 100%">
            <el-option
              v-for="type in resourceTypes"
              :key="type.id"
              :label="type.name"
              :value="type.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="房间名" required>
          <el-input
            v-model="newRoomName"
            maxlength="30"
            show-word-limit
            placeholder="如：A01"
            @keyup.enter="handleCreateRoom"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreateRoom">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="bindVisible"
      :title="bindRoom ? `绑定设备 - ${bindRoom.name}` : '绑定设备'"
      width="560px"
      append-to-body
      style="max-width: 92vw"
    >
      <template v-if="devices.length">
        <div class="bind-tip">
          已选 {{ bindSelection.length }} 台设备。已被其他房间占用的设备仍可保存；如需独占，请先在原房间解绑。
        </div>
        <el-checkbox-group v-model="bindSelection" class="device-list">
          <el-checkbox
            v-for="device in devices"
            :key="device.id"
            :value="Number(device.id)"
            class="device-item"
          >
            <span class="device-name">{{ deviceLabel(device) }}</span>
            <el-tag
              v-if="occupiedRoomName(Number(device.id))"
              type="warning"
              effect="light"
              size="small"
              class="occupied-tag"
            >
              已被 {{ occupiedRoomName(Number(device.id)) }} 占用
            </el-tag>
          </el-checkbox>
        </el-checkbox-group>
      </template>
      <el-empty v-else description="当前门店暂无可绑定设备" :image-size="80" />
      <template #footer>
        <el-button @click="bindVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="linking"
          :disabled="!bindRoom"
          @click="handleSaveBind"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  createResource,
  deleteResource,
  getDevices,
  getResources,
  getResourceTypes,
  getVoiceDevices,
  linkDevices
} from '@/api/merchant'
import { parseJson } from '@/utils/merchant'

defineOptions({ name: 'MerchantResources' })

interface ResourceTypeItem {
  id: number
  name: string
  [key: string]: any
}

interface DeviceItem {
  id: number
  device_sn?: string
  name?: string
  type?: string
  status?: any
  [key: string]: any
}

interface ResourceItem {
  id: number
  name: string
  resource_type_id?: number
  resource_type?: { id?: number; name?: string }
  device_ids?: string | number[] | null
  [key: string]: any
}

interface RoomRow extends ResourceItem {
  typeName: string
  deviceList: Array<{ id: number; label: string }>
}

const merchantStore = useMerchantStore()

const resourceTypes = ref<ResourceTypeItem[]>([])
const resources = ref<ResourceItem[]>([])
const devices = ref<DeviceItem[]>([])
const voiceDevices = ref<DeviceItem[]>([])
const loading = ref(false)
const typeFilter = ref<number | ''>('')
const keyword = ref('')

const createVisible = ref(false)
const creating = ref(false)
const newRoomName = ref('')
const newRoomTypeId = ref<number | ''>('')

const bindVisible = ref(false)
const linking = ref(false)
const bindRoom = ref<ResourceItem | null>(null)
const bindSelection = ref<number[]>([])

const deletingId = ref<number | null>(null)

let loadSeq = 0

const toList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const roomDeviceIds = (room: ResourceItem): number[] => {
  const raw = room?.device_ids
  const parsed = Array.isArray(raw) ? raw : parseJson<any[]>(raw, [])
  if (!Array.isArray(parsed)) return []
  return parsed.map((id) => Number(id)).filter((id) => Number.isFinite(id))
}

const deviceLabel = (device: DeviceItem) => {
  const name = device.name || device.device_sn || `设备 #${device.id}`
  const sn = device.device_sn || ''
  const label = sn && sn !== name ? `${name}（${sn}）` : name
  return device.type === 'speaker' ? `${label} [语音]` : label
}

const deviceMap = computed(() => {
  const map = new Map<number, DeviceItem>()
  devices.value.forEach((device) => map.set(Number(device.id), device))
  voiceDevices.value.forEach((device) =>
    map.set(Number(device.id), { ...device, type: device.type || 'speaker' })
  )
  return map
})

const roomRows = computed<RoomRow[]>(() =>
  resources.value.map((room) => {
    const deviceList = roomDeviceIds(room)
      .map((id) => {
        const device = deviceMap.value.get(id)
        return {
          id,
          label: device ? deviceLabel(device) : `设备 #${id}`
        }
      })
      .filter((item, index, list) => list.findIndex((x) => x.label === item.label) === index)
    const typeName =
      room.resource_type?.name ||
      resourceTypes.value.find((type) => Number(type.id) === Number(room.resource_type_id))
        ?.name ||
      '-'
    return { ...room, typeName, deviceList }
  })
)

const visibleRooms = computed(() => {
  const search = keyword.value.trim().toLowerCase()
  return roomRows.value.filter((room) => {
    const typeId = room.resource_type_id ?? room.resource_type?.id
    const matchType = typeFilter.value === '' || Number(typeId) === Number(typeFilter.value)
    const matchName = !search || String(room.name || '').toLowerCase().includes(search)
    return matchType && matchName
  })
})

const emptyText = computed(() =>
  resources.value.length ? '没有符合筛选条件的房间' : '还没有房间，点击右上角新增房间'
)

const loadAll = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const [typesRes, resourcesRes, devicesRes, voiceDevicesRes] = await Promise.all([
      getResourceTypes(),
      getResources(),
      getDevices(),
      getVoiceDevices().catch(() => [])
    ])
    if (seq !== loadSeq) return
    resourceTypes.value = toList(typesRes) as ResourceTypeItem[]
    resources.value = toList(resourcesRes) as ResourceItem[]
    devices.value = toList(devicesRes) as DeviceItem[]
    voiceDevices.value = toList(voiceDevicesRes) as DeviceItem[]
  } catch {
    return
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const openCreateRoom = () => {
  if (!resourceTypes.value.length) {
    ElMessage.warning('请先新增房型')
    return
  }
  newRoomName.value = ''
  newRoomTypeId.value =
    typeFilter.value === '' ? Number(resourceTypes.value[0]?.id) : typeFilter.value
  createVisible.value = true
}

const handleCreateRoom = async () => {
  if (creating.value) return
  const name = newRoomName.value.trim()
  if (!newRoomTypeId.value) {
    ElMessage.warning('请选择房型')
    return
  }
  if (!name) {
    ElMessage.warning('请输入房间名')
    return
  }
  creating.value = true
  try {
    await createResource({ name, resource_type_id: Number(newRoomTypeId.value) })
    ElMessage.success('房间已添加')
    createVisible.value = false
    newRoomName.value = ''
    await loadAll()
  } catch {
    return
  } finally {
    creating.value = false
  }
}

const openBind = (room: ResourceItem) => {
  bindRoom.value = room
  bindSelection.value = roomDeviceIds(room)
  bindVisible.value = true
}

const occupiedRoomName = (deviceId: number) => {
  const currentId = Number(bindRoom.value?.id)
  const owner = resources.value.find(
    (room) => Number(room.id) !== currentId && roomDeviceIds(room).includes(deviceId)
  )
  return owner ? String(owner.name || `房间 #${owner.id}`) : ''
}

const handleSaveBind = async () => {
  if (linking.value || !bindRoom.value) return
  linking.value = true
  try {
    await linkDevices(Number(bindRoom.value.id), bindSelection.value)
    ElMessage.success('设备绑定已保存')
    bindVisible.value = false
    bindRoom.value = null
    bindSelection.value = []
    await loadAll()
  } catch {
    return
  } finally {
    linking.value = false
  }
}

const handleDelete = async (room: ResourceItem) => {
  if (deletingId.value !== null) return
  try {
    await ElMessageBox.confirm(
      `确认删除房间「${room.name}」？删除后不可恢复。`,
      '删除房间',
      {
        type: 'warning',
        confirmButtonText: '确认删除',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }
  deletingId.value = Number(room.id)
  try {
    await deleteResource(room.id)
    ElMessage.success('房间已删除')
    await loadAll()
  } catch {
    return
  } finally {
    deletingId.value = null
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    typeFilter.value = ''
    keyword.value = ''
    createVisible.value = false
    bindVisible.value = false
    bindRoom.value = null
    bindSelection.value = []
    if (!id) {
      resourceTypes.value = []
      resources.value = []
      devices.value = []
      return
    }
    loadAll()
  }
)

onMounted(() => {
  loadAll()
})
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.toolbar-left {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.type-filter {
  width: 160px;
}

.search-input {
  width: 220px;
}

.room-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.device-tags {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
  padding: 4px 0;
}

.device-tags :deep(.el-tag) {
  max-width: 100%;
}

.muted {
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

.bind-tip {
  margin-bottom: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 360px;
  padding-right: 6px;
  overflow-y: auto;
}

.device-item {
  width: 100%;
  height: auto;
  padding: 8px 10px;
  margin-right: 0;
  border-radius: 6px;
}

.device-item:hover {
  background: var(--el-bg-color-page);
}

.device-item :deep(.el-checkbox__label) {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  white-space: normal;
}

.device-name {
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.occupied-tag {
  margin-left: 4px;
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
  }

  .toolbar-left {
    flex: 1;
  }

  .type-filter,
  .search-input {
    width: 100%;
  }
}
</style>


