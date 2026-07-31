<template>
  <div class="domains">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>域名管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            添加域名
          </el-button>
        </div>
      </template>

      <el-table :data="domains" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
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
        <el-table-column prop="cert_expiry" label="证书过期" width="180">
          <template #default="{ row }">
            {{ row.cert_expiry ? formatTime(row.cert_expiry) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑域名' : '添加域名'" width="400px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="请输入域名" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="HTTPS模式" prop="https_mode">
          <el-select v-model="form.https_mode" placeholder="请选择HTTPS模式">
            <el-option label="无证书" value="none" />
            <el-option label="自动证书" value="auto" />
            <el-option label="Cloudflare代理" value="cf_proxy" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">
          {{ isEdit ? '更新' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getDomains, createDomain, updateDomain, deleteDomain } from '../api/domains'

const domains = ref([])
const loading = ref(false)

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const editId = ref<number | null>(null)

const formRef = ref()

const form = reactive({
  domain: '',
  https_mode: 'none',
})

const rules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  https_mode: [{ required: true, message: '请选择HTTPS模式', trigger: 'change' }],
}

onMounted(() => {
  loadDomains()
})

const loadDomains = async () => {
  loading.value = true
  try {
    const res = await getDomains()
    domains.value = res.data
  } catch (error) {
    console.error('Failed to load domains:', error)
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  editId.value = null
  form.domain = ''
  form.https_mode = 'none'
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  editId.value = row.id
  form.domain = row.domain
  form.https_mode = row.https_mode
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()

  submitLoading.value = true
  try {
    if (isEdit.value && editId.value) {
      await updateDomain(editId.value, form)
      ElMessage.success('更新成功')
    } else {
      await createDomain(form)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    loadDomains()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该域名吗？', '提示', {
      type: 'warning',
    })
    await deleteDomain(row.id)
    ElMessage.success('删除成功')
    loadDomains()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
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
  }
  return map[status] || 'info'
}

const getCertStatusText = (status: string) => {
  const map: Record<string, string> = {
    none: '无',
    pending: '申请中',
    active: '有效',
    expired: '已过期',
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
