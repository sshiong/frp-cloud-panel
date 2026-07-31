<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>映射总数</span>
          </template>
          <div class="stat-value">{{ stats.totalMappings }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>运行中</span>
          </template>
          <div class="stat-value text-success">{{ stats.runningMappings }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>客户端数量</span>
          </template>
          <div class="stat-value">{{ stats.totalClients }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>域名数量</span>
          </template>
          <div class="stat-value">{{ stats.totalDomains }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>最近映射</span>
          </template>
          <el-table :data="recentMappings" style="width: 100%">
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="type" label="类型" width="80" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>最近日志</span>
          </template>
          <el-table :data="recentLogs" style="width: 100%">
            <el-table-column prop="action" label="操作" />
            <el-table-column prop="resource" label="资源" />
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMappings } from '../api/mappings'
import { getClients } from '../api/clients'
import { getDomains } from '../api/domains'
import { getLogs } from '../api/logs'

const stats = ref({
  totalMappings: 0,
  runningMappings: 0,
  totalClients: 0,
  totalDomains: 0,
})

const recentMappings = ref([])
const recentLogs = ref([])

onMounted(async () => {
  await loadStats()
})

const loadStats = async () => {
  try {
    const [mappingsRes, clientsRes, domainsRes, logsRes] = await Promise.all([
      getMappings({ page_size: 5 }),
      getClients(),
      getDomains(),
      getLogs({ page_size: 5 }),
    ])

    stats.value.totalMappings = mappingsRes.data.total
    stats.value.runningMappings = mappingsRes.data.items.filter(
      (m: any) => m.status === 'running'
    ).length
    stats.value.totalClients = clientsRes.data.total
    stats.value.totalDomains = domainsRes.data.length

    recentMappings.value = mappingsRes.data.items
    recentLogs.value = logsRes.data.items
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    running: 'success',
    pending_apply: 'warning',
    offline: 'info',
    config_error: 'danger',
    disabled: 'info',
  }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    running: '运行中',
    pending_apply: '待应用',
    offline: '离线',
    config_error: '配置错误',
    disabled: '已禁用',
  }
  return map[status] || status
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString()
}
</script>

<style scoped>
.stat-value {
  font-size: 32px;
  font-weight: bold;
  text-align: center;
  color: #303133;
}

.text-success {
  color: #67c23a;
}
</style>
