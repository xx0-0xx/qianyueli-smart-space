<template>
  <div v-loading="loading" element-loading-text="加载中..." class="merchant-dashboard">
    <div class="page-head">
      <div class="page-head__copy">
        <div class="page-title">经营看板</div>
        <div class="page-sub">
          {{ merchantStore.getStoreName || '当前门店' }} · {{ todayText }}
        </div>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadData">刷新</el-button>
    </div>

    <div class="stat-grid">
      <div
        v-for="item in stats"
        :key="item.label"
        class="stat-card"
        :class="{ 'is-clickable': !!item.path }"
        @click="onStatClick(item)"
      >
        <div class="stat-icon" :style="{ background: item.bg, color: item.color }">
          <el-icon :size="22">
            <component :is="item.icon" />
          </el-icon>
        </div>
        <div class="stat-copy">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-value">{{ item.value }}</div>
          <div class="stat-sub">{{ item.sub }}</div>
        </div>
      </div>
    </div>

    <div class="panel-grid">
      <ContentWrap title="待处理" :body-style="panelBodyStyle" class="!mb-0">
        <div class="todo-list">
          <div
            v-for="item in todoItems"
            :key="item.key"
            class="todo-item"
            @click="router.push(item.path)"
          >
            <div class="todo-icon" :style="{ background: item.bg, color: item.color }">
              <el-icon :size="18">
                <component :is="item.icon" />
              </el-icon>
            </div>
            <div class="todo-copy">
              <div class="todo-title">{{ item.title }}</div>
              <div class="todo-desc">{{ item.desc }}</div>
            </div>
            <el-tag
              round
              effect="light"
              :type="item.count > 0 ? item.tagType : 'info'"
              class="todo-count"
            >
              {{ item.count }}
            </el-tag>
            <el-icon class="todo-arrow"><ArrowRight /></el-icon>
          </div>
        </div>
      </ContentWrap>

      <ContentWrap title="快捷入口" :body-style="panelBodyStyle" class="!mb-0">
        <div class="quick-grid">
          <div
            v-for="item in quickEntries"
            :key="item.title"
            class="quick-item"
            @click="router.push(item.path)"
          >
            <div class="quick-icon" :style="{ background: item.bg, color: item.color }">
              <el-icon :size="19">
                <component :is="item.icon" />
              </el-icon>
            </div>
            <div>
              <div class="quick-title">{{ item.title }}</div>
              <div class="quick-desc">{{ item.desc }}</div>
            </div>
          </div>
        </div>
      </ContentWrap>
    </div>

    <ContentWrap title="房间状态" :body-style="panelBodyStyle">
      <template #header>
        <div class="room-summary">
          {{ activeRoomCount }} 间使用中 / 共 {{ roomViews.length }} 间
        </div>
      </template>

      <div class="room-toolbar">
        <el-input
          v-model="keyword"
          class="room-search"
          placeholder="搜索房间或房型"
          clearable
          :prefix-icon="Search"
        />
        <el-radio-group v-model="roomFilter" size="small">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="active">使用中</el-radio-button>
          <el-radio-button value="free">空闲</el-radio-button>
        </el-radio-group>
      </div>

      <div v-if="filteredRooms.length" class="room-grid">
        <div
          v-for="room in filteredRooms"
          :key="room.id"
          class="room-card"
          :class="{ 'is-expanded': isExpanded(room.id), 'is-active': room.active }"
        >
          <div class="room-main" @click="toggleRoom(room.id)">
            <span class="room-dot" :class="room.active ? 'is-busy' : 'is-free'"></span>
            <div class="room-meta">
              <div class="room-name">{{ room.name }}</div>
              <div class="room-type">{{ room.typeName }}</div>
            </div>
            <div class="room-state">
              <el-tag :type="room.active ? 'success' : 'info'" size="small" effect="light">
                {{ room.active ? '使用中' : '空闲' }}
              </el-tag>
              <span v-if="room.active" class="room-remain">剩余 {{ room.remainText }}</span>
              <span v-if="room.active" class="room-end">{{ room.endText }} 结束</span>
            </div>
            <el-icon class="room-chevron"><ArrowDown /></el-icon>
          </div>

          <div v-if="isExpanded(room.id)" class="device-panel" @click.stop>
            <div class="device-panel__head">
              <span>设备控制</span>
              <span class="device-total">{{ room.devices.length }} 台</span>
            </div>
            <el-empty
              v-if="room.devices.length === 0"
              description="未绑定设备"
              :image-size="56"
            />
            <div v-else class="device-list">
              <div v-for="dev in room.devices" :key="dev.id" class="device-row">
                <div class="device-info">
                  <span class="device-name">{{ dev.name }}</span>
                  <el-tag size="small" effect="plain">{{ deviceTypeLabel(dev.type) }}</el-tag>
                  <el-tag
                    size="small"
                    effect="light"
                    :type="isOnline(dev.status) ? 'success' : 'info'"
                  >
                    {{ isOnline(dev.status) ? '在线' : '离线' }}
                  </el-tag>
                </div>
                <div v-if="dev.buttons.length" class="device-actions">
                  <el-button
                    v-for="(btn, index) in dev.buttons"
                    :key="btn.cmd"
                    size="small"
                    plain
                    :type="index === 0 ? 'primary' : 'default'"
                    :loading="isSending(dev.id, btn.cmd)"
                    @click.stop="sendCommand(dev.id, btn.cmd)"
                  >
                    {{ btn.label }}
                  </el-button>
                </div>
                <span v-else class="device-empty-cmd">未配置控制命令</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <el-empty
        v-else
        :description="roomViews.length ? '没有匹配的房间' : '暂无房间数据'"
      />
    </ContentWrap>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowDown,
  ArrowRight,
  Calendar,
  Clock,
  Cpu,
  DataAnalysis,
  Money,
  OfficeBuilding,
  Refresh,
  RefreshLeft,
  Search,
  Tickets,
  Wallet
} from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  controlDevice,
  getAllOrders,
  getDevices,
  getRecharges,
  getRefundRequests,
  getResources
} from '@/api/merchant'
import { fenToYuan, formatDateTime, formatRemain, parseJson } from '@/utils/merchant'
import { buildDeviceButtons, deviceTypeLabel } from '@/utils/deviceCommands'
import type { DeviceButton } from '@/utils/deviceCommands'

