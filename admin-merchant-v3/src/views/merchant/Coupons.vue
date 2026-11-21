<template>
  <ContentWrap title="券码管理" :body-style="{ padding: '0 18px 18px' }">
    <div class="create-card">
      <div class="create-head">
        <div class="create-title">生成券码</div>
        <div class="create-desc">顾客在小程序「券码兑换」输入券码，金额直接充入余额</div>
      </div>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :inline="true"
        class="create-form"
        @submit.prevent
      >
        <el-form-item label="券名" prop="name">
          <el-input
            v-model="form.name"
            class="name-input"
            placeholder="如 新人礼包"
            maxlength="40"
            clearable
          />
        </el-form-item>
        <el-form-item label="面额" prop="discount">
          <el-input-number
            v-model="form.discount"
            class="number-input"
            :min="1"
            :max="99999"
            :precision="2"
            :step="10"
            controls-position="right"
          />
          <span class="unit">元</span>
        </el-form-item>
        <el-form-item label="数量" prop="count">
          <el-input-number
            v-model="form.count"
            class="count-input"
            :min="1"
            :max="100"
            :step="1"
            controls-position="right"
          />
          <span class="unit">张</span>
        </el-form-item>
        <el-form-item class="submit-item">
          <el-button type="primary" :loading="creating" @click="handleCreate">
            生成券码
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-tabs v-model="activeTab" class="coupon-tabs">
      <el-tab-pane :label="`未使用 (${unusedList.length})`" name="unused">
        <div class="toolbar">
          <span class="summary">共 {{ unusedList.length }} 张未使用券码</span>
          <el-button :icon="Download" :disabled="!unusedList.length" @click="handleExport">
            导出未使用券码
          </el-button>
        </div>

        <el-table
          v-loading="loading"
          :data="unusedList"
          row-key="id"
          empty-text="暂无未使用券码"
        >
          <el-table-column label="券码" min-width="190">
            <template #default="{ row }">
              <span class="code">{{ row.code || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="券名" min-width="150">
            <template #default="{ row }">{{ row.name || '-' }}</template>
          </el-table-column>
          <el-table-column label="面额" width="120">
            <template #default="{ row }">
              <span class="amount">¥{{ amountOf(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <div class="row-actions">
                <el-button link type="primary" @click="handleCopy(row)">复制</el-button>
                <el-button
                  link
                  type="danger"
                  :loading="deletingId === Number(row.id)"
                  :disabled="deletingId !== null"
                  @click="handleDelete(row)"
                >
                  删除
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="`已使用 (${usedList.length})`" name="used">
        <div class="toolbar">
          <span class="summary">共 {{ usedList.length }} 张已使用券码</span>
        </div>

        <el-table
          v-loading="loading"
          :data="usedList"
          row-key="id"
          empty-text="暂无已使用券码"
        >
          <el-table-column label="券码" min-width="190">
            <template #default="{ row }">
              <span class="code is-used">{{ row.code || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="券名" min-width="150">
            <template #default="{ row }">{{ row.name || '-' }}</template>
          </el-table-column>
          <el-table-column label="面额" width="120">
            <template #default="{ row }">
              <span class="amount is-used">¥{{ amountOf(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                :loading="deletingId === Number(row.id)"
                :disabled="deletingId !== null"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { createCoupon, deleteCoupon, getCoupons } from '@/api/merchant'
import { downloadCsv, fenToYuan, formatDate } from '@/utils/merchant'

defineOptions({ name: 'MerchantCoupons' })

interface Coupon {
  id: number
  code?: string
  name?: string
  discount?: number
  price?: number
  status?: number | string
  [key: string]: any
}

const merchantStore = useMerchantStore()

const formRef = ref<FormInstance>()
const form = reactive({
  name: '',
  discount: 10,
  count: 1
})
const rules: FormRules = {
  name: [{ required: true, message: '请输入券名', trigger: 'blur' }],
  discount: [{ required: true, message: '请输入券面额', trigger: 'change' }],
  count: [{ required: true, message: '请输入生成数量', trigger: 'change' }]
}

const activeTab = ref<'unused' | 'used'>('unused')
const unusedList = ref<Coupon[]>([])
const usedList = ref<Coupon[]>([])
const loading = ref(false)
const creating = ref(false)
const deletingId = ref<number | null>(null)

let loadSeq = 0

const asList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const amountOf = (coupon: Coupon) =>
  fenToYuan(Number(coupon.discount) > 0 ? coupon.discount : coupon.price)

const loadCoupons = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const res: any = await getCoupons()
    if (seq !== loadSeq) return
    const list = asList(res) as Coupon[]
    unusedList.value = list.filter((item) => Number(item.status) === 0)
    usedList.value = list.filter((item) => Number(item.status) !== 0)
  } catch {
    // 请求层已统一提示错误
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const handleCreate = async () => {
  if (creating.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  const name = form.name.trim()
  if (!name) {
    ElMessage.warning('请输入券名')
    return
  }
  if (Number(form.discount) < 1) {
    ElMessage.warning('券面额不能低于 1 元')
    return
  }
  const count = Math.min(100, Math.max(1, Math.round(Number(form.count) || 1)))

  creating.value = true
  try {
    const res: any = await createCoupon({
      name,
      discount: Math.round(Number(form.discount) * 100),
      type: 1,
      hours: 0,
      price: 0,
      resource_type_id: 0,
      count
    })
    const rawCount = Array.isArray(res)
      ? res.length
      : Number(res?.count ?? res?.data?.count)
    const createdCount = Number.isFinite(rawCount) && rawCount > 0 ? rawCount : count
    ElMessage.success(`已生成 ${createdCount} 张券码`)
    form.name = ''
    form.discount = 10
    form.count = 1
    activeTab.value = 'unused'
    await loadCoupons()
  } catch {
    // 请求层已统一提示错误
  } finally {
    creating.value = false
  }
}

const copyText = async (text: string) => {
  if (!text) return false
  try {
    if (navigator.clipboard?.writeText && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // 继续尝试兼容方案
  }
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}

const handleCopy = async (coupon: Coupon) => {
  const ok = await copyText(String(coupon.code || ''))
  if (ok) {
    ElMessage.success('券码已复制')
  } else {
    ElMessage.warning('复制失败，请手动选择券码')
  }
}

const handleDelete = async (coupon: Coupon) => {
  try {
    await ElMessageBox.confirm('确定删除该券码？删除后不可恢复。', '删除券码', {
      type: 'warning',
      confirmButtonText: '确定删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  deletingId.value = Number(coupon.id)
  try {
    await deleteCoupon(coupon.id)
    ElMessage.success('已删除')
    await loadCoupons()
  } catch {
    // 请求层已统一提示错误
  } finally {
    deletingId.value = null
  }
}

const handleExport = () => {
  if (!unusedList.value.length) {
    ElMessage.warning('没有未使用的券码')
    return
  }
  const rows = unusedList.value.map((coupon) => [
    String(coupon.code || ''),
    coupon.name || '',
    amountOf(coupon)
  ])
  downloadCsv(`未使用券码_${formatDate(Date.now())}.csv`, [['券码', '券名', '面额'], ...rows])
  ElMessage.success('已导出未使用券码')
}

watch(
  () => merchantStore.storeId,
  (id) => {
    if (!id) {
      unusedList.value = []
      usedList.value = []
      return
    }
    loadCoupons()
  }
)

onMounted(() => {
  loadCoupons()
})
</script>

<style lang="scss" scoped>
.create-card {
  padding: 16px 18px 4px;
  margin-bottom: 6px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
}

.create-head {
  margin-bottom: 14px;
}

.create-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.create-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0 8px;
  align-items: flex-start;
}

.create-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.name-input {
  width: 220px;
}

.number-input {
  width: 160px;
}

.count-input {
  width: 130px;
}

.unit {
  margin-left: 8px;
  font-size: 13px;
  line-height: 32px;
  color: var(--el-text-color-secondary);
}

.submit-item {
  margin-left: 8px;
}

.coupon-tabs :deep(.el-tabs__header) {
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

.code {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: 14px;
  font-weight: 600;
  color: #1a56db;
  letter-spacing: 0.5px;
}

.code.is-used {
  color: var(--el-text-color-placeholder);
}

.amount {
  font-weight: 600;
  color: #e11d48;
}

.amount.is-used {
  font-weight: 500;
  color: var(--el-text-color-placeholder);
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

@media (max-width: 768px) {
  .name-input,
  .number-input,
  .count-input {
    width: 100%;
  }

  .create-form :deep(.el-form-item) {
    width: 100%;
    margin-right: 0;
  }

  .submit-item {
    margin-left: 0;
  }

  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
