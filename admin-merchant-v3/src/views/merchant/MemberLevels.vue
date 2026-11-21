<template>
  <ContentWrap title="会员等级" :body-style="{ padding: '0 18px 18px' }">
    <template #header>
      <div class="toolbar">
        <span class="summary">共 {{ levels.length }} 个等级</span>
        <div class="toolbar-actions">
          <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">新增等级</el-button>
        </div>
      </div>
    </template>

    <el-alert
      class="level-alert"
      type="info"
      :closable="false"
      show-icon
      title="折扣以万分比设置：10000=不打折（10折），9000=9折，8500=8.5折（原价85%），1000=1折（原价10%）；已被会员使用的等级不能删除，可改为停用。"
    />

    <el-table v-loading="loading" :data="levels" row-key="id" empty-text="暂无会员等级">
      <el-table-column prop="name" label="等级名称" min-width="160" />
      <el-table-column label="折扣" width="120">
        <template #default="{ row }">
          <span class="discount">{{ formatDiscount(row.discount_bp) }} 折</span>
        </template>
      </el-table-column>
      <el-table-column prop="sort" label="排序" width="90" />
      <el-table-column label="会员数" width="100">
        <template #default="{ row }">{{ Number(row.member_count || 0) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="Number(row.status) === 1 ? 'success' : 'info'" effect="light">
            {{ Number(row.status) === 1 ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <div class="row-actions">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button
              link
              type="danger"
              :loading="deletingId === Number(row.id)"
              :disabled="deletingId !== null"
              @click="remove(row)"
            >
              删除
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </ContentWrap>

  <el-dialog
    v-model="visible"
    :title="editingId !== null ? '编辑等级' : '新增等级'"
    width="440px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="86px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" maxlength="32" show-word-limit placeholder="如 金卡" />
      </el-form-item>
      <el-form-item label="折扣" prop="discount_bp">
        <el-input-number
          v-model="form.discount_bp"
          class="discount-input"
          :min="1"
          :max="10000"
          :step="100"
          :precision="0"
          controls-position="right"
        />
        <span class="hint">{{ formatDiscount(form.discount_bp) }} 折</span>
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-input-number
          v-model="form.sort"
          :min="0"
          :step="1"
          :precision="0"
          controls-position="right"
        />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        <span class="hint">{{ form.status === 1 ? '启用' : '停用' }}</span>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { useMerchantStore } from '@/store/modules/merchant'
import {
  createMemberLevel,
  deleteMemberLevel,
  getMemberLevels,
  updateMemberLevel,
  type MemberLevel
} from '@/api/merchant'

defineOptions({ name: 'MerchantMemberLevels' })

const merchantStore = useMerchantStore()

const levels = ref<MemberLevel[]>([])
const loading = ref(false)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const visible = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({
  name: '',
  discount_bp: 10000,
  sort: 0,
  status: 1
})

const rules: FormRules = {
  name: [
    { required: true, message: '请输入等级名称', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        const name = String(value ?? '').trim()
        if (!name) return callback(new Error('请输入等级名称'))
        if ([...name].length > 32) return callback(new Error('等级名称最多 32 个字'))
        callback()
      },
      trigger: 'blur'
    }
  ],
  discount_bp: [
    {
      validator: (_rule, value, callback) => {
        const discount = Number(value)
        if (!Number.isInteger(discount) || discount < 1 || discount > 10000) {
          return callback(new Error('折扣需为 1–10000 的整数（万分比）'))
        }
        callback()
      },
      trigger: 'change'
    }
  ]
}

const formatDiscount = (discountBP: number) => {
  const value = Number(discountBP)
  if (!Number.isFinite(value) || value <= 0) return '-'
  return (value / 1000).toFixed(2)
}

let loadSeq = 0

const resetForm = () => {
  Object.assign(form, { name: '', discount_bp: 10000, sort: 0, status: 1 })
}

const load = async () => {
  const seq = ++loadSeq
  loading.value = true
  try {
    const res = await getMemberLevels()
    if (seq !== loadSeq) return
    levels.value = Array.isArray(res) ? res : []
  } catch {
    if (seq === loadSeq) levels.value = []
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const openCreate = () => {
  editingId.value = null
  resetForm()
  formRef.value?.clearValidate()
  visible.value = true
}

const openEdit = (row: MemberLevel) => {
  editingId.value = Number(row.id)
  Object.assign(form, {
    name: row.name || '',
    discount_bp: Number(row.discount_bp) || 10000,
    sort: Number(row.sort) || 0,
    status: Number(row.status) === 1 ? 1 : 0
  })
  formRef.value?.clearValidate()
  visible.value = true
}

const save = async () => {
  if (saving.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  const name = form.name.trim()
  const discountBP = Number(form.discount_bp)
  if (!name) {
    ElMessage.warning('请输入等级名称')
    return
  }
  if (!Number.isInteger(discountBP) || discountBP < 1 || discountBP > 10000) {
    ElMessage.warning('折扣需为 1–10000 的整数（万分比）')
    return
  }

  const payload = {
    name,
    discount_bp: discountBP,
    sort: Number(form.sort) || 0,
    status: Number(form.status) === 1 ? 1 : 0
  }

  saving.value = true
  try {
    if (editingId.value !== null) {
      await updateMemberLevel(editingId.value, payload)
    } else {
      await createMemberLevel(payload)
    }
    ElMessage.success('保存成功')
    visible.value = false
    await load()
  } catch {
    // 请求层已统一提示错误
  } finally {
    saving.value = false
  }
}

const remove = async (row: MemberLevel) => {
  try {
    await ElMessageBox.confirm(`确定删除等级「${row.name}」？删除后不可恢复。`, '删除等级', {
      type: 'warning',
      confirmButtonText: '确定删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  deletingId.value = Number(row.id)
  try {
    await deleteMemberLevel(Number(row.id))
    ElMessage.success('已删除')
    await load()
  } catch {
    // 请求层已统一提示错误
  } finally {
    deletingId.value = null
  }
}

watch(
  () => merchantStore.storeId,
  (id) => {
    loadSeq++
    levels.value = []
    loading.value = false
    visible.value = false
    editingId.value = null
    deletingId.value = null
    resetForm()
    formRef.value?.clearValidate()
    if (!id) return
    load()
  }
)

onMounted(load)
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.summary {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.level-alert {
  margin-bottom: 14px;
}

.discount {
  font-weight: 600;
  color: #e11d48;
}

.discount-input {
  width: 160px;
}

.hint {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: center;
}

@media (width <= 768px) {
  .toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .toolbar-actions {
    justify-content: flex-end;
  }
}
</style>
