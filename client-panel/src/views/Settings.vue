<template>
  <div class="settings">
    <el-card>
      <template #header>
        <span>设置</span>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="个人信息" name="profile">
          <el-form :model="profileForm" ref="profileFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="用户名">
              <el-input v-model="profileForm.username" disabled />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdateProfile" :loading="profileLoading">
                保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="设备信息" name="device">
          <el-form :model="deviceForm" ref="deviceFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="客户端ID">
              <el-input v-model="deviceForm.clientId" disabled />
            </el-form-item>
            <el-form-item label="设备名称">
              <el-input v-model="deviceForm.deviceName" placeholder="请输入设备名称" />
            </el-form-item>
            <el-form-item label="服务端地址">
              <el-input v-model="deviceForm.serverAddr" placeholder="请输入服务端地址" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdateDevice" :loading="deviceLoading">
                保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="FRPC 配置" name="frpc">
          <el-form :model="frpcForm" ref="frpcFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="FRPC 路径">
              <el-input v-model="frpcForm.path" placeholder="请输入 FRPC 路径" />
            </el-form-item>
            <el-form-item label="配置文件路径">
              <el-input v-model="frpcForm.configPath" placeholder="请输入配置文件路径" />
            </el-form-item>
            <el-form-item label="日志文件路径">
              <el-input v-model="frpcForm.logPath" placeholder="请输入日志文件路径" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdateFRPC" :loading="frpcLoading">
                保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../store/user'

const userStore = useUserStore()

const activeTab = ref('profile')

const profileLoading = ref(false)
const deviceLoading = ref(false)
const frpcLoading = ref(false)

const profileFormRef = ref()
const deviceFormRef = ref()
const frpcFormRef = ref()

const profileForm = reactive({
  username: '',
  email: '',
})

const deviceForm = reactive({
  clientId: '',
  deviceName: '',
  serverAddr: '',
})

const frpcForm = reactive({
  path: 'frpc',
  configPath: './data/frpc.toml',
  logPath: './logs/frpc.log',
})

onMounted(() => {
  loadSettings()
})

const loadSettings = () => {
  if (userStore.userInfo) {
    profileForm.username = userStore.userInfo.username
    profileForm.email = userStore.userInfo.email
  }
}

const handleUpdateProfile = async () => {
  profileLoading.value = true
  try {
    // TODO: 实现更新个人信息
    ElMessage.success('保存成功')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    profileLoading.value = false
  }
}

const handleUpdateDevice = async () => {
  deviceLoading.value = true
  try {
    // TODO: 实现更新设备信息
    ElMessage.success('保存成功')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    deviceLoading.value = false
  }
}

const handleUpdateFRPC = async () => {
  frpcLoading.value = true
  try {
    // TODO: 实现更新 FRPC 配置
    ElMessage.success('保存成功')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    frpcLoading.value = false
  }
}
</script>