defineOptions({ name: 'MerchantDashboard' })

interface RoomDevice {
  id: number
  name: string
  type: string
  status: any
  buttons: DeviceButton[]
}

interface RoomView {
  id: number | string
  name: string
  typeName: string
  active: boolean
  remainText: string
  endText: string
  devices: RoomDevice[]
}

const router = useRouter()
const merchantStore = useMerchantStore()

const loading = ref(false)
const nowTs = ref(Date.now())
const resources = ref<any[]>([])
const orders = ref<any[]>([])
const devices = ref<any[]>([])
const recharges = ref<any[]>([])
const refunds = ref<any[]>([])

const keyword = ref('')
const roomFilter = ref<'all' | 'active' | 'free'>('all')
const expandedId = ref('')
const sendingKeys = ref<string[]>([])

const panelBodyStyle = { padding: '16px' }

let loadSeq = 0
let refreshTimer: number | undefined

const asList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const parseDeviceIds = (raw: any): number[] => {
  const value = Array.isArray(raw) ? raw : parseJson<any[]>(raw, [])
  return (Array.isArray(value) ? value : [])
    .map((id) => Number(id))
    .filter((id) => Number.isFinite(id))
}

const isOnline = (status: any) => Number(status) === 1 || status === 'online'

const dayRange = computed(() => {
  const now = new Date(nowTs.value)
  const start = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  return { start, end: start + 86400000 }
})

