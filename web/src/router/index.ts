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
      path: '/agreement',
      name: 'Agreement',
      component: () => import('../pages/Agreement.vue')
    },
    {
      path: '/privacy',
      name: 'Privacy',
      component: () => import('../pages/Privacy.vue')
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
    {
      // 拼字房邀请链接：/s/8621 → 首页带 ?sprint=8621，App.vue 落 localStorage，
      // 进编辑器后 useSprint.init 自动弹「加入房间」面板
      path: '/s/:code',
      name: 'SprintInvite',
      redirect: to => ({ path: '/', query: { sprint: String(to.params.code) } })
    },
  ]
})

router.beforeEach((to, _from, next) => {
  // 小程序 web-view：营销首页没有存在意义（获客靠小程序商店页），直接去书架
  if (to.path === '/' && /miniProgram/i.test(navigator.userAgent)) {
    next('/bookshelf')
    return
  }
  if (to.meta.requiresAuth && !localStorage.getItem('access_token')) {
    next('/login')
  } else {
    next()
  }
})

export default router