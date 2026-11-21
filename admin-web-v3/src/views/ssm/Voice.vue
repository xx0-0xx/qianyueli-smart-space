<template>
  <div>
    <el-alert
      v-if="needsReconfigure"
      type="warning"
      :closable="false"
      show-icon
      class="mb-12"
      title="语音配置已改为按设备 SN 管理，请重新为每台控制器配置播报规则。旧全局设置不会自动迁移。"
    />

    <ContentWrap title="语音播报">
      <template #header>
        <div class="toolbar">
          <el-button :icon="Plus" type="primary" @click="openCreate">新增设备</el-button>
          <el-button :loading="loading" @click="load">重载</el-button>
          <el-button type="primary" :loading="saving" @click="saveAll">保存全部</el-button>
        </div>
      </template>

      <el-table :data="devices" v-loading="loading" stripe empty-text="尚未配置设备语音">
        <el-table-column prop="device_sn" label="设备 SN" min-width="170" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '已启用' : '已停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始播报" min-width="150">
          <template #default="{ row }">{{ row.on_start || '未设置' }}</template>
        </el-table-column>
        <el-table-column label="结束播报" min-width="150">
          <template #default="{ row }">{{ row.on_end || '未设置' }}</template>
        </el-table-column>
        <el-table-column label="余时提醒" min-width="240">
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
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ $index }">
            <el-button link type="primary" @click="openEdit($index)">编辑</el-button>
            <el-button link type="danger" @click="removeDevice($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <span class="muted">{{ devices.length }} 台设备</span>
      </div>
    </ContentWrap>

    <el-dialog
      v-model="dialogVisible"
      :title="editingIndex < 0 ? '新增设备语音' : '编辑设备语音'"
      width="720px"
      destroy-on-close
    >
      <el-form :model="draft" label-width="120px">
        <el-form-item label="设备 SN" required>
          <el-input
            v-model="draft.device_sn"
            maxlength="64"
            show-word-limit
            placeholder="输入设备管理中的 ASCII 控制器 SN"
            :disabled="editingIndex >= 0"
          />
          <div class="help">SN 必须对应设备管理中已添加的 ASCII 控制器</div>
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
        <el-button type="primary" @click="applyDraft">完成</el-button>
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
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { getVoiceConfig, saveVoiceConfig, type VoiceDevice } from '@/api/ssm'

defineOptions({ name: 'Voice' })

const devices = ref<VoiceDevice[]>([])
const loading = ref(false)
const saving = ref(false)
const needsReconfigure = ref(false)
const dialogVisible = ref(false)
const editingIndex = ref(-1)
const showList = ref(false)

const emptyDevice = (): VoiceDevice => ({
  device_sn: '',
  enabled: true,
  on_start: '',
  on_end: '',
  reminders: []
})

const draft = reactive<VoiceDevice>(emptyDevice())

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

const load = async () => {
  loading.value = true
  try {
    const data = (await getVoiceConfig()) || {}
    if (!Array.isArray(data.devices)) {
      devices.value = []
      needsReconfigure.value = Boolean(
        data.enabled || data.on_start || data.on_end || data.on_30min || data.on_10min || data.on_5min
      )
      return
    }
    needsReconfigure.value = false
    devices.value = data.devices.map((device) => ({
      ...emptyDevice(),
      ...device,
      reminders: Array.isArray(device.reminders)
        ? device.reminders.map((r) => ({ ...r }))
        : []
    }))
  } catch {
    ElMessage.error('语音配置加载失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  editingIndex.value = -1
  Object.assign(draft, emptyDevice())
  dialogVisible.value = true
}

const openEdit = (index: number) => {
  editingIndex.value = index
  Object.assign(draft, JSON.parse(JSON.stringify(devices.value[index])))
  dialogVisible.value = true
}

const addReminder = () => {
  if (draft.reminders.length >= 10) return
  draft.reminders.push({ minutes_before_end: 30, command: '' })
}

const removeReminder = (index: number) => {
  draft.reminders.splice(index, 1)
}

const applyDraft = () => {
  const sn = draft.device_sn.trim()
  if (!sn) {
    ElMessage.warning('请填写设备 SN')
    return
  }
  const minutes = draft.reminders.map((item) => Number(item.minutes_before_end))
  if (
    minutes.some((value) => !Number.isInteger(value) || value <= 0) ||
    new Set(minutes).size !== minutes.length
  ) {
    ElMessage.warning('提醒分钟必须为正整数且不能重复')
    return
  }
  if (draft.reminders.length > 10) {
    ElMessage.warning('每台设备最多添加 10 个提醒')
    return
  }
  const normalized = { ...JSON.parse(JSON.stringify(draft)), device_sn: sn }
  if (editingIndex.value < 0) {
    if (devices.value.some((item) => item.device_sn === sn)) {
      ElMessage.warning('该 SN 已存在，请直接编辑对应设备')
      return
    }
    devices.value.push(normalized)
  } else {
    devices.value.splice(editingIndex.value, 1, normalized)
  }
  dialogVisible.value = false
}

const removeDevice = async (index: number) => {
  try {
    await ElMessageBox.confirm(
      `确定删除设备 ${devices.value[index].device_sn} 的语音配置？`,
      '删除配置',
      { type: 'warning' }
    )
  } catch {
    return
  }
  devices.value.splice(index, 1)
}

const saveAll = async () => {
  if (saving.value) return
  saving.value = true
  try {
    await saveVoiceConfig(devices.value)
    needsReconfigure.value = false
    ElMessage.success('语音配置已保存')
    await load()
  } finally {
    saving.value = false
  }
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

.reminder-tag {
  margin: 2px 6px 2px 0;
}

.help {
  width: 100%;
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: #9ca3af;
}

.empty-reminders {
  padding: 0 0 12px 120px;
  font-size: 13px;
  color: #9ca3af;
}

.reminder-row {
  display: grid;
  grid-template-columns: 130px 130px minmax(160px, 1fr) 32px;
  gap: 10px;
  align-items: center;
  margin: 0 0 10px 120px;
}

.minute-label {
  font-size: 13px;
  color: #6b7280;
  white-space: nowrap;
}

.add-reminder {
  margin-left: 120px;
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
  color: #596579;
}

.voice-item code {
  font-size: 12px;
  color: #1a56db;
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
