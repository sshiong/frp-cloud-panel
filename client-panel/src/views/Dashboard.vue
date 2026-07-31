<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span>我的映射</span>
          </template>
          <div class="stat-value">{{ stats.totalMappings }}</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span>运行中</span>
          </template>
          <div class="stat-value text-success">{{ stats.runningMappings }}</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span>FRPC 状态</span>
          </template>
          <div class="stat-value" :class="frpcRunning ? 'text-success' : 'text-danger'">
            {{ frpcRunning ? '运行中' : '已停止' }}
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>我的映射列表</span>
          </template>
          <el-table :data="mappings" style="width: 100%">
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
            <span>快速操作</span>
          </template>
          <div class="quick-actions">
            <el-button type="primary" @click="$router.push('/mappings')">
              <el-icon><Plus /></el-icon>
              创建映射
            </el-button>
            <el-button :type="frpcRunning ? 'danger' : 'success'" @click="toggleFRPC">
              <el-icon><VideoPlay /></el-icon>
              {{ frpcRunning ? '停止 FRPC' : '启动 FRPC' }}
            </el-button>
            <el-button @click="$router.push('/logs')">
              <el-icon><Document /></el-icon>
              查看日志
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMappings } from '../api/mappings'

const stats = ref({
  totalMappings: 0,
  runningMappings: 0,
})

const mappings = ref([])
const frpcRunning = ref(false)

onMounted(() => {
  loadStats()
})

const loadStats = async () => {
  try {
    const res = await getMappings({ page_size: 5 })
    stats.value.totalMappings = res.data.total
    stats.value.runningMappings = res.data.items.filter(
      (m: any) => m.status === 'running'
    ).length
    mappings.value = res.data.items
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const toggleFRPC = () => {
  // TODO: 实现 FRPC 启停功能
  frpcRunning.value = !frpcRunning.value
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    running: 'success',
    pending_apply: 'warning',
    offline: 'info',
    config_error: 'danger',
  }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    running: '运行中',
    pending_apply: '待应用',
    offline: '离线',
    config_error: '配置错误',
  }
  return map[status] || status
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

.text-danger {
  color: #f56c6c;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
