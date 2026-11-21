<template>
  <div>
    <el-row :gutter="16">
      <el-col v-for="item in stats" :key="item.label" :xs="12" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-card__body">
            <div class="stat-card__icon" :style="{ background: item.bg, color: item.color }">
              <el-icon :size="22"><component :is="item.icon" /></el-icon>
            </div>
            <div>
              <div class="stat-card__label">{{ item.label }}</div>
              <div class="stat-card__value">{{ item.value }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :xs="24" :md="16">
        <ContentWrap title="门店概览">
          <el-table :data="stores" v-loading="loading" stripe>
            <el-table-column prop="name" label="门店" min-width="140" />
            <el-table-column prop="address" label="地址" min-width="200" show-overflow-tooltip />
            <el-table-column prop="phone" label="电话" width="140" />
            <el-table-column label="邀请码" width="140">
              <template #default="{ row }">
                <el-tag v-if="row.invite_code" effect="plain">{{ row.invite_code }}</el-tag>
                <span v-else class="muted">未生成</span>
              </template>
            </el-table-column>
          </el-table>
        </ContentWrap>
      </el-col>

      <el-col :xs="24" :md="8">
        <ContentWrap title="快捷入口">
          <div class="quick-list">
            <div class="quick-item" @click="router.push('/stores')">
              <el-icon :size="20"><Shop /></el-icon>
              <div>
                <div class="quick-title">门店管理</div>
                <div class="quick-desc">新增门店、邀请码、二维码</div>
              </div>
            </div>
            <div class="quick-item" @click="router.push('/devices')">
              <el-icon :size="20"><Cpu /></el-icon>
              <div>
                <div class="quick-title">设备管理</div>
                <div class="quick-desc">设备 SN、在线状态、远程控制</div>
              </div>
            </div>
            <div class="quick-item" @click="router.push('/resources')">
              <el-icon :size="20"><OfficeBuilding /></el-icon>
              <div>
                <div class="quick-title">房型 / 房间</div>
                <div class="quick-desc">房型配置、房间与设备绑定</div>
              </div>
            </div>
            <div class="quick-item" @click="router.push('/audio')">
              <el-icon :size="20"><Headset /></el-icon>
              <div>
                <div class="quick-title">音频 / 语音</div>
                <div class="quick-desc">音频上传、设备播放、播报规则</div>
              </div>
            </div>
            <div class="quick-item" @click="router.push('/leads')">
              <el-icon :size="20"><UserFilled /></el-icon>
              <div>
                <div class="quick-title">意向客户</div>
                <div class="quick-desc">官网留资线索跟进</div>
              </div>
            </div>
            <div class="quick-item" @click="router.push('/report')">
              <el-icon :size="20"><DataAnalysis /></el-icon>
              <div>
                <div class="quick-title">数据报表</div>
                <div class="quick-desc">门店营收、订单、退款统计</div>
              </div>
            </div>
          </div>
        </ContentWrap>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Cpu,
  DataAnalysis,
  Headset,
  OfficeBuilding,
  Shop,
  Tickets,
  User,
  UserFilled,
  Wallet
} from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { getOverview, getStores } from '@/api/ssm'

defineOptions({ name: 'Dashboard' })

const router = useRouter()
const loading = ref(false)
const stores = ref<any[]>([])
const overview = ref({
  total_stores: 0,
  total_users: 0,
  total_orders: 0,
  total_revenue: 0
})

const stats = computed(() => [
  {
    label: '门店总数',
    value: overview.value.total_stores,
    icon: Shop,
    bg: '#e8f3ff',
    color: '#1a56db'
  },
  {
    label: '用户总数',
    value: overview.value.total_users,
    icon: User,
    bg: '#e9f8ef',
    color: '#16a34a'
  },
  {
    label: '订单总数',
    value: overview.value.total_orders,
    icon: Tickets,
    bg: '#fff4e6',
    color: '#f59e0b'
  },
  {
    label: '累计营收',
    value: `¥${Number(overview.value.total_revenue || 0).toLocaleString()}`,
    icon: Wallet,
    bg: '#fdecef',
    color: '#e11d48'
  }
])

const load = async () => {
  loading.value = true
  try {
    const [statsRes, storesRes] = await Promise.all([getOverview(), getStores()])
    overview.value = statsRes
    stores.value = storesRes.list || []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style lang="scss" scoped>
.stat-card {
  margin-bottom: 16px;
  border: none;

  &__body {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  &__icon {
    display: flex;
    width: 46px;
    height: 46px;
    align-items: center;
    justify-content: center;
    border-radius: 12px;
  }

  &__label {
    font-size: 13px;
    color: #6b7280;
  }

  &__value {
    margin-top: 4px;
    font-size: 24px;
    font-weight: 700;
    color: #111827;
  }
}

.quick-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.quick-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px;
  cursor: pointer;
  border-radius: 10px;
  transition: background 0.2s;

  &:hover {
    background: #f3f6fb;
  }
}

.quick-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.quick-desc {
  margin-top: 2px;
  font-size: 12px;
  color: #9ca3af;
}

.muted {
  color: #c0c4cc;
}
</style>
