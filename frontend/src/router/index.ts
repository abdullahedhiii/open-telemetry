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
    },
    {
      path: '/watchlist',
      name: 'watchlist',
      component: () => import('@/views/Watchlist.vue'),
    },
    {
      path: '/details/stocks/:symbol',
      name: 'detail_stocks',
      component: () => import('@/views/Details.vue'),
    },
    {
      path: '/details/crypto/:symbolId',
      name: 'detail_crypto',
      component: () => import('@/views/Details.vue'),
    }
  
  
  ],
})

export default router
