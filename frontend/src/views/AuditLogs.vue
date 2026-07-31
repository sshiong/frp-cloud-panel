<template>
  <div class="audit-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>审计日志</span>
          <div class="header-actions">
            <el-button @click="loadLogs" :loading="loading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 筛选条件 -->
      <el-form :inline="true" :model="filters" style="margin-bottom: 20px;">
        <el-form-item label="操作类型">
          <el-select v-model="filters.action" placeholder="全部" clearable>
            <el-option label="登录" value="login" />
            <el-option label="创建映射" value="create_mapping" />
            <el-option label="删除映射" value="delete_mapping" />
            <el-option label="设置Token" value="set_cf_token" />
            <el-option label="DNS操作" value="dns_" />
            <el-option label="备份" value="backup" />
          </el-select>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="filters.resource" placeholder="全部" clearable>
            <el-option label="用户" value="user" />
            <el-option label="映射" value="proxy_mapping" />
            <el-option label="客户端" value="client" />
            <el-option label="域名" value="domain" />
            <el-option label="DNS记录" value="dns_record" />
            <el-option label="证书" value="certificate" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filters.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadLogs">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 日志列表 -->
      <el-table :data="logs" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="action" label="操作" width="120">
          <template #default="{ row }">
            <el-tag :type="getActionType(row.action)">
              {{ getActionText(row.action) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源" width="100" />
        <el-table-column prop="detail" label="详情" />
        <el-table-column prop="ip" label="IP地址" width="120" />
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadLogs"
        @current-change="loadLogs"
        style="margin-top: 20px; justify-content: flex-end;"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { getLogs } from '../api/logs'

const logs = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filters = reactive({
  action: '',
  resource: '',
  dateRange: null as any,
})

onMounted(() => {
  loadLogs()
})

const loadLogs = async () => {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      page_size: pageSize.value,
    }

    if (filters.action) {
      params.action = filters.action
    }

    if (filters.resource) {
      params.resource = filters.resource
    }

    if (filters.dateRange) {
      params.start_date = filters.dateRange[0].toISOString()
      params.end_date = filters.dateRange[1].toISOString()
    }

    const res = await getLogs(params)
    logs.value = res.data.items
    total.value = res.data.total
  } catch (error) {
    console.error('Failed to load logs:', error)
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.action = ''
  filters.resource = ''
  filters.dateRange = null
  loadLogs()
}

const getActionType = (action: string) => {
  if (action.includes('login')) return 'primary'
  if (action.includes('create')) return 'success'
  if (action.includes('delete')) return 'danger'
  if (action.includes('update')) return 'warning'
  return 'info'
}

const getActionText = (action: string) => {
  const map: Record<string, string> = {
    login: '登录',
    frps_login: 'FRPS登录',
    create_mapping: '创建映射',
    delete_mapping: '删除映射',
    set_cf_token: '设置Token',
    delete_cf_token: '删除Token',
    create_dns_record: '创建DNS记录',
    update_dns_record: '更新DNS记录',
    delete_dns_record: '删除DNS记录',
    renew_cert: '续期证书',
    create_backup: '创建备份',
    restore_backup: '恢复备份',
    delete_backup: '删除备份',
  }
  return map[action] || action
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString()
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}
</style>
