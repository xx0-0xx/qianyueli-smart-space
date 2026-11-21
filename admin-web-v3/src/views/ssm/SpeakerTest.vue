<template>
  <ContentWrap title="音响测试">
    <el-form label-width="90px">
      <el-form-item label="设备">
        <el-select v-model="deviceId" filterable placeholder="选择设备" style="width: 320px" @change="syncDevice">
          <el-option
            v-for="d in devices"
            :key="d.id"
            :label="`${d.name}（${d.device_sn}）`"
            :value="d.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="设备 SN">
        <el-input v-model="sn" placeholder="设备 SN" style="width: 320px" />
      </el-form-item>
      <el-form-item label="音频 URL">
        <el-input v-model="url" placeholder="音频 URL" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="send()">播放</el-button>
      </el-form-item>
    </el-form>

    <el-alert
      v-if="result"
      :type="result.ok ? 'success' : 'error'"
      :closable="false"
      :title="result.msg"
    />

    <div class="preset-title">预设音频</div>
    <div class="presets">
      <el-button
        v-for="f in presetFiles"
        :key="f.name"
        size="small"
        plain
        type="warning"
        @click="playPreset(f.name)"
      >
        {{ f.name }}
      </el-button>
      <span v-if="!presetFiles.length" class="muted">暂无已上传音频</span>
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import { API_BASE, getAudioFiles, getDevices, speakerTest } from '@/api/ssm'

defineOptions({ name: 'SpeakerTest' })

const devices = ref<any[]>([])
const presetFiles = ref<any[]>([])
const deviceId = ref<number | null>(null)
const sn = ref('')
const url = ref(`${API_BASE}/audio/111.mp3`)
const loading = ref(false)
const result = ref<{ ok: boolean; msg: string } | null>(null)

const syncDevice = () => {
  const device = devices.value.find((d) => d.id === deviceId.value)
  if (device) sn.value = device.device_sn
}

const load = async () => {
  const [deviceRes, files] = await Promise.all([getDevices(), getAudioFiles().catch(() => [])])
  devices.value = deviceRes.list || []
  presetFiles.value = files || []
  if (devices.value.length && !deviceId.value) {
    deviceId.value = devices.value[0].id
    syncDevice()
  }
}

const send = async (audioUrl?: string) => {
  if (!sn.value.trim()) {
    ElMessage.warning('请填写设备 SN')
    return
  }
  loading.value = true
  try {
    await speakerTest(sn.value.trim(), audioUrl || url.value)
    result.value = { ok: true, msg: `指令已下发到 ${sn.value}` }
  } catch (e: any) {
    result.value = { ok: false, msg: '失败：' + (e?.message || '') }
  } finally {
    loading.value = false
  }
}

const playPreset = (name: string) => {
  url.value = `${API_BASE}/audio/${encodeURIComponent(name)}`
  send(url.value)
}

onMounted(load)
</script>

<style lang="scss" scoped>
.preset-title {
  margin: 20px 0 10px;
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
}

.presets {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.muted {
  font-size: 13px;
  color: #9ca3af;
}
</style>
