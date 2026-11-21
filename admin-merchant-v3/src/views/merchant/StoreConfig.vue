<template>
  <ContentWrap title="门店设置" :body-style="{ padding: '0 18px 18px' }">
    <div v-loading="loading" class="settings-wrap">
      <el-alert
        class="settings-tip"
        type="info"
        :closable="false"
        show-icon
        title="WiFi 信息与客服电话将展示在顾客端，请确保信息真实有效"
      />

      <el-form :model="form" label-width="110px" class="settings-form" @submit.prevent>
        <div class="section">
          <div class="section-title">WiFi 信息</div>
          <el-form-item label="WiFi 名称">
            <el-input
              v-model="form.wifiName"
              placeholder="请输入 WiFi 名称"
              maxlength="32"
              clearable
            />
          </el-form-item>
          <el-form-item label="WiFi 密码">
            <el-input
              v-model="form.wifiPass"
              type="text"
              show-password
              placeholder="请输入 WiFi 密码"
              maxlength="64"
              autocomplete="off"
            />
          </el-form-item>
        </div>

        <div class="section">
          <div class="section-title">联系方式</div>
          <el-form-item label="客服电话">
            <el-input
              v-model="form.phone"
              placeholder="请输入门店客服电话"
              maxlength="20"
              clearable
            />
          </el-form-item>
          <div class="section-desc">该号码用于顾客端展示与联系，请勿填写他人号码。</div>
        </div>

        <div class="section">
          <div class="section-title">退款设置</div>
          <el-form-item label="自动同意退款">
            <div class="switch-row">
              <el-switch v-model="form.autoApproveRefund" />
              <span class="switch-tip">
                开启后，顾客申请退款将立即退回余额，无需商家逐单同意；距开始不足 15
                分钟的订单不可退款。
              </span>
            </div>
          </el-form-item>
        </div>

        <div class="form-actions">
          <el-button type="primary" :loading="saving" :disabled="loading" @click="handleSave">
            保存设置
          </el-button>
        </div>
      </el-form>

      <div class="legal-links">
        <a :href="agreementHref" target="_blank" rel="noopener noreferrer">《用户服务协议》</a>
        <span class="dot">·</span>
        <a :href="privacyHref" target="_blank" rel="noopener noreferrer">《隐私政策》</a>
      </div>
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import { getStoreConfig, updateStoreConfig } from '@/api/merchant'
import { parseJson } from '@/utils/merchant'

defineOptions({ name: 'MerchantStoreConfig' })

const router = useRouter()
const merchantStore = useMerchantStore()

const form = reactive({
  wifiName: '',
  wifiPass: '',
  phone: '',
  autoApproveRefund: true
})
const extraConfig = ref<Record<string, any>>({})
const loading = ref(false)
const saving = ref(false)

const agreementHref = computed(() => router.resolve('/agreement').href)
const privacyHref = computed(() => router.resolve('/privacy').href)

let loadSeq = 0

const normalizeAutoApprove = (value: any): boolean => {
  if (value === undefined || value === null || value === '') return true
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    return normalized === '1' || normalized === 'true'
  }
  return Boolean(value)
}

const loadConfig = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const res: any = await getStoreConfig()
    if (seq !== loadSeq) return
    const rawConfig = res?.config_json
    const config =
      rawConfig && typeof rawConfig === 'object'
        ? rawConfig
        : parseJson<Record<string, any>>(rawConfig, {}) || {}
    extraConfig.value = { ...config }
    form.wifiName = config.wifi_ssid || ''
    form.wifiPass = config.wifi_pass || ''
    form.phone = res?.phone || ''
    form.autoApproveRefund = normalizeAutoApprove(res?.auto_approve_refund)
  } catch {
    // 请求层已统一提示错误
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const handleSave = async () => {
  if (saving.value) return
  saving.value = true
  try {
    await updateStoreConfig({
      config_json: JSON.stringify({
        ...extraConfig.value,
        wifi_ssid: form.wifiName.trim(),
        wifi_pass: form.wifiPass
      }),
      phone: form.phone.trim(),
      auto_approve_refund: form.autoApproveRefund
    })
    ElMessage.success('门店设置已保存')
  } catch {
    // 请求层已统一提示错误
  } finally {
    saving.value = false
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    if (!id) return
    loadConfig()
  }
)

onMounted(() => {
  loadConfig()
})
</script>

<style lang="scss" scoped>
.settings-wrap {
  max-width: 760px;
  padding-top: 4px;
}

.settings-tip {
  margin-bottom: 18px;
}

.section {
  padding: 16px 18px 2px;
  margin-bottom: 14px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
}

.section-title {
  margin-bottom: 14px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-desc {
  padding: 0 0 14px;
  margin-top: -6px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.settings-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.switch-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.switch-tip {
  flex: 1;
  min-width: 220px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--el-text-color-placeholder);
}

.form-actions {
  padding: 6px 0 4px;
}

.legal-links {
  padding-top: 18px;
  margin-top: 8px;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
  text-align: center;
  border-top: 1px dashed var(--el-border-color-light);
}

.legal-links a {
  color: #1a56db;
  text-decoration: none;
}

.legal-links a:hover {
  text-decoration: underline;
}

.dot {
  margin: 0 8px;
}

@media (max-width: 640px) {
  .settings-wrap {
    max-width: none;
  }

  .settings-form :deep(.el-form-item__label) {
    float: none;
    width: auto !important;
    padding: 0 0 6px;
    line-height: 1.5;
  }

  .settings-form :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }

  .section {
    padding: 14px 14px 0;
  }
}
</style>
