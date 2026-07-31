<template>
  <div class="logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>日志查看</span>
          <div>
            <el-button @click="refreshLogs">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button @click="clearLogs">
              <el-icon><Delete /></el-icon>
              清空
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="FRPC 日志" name="frpc">
          <el-input
            v-model="frpcLogs"
            type="textarea"
            :rows="15"
            readonly
            placeholder="暂无 FRPC 日志"
          />
        </el-tab-pane>
        <el-tab-pane label="系统日志" name="system">
          <el-input
            v-model="systemLogs"
            type="textarea"
            :rows="15"
            readonly
            placeholder="暂无系统日志"
          />
        </el-tab-pane>
        <el-tab-pane label="错误日志" name="error">
          <el-input
            v-model="errorLogs"
            type="textarea"
            :rows="15"
            readonly
            placeholder="暂无错误日志"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const activeTab = ref('frpc')
const frpcLogs = ref('')
const systemLogs = ref('')
const errorLogs = ref('')

onMounted(() => {
  loadLogs()
})

const loadLogs = async () => {
  // TODO: 实现获取日志
  frpcLogs.value = '暂无 FRPC 日志'
  systemLogs.value = '暂无系统日志'
  errorLogs.value = '暂无错误日志'
}

const refreshLogs = () => {
  loadLogs()
  ElMessage.success('日志已刷新')
}

const clearLogs = () => {
  // TODO: 实现清空日志
  frpcLogs.value = ''
  systemLogs.value = ''
  errorLogs.value = ''
  ElMessage.success('日志已清空')
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
