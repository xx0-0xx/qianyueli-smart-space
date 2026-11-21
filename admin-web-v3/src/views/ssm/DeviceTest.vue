<template>
  <div>
    <ContentWrap title="设备连接状态">
      <div class="row">
        <el-select v-model="deviceId" filterable placeholder="选择设备" style="width: 300px" @change="syncDevice">
          <el-option
            v-for="d in devices"
            :key="d.id"
            :label="`${d.name}（${d.device_sn}）`"
            :value="d.id"
          />
        </el-select>
        <el-button :loading="checking" @click="checkStatus">检查状态</el-button>
        <el-tag v-if="statusText" :type="statusText === '在线' ? 'success' : 'danger'">
          {{ statusText }}
        </el-tag>
        <span v-if="currentDevice" class="muted">SN：{{ currentDevice.device_sn }}</span>
      </div>
    </ContentWrap>

    <ContentWrap title="发送音频播放指令">
      <el-form label-width="90px">
        <el-form-item label="设备 SN">
          <el-input v-model="sn" placeholder="设备 SN" style="width: 320px" />
        </el-form-item>
        <el-form-item label="音频 URL">
          <el-input v-model="testUrl" placeholder="音频 URL" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="sending" @click="sendSpeaker">发送播放指令</el-button>
        </el-form-item>
      </el-form>
      <el-alert v-if="sendResult" type="info" :closable="false" :title="sendResult" />
    </ContentWrap>

    <ContentWrap title="发送控制指令">
      <el-form label-width="90px">
        <el-form-item label="原始指令">
          <div class="row">
            <el-input v-model="rawCmd" placeholder="如：a1 / b1 / qk / mp3=00001" style="width: 260px" />
            <el-button type="primary" :loading="sendingCmd" @click="sendRaw">发送</el-button>
          </div>
        </el-form-item>
      </el-form>

      <div class="cmd-title">ASCII 成品控制器</div>
      <div class="cmd-grid">
        <el-button v-for="cmd in asciiCommands" :key="cmd.label" size="small" @click="sendCmd(cmd.value)">
          {{ cmd.label }}
        </el-button>
      </div>

      <div class="cmd-title">JSON 自制固件</div>
      <div class="cmd-grid">
        <el-button v-for="cmd in jsonCommands" :key="cmd" size="small" @click="sendCmd(cmd)">
          {{ cmd }}
        </el-button>
      </div>

      <el-alert v-if="cmdResult" type="info" :closable="false" :title="cmdResult" class="mt-12" />
    </ContentWrap>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import { API_BASE, controlDevice, getDevices, speakerTest } from '@/api/ssm'

defineOptions({ name: 'DeviceTest' })

const devices = ref<any[]>([])
const deviceId = ref<number | null>(null)
const sn = ref('')
const testUrl = ref(`${API_BASE}/audio/111.mp3`)
const rawCmd = ref('')
const checking = ref(false)
const sending = ref(false)
const sendingCmd = ref(false)
const statusText = ref('')
const sendResult = ref('')
const cmdResult = ref('')

const jsonCommands = [
  'light_on',
  'light_off',
  'door_open',
  'door_close',
  'ac_on',
  'ac_off',
  'exhaust_on',
  'exhaust_off',
  'session_start',
  'session_end',
  'reboot'
]

const asciiCommands = [
  { label: 'a1 开', value: 'a1' },
  { label: 'b1 关', value: 'b1' },
  { label: 'a2 开', value: 'a2' },
  { label: 'b2 关', value: 'b2' },
  { label: 'qk 全开', value: 'qk' },
  { label: 'qg 全关', value: 'qg' },
  { label: 'qa 查状态', value: 'qa' },
  { label: 'qc 查详情', value: 'qc' },
  { label: 'bbh 查版本', value: 'bbh' }
]

const currentDevice = computed(() => devices.value.find((d) => d.id === deviceId.value))

const syncDevice = () => {
  if (currentDevice.value) {
    sn.value = currentDevice.value.device_sn
    statusText.value = ''
  }
}

const load = async () => {
  const res = await getDevices()
  devices.value = res.list || []
  if (devices.value.length && !deviceId.value) {
    deviceId.value = devices.value[0].id
    syncDevice()
  }
}

const checkStatus = async () => {
  if (!deviceId.value) {
    ElMessage.warning('请先选择设备')
    return
  }
  checking.value = true
  try {
    await load()
    statusText.value = currentDevice.value?.status === 1 ? '在线' : '离线'
  } finally {
    checking.value = false
  }
}

const sendSpeaker = async () => {
  if (!sn.value.trim()) {
    ElMessage.warning('请填写设备 SN')
    return
  }
  sending.value = true
  try {
    const res: any = await speakerTest(sn.value.trim(), testUrl.value)
    sendResult.value = '已下发：' + JSON.stringify(res)
    ElMessage.success('指令已发送')
  } catch (e: any) {
    sendResult.value = '失败：' + (e?.message || '')
  } finally {
    sending.value = false
  }
}

const sendRaw = () => {
  if (!rawCmd.value.trim()) {
    ElMessage.warning('请输入指令')
    return
  }
  sendCmd(rawCmd.value.trim())
}

const sendCmd = async (cmd: string) => {
  if (!deviceId.value) {
    ElMessage.warning('请先选择设备')
    return
  }
  sendingCmd.value = true
  try {
    await controlDevice(deviceId.value, cmd)
    cmdResult.value = '已发送：' + cmd
    ElMessage.success('指令已发送')
  } catch (e: any) {
    cmdResult.value = '失败：' + cmd + '（' + (e?.message || '') + '）'
  } finally {
    sendingCmd.value = false
  }
}

onMounted(load)
</script>

<style lang="scss" scoped>
.row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.muted {
  font-size: 13px;
  color: #9ca3af;
}

.cmd-title {
  margin: 16px 0 8px;
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
}

.cmd-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.mt-12 {
  margin-top: 12px;
}
</style>
