<template>
  <ContentWrap title="房间时段" :body-style="{ padding: '0 18px 18px' }">
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button :icon="ArrowLeft" @click="shiftDay(-1)">上一天</el-button>
        <el-date-picker
          v-model="selectedDate"
          class="date-picker"
          type="date"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          :clearable="false"
          placeholder="选择日期"
          @change="handleDateChange"
        />
        <el-button :icon="ArrowRight" @click="shiftDay(1)">下一天</el-button>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
    </div>

    <div v-loading="loading" class="timeline-body">
      <div v-if="timelineRows.length" class="timeline-wrap">
        <div class="timeline-scroll">
          <div class="timeline-grid" :style="gridStyle">
            <div class="grid-cell corner">房间</div>
            <div v-for="hour in HOURS" :key="hour" class="grid-cell hour-head">
              {{ hour }}
            </div>

            <template v-for="row in timelineRows" :key="row.id">
              <div class="grid-cell room-cell" :title="row.name">{{ row.name }}</div>
              <div
                v-for="(cell, index) in row.cells"
                :key="`${row.id}-${index}`"
                class="grid-cell hour-cell"
              >
                <el-tooltip placement="top" effect="dark" :show-after="120">
                  <template #content>
                    <div v-if="cell.orders.length" class="cell-tooltip">
                      <div
                        v-for="order in cell.orders"
                        :key="order.id"
                        class="tooltip-order"
                      >
                        <div class="tooltip-room">{{ row.name }}</div>
                        <div>
                          {{ formatDateTime(order.start_time) }} -
                          {{ formatDateTime(order.end_time) }}
                        </div>
                        <div>状态：{{ statusText(order.status) }}</div>
                      </div>
                    </div>
                    <div v-else>{{ row.name }} · {{ hourRangeText(index) }} · 空闲</div>
                  </template>
                  <div class="cell-inner" :class="{ booked: cell.orders.length > 0 }"></div>
                </el-tooltip>
              </div>
            </template>
          </div>
        </div>

        <div class="legend">
          <span class="legend-item">
            <i class="legend-block booked"></i>
            已预订
          </span>
          <span class="legend-item">
            <i class="legend-block free"></i>
            空闲
          </span>
          <span class="legend-summary">
            {{ selectedDate }} · 共 {{ timelineRows.length }} 间房
          </span>
        </div>
      </div>

      <el-empty
        v-else-if="!loading"
        description="当前门店暂无房间，请先在房间列表中添加房间"
        :image-size="100"
      />
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight, Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { getAllOrders, getResources } from '@/api/merchant'
import { formatDateTime } from '@/utils/merchant'

defineOptions({ name: 'MerchantTimeline' })

interface ResourceItem {
  id: number
  name: string
  [key: string]: any
}

interface OrderItem {
  id: number
  resource_id?: number | string
  status?: number | string
  start_time?: number
  end_time?: number
  [key: string]: any
}

interface TimelineCell {
  orders: OrderItem[]
}

const HOURS = Array.from({ length: 24 }, (_, hour) => String(hour).padStart(2, '0'))
const ACTIVE_STATUS = new Set([0, 1, 2, 3, 5])
const ORDER_STATUS: Record<number, string> = {
  0: '待支付',
  1: '已支付',
  2: '进行中',
  3: '已完成',
  4: '已取消',
  5: '退款中',
  6: '已退款'
}

const merchantStore = useMerchantStore()

const resources = ref<ResourceItem[]>([])
const orders = ref<OrderItem[]>([])
const loading = ref(false)
const selectedDate = ref(formatDateValue(new Date()))

let loadSeq = 0

function pad(value: number) {
  return value < 10 ? `0${value}` : String(value)
}

