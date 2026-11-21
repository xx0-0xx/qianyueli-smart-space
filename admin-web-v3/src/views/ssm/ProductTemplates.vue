<template>
  <ContentWrap title="产品模板">
    <template #header>
      <div class="toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索产品名称"
          :prefix-icon="Search"
          clearable
          style="width: 220px"
        />
        <el-button type="primary" :icon="Plus" @click="add">新增产品模板</el-button>
      </div>
    </template>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="mb-12"
      title="产品模板 = 硬件型号。商家绑定设备时按模板自动生成子设备（如 3 路灯开关生成 3 个灯设备）"
    />

    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="170" show-overflow-tooltip />
      <el-table-column prop="name" label="产品名称" width="190" />
      <el-table-column label="协议" width="100">
        <template #default="{ row }">
          <el-tag effect="plain">{{ row.protocol === 'ascii' ? 'ASCII' : 'JSON' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="子设备" min-width="340">
        <template #default="{ row }">
          <div v-for="(d, i) in row.devices || []" :key="i" class="sub-device">
            {{ d.name }} · GPIO{{ d.gpio }} · {{ d.cmd_on || '—' }}/{{ d.cmd_off || '—' }} ·
            {{ typeLabel(d.type) }}
            <span v-if="!d.is_toggle" class="muted">（点动）</span>
          </div>
          <span v-if="!row.devices || row.devices.length === 0" class="muted">无子设备</span>
        </template>
      </el-table-column>
      <el-table-column label="展示顺序" width="120">
        <template #default="{ row }">
          <el-button link type="primary" :disabled="isFirst(row)" @click="move(row, 'up')">
            上移
          </el-button>
          <el-button link type="primary" :disabled="isLast(row)" @click="move(row, 'down')">
            下移
          </el-button>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="edit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ filtered.length }} 个产品模板</span>
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" :title="editingId ? '编辑产品模板' : '新增产品模板'" width="900px">
    <el-form label-width="90px">
      <el-form-item label="产品名称" required>
        <el-input v-model="form.name" placeholder="如：3路灯开关" />
      </el-form-item>
      <el-form-item label="通信协议">
        <el-select v-model="form.protocol" style="width: 100%">
          <el-option label="自制固件（JSON 指令，如 {cmd:relay1_on}）" value="json" />
          <el-option label="成品控制器（ASCII 纯文本，如 a1 / mp3=00001）" value="ascii" />
        </el-select>
      </el-form-item>
      <el-form-item label="设备类型">
        <el-select v-model="form.type" style="width: 100%">
          <el-option label="其他" value="other" />
          <el-option label="音响 / 云喇叭" value="speaker" />
          <el-option label="灯光" value="light" />
          <el-option label="门锁" value="door_lock" />
          <el-option label="插座" value="machine" />
          <el-option label="空调" value="ac" />
          <el-option label="排风" value="exhaust" />
        </el-select>
        <div class="muted">无子设备模板（如云喇叭）添加设备时使用此类型；有子设备时以子设备类型为准。</div>
      </el-form-item>
    </el-form>

    <div class="lines-title">子设备定义（商家绑定时按此生成）</div>
    <el-table :data="form.devices" border size="small">
      <el-table-column label="名称" width="130">
        <template #default="{ row }">
          <el-input v-model="row.name" size="small" placeholder="如 灯1" />
        </template>
      </el-table-column>
      <el-table-column label="GPIO" width="90">
        <template #default="{ row }">
          <el-input v-model.number="row.gpio" size="small" type="number" />
        </template>
      </el-table-column>
      <el-table-column label="开指令" width="140">
        <template #default="{ row }">
          <el-input v-model="row.cmd_on" size="small" placeholder="a1" />
        </template>
      </el-table-column>
      <el-table-column label="关指令" width="140">
        <template #default="{ row }">
          <el-input v-model="row.cmd_off" size="small" placeholder="b1" />
        </template>
      </el-table-column>
      <el-table-column label="类型" width="120">
        <template #default="{ row }">
          <el-select v-model="row.type" size="small">
            <el-option label="灯光" value="light" />
            <el-option label="门锁" value="door_lock" />
            <el-option label="插座" value="machine" />
            <el-option label="音响" value="speaker" />
            <el-option label="空调" value="ac" />
            <el-option label="排风" value="exhaust" />
            <el-option label="其他" value="other" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="开关型" width="110">
        <template #default="{ row }">
          <el-switch v-model="row.is_toggle" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="70">
        <template #default="{ $index }">
          <el-button link type="danger" @click="form.devices.splice($index, 1)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button class="add-line" size="small" @click="addDevice">+ 添加子设备</el-button>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import {
  createProductTemplate,
  deleteProductTemplate,
  getProductTemplates,
  moveProductTemplate,
  updateProductTemplate
} from '@/api/ssm'

defineOptions({ name: 'ProductTemplates' })

const templates = ref<any[]>([])
const keyword = ref('')
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref<number | null>(null)

interface TemplateDevice {
  name: string
  gpio: number
  cmd_on: string
  cmd_off: string
  type: string
  is_toggle: boolean
}

const form = reactive({
  name: '',
  protocol: 'json',
  type: 'other',
  devices: [] as TemplateDevice[]
})

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return templates.value
  return templates.value.filter((t) => String(t.name || '').toLowerCase().includes(kw))
})

