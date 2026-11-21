<template>
  <ContentWrap title="订单管理" :body-style="{ padding: '0 18px 18px' }">
    <el-tabs v-model="activeTab" class="order-tabs" @tab-change="handleTabChange">
      <el-tab-pane name="orders" label="订单">
        <div class="toolbar">
          <div class="toolbar-left">
            <el-select
              v-model="statusFilter"
              class="status-select"
              placeholder="全部状态"
              @change="handleStatusChange"
            >
              <el-option
                v-for="option in statusOptions"
                :key="String(option.value)"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </div>
          <el-button
            :icon="Refresh"
            :loading="ordersLoading || refundsLoading"
            @click="handleRefresh"
          >
            刷新
          </el-button>
        </div>

        <el-table v-loading="ordersLoading" :data="orders" row-key="id" empty-text="暂无订单">
          <el-table-column label="订单号" min-width="180">
            <template #default="{ row }">
              <span class="order-no">{{ row.order_no || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="房间" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">{{ row.resource_name || '-' }}</template>
          </el-table-column>
          <el-table-column label="使用时间" min-width="210">
            <template #default="{ row }">
              <span class="time-range">{{ orderTimeText(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="时长" width="100">
            <template #default="{ row }">{{ durationText(row) }}</template>
          </el-table-column>
          <el-table-column label="金额" width="120">
            <template #default="{ row }">
              <span class="amount">¥{{ fenToYuan(row.total_amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="getOrderStatus(row.status).type" effect="light">
                {{ getOrderStatus(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <div class="row-actions">
                <el-button
                  v-if="Number(row.status) === 0"
                  link
                  type="danger"
                  :loading="actionKey === `cancel-${row.id}`"
                  :disabled="!!actionKey"
                  @click="handleCancelOrder(row)"
                >
                  取消
                </el-button>
                <el-button
                  v-if="Number(row.status) === 1 || Number(row.status) === 2"
                  link
                  type="danger"
                  :loading="actionKey === `refund-${row.id}`"
                  :disabled="!!actionKey"
                  @click="handleRefundOrder(row)"
                >
                  退款
                </el-button>
                <el-button link type="primary" @click="openDetail(row)">详情</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div v-if="total > 0" class="pagination">
          <el-pagination
            v-model:current-page="page"
            :page-size="PAGE_SIZE"
            :total="total"
            background
            layout="total, prev, pager, next, jumper"
            @current-change="handlePageChange"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane name="refunds">
        <template #label>
          <span class="refund-tab-label">
            退款审核
            <span v-if="pendingRefundCount > 0" class="refund-count">
              {{ pendingRefundCount > 99 ? '99+' : pendingRefundCount }}
            </span>
          </span>
        </template>

        <el-alert
          class="refund-tip"
          type="warning"
          :closable="false"
          show-icon
          title="同意退款仅适用于已确认的余额支付，退款金额将退回用户余额。"
        />

        <el-table v-loading="refundsLoading" :data="refunds" row-key="id" empty-text="暂无退款申请">
          <el-table-column label="订单号" min-width="180">
            <template #default="{ row }">
              <span class="order-no">{{ refundOrderText(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="退款金额" width="120">
            <template #default="{ row }">
              <span class="amount">¥{{ fenToYuan(row.amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="原因" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">{{ row.reason || '-' }}</template>
          </el-table-column>
          <el-table-column label="申请时间" width="160">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="getRefundStatus(row.status).type" effect="light">
                {{ getRefundStatus(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="170" fixed="right">
            <template #default="{ row }">
              <div v-if="Number(row.status) === 0" class="row-actions">
                <el-button
                  link
                  type="success"
                  :loading="actionKey === `approve-${row.id}`"
                  :disabled="!!actionKey"
                  @click="handleApproveRefund(row)"
                >
                  同意
                </el-button>
                <el-button
                  link
                  type="danger"
                  :loading="actionKey === `reject-${row.id}`"
                  :disabled="!!actionKey"
                  @click="handleRejectRefund(row)"
                >
                  拒绝
                </el-button>
              </div>
              <el-tag v-else :type="getRefundStatus(row.status).type" effect="plain">
                {{ getRefundStatus(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog
      v-model="detailVisible"
      title="订单详情"
      width="560px"
      append-to-body
      style="max-width: 92vw"
    >
      <el-descriptions v-if="detailOrder" :column="1" border label-width="90px">
        <el-descriptions-item label="订单号">
          {{ detailOrder.order_no || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="房间">
          {{ detailOrder.resource_name || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="使用时间">
          {{ orderTimeText(detailOrder) }}
        </el-descriptions-item>
        <el-descriptions-item label="时长">
          {{ durationText(detailOrder) }}
        </el-descriptions-item>
        <el-descriptions-item label="金额">
          <span class="amount">¥{{ fenToYuan(detailOrder.total_amount) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getOrderStatus(detailOrder.status).type" effect="light">
            {{ getOrderStatus(detailOrder.status).text }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="下单时间">
          {{ formatDateTime(detailOrder.created_at) }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  approveRefund,
  getOrders,
  getRefundRequests,
  merchantCancelOrder,
  merchantDirectRefund,
  rejectRefund
} from '@/api/merchant'
import { fenToYuan, formatDateTime } from '@/utils/merchant'

defineOptions({ name: 'MerchantOrders' })

type TagType = 'primary' | 'success' | 'info' | 'warning' | 'danger'
type ActiveTab = 'orders' | 'refunds'

interface MerchantOrder {
  id: number
  order_no?: string
  resource_name?: string
  status?: number | string
  start_time?: number
  end_time?: number
  total_amount?: number
  duration?: number
  created_at?: number
  [key: string]: any
}

interface RefundRequest {
  id: number
  order_no?: string
  order_id?: number | string
  amount?: number
  reason?: string
  status?: number | string
  created_at?: number
  [key: string]: any
}

const PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()
const merchantStore = useMerchantStore()

const activeTab = ref<ActiveTab>(String(route.query.tab || '') === 'refunds' ? 'refunds' : 'orders')
const statusFilter = ref<number | ''>('')
const page = ref(1)
const total = ref(0)
const orders = ref<MerchantOrder[]>([])
const refunds = ref<RefundRequest[]>([])
const ordersLoading = ref(false)
const refundsLoading = ref(false)
const actionKey = ref('')
const detailVisible = ref(false)
const detailOrder = ref<MerchantOrder | null>(null)

const statusOptions = [
  { label: '全部', value: '' },
  { label: '待支付', value: 0 },
  { label: '已支付', value: 1 },
  { label: '进行中', value: 2 },
  { label: '已完成', value: 3 },
  { label: '已退款', value: 6 }
]

const ORDER_STATUS: Record<number, { text: string; type: TagType }> = {
  0: { text: '待支付', type: 'warning' },
  1: { text: '已支付', type: 'primary' },
  2: { text: '进行中', type: 'success' },
  3: { text: '已完成', type: 'success' },
  4: { text: '已取消', type: 'info' },
  5: { text: '退款中', type: 'warning' },
  6: { text: '已退款', type: 'danger' }
}

const REFUND_STATUS: Record<number, { text: string; type: TagType }> = {
  0: { text: '待审核', type: 'warning' },
  1: { text: '已退回余额', type: 'success' },
  2: { text: '已拒绝', type: 'info' }
}

const UNKNOWN_STATUS = { text: '未知', type: 'info' as TagType }

const getOrderStatus = (status: any) => ORDER_STATUS[Number(status)] || UNKNOWN_STATUS
const getRefundStatus = (status: any) => REFUND_STATUS[Number(status)] || UNKNOWN_STATUS

const pendingRefundCount = computed(
  () => refunds.value.filter((refund) => Number(refund.status) === 0).length
)

const asList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const orderTimeText = (order: MerchantOrder) => {
  if (!order.start_time && !order.end_time) return '-'
  const start = order.start_time ? formatDateTime(order.start_time) : '-'
  const end = order.end_time ? formatDateTime(order.end_time) : '-'
  return `${start} - ${end}`
}

const durationText = (order: MerchantOrder) => {
  const duration = Number(order.duration)
  return Number.isFinite(duration) && duration > 0 ? `${duration} 分钟` : '-'
}

const refundOrderText = (refund: RefundRequest) => {
  if (refund.order_no) return `#${refund.order_no}`
  if (refund.order_id !== undefined && refund.order_id !== null) return `#${refund.order_id}`
  return '-'
}

let ordersSeq = 0
let refundsSeq = 0

const loadOrders = async () => {
  const seq = ++ordersSeq
  ordersLoading.value = true
  try {
    const params: Record<string, any> = { page: page.value, size: PAGE_SIZE }
    if (statusFilter.value !== '') params.status = statusFilter.value
    const res: any = await getOrders(params)
    if (seq !== ordersSeq) return
    const list = asList(res)
    orders.value = list as MerchantOrder[]
    const rawTotal = Number(res?.total)
    total.value = Number.isFinite(rawTotal) ? rawTotal : list.length
    const rawPage = Number(res?.page)
    if (Number.isFinite(rawPage) && rawPage > 0) page.value = rawPage
  } catch {
    return
  } finally {
    if (seq === ordersSeq) ordersLoading.value = false
  }
}

const loadRefunds = async () => {
  const seq = ++refundsSeq
  refundsLoading.value = true
  try {
    const res: any = await getRefundRequests()
    if (seq !== refundsSeq) return
    refunds.value = asList(res) as RefundRequest[]
  } catch {
    return
  } finally {
    if (seq === refundsSeq) refundsLoading.value = false
  }
}

const loadAll = () => Promise.all([loadOrders(), loadRefunds()])

const reloadOrders = async () => {
  const requestedPage = page.value
  await loadOrders()
  if (requestedPage > 1 && orders.value.length === 0 && total.value > 0) {
    page.value = Math.min(requestedPage, Math.ceil(total.value / PAGE_SIZE))
    if (page.value !== requestedPage) await loadOrders()
  }
}

const handleStatusChange = () => {
  page.value = 1
  loadOrders()
}

const handlePageChange = (nextPage: number) => {
  page.value = nextPage
  loadOrders()
}

const handleRefresh = () => loadAll()

const openDetail = (order: MerchantOrder) => {
  detailOrder.value = order
  detailVisible.value = true
}

const handleTabChange = (name: string | number) => {
  const nextTab: ActiveTab = name === 'refunds' ? 'refunds' : 'orders'
  const currentTab: ActiveTab = route.query.tab === 'refunds' ? 'refunds' : 'orders'
  if (nextTab === currentTab) return
  router.replace({
    query: {
      ...route.query,
      tab: nextTab === 'refunds' ? 'refunds' : undefined
    }
  })
}

const runAction = async (
  key: string,
  message: string,
  title: string,
  confirmButtonText: string,
  action: () => Promise<any>,
  successMessage: string
) => {
  if (actionKey.value) return
  actionKey.value = key
  try {
    await ElMessageBox.confirm(message, title, {
      type: 'warning',
      confirmButtonText,
      cancelButtonText: '取消'
    })
    await action()
    ElMessage.success(successMessage)
    await Promise.all([reloadOrders(), loadRefunds()])
  } catch {
    return
  } finally {
    actionKey.value = ''
  }
}

const handleCancelOrder = (order: MerchantOrder) =>
  runAction(
    `cancel-${order.id}`,
    '确认取消该待支付订单？取消后不可恢复。',
    '取消订单',
    '确认取消',
    () => merchantCancelOrder(order.id),
    '订单已取消'
  )

const handleRefundOrder = (order: MerchantOrder) =>
  runAction(
    `refund-${order.id}`,
    '仅余额支付订单支持自动退回余额。确认后订单关闭且不可恢复。',
    '确认退款',
    '确认退款',
    () => merchantDirectRefund(order.id),
    '退款已处理，金额已退回用户余额'
  )

const handleApproveRefund = (refund: RefundRequest) =>
  runAction(
    `approve-${refund.id}`,
    '仅余额支付订单会自动退回余额，其他支付方式暂不支持自动退款。',
    '确认同意退款',
    '同意退款',
    () => approveRefund(refund.id),
    '退款已同意，金额已退回用户余额'
  )

const handleRejectRefund = (refund: RefundRequest) =>
  runAction(
    `reject-${refund.id}`,
    '确认拒绝该退款申请？',
    '拒绝退款',
    '确认拒绝',
    () => rejectRefund(refund.id),
    '已拒绝该退款申请'
  )

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = String(tab || '') === 'refunds' ? 'refunds' : 'orders'
  }
)

watch(
  () => merchantStore.storeId,
  (id) => {
    page.value = 1
    statusFilter.value = ''
    detailVisible.value = false
    if (!id) {
      orders.value = []
      refunds.value = []
      total.value = 0
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
.order-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 16px;
  }
}

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
  gap: 12px;
  align-items: center;
}

.status-select {
  width: 150px;
}

.order-no {
  font-family: 'JetBrains Mono', Consolas, monospace;
  color: var(--el-text-color-primary);
}

.time-range {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.amount {
  font-weight: 600;
  color: #e11d48;
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  overflow-x: auto;
}

.refund-tab-label {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.refund-count {
  display: inline-flex;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 12px;
  line-height: 18px;
  color: #fff;
  text-align: center;
  background: #f56c6c;
  border-radius: 9px;
}

.refund-tip {
  margin-bottom: 14px;
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
  }

  .toolbar-left {
    flex: 1;
  }

  .status-select {
    width: 100%;
  }

  .pagination {
    justify-content: flex-start;
  }
}
</style>
