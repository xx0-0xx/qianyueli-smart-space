<template>
  <ContentWrap title="房型管理" :body-style="{ padding: '0 18px 18px' }">
    <div class="toolbar">
      <span class="toolbar-summary">共 {{ resourceTypes.length }} 个房型</span>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增房型</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="resourceTypes"
      row-key="id"
      empty-text="暂未添加房型"
    >
      <el-table-column prop="id" label="ID" width="100" />
      <el-table-column prop="name" label="名称" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button
            link
            type="danger"
            :loading="deletingId === row.id"
            :disabled="deletingId !== null"
            @click="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>

      <template #empty>
        <el-empty description="还没有房型，点击右上角新增房型" :image-size="90" />
      </template>
    </el-table>

    <el-dialog
      v-model="createVisible"
      title="新增房型"
      width="420px"
      append-to-body
      style="max-width: 92vw"
    >
      <el-form label-width="80px" @submit.prevent>
        <el-form-item label="房型名称" required>
          <el-input
            v-model="newTypeName"
            maxlength="30"
            show-word-limit
            placeholder="如：大床房"
            @keyup.enter="handleCreate"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">保存</el-button>
      </template>
    </el-dialog>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  createResourceType,
  deleteResourceType,
  getResourceTypes
} from '@/api/merchant'

defineOptions({ name: 'MerchantResourceTypes' })

interface ResourceTypeItem {
  id: number
  name: string
  [key: string]: any
}

const merchantStore = useMerchantStore()

const resourceTypes = ref<ResourceTypeItem[]>([])
const loading = ref(false)
const creating = ref(false)
const createVisible = ref(false)
const newTypeName = ref('')
const deletingId = ref<number | null>(null)

let loadSeq = 0

const toList = (res: any): any[] => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.list)) return res.list
  if (Array.isArray(res?.data)) return res.data
  return []
}

const loadTypes = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const res = await getResourceTypes()
    if (seq !== loadSeq) return
    resourceTypes.value = toList(res) as ResourceTypeItem[]
  } catch {
    return
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const openCreate = () => {
  newTypeName.value = ''
  createVisible.value = true
}

const handleCreate = async () => {
  if (creating.value) return
  const name = newTypeName.value.trim()
  if (!name) {
    ElMessage.warning('请输入房型名称')
    return
  }
  creating.value = true
  try {
    await createResourceType({ name })
    ElMessage.success('房型已添加')
    createVisible.value = false
    newTypeName.value = ''
    await loadTypes()
  } catch {
    return
  } finally {
    creating.value = false
  }
}

const handleDelete = async (row: ResourceTypeItem) => {
  if (deletingId.value !== null) return
  try {
    await ElMessageBox.confirm(
      `确认删除房型「${row.name}」？删除后不可恢复。`,
      '删除房型',
      {
        type: 'warning',
        confirmButtonText: '确认删除',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }
  deletingId.value = row.id
  try {
    await deleteResourceType(row.id)
    ElMessage.success('房型已删除')
    await loadTypes()
  } catch {
    return
  } finally {
    deletingId.value = null
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    createVisible.value = false
    newTypeName.value = ''
    if (!id) {
      resourceTypes.value = []
      return
    }
    loadTypes()
  }
)

onMounted(() => {
  loadTypes()
})
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.toolbar-summary {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 640px) {
  .toolbar {
    align-items: stretch;
  }
}
</style>
