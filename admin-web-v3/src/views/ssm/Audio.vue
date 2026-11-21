<template>
  <ContentWrap title="音频文件">
    <template #header>
      <div class="toolbar">
        <input ref="fileInput" type="file" accept=".mp3" class="file-input" @change="onFileChange" />
        <span class="file-name">{{ selectedFile?.name || '未选择文件' }}</span>
        <el-button :disabled="!selectedFile" @click="fileInput?.click()">选择 MP3</el-button>
        <el-button type="primary" :disabled="!selectedFile" :loading="uploading" @click="upload">
          上传
        </el-button>
      </div>
    </template>

    <el-table :data="files" v-loading="loading" stripe>
      <el-table-column prop="name" label="文件名" min-width="220" />
      <el-table-column label="大小" width="110">
        <template #default="{ row }">{{ (Number(row.size) / 1024 / 1024).toFixed(1) }} MB</template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="play(row.name)">试听</el-button>
          <el-button link type="success" @click="playOnDevice(row.name)">设备播放</el-button>
          <el-button link type="danger" @click="remove(row.name)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ files.length }} 个音频文件</span>
    </div>
  </ContentWrap>

  <ContentWrap title="设备播放测试">
    <div class="play-row">
      <span class="label">目标设备 SN</span>
      <el-select v-model="deviceSN" filterable placeholder="选择设备" style="width: 260px">
        <el-option v-for="sn in deviceSNs" :key="sn" :label="sn" :value="sn" />
      </el-select>
      <span class="muted">选择音频文件后点「设备播放」即可下发</span>
    </div>
  </ContentWrap>

  <ContentWrap title="设备音频配置">
    <div v-if="!audioConfigs.length" class="empty">暂无设备音频配置</div>
    <div v-for="cfg in audioConfigs" :key="cfg.device_sn" class="config-row">
      <div class="config-sn">
        <div class="sn">{{ cfg.device_sn }}</div>
        <div class="room">{{ cfg.room_name || '未绑定房间' }}</div>
      </div>
      <el-select v-model="cfg._start" placeholder="开始播报：无" clearable style="width: 210px">
        <el-option v-for="f in files" :key="f.name" :label="f.name" :value="f.name" />
      </el-select>
      <el-select v-model="cfg._end" placeholder="结束播报：无" clearable style="width: 210px">
        <el-option v-for="f in files" :key="f.name" :label="f.name" :value="f.name" />
      </el-select>
      <el-button type="primary" plain @click="saveCfg(cfg)">保存</el-button>
    </div>
  </ContentWrap>

  <audio ref="player" class="hidden-audio" controls></audio>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ContentWrap } from '@/components/ContentWrap'
import {
  API_BASE,
  deleteAudio,
  getAudioConfigs,
  getAudioFiles,
  getDevices,
  saveAudioConfig,
  speakerTest,
  uploadAudio
} from '@/api/ssm'

defineOptions({ name: 'Audio' })

const files = ref<any[]>([])
const audioConfigs = ref<any[]>([])
const devices = ref<any[]>([])
const selectedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement>()
const player = ref<HTMLAudioElement>()
const deviceSN = ref('')
const loading = ref(false)
const uploading = ref(false)

const deviceSNs = computed(() => [...new Set(devices.value.map((d) => d.device_sn))])

const onFileChange = (e: Event) => {
  selectedFile.value = (e.target as HTMLInputElement).files?.[0] || null
}

const load = async () => {
  loading.value = true
  try {
    const [fileList, deviceList] = await Promise.all([
      getAudioFiles().catch(() => []),
      getDevices().catch(() => ({ list: [] }))
    ])
    files.value = fileList || []
    devices.value = deviceList?.list || (Array.isArray(deviceList) ? deviceList : [])
    if (!deviceSN.value && deviceSNs.value.length) {
      deviceSN.value = deviceSNs.value[0]
    }
  } finally {
    loading.value = false
  }
}

const loadConfigs = async () => {
  const list = (await getAudioConfigs().catch(() => [])) || []
  audioConfigs.value = list.map((cfg: any) => ({
    ...cfg,
    _start: cfg.start_audio || '',
    _end: cfg.end_audio || ''
  }))
}

const upload = async () => {
  if (!selectedFile.value) return
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', selectedFile.value)
    await uploadAudio(fd)
    ElMessage.success('上传成功')
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
    await load()
  } finally {
    uploading.value = false
  }
}

const play = (name: string) => {
  if (!player.value) return
  player.value.src = `${API_BASE}/audio/${encodeURIComponent(name)}`
  player.value.play()
}

const playOnDevice = async (name: string) => {
  if (!deviceSN.value) {
    ElMessage.warning('请先选择设备 SN')
    return
  }
  await speakerTest(deviceSN.value, `${API_BASE}/audio/${encodeURIComponent(name)}`)
  ElMessage.success('已下发到设备')
}

const remove = async (name: string) => {
  try {
    await ElMessageBox.confirm(`确定删除音频「${name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await deleteAudio(name)
  ElMessage.success('已删除')
  await load()
}

const saveCfg = async (cfg: any) => {
  await saveAudioConfig(cfg.device_sn, {
    start_audio: cfg._start || '',
    end_audio: cfg._end || ''
  })
  ElMessage.success('已保存')
}

onMounted(() => {
  load()
  loadConfigs()
})
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-grow: 1;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
}

.file-input {
  display: none;
}

.file-name {
  max-width: 220px;
  overflow: hidden;
  font-size: 13px;
  color: #6b7280;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}

.muted {
  color: #9ca3af;
}

.play-row {
  display: flex;
  gap: 12px;
  align-items: center;
}

.label {
  font-size: 13px;
  color: #6b7280;
}

.empty {
  padding: 8px 0;
  font-size: 13px;
  color: #9ca3af;
}

.config-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f0f2f5;

  &:last-child {
    border-bottom: none;
  }
}

.config-sn {
  min-width: 180px;
}

.sn {
  font-family: monospace;
  font-size: 14px;
  font-weight: 700;
  color: #1f2937;
}

.room {
  margin-top: 2px;
  font-size: 12px;
  color: #9ca3af;
}

.hidden-audio {
  display: none;
}
</style>
