<template>
  <div class="settings">
    <el-card>
      <template #header>
        <span>系统设置</span>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="个人信息" name="profile">
          <el-form :model="profileForm" :rules="profileRules" ref="profileFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="profileForm.username" disabled />
            </el-form-item>
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdateProfile" :loading="profileLoading">
                保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="修改密码" name="password">
          <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="旧密码" prop="oldPassword">
              <el-input v-model="passwordForm.oldPassword" type="password" placeholder="请输入旧密码" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="passwordForm.newPassword" type="password" placeholder="请输入新密码" show-password />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="passwordForm.confirmPassword" type="password" placeholder="请再次输入新密码" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdatePassword" :loading="passwordLoading">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="Cloudflare Token" name="cloudflare">
          <el-form :model="cfForm" ref="cfFormRef" label-width="100px" style="max-width: 500px;">
            <el-form-item label="Token" prop="token">
              <el-input v-model="cfForm.token" type="password" placeholder="请输入 Cloudflare API Token" show-password />
            </el-form-item>
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="cfForm.email" placeholder="请输入 Cloudflare 邮箱" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveCFToken" :loading="cfLoading">
                保存
              </el-button>
              <el-button @click="handleTestCFToken" :loading="cfTestLoading">
                测试连接
              </el-button>
            </el-form-item>
          </el-form>

          <el-divider />

          <div v-if="cfStatus">
            <h4>Token 状态</h4>
            <p>状态: <el-tag :type="cfStatus.status === 'active' ? 'success' : 'danger'">{{ cfStatus.status === 'active' ? '有效' : '无效' }}</el-tag></p>
            <p v-if="cfStatus.email">邮箱: {{ cfStatus.email }}</p>
            <p v-if="cfStatus.created_at">创建时间: {{ formatTime(cfStatus.created_at) }}</p>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../store/user'
import { updateUserInfo, updatePassword } from '../api/auth'
import { setCFToken, getCFTokenStatus, testCFToken } from '../api/cloudflare'

const userStore = useUserStore()

const activeTab = ref('profile')

// 个人信息
const profileFormRef = ref()
const profileLoading = ref(false)
const profileForm = reactive({
  username: '',
  email: '',
})
const profileRules = {
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '请输入正确的邮箱', trigger: 'blur' }],
}

// 修改密码
const passwordFormRef = ref()
const passwordLoading = ref(false)
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const passwordRules = {
  oldPassword: [{ required: true, message: '请输入旧密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '密码长度不能小于6位', trigger: 'blur' }],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule: any, value: string, callback: any) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

// Cloudflare Token
const cfFormRef = ref()
const cfLoading = ref(false)
const cfTestLoading = ref(false)
const cfStatus = ref<any>(null)
const cfForm = reactive({
  token: '',
  email: '',
})

onMounted(() => {
  loadUserInfo()
  loadCFStatus()
})

const loadUserInfo = () => {
  if (userStore.userInfo) {
    profileForm.username = userStore.userInfo.username
    profileForm.email = userStore.userInfo.email
  }
}

const loadCFStatus = async () => {
  try {
    const res = await getCFTokenStatus()
    cfStatus.value = res.data
  } catch (error) {
    console.error('Failed to load CF status:', error)
  }
}

const handleUpdateProfile = async () => {
  await profileFormRef.value.validate()

  profileLoading.value = true
  try {
    await updateUserInfo({ email: profileForm.email })
    ElMessage.success('保存成功')
    userStore.fetchUserInfo()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    profileLoading.value = false
  }
}

const handleUpdatePassword = async () => {
  await passwordFormRef.value.validate()

  passwordLoading.value = true
  try {
    await updatePassword(passwordForm.oldPassword, passwordForm.newPassword)
    ElMessage.success('密码修改成功')
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '密码修改失败')
  } finally {
    passwordLoading.value = false
  }
}

const handleSaveCFToken = async () => {
  cfLoading.value = true
  try {
    await setCFToken(cfForm.token, cfForm.email)
    ElMessage.success('保存成功')
    loadCFStatus()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    cfLoading.value = false
  }
}

const handleTestCFToken = async () => {
  cfTestLoading.value = true
  try {
    const res = await testCFToken()
    if (res.data.status === 'valid') {
      ElMessage.success('Token 有效')
    } else {
      ElMessage.error('Token 无效')
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '测试失败')
  } finally {
    cfTestLoading.value = false
  }
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString()
}
</script>
