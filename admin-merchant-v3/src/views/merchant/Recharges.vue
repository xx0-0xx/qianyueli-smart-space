<template>
  <ContentWrap title="充值确认" :body-style="{ padding: '0 18px 18px' }">
    <el-alert
      class="recharge-tip"
      type="info"
      :closable="false"
      show-icon
      title="确认到账后金额将计入用户余额；取消后该充值单不再处理"
    />

    <el-tabs v-model="activeTab" class="recharge-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="待确认" name="pending" />
      <el-tab-pane label="已到账" name="paid" />
    </el-tabs>

    <div class="toolbar">
      <span class="summary">
        {{ activeTab === 'pending' ? '待确认充值单' : '已到账充值单' }}：{{ list.length }} 条
      </span>
      <el-button :icon="Refresh" :loading="loading" @click="loadList">刷新</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="list"
      row-key="id"
      :empty-text="activeTab === 'pending' ? '暂无待确认充值' : '暂无已到账充值'"
    >
      <el-table-column label="金额" width="130">
        <template #default="{ row }">
          <span class="amount">¥{{ fenToYuan(row.amount) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="用户" min-width="140">
        <template #default="{ row }">{{ row.nickname || '用户' }}</template>
      </el-table-column>
      <el-table-column label="备注" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.note || '-' }}</template>
      </el-table-column>
      <el-table-column label="时间" width="170">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="Number(row.status) === 0 ? 'warning' : 'success'" effect="light">
            {{ Number(row.status) === 0 ? '待确认' : '已到账' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <div v-if="Number(row.status) === 0" class="row-actions">
            <el-button
              link
              type="success"
              :loading="actionKey === `confirm-${row.id}`"
              :disabled="!!actionKey"
              @click="handleConfirm(row)"
            >
              确认到账
            </el-button>
            <el-button
              link
              type="danger"
              :loading="actionKey === `cancel-${row.id}`"
              :disabled="!!actionKey"
              @click="handleCancel(row)"
            >
              取消
            </el-button>
          </div>
          <span v-else class="done-text">已处理</span>
        </template>
      </el-table-column>
    </el-table>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { cancelRecharge, confirmRecharge, getRecharges } from '@/api/merchant'
import { fenToYuan, formatDateTime } from '@/utils/merchant'

defineOptions({ name: 'MerchantRecharges' })

type ActiveTab = 'pending' | 'paid'

interface Recharge {
  id: number
  amount?: number
  status?: number | string
  nickname?: string
  note?: string
  created_at?: number
  [key: string]: any
}

const merchantStore = useMerchantStore()

const activeTab = ref<ActiveTab>('pending')
const list = ref<Recharge[]>([])
const loading = ref(false)
const actionKey = ref('')

let loadSeq = 0

const asList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const loadList = async () => {
  const seq = ++loadSeq
  const status = activeTab.value === 'pending' ? 0 : 1
  loading.value = true
  try {
    const res: any = await getRecharges(status)
    if (seq !== loadSeq) return
    list.value = asList(res) as Recharge[]
  } catch {
    // 请求层已统一提示错误
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const handleTabChange = () => {
  list.value = []
  loadList()
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
  try {
    await ElMessageBox.confirm(message, title, {
      type: 'warning',
      confirmButtonText,
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  actionKey.value = key
  try {
    await action()
    ElMessage.success(successMessage)
    await loadList()
  } catch {
    // 请求层已统一提示错误
  } finally {
    actionKey.value = ''
  }
}

const handleConfirm = (row: Recharge) =>
  runAction(
    `confirm-${row.id}`,
    `确认用户充值 ¥${fenToYuan(row.amount)} 已到账？确认后将计入用户余额。`,
    '确认到账',
    '确认到账',
    () => confirmRecharge(row.id),
    '已确认到账'
  )

const handleCancel = (row: Recharge) =>
  runAction(
    `cancel-${row.id}`,
    `确认取消该笔 ¥${fenToYuan(row.amount)} 的充值单？`,
    '取消充值',
    '确认取消',
    () => cancelRecharge(row.id),
    '已取消该充值单'
  )

watch(
  () => merchantStore.storeId,
  (id) => {
    if (!id) {
      list.value = []
      return
    }
    loadList()
  }
)

onMounted(() => {
  loadList()
})
</script>

<style lang="scss" scoped>
.recharge-tip {
  margin-bottom: 10px;
}

.recharge-tabs :deep(.el-tabs__header) {
  margin-bottom: 12px;
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

.amount {
  font-size: 15px;
  font-weight: 700;
  color: #e11d48;
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

.done-text {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
