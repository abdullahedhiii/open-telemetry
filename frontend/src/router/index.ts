import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path : '/stocks',
      name : 'stocks',
      component: () => import('@/views/Stocks.vue'),
    },
    {
      path : '/crypto',
      name : 'crypto',
      component: () => import('@/views/Crypto.vue'),
    }
  
  
  ],
})

export default router