const todayText = computed(() => {
  const d = new Date(nowTs.value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
})

const deviceMap = computed(() => {
  const map = new Map<number, any>()
  devices.value.forEach((dev) => map.set(Number(dev.id), dev))
  return map
})

const roomViews = computed<RoomView[]>(() => {
  const now = nowTs.value
  const activeOrderMap = new Map<string, any>()
  orders.value.forEach((order) => {
    if (![1, 2].includes(Number(order.status))) return
    if (!(Number(order.start_time) <= now && Number(order.end_time) > now)) return
    const key = String(order.resource_id)
    if (!activeOrderMap.has(key)) activeOrderMap.set(key, order)
  })

  return resources.value.map((resource) => {
    const activeOrder = activeOrderMap.get(String(resource.id)) || null
    const roomDevices = parseDeviceIds(resource.device_ids)
      .map((id) => deviceMap.value.get(id))
      .filter(Boolean)
      .map((dev: any) => ({
        id: Number(dev.id),
        name: dev.name || '未命名设备',
        type: dev.type || '',
        status: dev.status,
        buttons: buildDeviceButtons(dev)
      }))

    return {
      id: resource.id,
      name: resource.name || `房间 #${resource.id}`,
      typeName: resource.resource_type?.name || '未分类',
      active: !!activeOrder,
      remainText: activeOrder ? formatRemain(activeOrder.end_time) : '',
      endText: activeOrder ? formatDateTime(activeOrder.end_time) : '',
      devices: roomDevices
    }
  })
})

const todayOrders = computed(() => {
  const { start, end } = dayRange.value
  return orders.value.filter((order) => {
    return Number(order.start_time) < end && Number(order.end_time) > start
  })
})

const revenueFen = computed(() =>
  todayOrders.value.reduce((sum, order) => {
    return [1, 2, 3].includes(Number(order.status))
      ? sum + Number(order.total_amount || 0)
      : sum
  }, 0)
)

const todayOrderCount = computed(() => todayOrders.value.length)
const activeRoomCount = computed(() => roomViews.value.filter((room) => room.active).length)
const pendingOrderCount = computed(
  () => orders.value.filter((order) => Number(order.status) === 0).length
)
const pendingRefundCount = computed(
  () => refunds.value.filter((refund) => Number(refund.status) === 0).length
)
const rechargeCount = computed(() => recharges.value.length)

const stats = computed(() => [
  {
    label: '今日营收',
    value: `¥${fenToYuan(revenueFen.value)}`,
    sub: `今日 ${todayOrderCount.value} 笔订单`,
    icon: Wallet,
    bg: '#e9f8ef',
    color: '#16a34a',
    path: ''
  },
  {
    label: '今日订单',
    value: todayOrderCount.value,
    sub: '按订单时间统计',
    icon: Tickets,
    bg: '#e8f3ff',
    color: '#1a56db',
    path: '/orders'
  },
  {
    label: '使用中房间',
    value: `${activeRoomCount.value} / ${roomViews.value.length}`,
    sub: '订单进行中的房间',
    icon: OfficeBuilding,
    bg: '#fff7e6',
    color: '#f59e0b',
    path: ''
  },
  {
    label: '待支付订单',
    value: pendingOrderCount.value,
    sub: '需确认或跟进',
    icon: Clock,
    bg: '#fdecef',
    color: '#e11d48',
    path: '/orders'
  }
])

const todoItems = computed(() => [
  {
    key: 'orders',
    title: '待支付订单',
    desc: '需确认或跟进',
    count: pendingOrderCount.value,
    icon: Tickets,
    bg: '#fff7e6',
    color: '#f59e0b',
    tagType: 'warning' as const,
    path: '/orders'
  },
  {
    key: 'refunds',
    title: '退款申请',
    desc: '等待审核处理',
    count: pendingRefundCount.value,
    icon: RefreshLeft,
    bg: '#fdecef',
    color: '#e11d48',
    tagType: 'danger' as const,
    path: '/orders?tab=refunds'
  },
  {
    key: 'recharges',
    title: '充值确认',
    desc: '线下收款待确认',
    count: rechargeCount.value,
    icon: Money,
    bg: '#e8f3ff',
    color: '#1a56db',
    tagType: 'primary' as const,
    path: '/finance/recharges'
  }
])

const quickEntries = [
  {
    title: '房间管理',
    desc: '房间与设备绑定',
    icon: OfficeBuilding,
    bg: '#e8f3ff',
    color: '#1a56db',
    path: '/rooms/resources'
  },
  {
    title: '设备管理',
    desc: '状态与控制',
    icon: Cpu,
    bg: '#f3e8ff',
    color: '#7c3aed',
    path: '/devices'
  },
  {
    title: '房间时段',
    desc: '查看预约占用',
    icon: Calendar,
    bg: '#e0f7fa',
    color: '#0891b2',
    path: '/rooms/timeline'
  },
  {
    title: '月度报表',
    desc: '月度订单与营收',
    icon: DataAnalysis,
    bg: '#fff7e6',
    color: '#f59e0b',
    path: '/finance/report'
  }
]

const filteredRooms = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return roomViews.value.filter((room) => {
    const matchStatus =
      roomFilter.value === 'all' ||
      (roomFilter.value === 'active' ? room.active : !room.active)
    const matchKeyword =
      !q ||
      room.name.toLowerCase().includes(q) ||
      room.typeName.toLowerCase().includes(q)
    return matchStatus && matchKeyword
  })
})

const isExpanded = (id: number | string) => expandedId.value === String(id)
const isSending = (id: number, cmd: string) => sendingKeys.value.includes(`${id}:${cmd}`)

const toggleRoom = (id: number | string) => {
  const key = String(id)
  expandedId.value = expandedId.value === key ? '' : key
}

const onStatClick = (item: { path: string }) => {
  if (item.path) router.push(item.path)
}

