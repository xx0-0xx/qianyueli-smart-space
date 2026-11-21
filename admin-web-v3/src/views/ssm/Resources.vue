<template>
  <ContentWrap title="房间管理">
    <template #header>
      <div class="toolbar">
        <el-select v-model="storeId" placeholder="选择门店" style="width: 180px" @change="onStoreChange">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="Number(s.id)" />
        </el-select>
        <el-select v-model="typeId" placeholder="选择房型" style="width: 180px" @change="loadRooms">
          <el-option v-for="t in types" :key="t.id" :label="t.name" :value="Number(t.id)" />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="搜索房间名称"
          :prefix-icon="Search"
          clearable
          style="width: 190px"
        />
        <el-button type="primary" :icon="Plus" :disabled="!storeId || !typeId" @click="add">
          新增房间
        </el-button>
      </div>
    </template>

    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="170" show-overflow-tooltip />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="绑定设备" min-width="220">
        <template #default="{ row }">
          <template v-if="parseIds(row.device_ids).length">
            <el-tag
              v-for="id in parseIds(row.device_ids)"
              :key="id"
              size="small"
              effect="plain"
              class="device-tag"
            >
              {{ deviceLabel(id) }}
            </el-tag>
          </template>
          <span v-else class="muted">未绑定</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openLink(row)">绑定设备</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ filtered.length }} 个房间</span>
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" title="新增房间" width="400px">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="70px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="如：101 / 大厅A" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="linkVisible" title="绑定设备" width="520px">
    <el-select v-model="linkIds" multiple filterable placeholder="选择设备" style="width: 100%">
      <el-option
        v-for="d in storeDevices"
        :key="d.id"
        :label="`${d.name}（${d.device_sn}）`"
        :value="d.id"
      />
    </el-select>
    <div class="help">同一路设备可绑定到多个房间，请按实际控制关系选择</div>
    <template #footer>
      <el-button @click="linkVisible = false">取消</el-button>
      <el-button type="primary" @click="doLink">保存</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import {
  createResource,
  deleteResource,
  getDevices,
  getResourceTypes,
  getResources,
  getStores,
  linkDevices
} from '@/api/ssm'

defineOptions({ name: 'Resources' })

const stores = ref<any[]>([])
const types = ref<any[]>([])
const rooms = ref<any[]>([])
const allDevices = ref<any[]>([])
const storeId = ref<number | null>(null)
const typeId = ref<number | null>(null)
const keyword = ref('')
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const linkVisible = ref(false)
const linkIds = ref<number[]>([])
const linkRow = ref<any>(null)
const formRef = ref<FormInstance>()
const form = reactive({ name: '' })

const rules: FormRules = {
  name: [{ required: true, message: '请输入房间名称', trigger: 'blur' }]
}

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return rooms.value
  return rooms.value.filter((r) => String(r.name || '').toLowerCase().includes(kw))
})

const storeDevices = computed(() =>
  allDevices.value.filter((d) => String(d.store_id) === String(storeId.value))
)

const parseIds = (value: string) => {
  try {
    const ids = JSON.parse(value || '[]')
    return Array.isArray(ids) ? ids : []
  } catch {
    return []
  }
}

const deviceLabel = (id: number) =>
  allDevices.value.find((d) => String(d.id) === String(id))?.name || `#${id}`

const onStoreChange = async () => {
  typeId.value = null
  types.value = []
  rooms.value = []
  if (!storeId.value) return
  const [typeList, deviceList] = await Promise.all([
    getResourceTypes(storeId.value),
    getDevices()
  ])
  types.value = typeList || []
  allDevices.value = deviceList.list || []
  if (types.value.length) {
    typeId.value = Number(types.value[0].id)
    await loadRooms()
  }
}

const loadRooms = async () => {
  if (!storeId.value || !typeId.value) {
    rooms.value = []
    return
  }
  loading.value = true
  try {
    const groups = (await getResources(storeId.value)) || []
    rooms.value = groups
      .reduce((arr: any[], g: any) => arr.concat(g.resources || []), [])
      .filter((room: any) => String(room.resource_type_id) === String(typeId.value))
  } finally {
    loading.value = false
  }
}

const add = () => {
  form.name = ''
  visible.value = true
}

const save = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await createResource({
      name: form.name,
      store_id: Number(storeId.value),
      resource_type_id: Number(typeId.value)
    })
    ElMessage.success('保存成功')
    visible.value = false
    await loadRooms()
  } finally {
    saving.value = false
  }
}

const openLink = (row: any) => {
  linkRow.value = row
  linkIds.value = parseIds(row.device_ids)
  linkVisible.value = true
}

const doLink = async () => {
  await linkDevices(linkRow.value.id, linkIds.value)
  ElMessage.success('已绑定')
  linkVisible.value = false
  await loadRooms()
}

const remove = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定删除房间「${row.name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await deleteResource(row.id)
  ElMessage.success('已删除')
  await loadRooms()
}

onMounted(async () => {
  const res = await getStores()
  stores.value = res.list || []
  if (stores.value.length) {
    storeId.value = Number(stores.value[0].id)
    await onStoreChange()
  }
})
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

.device-tag {
  margin: 2px 4px 2px 0;
}

.help {
  margin-top: 8px;
  font-size: 12px;
  color: #9ca3af;
}
</style>
