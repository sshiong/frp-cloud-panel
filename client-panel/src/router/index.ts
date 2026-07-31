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
          path: 'frpc',
          name: 'FRPC',
          component: () => import('../views/FRPC.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'logs',
          name: 'Logs',
          component: () => import('../views/Logs.vue'),
          meta: { requiresAuth: true },
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('../views/Settings.vue'),
          meta: { requiresAuth: true },
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
