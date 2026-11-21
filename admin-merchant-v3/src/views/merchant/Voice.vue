<template>
  <div>
    <ContentWrap title="语音播报">
      <template #header>
        <div class="toolbar">
          <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="mb-12"
        title="配置本门店云喇叭的预置语音码，例如 mp3=00005。仅对本门店音响设备生效。"
      />

      <el-table :data="devices" v-loading="loading" stripe empty-text="当前门店暂无云喇叭设备，请先在设备管理中添加音响设备">
        <el-table-column label="设备" min-width="200">
          <template #default="{ row }">
            <div class="device-name">{{ row.name || deviceName(row.device_sn) }}</div>
            <div class="device-sn">{{ row.device_sn }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="light">
              {{ row.enabled ? '已启用' : '已停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始播报" min-width="140">
          <template #default="{ row }">{{ row.on_start || '未设置' }}</template>
        </el-table-column>
        <el-table-column label="结束播报" min-width="140">
          <template #default="{ row }">{{ row.on_end || '未设置' }}</template>
        </el-table-column>
        <el-table-column label="余时提醒" min-width="230">
          <template #default="{ row }">
            <el-tag
              v-for="reminder in row.reminders || []"
              :key="reminder.minutes_before_end"
              class="reminder-tag"
              effect="plain"
            >
              剩 {{ reminder.minutes_before_end }} 分钟
            </el-tag>
            <span v-if="!row.reminders?.length" class="muted">未设置</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ $index }">
            <el-button link type="primary" @click="openEdit($index)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <span class="muted">{{ devices.length }} 台云喇叭</span>
      </div>
    </ContentWrap>

    <el-dialog v-model="dialogVisible" title="编辑语音播报" width="720px" destroy-on-close>
      <el-form v-if="draft" label-width="110px">
        <el-form-item label="设备">
          <div>
            <div class="dialog-device-name">{{ draft.name || deviceName(draft.device_sn) }}</div>
            <div class="device-sn">{{ draft.device_sn }}</div>
          </div>
        </el-form-item>
        <el-form-item label="启用播报">
          <el-switch v-model="draft.enabled" />
        </el-form-item>
        <el-divider content-position="left">订单开始与结束</el-divider>
        <el-form-item label="开始使用">
          <el-input v-model="draft.on_start" placeholder="例如 mp3=00005" maxlength="64" />
        </el-form-item>
        <el-form-item label="结束使用">
          <el-input v-model="draft.on_end" placeholder="例如 mp3=00012" maxlength="64" />
        </el-form-item>
        <el-divider content-position="left">订单剩余时间提醒</el-divider>
        <div v-if="draft.reminders.length === 0" class="empty-reminders">尚未添加提醒</div>
        <div v-for="(reminder, index) in draft.reminders" :key="index" class="reminder-row">
          <el-input-number
            v-model="reminder.minutes_before_end"
            :min="1"
            :max="10080"
            controls-position="right"
          />
          <span class="minute-label">分钟后结束时播报</span>
          <el-input v-model="reminder.command" placeholder="例如 mp3=00009" maxlength="64" />
          <el-button circle text type="danger" title="删除提醒" @click="removeReminder(index)">
            ×
          </el-button>
        </div>
        <el-button class="add-reminder" :disabled="draft.reminders.length >= 10" @click="addReminder">
          添加提醒
        </el-button>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="applyDraft">完成并保存</el-button>
      </template>
    </el-dialog>

    <ContentWrap title="预置语音码对照">
      <template #header>
        <el-button link type="primary" @click="showList = !showList">
          {{ showList ? '收起' : '展开' }}
        </el-button>
      </template>
      <div v-if="showList" class="voice-list">
        <div v-for="voice in voiceList" :key="voice.code" class="voice-item">
          <code>mp3={{ voice.code }}</code>
          <span>{{ voice.text }}</span>
        </div>
      </div>
    </ContentWrap>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  getDevices,
  getVoiceConfig,
  saveVoiceConfig,
  type VoiceDevice
} from '@/api/merchant'

defineOptions({ name: 'MerchantVoiceConfig' })

const merchantStore = useMerchantStore()

const devices = ref<VoiceDevice[]>([])
const deviceNames = ref<Record<string, string>>({})
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editingIndex = ref(-1)
const draft = ref<VoiceDevice | null>(null)
const showList = ref(false)

const voiceList = [
  { code: '00001', text: '开机成功' },
  { code: '00002', text: '您好！欢迎光临' },
  { code: '00003', text: '本次时间已结束' },
  { code: '00004', text: '谢谢光临！请带好随身物品' },
  { code: '00005', text: '欢迎光临！已成功开机，请大家一起享受欢乐的旅程吧' },
  { code: '00006', text: '重要提示！本场所禁止黄赌毒等违法行为…' },
  { code: '00007', text: '续时成功！' },
  { code: '00008', text: '恭喜你续时成功！' },
  { code: '00009', text: '距本次结束时间还有30分钟，请及时续费' },
  { code: '00010', text: '距本次结束时间还有10分钟，请及时续费' },
  { code: '00011', text: '距本次结束时间仅剩5分钟，请及时续费' },
  { code: '00012', text: '本次时间已结束，请带好随身物品，欢迎下次光临' },
  { code: '00013', text: '包间距结束时间还有10分钟' },
  { code: '00014', text: '包间距结束时间还有5分钟' },
  { code: '00015', text: '包间时间已结束，系统将自动断电' },
  { code: '00016', text: '品茶时间还有10分钟结束' },
  { code: '00017', text: '包间时间还有10分钟即将结束' },
  { code: '00018', text: '包间还有5分钟即将结束' },
  { code: '00019', text: '预定时间马上结束' },
  { code: '00020', text: '欢迎光临，你的包间已通电' },
  { code: '00021', text: '包间还有10分钟即将结束' },
  { code: '00022', text: '免费使用时间即将结束' },
  { code: '00023', text: '免费使用时间即将结束，请到前台办理' },
  { code: '00024', text: '包间时间已经结束' },
  { code: '00025', text: '上班时间到了' },
  { code: '00026', text: '下班时间到了' },
  { code: '00027', text: '中午到了' },
  { code: '00028', text: '大家好！中午到了' },
  { code: '00029', text: '大家好！上班时间到了' }
]

const emptyDevice = (sn: string): VoiceDevice => ({
  device_sn: sn,
  name: '',
  enabled: false,
  on_start: '',
  on_end: '',
  reminders: []
})

const deviceName = (sn: string) => deviceNames.value[sn] || sn

const load = async () => {
  loading.value = true
  try {
    const [voiceRes, deviceRes] = await Promise.all([getVoiceConfig(), getDevices()])
    const voiceDevices = voiceRes?.devices || []
    const list = voiceDevices.map((item) => ({
      ...emptyDevice(item.device_sn),
      ...item,
      reminders: (item.reminders || []).map((r) => ({ ...r }))
    }))
    devices.value = list
    const names: Record<string, string> = {}
    voiceDevices.forEach((device: any) => {
      if (device.device_sn && device.name) names[device.device_sn] = device.name
    })
    ;(deviceRes || []).forEach((device: any) => {
      if (device.device_sn && !names[device.device_sn]) {
        names[device.device_sn] = device.name || device.device_sn
      }
    })
    deviceNames.value = names
  } finally {
    loading.value = false
  }
}

const openEdit = (index: number) => {
  editingIndex.value = index
  draft.value = JSON.parse(JSON.stringify(devices.value[index]))
  dialogVisible.value = true
}

const addReminder = () => {
  if (!draft.value || draft.value.reminders.length >= 10) return
  draft.value.reminders.push({ minutes_before_end: 30, command: '' })
}

const removeReminder = (index: number) => {
  draft.value?.reminders.splice(index, 1)
}

const applyDraft = async () => {
  if (saving.value) return
  if (!draft.value) return
  const sn = draft.value.device_sn
  if (!sn) {
    ElMessage.warning('设备 SN 不能为空')
    return
  }
  if (draft.value.enabled && (!draft.value.on_start.trim() || !draft.value.on_end.trim())) {
    ElMessage.warning('启用播报时必须填写开始和结束语音码')
    return
  }
  const minutes = draft.value.reminders.map((item) => Number(item.minutes_before_end))
  if (
    minutes.some((value) => !Number.isInteger(value) || value <= 0) ||
    new Set(minutes).size !== minutes.length
  ) {
    ElMessage.warning('提醒分钟必须为正整数且不能重复')
    return
  }
  const normalized = JSON.parse(JSON.stringify(draft.value)) as VoiceDevice
  if (editingIndex.value >= 0) {
    devices.value.splice(editingIndex.value, 1, normalized)
  }
  saving.value = true
  try {
    await saveVoiceConfig(devices.value)
    ElMessage.success('语音播报配置已保存')
    dialogVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    devices.value = []
    deviceNames.value = {}
    if (!id) return
    load()
  }
)

onMounted(load)
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-grow: 1;
  gap: 10px;
  justify-content: flex-end;
}

.mb-12 {
  margin-bottom: 12px;
}

.device-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.device-sn {
  margin-top: 2px;
  font-family: Consolas, Monaco, monospace;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.dialog-device-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.reminder-tag {
  margin: 2px 6px 2px 0;
}

.muted {
  color: var(--el-text-color-placeholder);
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}

.empty-reminders {
  padding: 0 0 12px 110px;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}

.reminder-row {
  display: grid;
  grid-template-columns: 130px 130px minmax(160px, 1fr) 32px;
  gap: 10px;
  align-items: center;
  margin: 0 0 10px 110px;
}

.minute-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.add-reminder {
  margin-left: 110px;
}

.voice-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 24px;
  max-height: 420px;
  overflow-y: auto;
}

.voice-item {
  display: grid;
  grid-template-columns: 100px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.voice-item code {
  font-size: 12px;
  color: var(--el-color-primary);
}

@media (max-width: 720px) {
  .reminder-row {
    grid-template-columns: 1fr;
    margin-left: 0;
  }

  .empty-reminders,
  .add-reminder {
    margin-left: 0;
    padding-left: 0;
  }

  .voice-list {
    grid-template-columns: 1fr;
  }
}
</style>
