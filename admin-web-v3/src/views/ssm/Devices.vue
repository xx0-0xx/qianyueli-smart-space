<template>
  <ContentWrap title="设备管理">
    <template #header>
      <div class="toolbar">
        <el-select v-model="storeId" placeholder="全部门店" clearable style="width: 180px">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="搜索设备名称 / SN"
          :prefix-icon="Search"
          clearable
          style="width: 210px"
        />
        <el-button type="primary" :icon="Plus" :disabled="!storeId" @click="add">
          新增设备
        </el-button>
      </div>
    </template>

    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="device_sn" label="设备 ID" width="170" />
      <el-table-column prop="name" label="名称" min-width="130" />
      <el-table-column prop="type" label="作用" width="100">
        <template #default="{ row }">
          <el-tag effect="plain">{{ typeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="指令" width="140">
        <template #default="{ row }">
          <code class="cmd-text">{{ cmdText(row) || '—' }}</code>
        </template>
      </el-table-column>
      <el-table-column label="协议" width="90">
        <template #default="{ row }">{{ row.protocol || 'json' }}</template>
      </el-table-column>
      <el-table-column label="门店" min-width="140">
        <template #default="{ row }">{{ storeName(row.store_id) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
            {{ row.status === 1 ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="editSn(row.device_sn)">编辑</el-button>
          <el-button link type="danger" @click="delSn(row.device_sn)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ filtered.length }} 路（同 SN 为一台多路设备）</span>
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" :title="editingSn ? '编辑设备' : '新增设备'" width="760px">
    <div class="probe-row">
      <el-input
        v-model="form.device_sn"
        placeholder="设备 ID（设备背面，如 d8bfc01acfb3）"
        :disabled="!!editingSn"
      />
      <el-button type="primary" :loading="probing" @click="doProbe">检测硬件</el-button>
    </div>

    <el-alert
      v-if="probeResult"
      type="success"
      :closable="false"
      show-icon
      class="probe-alert"
      :title="`检测到 ${probeResult.channels} 路，原始返回：${probeResult.raw || '-'}`"
    />
    <el-alert
      v-if="probeMsg"
      type="error"
      :closable="false"
      show-icon
      class="probe-alert"
      :title="probeMsg"
    />

    <el-form label-width="90px" class="device-form">
      <el-form-item label="型号">
        <el-select v-model="form.model" style="width: 100%">
          <el-option v-for="m in models" :key="m" :label="m" :value="m" />
        </el-select>
      </el-form-item>
      <el-form-item label="路数">
        <el-input-number v-model="form.channels" :min="1" :max="12" @change="syncChannels" />
        <span class="muted ml-10">检测后自动填充，也可手动修改</span>
      </el-form-item>
    </el-form>

    <div class="lines-title">每一路配置：</div>
    <el-table :data="form.lines" border size="small">
      <el-table-column label="路" width="50">
        <template #default="{ $index }">{{ $index + 1 }}</template>
      </el-table-column>
      <el-table-column label="名称" width="150">
        <template #default="{ row }">
          <el-input v-model="row.name" size="small" placeholder="如 灯1" />
        </template>
      </el-table-column>
      <el-table-column label="作用" width="130">
        <template #default="{ row }">
          <el-select v-model="row.type" size="small">
            <el-option label="灯光" value="light" />
            <el-option label="门锁" value="door_lock" />
            <el-option label="空调" value="ac" />
            <el-option label="电源/插座" value="machine" />
            <el-option label="排风" value="exhaust" />
            <el-option label="其他" value="other" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="动作" width="130">
        <template #default="{ row }">
          <el-select v-model="row.action" size="small">
            <el-option label="保持(开/关)" value="hold" />
            <el-option label="点动(脉冲)" value="pulse" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="指令" width="120">
        <template #default="{ row, $index }">
          <code class="cmd-text">{{ lineCmd(row, $index) }}</code>
        </template>
      </el-table-column>
    </el-table>

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
  createDevice,
  deleteDevice,
  getDevices,
  getStores,
  probeDevice
} from '@/api/ssm'

defineOptions({ name: 'Devices' })

const models = [
  '4路220V',
  '4路220V+喇叭',
  '4路12V',
  '4路12V+喇叭',
  '2路220V',
  '3路220V',
  '8路220V',
  '12路220V',
  '其他'
]

const loading = ref(false)
const saving = ref(false)
const probing = ref(false)
const visible = ref(false)
const editingSn = ref<string | null>(null)
const keyword = ref('')
const storeId = ref<number | null>(null)
const stores = ref<any[]>([])
const devices = ref<any[]>([])
const probeResult = ref<{ channels: number; raw?: string } | null>(null)
const probeMsg = ref('')

const form = reactive({
  device_sn: '',
  model: '4路220V',
  channels: 4,
  lines: [] as Array<{ name: string; type: string; action: string }>
})

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return devices.value.filter((d) => {
    if (storeId.value && String(d.store_id) !== String(storeId.value)) return false
    if (!kw) return true
    return (
      String(d.name || '').toLowerCase().includes(kw) ||
      String(d.device_sn || '').toLowerCase().includes(kw)
    )
  })
})

const typeLabel = (type: string) =>
  (
    {
      light: '灯光',
      door_lock: '门锁',
      ac: '空调',
      machine: '电源/插座',
      exhaust: '排风',
      speaker: '音响',
      combo: '组合',
      other: '其他'
    } as Record<string, string>
  )[type] || type || '-'

const storeName = (id: number) =>
  stores.value.find((s) => String(s.id) === String(id))?.name || '-'

const cmdText = (row: any) => {
  try {
    const c = JSON.parse(row.config_json || '{}')
    return `${c.cmd_on || ''}${c.cmd_off ? ' / ' + c.cmd_off : ''}`
  } catch {
    return ''
  }
}

const lineCmd = (row: { action: string }, index: number) => {
  const n = index + 1
  return row.action === 'pulse' ? `c${n}` : `a${n} / b${n}`
}

const syncChannels = () => {
  const old = form.lines
  const lines: typeof form.lines = []
  for (let i = 0; i < form.channels; i++) {
    lines.push(old[i] || { name: '', type: 'light', action: 'hold' })
  }
  form.lines = lines
}

const load = async () => {
  loading.value = true
  try {
    const [storesRes, devicesRes] = await Promise.all([getStores(), getDevices()])
    stores.value = storesRes.list || []
    devices.value = devicesRes.list || []
  } finally {
    loading.value = false
  }
}

const add = () => {
  editingSn.value = null
  form.device_sn = ''
  form.model = '4路220V'
  form.channels = 4
  form.lines = []
  probeResult.value = null
  probeMsg.value = ''
  syncChannels()
  visible.value = true
}

const doProbe = async () => {
  const sn = form.device_sn.trim()
  if (!sn) {
    ElMessage.warning('请先填写设备 ID')
    return
  }
  probing.value = true
  probeResult.value = null
  probeMsg.value = ''
  try {
    const data = await probeDevice(sn)
    if (data.channels > 0) {
      probeResult.value = data
      form.channels = data.channels
      syncChannels()
    } else {
      probeMsg.value =
        '未收到设备响应，请确认：①设备已配网并连上服务器 ②设备在线 ③再试一次'
    }
  } catch (e: any) {
    probeMsg.value = '检测失败：' + (e?.message || '')
  } finally {
    probing.value = false
  }
}

const editSn = (sn: string) => {
  const rows = devices.value.filter((d) => d.device_sn === sn)
  if (!rows.length) return
  editingSn.value = sn
  form.device_sn = sn
  let model = '4路220V'
  try {
    model = JSON.parse(rows[0].config_json || '{}').model || '4路220V'
  } catch {
    // ignore
  }
  form.model = model
  form.channels = rows.length
  form.lines = rows
    .sort((a, b) => a.id - b.id)
    .map((r) => {
      let cfg: any = {}
      try {
        cfg = JSON.parse(r.config_json || '{}')
      } catch {
        // ignore
      }
      return { name: r.name, type: r.type, action: cfg.cmd_off ? 'hold' : 'pulse' }
    })
  probeResult.value = null
  probeMsg.value = ''
  visible.value = true
}

const save = async () => {
  const sn = form.device_sn.trim()
  if (!sn) {
    ElMessage.warning('请填写设备 ID')
    return
  }
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }

  saving.value = true
  try {
    if (editingSn.value) {
      const olds = devices.value.filter((d) => d.device_sn === editingSn.value)
      for (const o of olds) {
        await deleteDevice(o.id)
      }
    }
    for (let i = 0; i < form.lines.length; i++) {
      const row = form.lines[i]
      const n = i + 1
      const isPulse = row.action === 'pulse'
      await createDevice({
        device_sn: sn,
        name: row.name || `第${n}路`,
        type: row.type || 'other',
        store_id: Number(storeId.value),
        protocol: 'ascii',
        config_json: JSON.stringify({
          cmd_on: isPulse ? `c${n}` : `a${n}`,
          cmd_off: isPulse ? '' : `b${n}`,
          auto_close: false,
          model: form.model
        })
      })
    }
    ElMessage.success('保存成功')
    visible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

const delSn = async (sn: string) => {
  try {
    await ElMessageBox.confirm(`确定删除设备「${sn}」的全部路？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  const olds = devices.value.filter((d) => d.device_sn === sn)
  for (const o of olds) {
    await deleteDevice(o.id)
  }
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

.cmd-text {
  font-family: monospace;
  font-size: 12px;
  color: #1a56db;
}

.probe-row {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.probe-alert {
  margin-bottom: 14px;
}

.device-form {
  margin-bottom: 4px;
}

.lines-title {
  margin: 8px 0;
  font-size: 13px;
  color: #6b7280;
}

.ml-10 {
  margin-left: 10px;
}
</style>
