import 'virtual:uno.css'
import '@/styles/index.scss'

import App from '@/App.vue'
import { createAppI18n } from '@/plugins/i18n'
import vuetify from '@/plugins/vuetify'
import { createAppRouter } from '@/router'

createApp(App).use(createAppI18n()).use(vuetify).use(createAppRouter()).mount('#app')
