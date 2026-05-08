<template>
  <!-- Add/Edit Peer Dialog -->
  <v-dialog v-model="show" transition="dialog-bottom-transition" width="680" persistent>
    <v-card class="rounded-xl" elevation="8">
      <v-card-title class="d-flex align-center gap-2 pa-4">
        <v-icon color="primary">mdi-shield-key</v-icon>
        {{ isEdit ? 'Edit Peer' : 'Add AmneziaWG Peer' }}
      </v-card-title>
      <v-divider />

      <v-card-text class="pa-4">
        <v-row dense>
          <!-- Name -->
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="peer.name"
              label="Peer Name"
              prepend-inner-icon="mdi-account"
              variant="outlined"
              density="compact"
              hide-details
            />
          </v-col>

          <!-- Remark -->
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="peer.remark"
              label="Remark (optional)"
              prepend-inner-icon="mdi-note-text"
              variant="outlined"
              density="compact"
              hide-details
            />
          </v-col>

          <!-- Private Key -->
          <v-col cols="12">
            <v-text-field
              v-model="peer.privateKey"
              label="Private Key"
              prepend-inner-icon="mdi-key"
              variant="outlined"
              density="compact"
              hint='Leave "auto" for auto-generation'
              persistent-hint
              :append-inner-icon="'mdi-refresh'"
              @click:append-inner="genKeypair"
            />
          </v-col>

          <!-- Public Key (read-only display) -->
          <v-col cols="12" v-if="peer.publicKey && peer.publicKey !== 'auto'">
            <v-text-field
              :model-value="peer.publicKey"
              label="Public Key (auto-derived)"
              prepend-inner-icon="mdi-key-variant"
              variant="outlined"
              density="compact"
              hide-details
              readonly
            />
          </v-col>

          <!-- Allowed IPs -->
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="peer.allowedIPs"
              label="Allowed IPs"
              prepend-inner-icon="mdi-ip-network"
              variant="outlined"
              density="compact"
              hide-details
              placeholder="auto or 10.8.0.2/32"
            />
          </v-col>

          <!-- Enable toggle -->
          <v-col cols="12" sm="6" class="d-flex align-center">
            <v-switch
              v-model="peer.enable"
              color="success"
              label="Enable Peer"
              hide-details
              inset
            />
          </v-col>

          <!-- Expiry -->
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="expiryDate"
              label="Expiry Date (empty = no expiry)"
              prepend-inner-icon="mdi-calendar-clock"
              variant="outlined"
              density="compact"
              hide-details
              type="datetime-local"
              clearable
              @click:clear="peer.expiry = 0"
            />
          </v-col>

          <!-- Data Volume Limit -->
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="volumeGB"
              label="Data Limit (GB, 0 = unlimited)"
              prepend-inner-icon="mdi-database"
              variant="outlined"
              density="compact"
              hide-details
              type="number"
              min="0"
            />
          </v-col>
        </v-row>
      </v-card-text>

      <v-divider />
      <v-card-actions class="pa-4">
        <v-spacer />
        <v-btn variant="text" @click="close">Cancel</v-btn>
        <v-btn
          color="primary"
          variant="tonal"
          :loading="loading"
          @click="save"
          prepend-icon="mdi-content-save"
        >
          {{ isEdit ? 'Update' : 'Add Peer' }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import { AmneziaPeer, createEmptyPeer } from '@/types/amnezia'
import HttpUtils from '@/plugins/httputil'
import { push } from 'notivue'

const props = defineProps<{
  modelValue: boolean
  peer?: AmneziaPeer | null
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  'saved': []
}>()

const show = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const loading = ref(false)
const peer = ref<AmneziaPeer>(createEmptyPeer())
const isEdit = computed(() => !!props.peer && props.peer.id > 0)

// Expiry date helper (datetime-local ↔ unix timestamp)
const expiryDate = computed({
  get: () => {
    if (!peer.value.expiry) return ''
    const d = new Date(peer.value.expiry * 1000)
    return d.toISOString().slice(0, 16)
  },
  set: (v: string) => {
    peer.value.expiry = v ? Math.floor(new Date(v).getTime() / 1000) : 0
  },
})

// Data volume in GB ↔ bytes
const volumeGB = computed({
  get: () => peer.value.volume ? (peer.value.volume / 1073741824).toFixed(2) : '0',
  set: (v: string) => {
    peer.value.volume = parseFloat(v) > 0 ? Math.floor(parseFloat(v) * 1073741824) : 0
  },
})

watch(() => props.modelValue, (v) => {
  if (v) {
    peer.value = props.peer ? { ...props.peer } : createEmptyPeer()
  }
})

const genKeypair = async () => {
  loading.value = true
  const msg = await HttpUtils.get('apiv2/amnezia/keypair')
  loading.value = false
  if (msg.success && msg.obj) {
    peer.value.privateKey = msg.obj.privateKey
    peer.value.publicKey = msg.obj.publicKey
  }
}

const save = async () => {
  if (!peer.value.name) {
    push.error({ message: 'Peer name is required' })
    return
  }
  loading.value = true
  let msg
  if (isEdit.value) {
    msg = await HttpUtils.post(`apiv2/amnezia/peers/${peer.value.id}`, peer.value)
  } else {
    msg = await HttpUtils.post('apiv2/amnezia/peers', peer.value)
  }
  loading.value = false
  if (msg.success) {
    emit('saved')
    close()
  }
}

const close = () => {
  emit('update:modelValue', false)
}
</script>
