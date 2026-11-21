<template>
  <div class="store-switcher">
    <el-dropdown trigger="click" @command="handleCommand">
      <span class="store-trigger">
        <Icon icon="ep:shop" :size="16" />
        <span class="store-name">{{ merchantStore.getStoreName || '选择门店' }}</span>
        <Icon icon="ep:arrow-down" :size="12" />
      </span>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="s in merchantStore.getStores"
            :key="s.id"
            :command="`store:${s.id}`"
          >
            <div class="store-option">
              <span>{{ s.name }}</span>
              <span v-if="String(s.id) === String(merchantStore.getStoreId)" class="check">✓</span>
            </div>
          </el-dropdown-item>
          <el-dropdown-item command="bind" divided>+ 添加门店邀请码</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <el-dialog v-model="bindVisible" title="绑定门店" width="420px">
      <el-input v-model="bindCode" placeholder="请输入门店邀请码" maxlength="32" />
      <div class="help">绑定后可在门店切换器中切换管理</div>
      <template #footer>
        <el-button @click="bindVisible = false">取消</el-button>
        <el-button type="primary" :loading="binding" @click="doBind">绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Icon } from '@/components/Icon'
import { useMerchantStore } from '@/store/modules/merchant'
import { bindMerchantStore } from '@/api/merchant'

defineOptions({ name: 'StoreSwitcher' })

const merchantStore = useMerchantStore()
const bindVisible = ref(false)
const bindCode = ref('')
const binding = ref(false)

const handleCommand = async (command: string) => {
  if (command === 'bind') {
    bindVisible.value = true
    return
  }
  const id = Number(command.replace('store:', ''))
  const target = merchantStore.getStores.find((s) => Number(s.id) === id)
  if (!target) return
  if (Number(merchantStore.getStoreId) === id) return
  merchantStore.setStore(id, target.name)
  ElMessage.success(`已切换到「${target.name}」`)
  window.location.reload()
}

const doBind = async () => {
  if (!bindCode.value.trim()) {
    ElMessage.warning('请输入门店邀请码')
    return
  }
  binding.value = true
  try {
    await bindMerchantStore(bindCode.value.trim().toUpperCase())
    ElMessage.success('绑定成功')
    bindVisible.value = false
    bindCode.value = ''
    await merchantStore.loadStores()
  } finally {
    binding.value = false
  }
}

onMounted(() => {
  merchantStore.loadStores()
})
</script>

<style lang="scss" scoped>
.store-switcher {
  display: inline-flex;
  align-items: center;
  margin-right: 12px;
}

.store-trigger {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  max-width: 220px;
  padding: 4px 8px;
  color: var(--top-header-text-color);
  cursor: pointer;
  border-radius: 6px;
}

.store-trigger:hover {
  background: var(--top-header-hover-color);
}

.store-name {
  max-width: 160px;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.store-option {
  display: flex;
  gap: 20px;
  align-items: center;
  justify-content: space-between;
  min-width: 160px;
}

.check {
  color: var(--el-color-primary);
}

.help {
  margin-top: 8px;
  font-size: 12px;
  color: #9ca3af;
}
</style>
