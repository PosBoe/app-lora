<script setup lang="ts">
const router = useRouter()
const { me, logout } = useSession()
const { language, setLanguage } = useLanguage()

// The Vuetify locale and the lang attribute start equal to the vue-i18n
// locale.
setLanguage(language.value)

// signedIn is a session with a password the user chose: the menu shows only
// then.
const signedIn = computed(() => me.value !== null && !me.value.password_temporary)

async function signOut() {
  await logout()
  await router.push({ name: 'Login' })
}
</script>

<template>
  <v-app>
    <v-app-bar flat density="compact" class="border-b border-subtle">
      <v-app-bar-title class="flex-none">
        <router-link to="/" class="text-strong no-underline">{{ $t('app.name') }}</router-link>
      </v-app-bar-title>

      <nav v-if="signedIn" class="ml-4 flex gap-1">
        <v-btn
          v-if="me?.is_admin"
          :to="{ name: 'Organizations' }"
          variant="text"
          size="small"
          data-test="nav-organizations"
        >
          {{ $t('nav.organizations') }}
        </v-btn>
        <v-btn
          v-if="me?.is_admin"
          :to="{ name: 'AdminDeviceModels' }"
          variant="text"
          size="small"
          data-test="nav-admin-models"
        >
          {{ $t('nav.adminModels') }}
        </v-btn>
        <template v-else>
          <v-btn :to="{ name: 'Home' }" exact variant="text" size="small" data-test="nav-devices">
            {{ $t('nav.devices') }}
          </v-btn>
          <v-btn :to="{ name: 'DeviceGroups' }" variant="text" size="small" data-test="nav-groups">
            {{ $t('nav.groups') }}
          </v-btn>
          <v-btn :to="{ name: 'DeviceModels' }" variant="text" size="small" data-test="nav-models">
            {{ $t('nav.models') }}
          </v-btn>
          <v-btn
            :to="{ name: 'Integrations' }"
            variant="text"
            size="small"
            data-test="nav-integrations"
          >
            {{ $t('nav.integrations') }}
          </v-btn>
        </template>
      </nav>

      <template #append>
        <div class="flex items-center gap-3 pr-2">
          <LanguageSwitch />
          <v-menu v-if="me" location="bottom end">
            <template #activator="{ props: menu }">
              <v-btn v-bind="menu" variant="text" size="small" append-icon="mdi-chevron-down">
                <span data-test="user-login">{{ me.login }}</span>
              </v-btn>
            </template>
            <v-list density="compact">
              <v-list-item v-if="me.organization" :subtitle="me.organization.name" disabled />
              <v-list-item
                prepend-icon="mdi-logout"
                :title="$t('app.logout')"
                data-test="logout"
                @click="signOut"
              />
            </v-list>
          </v-menu>
        </div>
      </template>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>
