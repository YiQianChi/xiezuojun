import { createRouter, createWebHistory, createWebHashHistory } from 'vue-router'

// ponytail-harmony: 鸿蒙壳走 resource://rawfile/ 协议，无 HTTP server 兜底 history 路由——
// 构建时 VITE_ROUTER=hash 切 hash 路由（web/.env.harmony 注入），网页/桌面保持 history
const useHash = !!(import.meta as any).env?.VITE_ROUTER

const router = createRouter({
  history: useHash ? createWebHashHistory() : createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'Home',
      component: () => import('../pages/Home.vue')
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('../pages/Login.vue')
    },
    {
      path: '/register',
      name: 'Register',
      component: () => import('../pages/Register.vue')
    },
    {
      path: '/bookshelf',
      name: 'Bookshelf',
      component: () => import('../pages/Bookshelf.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/editor/:projectId/:docId?',
      name: 'Editor',
      component: () => import('../pages/Editor.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/admin',
      name: 'Admin',
      component: () => import('../pages/Admin.vue'),
      meta: { requiresAuth: true }
    },
  ]
})

router.beforeEach((to, _from, next) => {
  if (to.meta.requiresAuth && !localStorage.getItem('access_token')) {
    next('/login')
  } else {
    next()
  }
})

export default router