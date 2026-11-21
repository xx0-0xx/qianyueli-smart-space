<template>
  <ContentWrap title="客户管理" :body-style="{ padding: '0 18px 18px' }">
    <div class="toolbar">
      <div class="search">
        <el-input
          v-model="keyword"
          class="search-input"
          placeholder="搜索昵称 / 手机号"
          clearable
          @keyup.enter="load(1)"
          @clear="load(1)"
        />
        <el-button type="primary" :icon="Search" @click="load(1)">搜索</el-button>
        <el-button :icon="Refresh" :loading="loading" @click="refresh">刷新</el-button>
      </div>
      <span class="summary">共 {{ total }} 位客户</span>
    </div>

    <el-alert
      v-if="loadError"
      class="load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    />

    <el-table v-loading="loading" :data="rows" row-key="id" empty-text="暂无客户">
      <el-table-column label="客户" min-width="200">
        <template #default="{ row }">
          <div class="customer">
            <el-avatar v-if="row.avatar" :size="34" :src="row.avatar" />
            <div class="customer-info">
              <div class="customer-name">{{ row.nickname || '未设置昵称' }}</div>
              <div class="muted">{{ row.phone || '未绑定手机号' }}</div>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="订单数" width="90">
        <template #default="{ row }">{{ Number(row.order_count || 0) }}</template>
      </el-table-column>
      <el-table-column label="净消费" width="120">
        <template #default="{ row }">
          <span class="amount">¥{{ fenToYuan(row.total_spent) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="最近消费" width="170">
        <template #default="{ row }">{{ formatDateTime(row.last_order_at) }}</template>
      </el-table-column>
      <el-table-column label="会员等级" min-width="150">
        <template #default="{ row }">
          <template v-if="row.level_id && Number(row.member_status) === 1">
            <el-tag v-if="levelInactive(row)" type="info" effect="light">等级已停用</el-tag>
            <el-tag v-else-if="memberIsExpired(row)" type="info" effect="light">
              {{ row.level_name || '会员' }}（已过期）
            </el-tag>
            <el-tag v-else type="success" effect="light">
              {{ row.level_name || '会员' }}
            </el-tag>
          </template>
          <span v-else class="muted">普通用户</span>
        </template>
      </el-table-column>
      <el-table-column label="到期日" width="120">
        <template #default="{ row }">{{ expiryText(row) }}</template>
      </el-table-column>
      <el-table-column label="备注" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.remark || '-' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button link type="primary" @click="openDetail(row)">查看</el-button>
            <el-tooltip
              v-if="!String(row.phone || '').trim()"
              content="客户未绑定手机号，请先在用户端授权"
              placement="top"
            >
              <span class="tooltip-wrap">
                <el-button link type="primary" disabled>设置会员</el-button>
              </span>
            </el-tooltip>
            <el-button v-else link type="primary" @click="openSet(row)">设置会员</el-button>
            <el-button
              v-if="row.level_id && Number(row.member_status) === 1"
              link
              type="danger"
              :loading="removingId === Number(row.id)"
              :disabled="removingId !== null"
              @click="remove(row)"
            >
              取消
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        background
        @current-change="handleCurrentChange"
        @size-change="handleSizeChange"
      />
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" title="设置会员" width="480px" :close-on-click-modal="false">
    <div v-if="target" class="dialog-customer">
      <span class="dialog-name">{{ target.nickname || '未设置昵称' }}</span>
      <span class="muted">{{ target.phone || '未绑定手机号' }}</span>
    </div>
    <el-form label-width="90px">
      <el-form-item label="会员等级" required>
        <el-select v-model="form.levelId" class="full" placeholder="请选择会员等级">
          <el-option
            v-for="level in activeLevels"
            :key="level.id"
            :label="`${level.name}（${formatDiscount(level.discount_bp)} 折）`"
            :value="Number(level.id)"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="到期日">
        <el-date-picker
          v-model="form.expiresAt"
          class="full"
          type="date"
          value-format="x"
          placeholder="留空表示永久有效"
          clearable
        />
        <div class="form-hint">留空表示永久有效；过期后会员自动失效。</div>
      </el-form-item>
      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          maxlength="255"
          show-word-limit
          placeholder="可选，如 老客户"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">确认</el-button>
    </template>
  </el-dialog>

  <el-drawer v-model="detailVisible" title="客户详情" size="680px" :destroy-on-close="true">
    <div v-loading="detailLoading" class="detail">
      <div v-if="detailError" class="detail-error">
        <el-alert :title="detailError" type="error" :closable="false" show-icon />
        <el-button class="retry-button" size="small" @click="retryDetail">重试</el-button>
      </div>
      <template v-if="detail">
        <div class="detail-head">
          <el-avatar v-if="detail.avatar" :size="46" :src="detail.avatar" />
          <div>
            <div class="detail-name">{{ detail.nickname || '未设置昵称' }}</div>
            <div class="muted">{{ detail.phone || '未绑定手机号' }}</div>
          </div>
        </div>

        <el-descriptions :column="1" border class="detail-desc">
          <el-descriptions-item label="会员等级">
            <template v-if="detail.level_id && Number(detail.member_status) === 1">
              <el-tag v-if="levelInactive(detail)" type="info" effect="light"> 等级已停用 </el-tag>
              <el-tag v-else :type="memberIsExpired(detail) ? 'info' : 'success'" effect="light">
                {{ detail.level_name || '会员' }}{{ memberIsExpired(detail) ? '（已过期）' : '' }}
              </el-tag>
            </template>
            <span v-else class="muted">普通用户</span>
          </el-descriptions-item>
          <el-descriptions-item label="到期日">{{ expiryText(detail) }}</el-descriptions-item>
          <el-descriptions-item label="备注">{{ detail.remark || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-section-title">最近订单</div>
        <el-table :data="detail.orders || []" size="small" empty-text="暂无订单">
          <el-table-column label="订单号" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">{{ row.order_no || '-' }}</template>
          </el-table-column>
          <el-table-column label="房型" min-width="110" show-overflow-tooltip>
            <template #default="{ row }">{{ row.resource_name || '-' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="92">
            <template #default="{ row }">
              <el-tag :type="getOrderStatus(row.status).type" effect="light">
                {{ getOrderStatus(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="金额" width="100">
            <template #default="{ row }">
              <span class="amount">¥{{ fenToYuan(row.total_amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="使用时间" min-width="150">
            <template #default="{ row }">{{ orderTimeText(row) }}</template>
          </el-table-column>
          <el-table-column label="下单时间" width="150">
            <template #default="{ row }">{{ detailTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>

        <div class="detail-section-title">充值记录</div>
        <el-table :data="detail.recharges || []" size="small" empty-text="暂无充值记录">
          <el-table-column label="金额" width="110">
            <template #default="{ row }">
              <span class="amount">¥{{ fenToYuan(row.amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getRechargeStatus(row.status).type" effect="light">
                {{ getRechargeStatus(row.status).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间">
            <template #default="{ row }">{{ detailTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </template>
      <el-empty
        v-else-if="!detailLoading && !detailError"
        description="暂无客户详情"
        :image-size="90"
      />
    </div>
  </el-drawer>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  getCustomer,
  getCustomers,
  getMemberLevels,
  removeCustomerMembership,
  setCustomerMembership,
  type CustomerDetail,
  type CustomerOrder,
  type MemberLevel,
  type StoreCustomer
} from '@/api/merchant'
import { fenToYuan, formatDate, formatDateTime } from '@/utils/merchant'

defineOptions({ name: 'MerchantCustomers' })

const merchantStore = useMerchantStore()

const rows = ref<StoreCustomer[]>([])
const levels = ref<MemberLevel[]>([])
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const removingId = ref<number | null>(null)
const visible = ref(false)
const target = ref<StoreCustomer | null>(null)
const detailVisible = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detail = ref<CustomerDetail | null>(null)
const detailTargetId = ref<number | null>(null)
const loadError = ref('')
const form = reactive({
  levelId: 0,
  expiresAt: null as number | string | null,
  remark: ''
})

const activeLevels = computed(() => levels.value.filter((level) => Number(level.status) === 1))

type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

const ORDER_STATUS: Record<number, { text: string; type: TagType }> = {
  0: { text: '待支付', type: 'warning' },
  1: { text: '已支付', type: 'primary' },
  2: { text: '进行中', type: 'success' },
  3: { text: '已完成', type: 'success' },
  4: { text: '已取消', type: 'info' },
  5: { text: '退款中', type: 'warning' },
  6: { text: '已退款', type: 'danger' }
}

const RECHARGE_STATUS: Record<number, { text: string; type: TagType }> = {
  0: { text: '待确认', type: 'warning' },
  1: { text: '已到账', type: 'success' },
  2: { text: '已取消', type: 'info' }
}

const UNKNOWN_STATUS: { text: string; type: TagType } = { text: '未知', type: 'info' }

const getOrderStatus = (status: any) => ORDER_STATUS[Number(status)] || UNKNOWN_STATUS
const getRechargeStatus = (status: any) => RECHARGE_STATUS[Number(status)] || UNKNOWN_STATUS

const toTimestamp = (value: any): number | null => {
  if (value === null || value === undefined || value === '') return null
  if (typeof value === 'number') return Number.isFinite(value) ? value : null
  const numeric = Number(value)
  if (Number.isFinite(numeric)) return numeric
  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : null
}

const detailTime = (value: any) => {
  const ms = toTimestamp(value)
  return ms === null ? '-' : formatDateTime(ms)
}

const orderTimeText = (order: CustomerOrder) => {
  if (!order.start_time && !order.end_time) return '-'
  const start = order.start_time ? detailTime(order.start_time) : '-'
  const end = order.end_time ? detailTime(order.end_time) : '-'
  return `${start} - ${end}`
}

const formatDiscount = (discountBP: number) => {
  const value = Number(discountBP)
  if (!Number.isFinite(value) || value <= 0) return '-'
  return (value / 1000).toFixed(2)
}

interface MemberState {
  level_id?: number
  level_status?: number
  member_status?: number
  expires_at?: number | null
}

const levelInactive = (row: MemberState) =>
  !!row.level_id && Number(row.member_status) === 1 && Number(row.level_status) !== 1

const memberIsExpired = (row: MemberState) =>
  Number(row.member_status) !== 1 || (!!row.expires_at && Number(row.expires_at) <= Date.now())

const expiryText = (row: MemberState) => {
  if (!row.level_id || Number(row.member_status) !== 1) return '-'
  if (Number(row.level_status) !== 1) return '-'
  return row.expires_at ? formatDate(row.expires_at) : '永久'
}

let loadSeq = 0
let levelSeq = 0
let paginationTimer: ReturnType<typeof setTimeout> | null = null
let sizeChangeGuard = false

const loadLevels = async () => {
  const seq = ++levelSeq
  try {
    const res = await getMemberLevels()
    if (seq !== levelSeq) return
    levels.value = Array.isArray(res) ? res : []
  } catch {
    if (seq === levelSeq) levels.value = []
  }
}

const load = async (nextPage = page.value) => {
  const seq = ++loadSeq
  page.value = nextPage
  loading.value = true
  loadError.value = ''
  try {
    const res = await getCustomers({
      keyword: keyword.value.trim(),
      page: page.value,
      page_size: pageSize.value
    })
    if (seq !== loadSeq) return
    rows.value = Array.isArray(res?.list) ? res.list : []
    total.value = Number(res?.total || 0)
  } catch (e: any) {
    if (seq !== loadSeq) return
    rows.value = []
    total.value = 0
    loadError.value = e?.message || '客户列表加载失败，请稍后重试'
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const refresh = async () => {
  await Promise.all([loadLevels(), load(page.value)])
}

const queuePaginationLoad = (nextPage: number) => {
  if (paginationTimer) clearTimeout(paginationTimer)
  paginationTimer = setTimeout(() => {
    paginationTimer = null
    load(nextPage)
  }, 0)
}

const handleCurrentChange = (nextPage: number) => {
  if (sizeChangeGuard) return
  queuePaginationLoad(nextPage)
}

const handleSizeChange = () => {
  sizeChangeGuard = true
  page.value = 1
  queuePaginationLoad(1)
  setTimeout(() => {
    sizeChangeGuard = false
  }, 0)
}

let detailSeq = 0

const loadDetail = async (userId: number) => {
  const seq = ++detailSeq
  detailLoading.value = true
  detailError.value = ''
  try {
    const res = await getCustomer(userId)
    if (seq !== detailSeq) return
    detail.value = res || null
  } catch (e: any) {
    if (seq !== detailSeq) return
    detail.value = null
    detailError.value = e?.message || '客户详情加载失败，请稍后重试'
  } finally {
    if (seq === detailSeq) detailLoading.value = false
  }
}

const openDetail = (row: StoreCustomer) => {
  detailTargetId.value = Number(row.id)
  detail.value = null
  detailError.value = ''
  detailVisible.value = true
  loadDetail(detailTargetId.value)
}

const retryDetail = () => {
  if (detailTargetId.value !== null) loadDetail(detailTargetId.value)
}

const openSet = (row: StoreCustomer) => {
  if (!String(row.phone || '').trim()) {
    ElMessage.warning('客户未绑定手机号，请先在用户端授权')
    return
  }
  target.value = row
  const levelId = Number(row.level_id)
  form.levelId = activeLevels.value.some((level) => Number(level.id) === levelId) ? levelId : 0
  form.expiresAt = row.expires_at ?? null
  form.remark = row.remark || ''
  visible.value = true
}

const save = async () => {
  if (saving.value || !target.value) return
  const levelId = Number(form.levelId)
  if (!levelId) {
    ElMessage.warning('请选择会员等级')
    return
  }

  let expiresAt: number | null = null
  if (form.expiresAt !== null && form.expiresAt !== undefined && form.expiresAt !== '') {
    const ms = Number(form.expiresAt)
    if (!Number.isFinite(ms) || ms <= 0) {
      ElMessage.warning('到期日格式不正确')
      return
    }
    // 到期日按当天 23:59:59.999 处理，避免当天零点即失效。
    const endOfDay = new Date(ms)
    endOfDay.setHours(23, 59, 59, 999)
    expiresAt = endOfDay.getTime()
  }

  saving.value = true
  try {
    await setCustomerMembership(Number(target.value.id), {
      level_id: levelId,
      expires_at: expiresAt,
      remark: form.remark.trim()
    })
    ElMessage.success('会员设置成功')
    visible.value = false
    await load()
  } catch {
    // 请求层已统一提示错误
  } finally {
    saving.value = false
  }
}

const remove = async (row: StoreCustomer) => {
  try {
    await ElMessageBox.confirm(
      `确定取消「${row.nickname || row.phone || '该客户'}」的会员？`,
      '取消会员',
      {
        type: 'warning',
        confirmButtonText: '确定取消',
        cancelButtonText: '返回'
      }
    )
  } catch {
    return
  }

  removingId.value = Number(row.id)
  try {
    await removeCustomerMembership(Number(row.id))
    ElMessage.success('已取消会员')
    await load()
  } catch {
    // 请求层已统一提示错误
  } finally {
    removingId.value = null
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    loadSeq++
    levelSeq++
    detailSeq++
    if (paginationTimer) {
      clearTimeout(paginationTimer)
      paginationTimer = null
    }
    keyword.value = ''
    page.value = 1
    rows.value = []
    total.value = 0
    levels.value = []
    loadError.value = ''
    loading.value = false
    visible.value = false
    target.value = null
    form.levelId = 0
    form.expiresAt = null
    form.remark = ''
    detailVisible.value = false
    detailLoading.value = false
    detailError.value = ''
    detail.value = null
    detailTargetId.value = null
    if (!id) return
    refresh()
  }
)

onMounted(async () => {
  await loadLevels()
  await load(1)
})
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.search {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.search-input {
  width: 260px;
}

.summary {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.load-error {
  margin-bottom: 12px;
}

.customer {
  display: flex;
  gap: 10px;
  align-items: center;
}

.customer-info {
  min-width: 0;
}

.customer-name {
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.muted {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
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

.tooltip-wrap {
  display: inline-flex;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.dialog-customer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
  padding: 10px 12px;
  margin-bottom: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.dialog-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.full {
  width: 100%;
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-placeholder);
}

.detail {
  min-height: 160px;
}

.detail-head {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  margin-bottom: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.detail-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.detail-desc {
  margin-bottom: 20px;
}

.detail-section-title {
  margin: 18px 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.detail-error {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: flex-start;
}

.retry-button {
  align-self: flex-end;
}

@media (width <= 768px) {
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .search-input {
    width: 100%;
  }

  .pagination {
    justify-content: center;
  }
}
</style>
