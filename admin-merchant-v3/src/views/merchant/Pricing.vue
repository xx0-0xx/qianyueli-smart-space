<template>
  <ContentWrap title="定价管理" :body-style="{ padding: '0 18px 18px' }">
    <el-alert
      class="price-alert"
      type="info"
      :closable="false"
      show-icon
      title="按房型设置每小时价格，该房型下所有房间统一价格。"
    />

    <div class="toolbar">
      <span class="summary">共 {{ priceRows.length }} 个房型</span>
      <el-button :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
    </div>

    <div v-loading="loading" class="price-list">
      <template v-if="priceRows.length">
        <div v-for="row in priceRows" :key="row.typeId" class="price-row">
          <div class="price-name">
            <span class="price-name__text">{{ row.name }}</span>
            <span class="price-name__hint">该房型下所有房间统一价格</span>
          </div>
          <div class="price-input">
            <el-input-number
              v-model="row.price"
              :min="0"
              :max="99999"
              :precision="2"
              :step="1"
              controls-position="right"
            />
            <span class="price-unit">元 / 小时</span>
          </div>
        </div>
      </template>

      <el-empty
        v-else-if="!loading"
        description="暂未添加房型，请先在房型管理中新增"
        :image-size="90"
      />
    </div>

    <div class="footer">
      <span class="footer-tip">价格为 0 的房型不会被保存。</span>
      <el-button
        type="primary"
        :loading="saving"
        :disabled="!priceRows.length"
        @click="handleSave"
      >
        保存定价
      </el-button>
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { getPriceRules, getResourceTypes, replacePriceRules } from '@/api/merchant'

defineOptions({ name: 'MerchantPricing' })

interface ResourceTypeItem {
  id: number
  name: string
  [key: string]: any
}

interface PriceRow {
  typeId: number
  name: string
  price: number
}

const merchantStore = useMerchantStore()

const priceRows = ref<PriceRow[]>([])
const loading = ref(false)
const saving = ref(false)

let loadSeq = 0

const toList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const loadAll = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const [typesRes, rulesRes] = await Promise.all([getResourceTypes(), getPriceRules()])
    if (seq !== loadSeq) return
    const types = toList(typesRes) as ResourceTypeItem[]
    const rules = toList(rulesRes)

    priceRows.value = types.map((type) => {
      const rule = rules.find(
        (item) => Number(item?.resource_type_id) === Number(type.id)
      )
      const price = rule
        ? Number((Number(rule.base_price || 0) / 100).toFixed(2))
        : 50
      return {
        typeId: Number(type.id),
        name: type.name || `房型 #${type.id}`,
        price: Number.isFinite(price) ? price : 0
      }
    })
  } catch {
    return
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const handleSave = async () => {
  if (saving.value) return
  const rules = priceRows.value
    .map((row) => ({
      resource_type_id: Number(row.typeId),
      is_default: false,
      base_price: Math.round(Number(row.price || 0) * 100),
      time_unit: 60,
      name: row.name,
      status: 1
    }))
    .filter((rule) => rule.base_price > 0)

  if (!rules.length) {
    ElMessage.warning('请先设置价格')
    return
  }

  saving.value = true
  try {
    await replacePriceRules(rules)
    ElMessage.success('定价已保存')
    await loadAll()
  } catch {
    return
  } finally {
    saving.value = false
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    priceRows.value = []
    if (!id) return
    loadAll()
  }
)

onMounted(() => {
  loadAll()
})
</script>

<style lang="scss" scoped>
.price-alert {
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

.price-list {
  min-height: 120px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
}

.price-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.price-row:first-child {
  border-top: none;
}

.price-name {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 180px;
}

.price-name__text {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.price-name__hint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.price-input {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.price-input :deep(.el-input-number) {
  width: 170px;
}

.price-unit {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.footer {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: flex-end;
  padding-top: 16px;
}

.footer-tip {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

@media (width <= 768px) {
  .price-row {
    align-items: stretch;
  }

  .price-input,
  .price-input :deep(.el-input-number) {
    width: 100%;
  }

  .footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
