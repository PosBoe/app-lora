<template>
  <v-app>
    <v-app-bar flat density="compact" class="border-b border-subtle">
      <v-app-bar-title>
        <router-link to="/" class="text-strong no-underline">lora</router-link>
      </v-app-bar-title>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>
