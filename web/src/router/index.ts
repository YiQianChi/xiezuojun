import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
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
      component: () => import('../pages/Bookshelf.vue')
    },
    {
      path: '/editor/:projectId/:docId?',
      name: 'Editor',
      component: () => import('../pages/Editor.vue')
    }
  ]
})

export default router