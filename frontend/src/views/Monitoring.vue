<template>
  <div class="monitoring">
    <el-row :gutter="20">
      <!-- 系统状态 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>系统状态</span>
          </template>
          <div class="stat-item">
            <span class="label">CPU 使用率</span>
            <el-progress :percentage="stats.cpu" :color="getProgressColor(stats.cpu)" />
          </div>
          <div class="stat-item">
            <span class="label">内存使用率</span>
            <el-progress :percentage="stats.memory" :color="getProgressColor(stats.memory)" />
          </div>
          <div class="stat-item">
            <span class="label">磁盘使用率</span>
            <el-progress :percentage="stats.disk" :color="getProgressColor(stats.disk)" />
          </div>
        </el-card>
      </el-col>

      <!-- 连接统计 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>连接统计</span>
          </template>
          <div class="stat-item">
            <span class="label">在线客户端</span>
            <span class="value">{{ stats.onlineClients }}</span>
          </div>
          <div class="stat-item">
            <span class="label">活跃映射</span>
            <span class="value">{{ stats.activeMappings }}</span>
          </div>
          <div class="stat-item">
            <span class="label">WebSocket 连接</span>
            <span class="value">{{ stats.wsConnections }}</span>
          </div>
        </el-card>
      </el-col>

      <!-- 性能指标 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>性能指标</span>
          </template>
          <div class="stat-item">
            <span class="label">平均响应时间</span>
            <span class="value">{{ stats.avgResponseTime }}ms</span>
          </div>
          <div class="stat-item">
            <span class="label">请求总数</span>
            <span class="value">{{ stats.totalRequests }}</span>
          </div>
          <div class="stat-item">
            <span class="label">错误率</span>
            <span class="value">{{ stats.errorRate }}%</span>
          </div>
        </el-card>
      </el-col>

      <!-- 数据库状态 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <span>数据库状态</span>
          </template>
          <div class="stat-item">
            <span class="label">数据库大小</span>
            <span class="value">{{ stats.dbSize }}</span>
          </div>
          <div class="stat-item">
            <span class="label">查询次数</span>
            <span class="value">{{ stats.dbQueries }}</span>
          </div>
          <div class="stat-item">
            <span class="label">连接数</span>
            <span class="value">{{ stats.dbConnections }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 实时图表 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>CPU 使用率趋势</span>
          </template>
          <div class="chart-placeholder">
            <p>图表加载中...</p>
            <p>（需要集成 ECharts 或其他图表库）</p>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>内存使用率趋势</span>
          </template>
          <div class="chart-placeholder">
            <p>图表加载中...</p>
            <p>（需要集成 ECharts 或其他图表库）</p>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getRouterStats } from '../api/monitoring'

const stats = ref({
  cpu: 45,
  memory: 62,
  disk: 38,
  onlineClients: 12,
  activeMappings: 25,
  wsConnections: 8,
  avgResponseTime: 120,
  totalRequests: 15000,
  errorRate: 0.5,
  dbSize: '2.5 MB',
  dbQueries: 50000,
  dbConnections: 5,
})

onMounted(() => {
  loadStats()
})

const loadStats = async () => {
  try {
    const res = await getRouterStats()
    // 合并统计数据
    stats.value = {
      ...stats.value,
      ...res.data,
    }
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const getProgressColor = (percentage: number) => {
  if (percentage < 50) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}
</script>

<style scoped>
.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.stat-item .label {
  color: #606266;
  font-size: 14px;
}

.stat-item .value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.chart-placeholder {
  height: 200px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: #909399;
}
</style>
