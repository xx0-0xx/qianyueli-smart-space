<template>
  <ContentWrap title="团购核销">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="Elys API Key 由服务器环境变量配置，页面不展示。门店需先在无限聚核创建并拿到 shopId。"
      style="margin-bottom: 12px"
    />

    <el-tabs v-model="activeTab">
      <el-tab-pane label="门店绑定" name="bindings">
        <div class="toolbar">
          <el-button type="primary" @click="openAdd">新增绑定</el-button>
          <el-button @click="loadBindings">刷新</el-button>
        </div>

        <el-table :data="bindings" v-loading="loadingBindings" row-key="id">
          <el-table-column prop="store_name" label="门店" min-width="140" />
          <el-table-column prop="store_id" label="门店ID" width="150" />
          <el-table-column prop="elys_shop_id" label="Elys shopId" width="140" />
          <el-table-column prop="elys_master_shop_id" label="Elys 商户ID" width="140" />
          <el-table-column label="美团" width="90">
            <template #default="{ row }">
              <el-tag :type="row.meituan_authorized ? 'success' : 'info'">
                {{ row.meituan_authorized ? '已授权' : '未授权' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="抖音" width="90">
            <template #default="{ row }">
              <el-tag :type="row.douyin_authorized ? 'success' : 'info'">
                {{ row.douyin_authorized ? '已授权' : '未授权' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="360" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="auth(row, 'meituan')">美团授权</el-button>
              <el-button link type="primary" @click="auth(row, 'douyin')">抖音授权</el-button>
              <el-button link @click="openEdit(row)">编辑</el-button>
              <el-button link type="danger" @click="remove(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="核销记录" name="records">
        <el-form :inline="true" @submit.prevent>
          <el-form-item label="平台">
            <el-select v-model="recordQuery.platform" clearable placeholder="全部" style="width: 120px">
              <el-option label="美团" value="meituan" />
              <el-option label="抖音" value="douyin" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="recordQuery.status" clearable placeholder="全部" style="width: 140px">
              <el-option label="已核销" :value="20" />
              <el-option label="已入账" :value="30" />
              <el-option label="待入账" :value="40" />
              <el-option label="失败" :value="50" />
              <el-option label="结果未知" :value="60" />
              <el-option label="重复" :value="70" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadRecords">查询</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="records" v-loading="loadingRecords" row-key="id">
          <el-table-column label="时间" width="170">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="平台" width="90">
            <template #default="{ row }">{{ platformText(row.platform) }}</template>
          </el-table-column>
          <el-table-column prop="voucher_code" label="券码" min-width="180" show-overflow-tooltip />
          <el-table-column prop="ticket_name" label="套餐" min-width="140" show-overflow-tooltip />
          <el-table-column label="面额" width="100">
            <template #default="{ row }">¥{{ yuan(row.pay_amount) }}</template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户ID" width="150" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="error_message" label="错误" min-width="160" show-overflow-tooltip />
          <el-table-column label="操作" width="110" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.status === 20 || row.status === 40"
                link
                type="primary"
                @click="retry(row)"
              >
                重试入账
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑绑定' : '新增绑定'" width="520px">
      <el-form label-width="110px">
        <el-form-item label="门店">
          <el-select v-model="form.store_id" filterable :disabled="editing" style="width: 100%">
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Elys shopId">
          <el-input v-model="form.elys_shop_id" placeholder="Elys 系统门店 ID" />
        </el-form-item>
        <el-form-item label="Elys 商户ID">
          <el-input v-model="form.elys_master_shop_id" placeholder="Elys 商户 ID" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import {
  deleteElysBinding,
  getElysAuthUrl,
  getElysBindings,
  getElysVerifications,
  getStores,
  retryElysCredit,
  upsertElysBinding,
  type ElysBindingVO
} from '@/api/ssm'

defineOptions({ name: 'SsmElys' })

const activeTab = ref<'bindings' | 'records'>('bindings')
const bindings = ref<ElysBindingVO[]>([])
const records = ref<any[]>([])
const stores = ref<any[]>([])
const loadingBindings = ref(false)
const loadingRecords = ref(false)
const dialogVisible = ref(false)
const editing = ref(false)
const form = reactive({
  store_id: '' as string | number,
  elys_shop_id: '',
  elys_master_shop_id: ''
})
const recordQuery = reactive({ platform: '', status: '' as number | '' })

const yuan = (v: any) => (Number(v || 0) / 100).toFixed(2)

const loadBindings = async () => {
  loadingBindings.value = true
  try {
    bindings.value = (await getElysBindings()) || []
  } finally {
    loadingBindings.value = false
  }
}

const loadStores = async () => {
  const res: any = await getStores()
  stores.value = res.list || []
}

const loadRecords = async () => {
  loadingRecords.value = true
  try {
    const res: any = await getElysVerifications({
      platform: recordQuery.platform || undefined,
      status: recordQuery.status || undefined,
      page: 1,
      page_size: 100
    })
    records.value = res.list || []
  } finally {
    loadingRecords.value = false
  }
}

const openAdd = () => {
  editing.value = false
  form.store_id = ''
  form.elys_shop_id = ''
  form.elys_master_shop_id = ''
  dialogVisible.value = true
}

const openEdit = (row: ElysBindingVO) => {
  editing.value = true
  form.store_id = row.store_id
  form.elys_shop_id = String(row.elys_shop_id)
  form.elys_master_shop_id = String(row.elys_master_shop_id)
  dialogVisible.value = true
}

const save = async () => {
  if (!form.store_id || !form.elys_shop_id) {
    ElMessage.warning('门店和 Elys shopId 必填')
    return
  }
  await upsertElysBinding({
    store_id: Number(form.store_id),
    elys_shop_id: Number(form.elys_shop_id),
    elys_master_shop_id: Number(form.elys_master_shop_id || 0)
  })
  ElMessage.success('已保存')
  dialogVisible.value = false
  await loadBindings()
}

const remove = async (row: ElysBindingVO) => {
  try {
    await ElMessageBox.confirm(`确认删除门店「${row.store_name}」的 Elys 绑定？`, '提示', {
      type: 'warning'
    })
  } catch {
    return
  }
  await deleteElysBinding(row.store_id)
  ElMessage.success('已删除')
  await loadBindings()
}

const auth = async (row: ElysBindingVO, platform: 'meituan' | 'douyin') => {
  const res: any = await getElysAuthUrl(row.store_id, platform)
  const url = res.url
  if (!url) {
    ElMessage.warning('未获取到授权链接')
    return
  }
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('授权链接已复制，请发给商家打开授权')
  } catch {
    ElMessage.info(url)
  }
}

const retry = async (row: any) => {
  await retryElysCredit(row.id)
  ElMessage.success('已触发重试')
  await loadRecords()
}

const platformText = (p: string) => (p === 'meituan' ? '美团' : p === 'douyin' ? '抖音' : p)
const statusText = (s: number) => {
  const map: Record<number, string> = {
    10: '核销中',
    20: '已核销',
    30: '已入账',
    40: '待入账',
    50: '失败',
    60: '结果未知',
    70: '重复'
  }
  return map[s] || String(s)
}
const statusType = (s: number): 'success' | 'warning' | 'danger' | 'info' => {
  if (s === 30) return 'success'
  if (s === 20 || s === 40) return 'warning'
  if (s === 50 || s === 60) return 'danger'
  return 'info'
}
const formatTime = (v: string) => {
  if (!v) return '-'
  const d = new Date(v)
  if (isNaN(d.getTime())) return v
  const pad = (n: number) => (n < 10 ? '0' + n : String(n))
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

onMounted(() => {
  loadBindings()
  loadStores()
  loadRecords()
})
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
</style>
