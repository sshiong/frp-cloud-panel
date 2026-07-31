<template>
  <div class="frpc">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>FRPC 管理</span>
          <el-button :type="frpcRunning ? 'danger' : 'success'" @click="toggleFRPC">
            <el-icon><VideoPlay /></el-icon>
            {{ frpcRunning ? '停止 FRPC' : '启动 FRPC' }}
          </el-button>
        </div>
      </template>

      <el-row :gutter="20">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>
              <span>FRPC 状态</span>
            </template>
            <div class="status-item">
              <span class="label">运行状态</span>
              <el-tag :type="frpcRunning ? 'success' : 'danger'">
                {{ frpcRunning ? '运行中' : '已停止' }}
              </el-tag>
            </div>
            <div class="status-item">
              <span class="label">进程ID</span>
              <span class="value">{{ frpcPid || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">启动时间</span>
              <span class="value">{{ frpcStartTime || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">运行时长</span>
              <span class="value">{{ frpcUptime || '-' }}</span>
            </div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>
              <span>配置信息</span>
            </template>
            <div class="status-item">
              <span class="label">服务端地址</span>
              <span class="value">{{ config.serverAddr || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">配置版本</span>
              <span class="value">{{ config.version || '-' }}</span>
            </div>
            <div class="status-item">
              <span class="label">代理数量</span>
              <span class="value">{{ config.proxyCount || 0 }}</span>
            </div>
            <div class="status-item">
              <span class="label">最后同步</span>
              <span class="value">{{ config.lastSync || '-' }}</span>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-card style="margin-top: 20px;">
        <template #header>
          <div class="card-header">
            <span>FRPC 日志</span>
            <el-button @click="refreshLogs">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </template>
        <el-input
          v-model="logs"
          type="textarea"
          :rows="10"
          readonly
          placeholder="暂无日志"
        />
      </el-card>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const frpcRunning = ref(false)
const frpcPid = ref('')
const frpcStartTime = ref('')
const frpcUptime = ref('')
const logs = ref('')

const config = ref({
  serverAddr: '',
  version: 0,
  proxyCount: 0,
  lastSync: '',
})

onMounted(() => {
  loadFRPCStatus()
  loadLogs()
})

const loadFRPCStatus = async () => {
  // TODO: 实现获取 FRPC 状态
  frpcRunning.value = false
}

const loadLogs = async () => {
  // TODO: 实现获取 FRPC 日志
  logs.value = '暂无日志'
}

const toggleFRPC = async () => {
  // TODO: 实现 FRPC 启停
  frpcRunning.value = !frpcRunning.value
  ElMessage.success(frpcRunning.value ? 'FRPC 已启动' : 'FRPC 已停止')
}

const refreshLogs = () => {
  loadLogs()
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.status-item .label {
  color: #606266;
  font-size: 14px;
}

.status-item .value {
  font-size: 14px;
  color: #303133;
}
</style>
