import './assets/styles/main.css'

import { createApp } from 'vue'
import App from './App.vue'
import router from './router'

import 'element-plus/dist/index.css'
import * as echarts from 'echarts'

const app = createApp(App)

app.use(router)
app.config.globalProperties.$echarts = echarts

app.mount('#app')
