<template>
  <div class="backups">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>备份管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建备份
          </el-button>
        </div>
      </template>

      <!-- 备份列表 -->
      <el-table :data="backups" style="width: 100%" v-loading="loading">
        <el-table-column prop="filename" label="文件名" />
        <el-table-column prop="size" label="大小" width="120">
          <template #default="{ row }">
            {{ formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="handleRestore(row)">恢复</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建备份对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建备份" width="400px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="备份密码" prop="password">
          <el-input v-model="createForm.password" type="password" placeholder="请输入备份密码" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="createForm.confirmPassword" type="password" placeholder="请再次输入密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="createLoading">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 恢复备份对话框 -->
    <el-dialog v-model="restoreDialogVisible" title="恢复备份" width="400px">
      <el-alert
        title="警告"
        type="warning"
        description="恢复备份将覆盖当前所有数据，此操作不可恢复！"
        show-icon
        :closable="false"
        style="margin-bottom: 20px;"
      />
      <el-form :model="restoreForm" :rules="restoreRules" ref="restoreFormRef" label-width="100px">
        <el-form-item label="备份文件" prop="filename">
          <el-input v-model="restoreForm.filename" disabled />
        </el-form-item>
        <el-form-item label="备份密码" prop="password">
          <el-input v-model="restoreForm.password" type="password" placeholder="请输入备份密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="restoreDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="handleRestoreConfirm" :loading="restoreLoading">
          确认恢复
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getBackups, createBackup, restoreBackup, deleteBackup } from '../api/backups'

const backups = ref([])
const loading = ref(false)

const createDialogVisible = ref(false)
const createLoading = ref(false)
const createFormRef = ref()

const restoreDialogVisible = ref(false)
const restoreLoading = ref(false)
const restoreFormRef = ref()

const createForm = reactive({
  password: '',
  confirmPassword: '',
})

const restoreForm = reactive({
  filename: '',
  password: '',
})

const createRules = {
  password: [
    { required: true, message: '请输入备份密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能小于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (rule: any, value: string, callback: any) => {
        if (value !== createForm.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

const restoreRules = {
  password: [
    { required: true, message: '请输入备份密码', trigger: 'blur' },
  ],
}

onMounted(() => {
  loadBackups()
})

const loadBackups = async () => {
  loading.value = true
  try {
    const res = await getBackups()
    backups.value = res.data
  } catch (error) {
    console.error('Failed to load backups:', error)
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  createForm.password = ''
  createForm.confirmPassword = ''
  createDialogVisible.value = true
}

const handleCreate = async () => {
  await createFormRef.value.validate()

  createLoading.value = true
  try {
    await createBackup(createForm.password)
    ElMessage.success('备份创建成功')
    createDialogVisible.value = false
    loadBackups()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '备份创建失败')
  } finally {
    createLoading.value = false
  }
}

const handleRestore = (row: any) => {
  restoreForm.filename = row.filename
  restoreForm.password = ''
  restoreDialogVisible.value = true
}

const handleRestoreConfirm = async () => {
  await restoreFormRef.value.validate()

  try {
    await ElMessageBox.confirm('确定要恢复此备份吗？当前数据将被覆盖！', '警告', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  restoreLoading.value = true
  try {
    await restoreBackup(restoreForm.filename, restoreForm.password)
    ElMessage.success('备份恢复成功')
    restoreDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '备份恢复失败')
  } finally {
    restoreLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除此备份吗？', '提示', {
      type: 'warning',
    })
    await deleteBackup(row.filename)
    ElMessage.success('备份删除成功')
    loadBackups()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '备份删除失败')
    }
  }
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
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
