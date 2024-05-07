import { createApp } from 'vue'
import { inject } from '@vercel/analytics'
import './style.css'
import App from './App.vue'
import router from './routes';

inject()

const app = createApp(App)

app.use(router)
app.mount('#app')
