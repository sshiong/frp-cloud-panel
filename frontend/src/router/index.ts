import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../store/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue'),
    },
    {
      path: '/',
      component: () => import('../views/Layout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('../views/Dashboard.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'mappings',
          name: 'Mappings',
          component: () => import('../views/Mappings.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'clients',
          name: 'Clients',
          component: () => import('../views/Clients.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'domains',
          name: 'Domains',
          component: () => import('../views/Domains.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('../views/Settings.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'audit-logs',
          name: 'AuditLogs',
          component: () => import('../views/AuditLogs.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'backups',
          name: 'Backups',
          component: () => import('../views/Backups.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'certificates',
          name: 'Certificates',
          component: () => import('../views/Certificates.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'dns-records',
          name: 'DNSRecords',
          component: () => import('../views/DNSRecords.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'monitoring',
          name: 'Monitoring',
          component: () => import('../views/Monitoring.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'users',
          name: 'Users',
          component: () => import('../views/Users.vue'),
          meta: { requiresAuth: true, requiresAdmin: true },
        },
      ],
    },
  ],
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && userStore.isAuthenticated) {
    next('/')
  } else {
    next()
  }
})

export default router
