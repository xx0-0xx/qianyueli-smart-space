<template>
  <ContentWrap title="月度报表" :body-style="{ padding: '0 18px 18px' }">
    <el-alert
      class="report-tip"
      type="info"
      :closable="false"
      show-icon
      title="营收统计已支付、进行中、已完成订单，退款订单单独计数"
    />

    <div class="toolbar">
      <span class="summary">
        共 {{ months.length }} 个月 · {{ totalOrders }} 笔订单
      </span>
      <el-button :icon="Refresh" :loading="loading" @click="loadReport">刷新</el-button>
    </div>

    <el-table
      v-if="months.length || loading"
      v-loading="loading"
      :data="months"
      row-key="month"
      empty-text="暂无订单数据"
    >
      <el-table-column label="月份" width="140">
        <template #default="{ row }">
          <span class="month">{{ row.month }}</span>
        </template>
      </el-table-column>
      <el-table-column label="总订单" width="110" align="center">
        <template #default="{ row }">{{ row.total }}</template>
      </el-table-column>
      <el-table-column label="已支付" width="110" align="center">
        <template #default="{ row }">
          <span class="paid">{{ row.paid }}</span>
        </template>
      </el-table-column>
      <el-table-column label="已退款" width="110" align="center">
        <template #default="{ row }">
          <span :class="['refund', { 'is-zero': row.refund === 0 }]">{{ row.refund }}</span>
        </template>
      </el-table-column>
      <el-table-column label="营收" min-width="150" align="right">
        <template #default="{ row }">
          <span class="revenue">¥{{ row.revenue }}</span>
        </template>
      </el-table-column>
      <el-table-column label="客单价" min-width="130" align="right">
        <template #default="{ row }">¥{{ row.avgPrice }}</template>
      </el-table-column>
    </el-table>

    <el-empty
      v-else
      description="暂无订单数据，产生订单后将自动生成月度统计"
      :image-size="100"
    />
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { getAllOrders } from '@/api/merchant'
import { fenToYuan } from '@/utils/merchant'

defineOptions({ name: 'MerchantReport' })

interface MonthRow {
  month: string
  total: number
  paid: number
  refund: number
  revenue: string
  avgPrice: string
}

const merchantStore = useMerchantStore()

const months = ref<MonthRow[]>([])
const loading = ref(false)

let loadSeq = 0

const totalOrders = computed(() => months.value.reduce((sum, item) => sum + item.total, 0))

const asList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const toMs = (value: any): number => {
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return 0
  return num < 1e12 ? num * 1000 : num
}

const pad = (value: number) => String(value).padStart(2, '0')

const loadReport = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const res: any = await getAllOrders()
    if (seq !== loadSeq) return

    const map = new Map<
      string,
      { total: number; paid: number; refund: number; revenueFen: number }
    >()

    asList(res).forEach((order) => {
      const ms = toMs(order?.start_time)
      if (!ms) return
      const date = new Date(ms)
      if (Number.isNaN(date.getTime())) return

      const month = `${date.getFullYear()}-${pad(date.getMonth() + 1)}`
      let item = map.get(month)
      if (!item) {
        item = { total: 0, paid: 0, refund: 0, revenueFen: 0 }
        map.set(month, item)
      }

      item.total += 1
      const status = Number(order?.status)
      if ([1, 2, 3].includes(status)) {
        item.paid += 1
        item.revenueFen += Number(order?.total_amount || 0)
      }
      if (status === 6) {
        item.refund += 1
      }
    })

    months.value = Array.from(map.entries())
      .sort((a, b) => b[0].localeCompare(a[0]))
      .map(([month, value]) => ({
        month,
        total: value.total,
        paid: value.paid,
        refund: value.refund,
        revenue: fenToYuan(value.revenueFen),
        avgPrice: value.paid ? (value.revenueFen / 100 / value.paid).toFixed(2) : '0.00'
      }))
  } catch {
    // 请求层已统一提示错误
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    if (!id) {
      months.value = []
      return
    }
    loadReport()
  }
)

onMounted(() => {
  loadReport()
})
</script>

<style lang="scss" scoped>
.report-tip {
  margin-bottom: 14px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.summary {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.month {
  font-size: 15px;
  font-weight: 700;
  color: #1a56db;
  letter-spacing: 0.3px;
}

.paid {
  font-weight: 600;
  color: #16a34a;
}

.refund {
  font-weight: 600;
  color: #e11d48;
}

.refund.is-zero {
  font-weight: 400;
  color: var(--el-text-color-placeholder);
}

.revenue {
  font-size: 16px;
  font-weight: 700;
  color: #e11d48;
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
