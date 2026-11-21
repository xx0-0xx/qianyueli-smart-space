<template>
  <ContentWrap title="意向客户">
    <template #header>
      <div class="toolbar">
        <el-radio-group v-model="status" @change="onFilter">
          <el-radio-button :value="-1">全部</el-radio-button>
          <el-radio-button :value="0">新线索</el-radio-button>
          <el-radio-button :value="1">已联系</el-radio-button>
          <el-radio-button :value="2">已成交</el-radio-button>
          <el-radio-button :value="3">无效</el-radio-button>
        </el-radio-group>
        <el-input
          v-model="keyword"
          placeholder="搜索姓名/手机号/公司"
          :prefix-icon="Search"
          clearable
          style="width: 230px"
          @keyup.enter="onFilter"
        />
        <el-button :icon="Refresh" @click="load">刷新</el-button>
      </div>
    </template>

    <div class="hint">来自官网「预约演示」表单的留资线索</div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="姓名" width="100" />
      <el-table-column label="手机号" width="190">
        <template #default="{ row }">
          <a :href="`tel:${row.phone}`" class="phone">{{ row.phone }}</a>
          <el-button link type="primary" size="small" @click="copy(row.phone)">复制</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="company" label="公司/门店" min-width="140" show-overflow-tooltip />
      <el-table-column prop="city" label="城市" width="90" />
      <el-table-column prop="store_type" label="类型" width="100" />
      <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="提交时间" width="170">
        <template #default="{ row }">{{ fmt(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="setStatus(row, 1)">已联系</el-button>
          <el-button link type="success" @click="setStatus(row, 2)">成交</el-button>
          <el-button link type="danger" @click="setStatus(row, 3)">无效</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-if="total > 0"
        layout="total, prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        @current-change="onPageChange"
      />
    </div>
  </ContentWrap>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ContentWrap } from '@/components/ContentWrap'
import { getLeads, updateLead } from '@/api/ssm'

defineOptions({ name: 'Leads' })

const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const status = ref(-1)
const keyword = ref('')
const loading = ref(false)

const load = async () => {
  loading.value = true
  try {
    const params: Record<string, any> = { page: page.value, page_size: size.value }
    if (status.value >= 0) params.status = status.value
    if (keyword.value.trim()) params.keyword = keyword.value.trim()
    const res = await getLeads(params)
    list.value = res.list || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const onFilter = () => {
  page.value = 1
  load()
}

const onPageChange = (p: number) => {
  page.value = p
  load()
}

const statusText = (s: number) => ['新线索', '已联系', '已成交', '无效'][s] || '未知'
const statusType = (s: number) => (['info', 'warning', 'success', 'danger'] as const)[s] || 'info'

const fmt = (t: string) =>
  t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : ''

const setStatus = async (row: any, s: number) => {
  await updateLead(row.id, { status: s, remark: row.remark })
  ElMessage.success('已更新')
  await load()
}

const copy = (text: string) => {
  navigator.clipboard.writeText(text).then(
    () => ElMessage.success('已复制'),
    () => ElMessage.warning('复制失败，请手动复制')
  )
}

onMounted(load)
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  flex-grow: 1;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
}

.hint {
  margin-bottom: 12px;
  font-size: 13px;
  color: #9ca3af;
}

.phone {
  margin-right: 6px;
  color: #1a56db;
  text-decoration: none;
}

.pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
}
</style>
