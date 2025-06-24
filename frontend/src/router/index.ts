import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/Home.vue'),
    },
    {
      path : '/stocks',
      name : 'stocks',
      component: () => import('@/views/Stocks.vue'),
    }
  
  ],
})

export default router