const isFirst = (row: any) =>
  templates.value.findIndex((item) => String(item.id) === String(row.id)) <= 0

const isLast = (row: any) => {
  const index = templates.value.findIndex((item) => String(item.id) === String(row.id))
  return index < 0 || index >= templates.value.length - 1
}

const move = async (row: any, direction: 'up' | 'down') => {
  if (loading.value) return
  loading.value = true
  try {
    await moveProductTemplate(Number(row.id), direction)
    await load()
  } finally {
    loading.value = false
  }
}

const typeLabel = (type: string) =>
  (
    {
      light: '灯光',
      door_lock: '门锁',
      machine: '插座',
      speaker: '音响',
      ac: '空调',
      exhaust: '排风',
      other: '其他'
    } as Record<string, string>
  )[type] || type || '—'

const emptyDevice = (): TemplateDevice => ({
  name: '',
  gpio: 0,
  cmd_on: '',
  cmd_off: '',
  type: 'light',
  is_toggle: true
})

const load = async () => {
  loading.value = true
  try {
    templates.value = (await getProductTemplates()) || []
  } finally {
    loading.value = false
  }
}

const add = () => {
  editingId.value = null
  form.name = ''
  form.protocol = 'json'
  form.type = 'other'
  form.devices = [emptyDevice()]
  visible.value = true
}

const addDevice = () => form.devices.push(emptyDevice())

const edit = (row: any) => {
  editingId.value = row.id
  form.name = row.name || ''
  form.protocol = row.protocol || 'json'
  form.type = row.type || 'other'
  form.devices = (row.devices || []).map((d: any) => ({ ...emptyDevice(), ...d }))
  visible.value = true
}

const save = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请填写产品名称')
    return
  }
  saving.value = true
  try {
    const body = {
      name: form.name.trim(),
      devices: form.devices.filter((d) => d.name || d.cmd_on),
      status: 1,
      protocol: form.protocol || 'json',
      type: form.type || 'other'
    }
    if (editingId.value) {
      await updateProductTemplate(editingId.value, body)
    } else {
      await createProductTemplate(body)
    }
    ElMessage.success('保存成功')
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const remove = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定删除产品模板「${row.name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await deleteProductTemplate(row.id)
  ElMessage.success('已删除')
  await load()
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

.mb-12 {
  margin-bottom: 12px;
}

.sub-device {
  font-size: 12px;
  line-height: 1.8;
  color: #6b7280;
}

.lines-title {
  margin: 8px 0;
  font-size: 13px;
  color: #6b7280;
}

.add-line {
  margin-top: 10px;
}
</style>