function formatDateValue(date: Date) {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function parseDateValue(value: string) {
  const [year, month, day] = String(value || '').split('-').map(Number)
  if (!year || !month || !day) return new Date()
  return new Date(year, month - 1, day)
}

function toList(res: any): any[] {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const hourRanges = computed(() => {
  const date = parseDateValue(selectedDate.value)
  const year = date.getFullYear()
  const month = date.getMonth()
  const day = date.getDate()
  return Array.from({ length: 24 }, (_, hour) => ({
    start: new Date(year, month, day, hour).getTime(),
    end: new Date(year, month, day, hour + 1).getTime()
  }))
})

const timelineRows = computed(() => {
  const ranges = hourRanges.value
  return resources.value.map((room) => {
    const roomId = Number(room.id)
    const roomOrders = orders.value.filter((order) => {
      if (Number(order.resource_id) !== roomId) return false
      if (!ACTIVE_STATUS.has(Number(order.status))) return false
      return Number.isFinite(Number(order.start_time)) && Number.isFinite(Number(order.end_time))
    })
    const cells: TimelineCell[] = ranges.map((range) => ({
      orders: roomOrders.filter((order) => {
        const start = Number(order.start_time)
        const end = Number(order.end_time)
        return start < range.end && end > range.start
      })
    }))
    return {
      id: room.id,
      name: room.name || `房间 #${room.id}`,
      cells
    }
  })
})

const gridStyle = computed(() => ({
  gridTemplateColumns: '140px repeat(24, minmax(42px, 1fr))',
  minWidth: '1148px'
}))

const statusText = (status: any) => ORDER_STATUS[Number(status)] || '未知'

const hourRangeText = (index: number) => `${pad(index)}:00 - ${pad(index + 1)}:00`

const loadAll = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const [resourcesRes, ordersRes] = await Promise.all([getResources(), getAllOrders()])
    if (seq !== loadSeq) return
    resources.value = toList(resourcesRes) as ResourceItem[]
    orders.value = (Array.isArray(ordersRes) ? ordersRes : toList(ordersRes)) as OrderItem[]
  } catch {
    return
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const shiftDay = (offset: number) => {
  const date = parseDateValue(selectedDate.value)
  date.setDate(date.getDate() + offset)
  selectedDate.value = formatDateValue(date)
}

const handleDateChange = (value: string | null) => {
  if (!value) selectedDate.value = formatDateValue(new Date())
}

watch(
  () => merchantStore.storeId,
  (id) => {
    if (!id) {
      resources.value = []
      orders.value = []
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
  gap: 10px;
  align-items: center;
}

.date-picker {
  width: 160px;
}

.timeline-body {
  min-height: 160px;
}

.timeline-wrap {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.timeline-scroll {
  overflow-x: auto;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}

.timeline-grid {
  display: grid;
  grid-auto-rows: 40px;
  min-height: 40px;
}

.grid-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid var(--el-border-color-light);
  border-bottom: 1px solid var(--el-border-color-light);
}

.hour-head {
  font-size: 12px;
  font-weight: 600;
  color: #909399;
  background: var(--el-bg-color-page);
}

.corner {
  position: sticky;
  left: 0;
  z-index: 3;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 600;
  color: #606266;
  background: var(--el-bg-color-page);
  justify-content: flex-start;
}

.room-cell {
  position: sticky;
  left: 0;
  z-index: 2;
  justify-content: flex-start;
  padding: 0 12px;
  overflow: hidden;
  font-size: 13px;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
  background: var(--el-bg-color);
  box-shadow: 1px 0 0 #ebeef5;
}

.hour-cell {
  padding: 3px;
}

.cell-inner {
  width: 100%;
  height: 100%;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  transition: background-color 0.15s ease;
}

.cell-inner.booked {
  background: var(--el-color-primary, #409eff);
}

.cell-inner.booked:hover {
  filter: brightness(0.92);
}

.cell-tooltip {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 260px;
}

.tooltip-order + .tooltip-order {
  padding-top: 8px;
  border-top: 1px solid rgb(255 255 255 / 20%);
}

.tooltip-room {
  font-weight: 600;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  align-items: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.legend-item {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.legend-block {
  display: inline-block;
  width: 14px;
  height: 14px;
  border-radius: 3px;
}

.legend-block.booked {
  background: var(--el-color-primary, #409eff);
}

.legend-block.free {
  background: var(--el-fill-color-light);
  border: 1px solid #dcdfe6;
}

.legend-summary {
  margin-left: auto;
  color: var(--el-text-color-placeholder);
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
  }

  .toolbar-left {
    flex: 1;
  }

  .date-picker {
    flex: 1;
    width: 100%;
  }

  .legend-summary {
    width: 100%;
    margin-left: 0;
  }
}
</style>
