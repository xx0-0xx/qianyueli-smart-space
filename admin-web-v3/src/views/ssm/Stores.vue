<template>
  <ContentWrap title="门店管理">
    <template #header>
      <div class="toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索门店名称 / 地址"
          :prefix-icon="Search"
          clearable
          style="width: 240px"
        />
        <el-button type="primary" :icon="Plus" @click="add">新增门店</el-button>
      </div>
    </template>

    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="150" show-overflow-tooltip />
      <el-table-column prop="name" label="名称" min-width="130" />
      <el-table-column prop="address" label="地址" min-width="180" show-overflow-tooltip />
      <el-table-column prop="phone" label="电话" width="130" />
      <el-table-column label="余额组" width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ groupName(row.balance_group_id) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '营业' : '已下架' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="邀请码" width="160">
        <template #default="{ row }">
          <template v-if="row.invite_code">
            <el-tag effect="plain" class="code-tag">{{ row.invite_code }}</el-tag>
            <el-button link type="primary" size="small" @click="copy(row.invite_code)">
              复制
            </el-button>
          </template>
          <span v-else class="muted">未生成</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="edit(row)">编辑</el-button>
          <el-button link :type="row.status === 1 ? 'info' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 1 ? '下架' : '上架' }}
          </el-button>
          <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
            <el-button link type="primary">更多</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="regen">刷新邀请码</el-dropdown-item>
                <el-dropdown-item command="qr">小程序码</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                <el-dropdown-item command="force-delete">强制删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ filtered.length }} 家门店</span>
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" :title="editingId ? '编辑门店' : '新增门店'" width="620px">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="请输入门店名称" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="电话">
            <el-input v-model="form.phone" placeholder="请输入联系电话" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="地址" prop="address">
        <el-input v-model="form.address" placeholder="请输入门店地址" />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="纬度">
            <el-input v-model="form.latitude" placeholder="39.9042" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="经度">
            <el-input v-model="form.longitude" placeholder="116.4074" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="封面图">
        <div class="cover-row">
          <input ref="coverInput" type="file" accept="image/*" class="hidden-input" @change="onCoverChange" />
          <el-button size="small" @click="coverInput?.click()">上传图片</el-button>
          <el-input v-model="form.cover" placeholder="封面 URL（或上传）" />
        </div>
        <img v-if="form.cover" :src="form.cover" class="cover-preview" alt="" />
      </el-form-item>

      <el-form-item label="告警 Webhook">
        <el-input
          v-model="form.notify_webhook"
          type="textarea"
          :rows="2"
          placeholder="企业微信/钉钉群机器人地址（可选）"
        />
        <div class="help">填后：该门店设备离线、每日营收会推送到这个群；留空则推送至系统总群</div>
      </el-form-item>

      <el-form-item label="余额组">
        <el-select v-model="form.balance_group_id" style="width: 100%">
          <el-option label="独立（新建组）" :value="0" />
          <el-option v-for="g in groups" :key="g.id" :label="groupLabel(g)" :value="Number(g.id)" />
        </el-select>
        <div class="help">同一余额组的门店，用户余额互通；不同组互相独立</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="qrVisible" :title="`${qrName} 的小程序码`" width="380px" align-center>
    <div class="qr-box">
      <img v-if="qrUrl" :src="qrUrl" class="qr-img" alt="" />
      <el-button v-if="qrUrl" size="small" @click="downloadQr">下载图片</el-button>
    </div>
  </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { getAccessToken } from '@/utils/auth'
import {
  API_BASE,
  createStore,
  deleteStore,
  forceDeleteStore,
  getBalanceGroups,
  getStores,
  regenerateInviteCode,
  setStoreBalanceGroup,
  storeQRCodeUrl,
  updateStore,
  uploadAudio
} from '@/api/ssm'

defineOptions({ name: 'Stores' })

const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref<number | null>(null)
const keyword = ref('')
const stores = ref<any[]>([])
const groups = ref<any[]>([])
const formRef = ref<FormInstance>()
const coverInput = ref<HTMLInputElement>()

const form = reactive({
  name: '',
  address: '',
  phone: '',
  latitude: '',
  longitude: '',
  cover: '',
  notify_webhook: '',
  balance_group_id: 0
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入门店名称', trigger: 'blur' }],
  address: [{ required: true, message: '请输入门店地址', trigger: 'blur' }]
}

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return stores.value
  return stores.value.filter(
    (s) =>
      String(s.name || '').toLowerCase().includes(kw) ||
      String(s.address || '').toLowerCase().includes(kw)
  )
})

const groupName = (id: number) => groups.value.find((g) => Number(g.id) === Number(id))?.name || (id ? `组#${id}` : '—')
const groupLabel = (g: any) => `${g.name}（${(g.stores || []).length} 店）`

