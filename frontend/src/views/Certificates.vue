<template>
  <div class="certificates">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>证书管理</span>
          <el-button @click="checkCertificates" :loading="checking">
            <el-icon><Refresh /></el-icon>
            检查证书
          </el-button>
        </div>
      </template>

      <!-- 证书列表 -->
      <el-table :data="certificates" style="width: 100%" v-loading="loading">
        <el-table-column prop="domain" label="域名" />
        <el-table-column prop="https_mode" label="HTTPS模式" width="120">
          <template #default="{ row }">
            <el-tag :type="getHTTPSModeType(row.https_mode)">
              {{ getHTTPSModeText(row.https_mode) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cert_status" label="证书状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getCertStatusType(row.cert_status)">
              {{ getCertStatusText(row.cert_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cert_expiry" label="过期时间" width="180">
          <template #default="{ row }">
            {{ row.cert_expiry ? formatTime(row.cert_expiry) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button
              size="small"
              @click="handleRenew(row)"
              :disabled="row.https_mode !== 'auto'"
              :loading="row.renewing"
            >
              续期
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getDomains } from '../api/domains'
import { renewCertificate, checkCerts } from '../api/certificates'

const certificates = ref([])
const loading = ref(false)
const checking = ref(false)

onMounted(() => {
  loadCertificates()
})

const loadCertificates = async () => {
  loading.value = true
  try {
    const res = await getDomains()
    certificates.value = res.data.map((d: any) => ({
      ...d,
      renewing: false,
    }))
  } catch (error) {
    console.error('Failed to load certificates:', error)
  } finally {
    loading.value = false
  }
}

const handleRenew = async (row: any) => {
  row.renewing = true
  try {
    await renewCertificate(row.domain)
    ElMessage.success('证书续期成功')
    loadCertificates()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '证书续期失败')
  } finally {
    row.renewing = false
  }
}

const checkCertificates = async () => {
  checking.value = true
  try {
    await checkCerts()
    ElMessage.success('证书检查已启动')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '证书检查失败')
  } finally {
    checking.value = false
  }
}

const getHTTPSModeType = (mode: string) => {
  const map: Record<string, string> = {
    none: 'info',
    auto: 'success',
    cf_proxy: 'warning',
  }
  return map[mode] || 'info'
}

const getHTTPSModeText = (mode: string) => {
  const map: Record<string, string> = {
    none: '无证书',
    auto: '自动证书',
    cf_proxy: 'CF代理',
  }
  return map[mode] || mode
}

const getCertStatusType = (status: string) => {
  const map: Record<string, string> = {
    none: 'info',
    pending: 'warning',
    active: 'success',
    expired: 'danger',
    error: 'danger',
  }
  return map[status] || 'info'
}

const getCertStatusText = (status: string) => {
  const map: Record<string, string> = {
    none: '无',
    pending: '申请中',
    active: '有效',
    expired: '已过期',
    error: '错误',
  }
  return map[status] || status
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
</style>
