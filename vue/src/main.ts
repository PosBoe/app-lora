import 'virtual:uno.css'
import '@/styles/index.scss'

import App from '@/App.vue'
import vuetify from '@/plugins/vuetify'
import { createAppRouter } from '@/router'

createApp(App).use(vuetify).use(createAppRouter()).mount('#app')