const sendCommand = async (deviceId: number, cmd: string) => {
  const key = `${deviceId}:${cmd}`
  if (sendingKeys.value.includes(key)) return
  sendingKeys.value = [...sendingKeys.value, key]
  try {
    await controlDevice(deviceId, cmd)
    ElMessage.success('指令已发送')
  } catch {
    // 请求层已统一提示错误
  } finally {
    sendingKeys.value = sendingKeys.value.filter((item) => item !== key)
  }
}

const loadData = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const [resourceRes, orderRes, deviceRes, rechargeRes, refundRes] = await Promise.all([
      getResources(),
      getAllOrders(),
      getDevices(),
      getRecharges('0'),
      getRefundRequests()
    ])
    if (seq !== loadSeq) return
    resources.value = asList(resourceRes)
    orders.value = asList(orderRes)
    devices.value = asList(deviceRes)
    recharges.value = asList(rechargeRes)
    refunds.value = asList(refundRes)
  } catch {
    // 请求层已统一提示错误，避免重复弹窗
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    expandedId.value = ''
    keyword.value = ''
    roomFilter.value = 'all'
    if (id) loadData()
  }
)

onMounted(() => {
  loadData()
  refreshTimer = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>

<style lang="scss" scoped>
.merchant-dashboard {
  min-height: 100%;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 15px;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.page-sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 18px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  transition: box-shadow 0.2s, border-color 0.2s;
}

.stat-card.is-clickable {
  cursor: pointer;
}

.stat-card.is-clickable:hover {
  border-color: #c7d7fe;
  box-shadow: 0 6px 18px rgb(26 86 219 / 8%);
}

.stat-icon {
  display: flex;
  flex: 0 0 46px;
  width: 46px;
  height: 46px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}

.stat-copy {
  min-width: 0;
}

.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.stat-value {
  margin-top: 3px;
  font-size: 23px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--el-text-color-primary);
}

.stat-sub {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
  gap: 16px;
  margin-bottom: 15px;
}

.todo-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.todo-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px;
  cursor: pointer;
  border-radius: 10px;
  transition: background 0.2s;
}

.todo-item:hover {
  background: var(--el-fill-color-light);
}

.todo-icon {
  display: flex;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
}

.todo-copy {
  flex: 1;
  min-width: 0;
}

.todo-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.todo-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.todo-count {
  min-width: 28px;
  justify-content: center;
}

.todo-arrow {
  color: var(--el-text-color-placeholder);
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.quick-item {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px;
  cursor: pointer;
  border-radius: 10px;
  transition: background 0.2s;
}

.quick-item:hover {
  background: var(--el-fill-color-light);
}

.quick-icon {
  display: flex;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
}

.quick-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.quick-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.room-summary {
  width: 100%;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  text-align: right;
}

.room-toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.room-search {
  width: 260px;
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 12px;
}

.room-card {
  overflow: hidden;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.room-card:hover {
  border-color: #c7d7fe;
  box-shadow: 0 4px 14px rgb(15 23 42 / 6%);
}

.room-card.is-expanded {
  border-color: #93b4fb;
}

.room-main {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 14px;
  cursor: pointer;
}

.room-dot {
  flex: 0 0 9px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
}

.room-dot.is-busy {
  background: #16a34a;
  box-shadow: 0 0 0 4px rgb(22 163 74 / 12%);
}

.room-dot.is-free {
  background: #cbd5e1;
}

.room-meta {
  flex: 1;
  min-width: 0;
}

.room-name {
  overflow: hidden;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-type {
  margin-top: 2px;
  overflow: hidden;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-state {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  align-items: flex-end;
  gap: 3px;
}

.room-remain {
  font-size: 12px;
  font-weight: 600;
  color: #16a34a;
}

.room-end {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.room-chevron {
  color: var(--el-text-color-placeholder);
  transition: transform 0.2s;
}

.room-card.is-expanded .room-chevron {
  transform: rotate(180deg);
}

.device-panel {
  padding: 0 14px 14px;
  border-top: 1px dashed var(--el-border-color-light);
}

.device-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.device-total {
  font-weight: 400;
  color: var(--el-text-color-placeholder);
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.device-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.device-info {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.device-name {
  margin-right: 2px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.device-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.device-actions :deep(.el-button) {
  margin-left: 0;
}

.device-empty-cmd {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.merchant-dashboard :deep(.el-empty) {
  padding: 18px 0;
}

@media (max-width: 1100px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .panel-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .page-head {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }

  .stat-grid {
    grid-template-columns: 1fr;
  }

  .room-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .room-search {
    width: 100%;
  }

  .quick-grid {
    grid-template-columns: 1fr;
  }

  .room-grid {
    grid-template-columns: 1fr;
  }
}
</style>
