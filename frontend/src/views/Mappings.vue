<template>
  <div class="mappings">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>映射管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建映射
          </el-button>
        </div>
      </template>

      <el-table :data="mappings" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="80" />
        <el-table-column prop="local_ip" label="本地IP" />
        <el-table-column prop="local_port" label="本地端口" width="100" />
        <el-table-column prop="remote_port" label="远程端口" width="100" />
        <el-table-column prop="domain" label="域名" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadMappings"
        @current-change="loadMappings"
        style="margin-top: 20px; justify-content: flex-end;"
      />
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑映射' : '创建映射'" width="500px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入映射名称" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" placeholder="请选择类型">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
            <el-option label="HTTP" value="http" />
            <el-option label="HTTPS" value="https" />
          </el-select>
        </el-form-item>
        <el-form-item label="本地IP" prop="local_ip">
          <el-input v-model="form.local_ip" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="本地端口" prop="local_port">
          <el-input-number v-model="form.local_port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="远程端口" prop="remote_port">
          <el-input-number v-model="form.remote_port" :min="10000" :max="20000" placeholder="留空自动分配" />
        </el-form-item>
        <el-form-item label="域名" prop="domain" v-if="form.type === 'http' || form.type === 'https'">
          <el-input v-model="form.domain" placeholder="请输入域名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">
          {{ isEdit ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getMappings, createMapping, updateMapping, deleteMapping } from '../api/mappings'

const mappings = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const editId = ref<number | null>(null)

const formRef = ref()

const form = reactive({
  name: '',
  type: 'tcp',
  local_ip: '127.0.0.1',
  local_port: 8080,
  remote_port: undefined as number | undefined,
  domain: '',
})

const rules = {
  name: [{ required: true, message: '请输入映射名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  local_ip: [{ required: true, message: '请输入本地IP', trigger: 'blur' }],
  local_port: [{ required: true, message: '请输入本地端口', trigger: 'blur' }],
}

onMounted(() => {
  loadMappings()
})

const loadMappings = async () => {
  loading.value = true
  try {
    const res = await getMappings({
      page: currentPage.value,
      page_size: pageSize.value,
    })
    mappings.value = res.data.items
    total.value = res.data.total
  } catch (error) {
    console.error('Failed to load mappings:', error)
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  editId.value = null
  form.name = ''
  form.type = 'tcp'
  form.local_ip = '127.0.0.1'
  form.local_port = 8080
  form.remote_port = undefined
  form.domain = ''
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  editId.value = row.id
  form.name = row.name
  form.type = row.type
  form.local_ip = row.local_ip
  form.local_port = row.local_port
  form.remote_port = row.remote_port
  form.domain = row.domain
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()

  submitLoading.value = true
  try {
    if (isEdit.value && editId.value) {
      await updateMapping(editId.value, form)
      ElMessage.success('更新成功')
    } else {
      await createMapping(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadMappings()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该映射吗？', '提示', {
      type: 'warning',
    })
    await deleteMapping(row.id)
    ElMessage.success('删除成功')
    loadMappings()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    running: 'success',
    pending_apply: 'warning',
    offline: 'info',
    config_error: 'danger',
    disabled: 'info',
    deleting: 'danger',
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
    deleting: '删除中',
  }
  return map[status] || status
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
