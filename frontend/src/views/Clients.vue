<template>
  <div class="clients">
    <el-card>
      <template #header>
        <span>客户端管理</span>
      </template>

      <el-table :data="clients" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="client_id" label="客户端ID" />
        <el-table-column prop="device_name" label="设备名称" />
        <el-table-column prop="ip" label="IP地址" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'">
              {{ row.status === 'active' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_seen_at" label="最后在线" width="180">
          <template #default="{ row }">
            {{ row.last_seen_at ? formatTime(row.last_seen_at) : '从未上线' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
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
        @size-change="loadClients"
        @current-change="loadClients"
        style="margin-top: 20px; justify-content: flex-end;"
      />
    </el-card>

    <!-- 编辑对话框 -->
    <el-dialog v-model="dialogVisible" title="编辑客户端" width="400px">
      <el-form :model="form" ref="formRef" label-width="100px">
        <el-form-item label="设备名称" prop="device_name">
          <el-input v-model="form.device_name" placeholder="请输入设备名称" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="form.status" placeholder="请选择状态">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">
          更新
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getClients, updateClient, deleteClient } from '../api/clients'

const clients = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const dialogVisible = ref(false)
const submitLoading = ref(false)
const editId = ref<number | null>(null)

const formRef = ref()

const form = reactive({
  device_name: '',
  status: 'active',
})

onMounted(() => {
  loadClients()
})

const loadClients = async () => {
  loading.value = true
  try {
    const res = await getClients({
      page: currentPage.value,
      page_size: pageSize.value,
    })
    clients.value = res.data.items
    total.value = res.data.total
  } catch (error) {
    console.error('Failed to load clients:', error)
  } finally {
    loading.value = false
  }
}

const handleEdit = (row: any) => {
  editId.value = row.id
  form.device_name = row.device_name
  form.status = row.status
  dialogVisible.value = true
}

const handleSubmit = async () => {
  submitLoading.value = true
  try {
    if (editId.value) {
      await updateClient(editId.value, form)
      ElMessage.success('更新成功')
      dialogVisible.value = false
      loadClients()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '更新失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该客户端吗？', '提示', {
      type: 'warning',
    })
    await deleteClient(row.id)
    ElMessage.success('删除成功')
    loadClients()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString()
}
</script>
