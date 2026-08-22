import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import './styles.css'
import App from './App.vue'
import { monitoringRouter } from './router'

const consoleApp = createApp(App)
const monitoringState = createPinia()

consoleApp.config.errorHandler = (error) => {
  console.error('monitoring console render failure', error)
}
consoleApp.use(monitoringState)
consoleApp.use(monitoringRouter)
consoleApp.use(ElementPlus, { locale: zhCn })
consoleApp.mount('#app')
