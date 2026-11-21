<template>
  <ContentWrap title="房型管理">
    <template #header>
      <div class="toolbar">
        <el-select v-model="storeId" placeholder="选择门店" style="width: 200px" @change="loadTypes">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="Number(s.id)" />
        </el-select>
        <el-input
          v-model="keyword"
          placeholder="搜索房型名称"
          :prefix-icon="Search"
          clearable
          style="width: 200px"
        />
        <el-button type="primary" :icon="Plus" :disabled="!storeId" @click="add">
          新增房型
        </el-button>
      </div>
    </template>

    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="170" show-overflow-tooltip />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <span class="muted">共 {{ filtered.length }} 个房型</span>
    </div>
  </ContentWrap>

  <el-dialog v-model="visible" title="新增房型" width="420px">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="70px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="如：标准包间 / 大包" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" placeholder="可选" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import {
  createResourceType,
  deleteResourceType,
  getResourceTypes,
  getStores
} from '@/api/ssm'

defineOptions({ name: 'ResourceTypes' })

const stores = ref<any[]>([])
const types = ref<any[]>([])
const storeId = ref<number | null>(null)
const keyword = ref('')
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({ name: '', description: '' })

const rules: FormRules = {
  name: [{ required: true, message: '请输入房型名称', trigger: 'blur' }]
}

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return types.value
  return types.value.filter((t) => String(t.name || '').toLowerCase().includes(kw))
})

const loadTypes = async () => {
  if (!storeId.value) {
    types.value = []
    return
  }
  loading.value = true
  try {
    types.value = (await getResourceTypes(storeId.value)) || []
  } finally {
    loading.value = false
  }
}

const add = () => {
  form.name = ''
  form.description = ''
  visible.value = true
}

const save = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await createResourceType({
      name: form.name,
      description: form.description,
      store_id: Number(storeId.value)
    })
    ElMessage.success('保存成功')
    visible.value = false
    await loadTypes()
  } finally {
    saving.value = false
  }
}

const remove = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定删除房型「${row.name}」？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  await deleteResourceType(row.id)
  ElMessage.success('已删除')
  await loadTypes()
}

onMounted(async () => {
  const res = await getStores()
  stores.value = res.list || []
  if (stores.value.length) {
    storeId.value = Number(stores.value[0].id)
    await loadTypes()
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
</style>
