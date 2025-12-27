import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import HomeView from '@/views/home/HomeView.vue'
import DataSetView from '@/views/home/DataSetView.vue'
import ApplicationView from '@/views/home/ApplicationView.vue'
import UserView from '@/views/home/UserView.vue'
import StatisticView from '@/views/home/StatisticView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'login',
      component: LoginView,
      alias: '/login',
    },
    {
      path: '/register',
      name: 'register',
      component: RegisterView,
    },
    {
      path: '/home', // 所有登录后的页面都将基于这个根路径
      name: 'home',
      component: HomeView,
      redirect: '/home/datasets', // 访问根路径时，默认重定向到数据集列表
      // meta: { requiresAuth: true }, // 这个布局下的所有页面都需要登录
      children: [
        {
          path: 'datasets',
          name: 'dataset-list',
          component: DataSetView,
        },
        {
          path: 'applications',
          name: 'application-list',
          component: ApplicationView,
        },
        {
          path: 'users',
          name: 'user-list',
          component: UserView,
        },
        {
          path: 'statistic',
          name: 'statistic',
          component: StatisticView,
        },
      ],
    },
  ],
})

export default router
