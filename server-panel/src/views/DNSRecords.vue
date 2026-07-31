<template>
  <div class="dns-records">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>DNS 记录管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            添加记录
          </el-button>
        </div>
      </template>

      <!-- 域名选择 -->
      <el-form :inline="true" style="margin-bottom: 20px;">
        <el-form-item label="域名">
          <el-select v-model="selectedDomain" placeholder="请选择域名" @change="loadRecords">
            <el-option
              v-for="domain in domains"
              :key="domain.domain"
              :label="domain.domain"
              :value="domain.domain"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <!-- DNS 记录列表 -->
      <el-table :data="records" style="width: 100%" v-loading="loading">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="80" />
        <el-table-column prop="content" label="内容" />
        <el-table-column prop="ttl" label="TTL" width="80" />
        <el-table-column prop="proxied" label="代理" width="80">
          <template #default="{ row }">
            <el-tag :type="row.proxied ? 'success' : 'info'">
              {{ row.proxied ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑记录' : '添加记录'" width="500px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" disabled />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择类型">
            <el-option label="A" value="A" />
            <el-option label="AAAA" value="AAAA" />
            <el-option label="CNAME" value="CNAME" />
            <el-option label="MX" value="MX" />
            <el-option label="TXT" value="TXT" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入记录名称" />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input v-model="form.content" placeholder="请输入记录内容" />
        </el-form-item>
        <el-form-item label="TTL" prop="ttl">
          <el-input-number v-model="form.ttl" :min="1" :max="86400" />
        </el-form-item>
        <el-form-item label="代理" prop="proxied">
          <el-switch v-model="form.proxied" />
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
import { getDomains } from '../api/domains'
import { getDNSRecords, createDNSRecord, updateDNSRecord, deleteDNSRecord } from '../api/dns'

const domains = ref([])
const selectedDomain = ref('')
const records = ref([])
const loading = ref(false)

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const editRecordId = ref('')

const formRef = ref()

const form = reactive({
  domain: '',
  type: 'A',
  name: '',
  content: '',
  ttl: 1,
  proxied: false,
})

const rules = {
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  content: [{ required: true, message: '请输入内容', trigger: 'blur' }],
}

onMounted(() => {
  loadDomains()
})

const loadDomains = async () => {
  try {
    const res = await getDomains()
    domains.value = res.data
    if (domains.value.length > 0) {
      selectedDomain.value = domains.value[0].domain
      loadRecords()
    }
  } catch (error) {
    console.error('Failed to load domains:', error)
  }
}

const loadRecords = async () => {
  if (!selectedDomain.value) return

  loading.value = true
  try {
    const res = await getDNSRecords(selectedDomain.value)
    records.value = res.data
  } catch (error) {
    console.error('Failed to load records:', error)
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  editRecordId.value = ''
  form.domain = selectedDomain.value
  form.type = 'A'
  form.name = ''
  form.content = ''
  form.ttl = 1
  form.proxied = false
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  editRecordId.value = row.id
  form.domain = selectedDomain.value
  form.type = row.type
  form.name = row.name
  form.content = row.content
  form.ttl = row.ttl
  form.proxied = row.proxied
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()

  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updateDNSRecord(editRecordId.value, form)
      ElMessage.success('记录更新成功')
    } else {
      await createDNSRecord(form)
      ElMessage.success('记录添加成功')
    }
    dialogVisible.value = false
    loadRecords()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除此记录吗？', '提示', {
      type: 'warning',
    })
    await deleteDNSRecord(row.id)
    ElMessage.success('记录删除成功')
    loadRecords()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