const load = async () => {
  loading.value = true
  try {
    const [storesRes, groupsRes] = await Promise.all([
      getStores(),
      getBalanceGroups().catch(() => [])
    ])
    stores.value = storesRes.list || []
    groups.value = groupsRes || []
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(form, {
    name: '',
    address: '',
    phone: '',
    latitude: '',
    longitude: '',
    cover: '',
    notify_webhook: '',
    balance_group_id: 0
  })
}

const add = () => {
  editingId.value = null
  resetForm()
  visible.value = true
}

const edit = (row: any) => {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name || '',
    address: row.address || '',
    phone: row.phone || '',
    latitude: row.latitude ? String(row.latitude) : '',
    longitude: row.longitude ? String(row.longitude) : '',
    cover: row.cover || '',
    notify_webhook: row.notify_webhook || '',
    balance_group_id: Number(row.balance_group_id) || 0
  })
  visible.value = true
}

const onCoverChange = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res: any = await uploadAudio(fd)
    const name = res?.data?.name || res?.data?.filename || file.name
    form.cover = `${API_BASE}/audio/${name}`
    ElMessage.success('图片已上传')
  } catch {
    ElMessage.error('图片上传失败')
  }
}

const save = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const body = {
      name: form.name,
      address: form.address,
      phone: form.phone,
      latitude: Number(form.latitude) || 0,
      longitude: Number(form.longitude) || 0,
      cover: form.cover,
      notify_webhook: form.notify_webhook
    }
    if (editingId.value) {
      const old = stores.value.find((s) => s.id === editingId.value)
      await updateStore(editingId.value, body)
      if (Number(old?.balance_group_id || 0) !== Number(form.balance_group_id || 0)) {
        await setStoreBalanceGroup(editingId.value, Number(form.balance_group_id) || 0)
      }
    } else {
      const created = await createStore(body)
      const gid = Number(form.balance_group_id) || 0
      if (created?.id && gid) {
        await setStoreBalanceGroup(created.id, gid)
      }
    }
    ElMessage.success('保存成功')
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (row: any) => {
  const target = row.status === 1 ? 2 : 1
  try {
    await ElMessageBox.confirm(
      `确定${target === 2 ? '下架' : '上架'}门店「${row.name}」？${
        target === 2 ? '下架后顾客端将不再显示、也无法预约该门店。' : ''
      }`,
      '提示',
      { type: 'warning' }
    )
  } catch {
    return
  }
  await updateStore(row.id, { status: target })
  ElMessage.success(target === 2 ? '已下架' : '已上架')
  await load()
}

const regen = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定刷新「${row.name}」的邀请码？旧邀请码将立即失效。`, '刷新邀请码', {
      type: 'warning'
    })
  } catch {
    return
  }
  const data = await regenerateInviteCode(row.id)
  ElMessage.success('新邀请码：' + (data.invite_code || ''))
  await load()
}

const remove = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定删除门店「${row.name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await deleteStore(row.id)
  ElMessage.success('已删除')
  await load()
}

const forceRemove = async (row: any) => {
  try {
    const { value } = await ElMessageBox.prompt(
      `此操作将彻底删除门店「${row.name}」及其全部数据（订单/支付/充值/退款/房间/设备/优惠券等），不可恢复。请输入门店名称以确认：`,
      '强制删除门店',
      {
        type: 'error',
        confirmButtonText: '确认强制删除',
        cancelButtonText: '取消',
        inputPlaceholder: row.name
      }
    )
    if (value !== row.name) {
      ElMessage.error('门店名称不匹配，已取消')
      return
    }
    await forceDeleteStore(row.id)
    ElMessage.success('已强制删除')
    await load()
  } catch {
    // 用户取消
  }
}

const handleCommand = (command: string, row: any) => {
  if (command === 'regen') regen(row)
  else if (command === 'qr') showQr(row)
  else if (command === 'delete') remove(row)
  else if (command === 'force-delete') forceRemove(row)
}

const qrVisible = ref(false)
const qrUrl = ref('')
const qrName = ref('')

const showQr = (row: any) => {
  qrUrl.value = storeQRCodeUrl(row.id, getAccessToken() || '')
  qrName.value = row.name || '门店'
  qrVisible.value = true
}

const downloadQr = () => {
  const a = document.createElement('a')
  a.href = qrUrl.value
  a.download = qrName.value + '-qrcode.png'
  a.click()
}

const copy = (code: string) => {
  navigator.clipboard.writeText(code).then(
    () => ElMessage.success('已复制：' + code),
    () => ElMessage.error('复制失败，请手动选中复制')
  )
}

onMounted(load)
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-grow: 1;
  gap: 10px;
  justify-content: flex-end;
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}

.muted {
  color: #9ca3af;
}

.code-tag {
  font-family: monospace;
  letter-spacing: 1px;
}

.cover-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.hidden-input {
  display: none;
}

.cover-preview {
  width: 100%;
  height: 110px;
  object-fit: cover;
  margin-top: 8px;
  border-radius: 8px;
}

.help {
  width: 100%;
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: #9ca3af;
}

.qr-box {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
}

.qr-img {
  width: 260px;
  height: 260px;
  border-radius: 8px;
}
</style>
