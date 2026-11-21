<template>
  <div>
    <el-row :gutter="16">
      <el-col v-for="item in statCards" :key="item.label" :xs="24" :sm="8">
        <el-card shadow="never" class="stat-card">
          <div class="stat-card__label">{{ item.label }}</div>
          <div class="stat-card__value" :style="{ color: item.color }">{{ item.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <ContentWrap title="门店收入明细">
      <template #header>
        <div class="toolbar">
          <el-date-picker
            v-model="month"
            type="month"
            placeholder="选择月份"
            format="YYYY-MM"
            value-format="YYYY-MM"
            clearable
            style="width: 180px"
            @change="load"
          />
          <el-button :icon="Refresh" @click="load">刷新</el-button>
        </div>
      </template>

      <el-table :data="rows" v-loading="loading" stripe>
        <el-table-column prop="name" label="门店" min-width="160" />
        <el-table-column prop="count" label="已完成订单" width="120" />
        <el-table-column prop="refund" label="退款订单" width="110" />
        <el-table-column label="收入" width="160">
          <template #default="{ row }">
            <span class="revenue">¥{{ Number(row.revenue).toLocaleString() }}</span>
          </template>
        </el-table-column>
      </el-table>
    </ContentWrap>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { getOrders, getStores } from '@/api/ssm'

defineOptions({ name: 'Report' })

const loading = ref(false)
const month = ref('')
const total = ref(0)
const doneCount = ref(0)
const refundCount = ref(0)
const rows = ref<any[]>([])

const statCards = computed(() => [
  { label: '总收入', value: `¥${total.value.toLocaleString()}`, color: '#1a56db' },
  { label: '已完成订单', value: doneCount.value, color: '#16a34a' },
  { label: '已退款', value: refundCount.value, color: '#e11d48' }
])

const load = async () => {
  loading.value = true
  try {
    const [storesRes, ordersRes] = await Promise.all([getStores(), getOrders()])
    const stores = storesRes.list || []
    let orders = ordersRes.list || []

    if (month.value) {
      orders = orders.filter((o) => String(o.created_at || '').startsWith(month.value))
    }

    const map: Record<string, { revenue: number; count: number; refund: number }> = {}
    orders.forEach((o) => {
      const sid = String(o.store_id)
      if (!map[sid]) map[sid] = { revenue: 0, count: 0, refund: 0 }
      if ([1, 2, 3].includes(o.status)) {
        map[sid].revenue += o.total_amount || 0
        map[sid].count++
      } else if (o.status === 6) {
        map[sid].refund++
      }
    })

    rows.value = stores.map((s) => ({
      id: s.id,
      name: s.name,
      revenue: Math.round((map[String(s.id)]?.revenue || 0) / 100),
      count: map[String(s.id)]?.count || 0,
      refund: map[String(s.id)]?.refund || 0
    }))

    total.value = rows.value.reduce((sum, r) => sum + r.revenue, 0)
    doneCount.value = rows.value.reduce((sum, r) => sum + r.count, 0)
    refundCount.value = rows.value.reduce((sum, r) => sum + r.refund, 0)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style lang="scss" scoped>
.stat-card {
  margin-bottom: 16px;
  border: none;

  &__label {
    font-size: 13px;
    color: #6b7280;
  }

  &__value {
    margin-top: 6px;
    font-size: 28px;
    font-weight: 700;
  }
}

.toolbar {
  display: flex;
  flex-grow: 1;
  gap: 10px;
  justify-content: flex-end;
}

.revenue {
  font-weight: 600;
  color: #1a56db;
}
</style>
