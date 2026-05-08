<template>
  <!-- Config Dialog -->
  <v-dialog v-model="showConfig" transition="dialog-bottom-transition" width="760" persistent>
    <v-card class="rounded-xl" elevation="8">
      <v-card-title class="d-flex align-center gap-2 pa-4">
        <v-icon color="primary">mdi-shield-lock</v-icon>
        AmneziaWG Server Config
      </v-card-title>
      <v-divider />
      <v-card-text class="pa-4">
        <v-row dense>
          <v-col cols="12" sm="4">
            <v-text-field v-model="cfg.interface" label="Interface" variant="outlined" density="compact" hide-details placeholder="awg0" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field v-model="cfg.address" label="Server Address (CIDR)" variant="outlined" density="compact" hide-details placeholder="10.8.0.1/24" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field v-model.number="cfg.listenPort" label="Listen Port" variant="outlined" density="compact" hide-details type="number" />
          </v-col>
          <v-col cols="12">
            <v-text-field
              v-model="cfg.privateKey"
              label="Private Key (leave blank or 'auto' to generate)"
              variant="outlined"
              density="compact"
              hide-details
              :append-inner-icon="'mdi-refresh'"
              @click:append-inner="genServerKey"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="cfg.dns" label="Client DNS" variant="outlined" density="compact" hide-details placeholder="1.1.1.1" />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model.number="cfg.mtu" label="MTU (0=1420)" variant="outlined" density="compact" hide-details type="number" />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="cfg.postUp" label="PostUp (iptables)" variant="outlined" density="compact" hide-details />
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field v-model="cfg.postDown" label="PostDown (iptables)" variant="outlined" density="compact" hide-details />
          </v-col>

          <!-- AmneziaWG obfuscation (collapsible) -->
          <v-col cols="12">
            <v-expansion-panels variant="accordion">
              <v-expansion-panel title="AmneziaWG Obfuscation (optional)">
                <v-expansion-panel-text>
                  <v-row dense>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.jc" label="Jc" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.jmin" label="Jmin" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.jmax" label="Jmax" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.s1" label="S1" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.s2" label="S2" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.h1" label="H1" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.h2" label="H2" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.h3" label="H3" variant="outlined" density="compact" hide-details type="number" /></v-col>
                    <v-col cols="6" sm="4"><v-text-field v-model.number="cfg.h4" label="H4" variant="outlined" density="compact" hide-details type="number" /></v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>
          </v-col>
        </v-row>
      </v-card-text>
      <v-divider />
      <v-card-actions class="pa-4">
        <v-spacer />
        <v-btn variant="text" @click="showConfig = false">Cancel</v-btn>
        <v-btn color="primary" variant="tonal" :loading="cfgLoading" @click="saveConfig" prepend-icon="mdi-content-save">Save Config</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Delete Confirm Dialog -->
  <v-dialog v-model="deleteDialog" max-width="380">
    <v-card class="rounded-xl" elevation="8">
      <v-card-title class="pa-4">
        <v-icon color="error" class="me-2">mdi-delete-alert</v-icon>
        Delete Peer
      </v-card-title>
      <v-card-text>Are you sure you want to delete <strong>{{ peerToDelete?.name }}</strong>?</v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
        <v-btn color="error" variant="tonal" :loading="delLoading" @click="confirmDelete">Delete</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Add/Edit Peer Modal -->
  <AmneziaPeerModal v-model="peerModal" :peer="editingPeer" @saved="loadPeers" />

  <!-- QR Code Dialog -->
  <v-dialog v-model="qrDialog" max-width="400">
    <v-card class="rounded-xl text-center pa-4" elevation="8">
      <v-card-title>
        <v-icon class="me-2">mdi-qrcode</v-icon>
        {{ qrPeer?.name }} Config
      </v-card-title>
      <v-card-text>
        <div v-if="qrCode" v-html="qrCode" class="d-flex justify-center"></div>
        <v-progress-circular v-else indeterminate color="primary" />
      </v-card-text>
      <v-card-actions>
        <v-btn variant="text" @click="downloadConf">
          <v-icon class="me-1">mdi-download</v-icon>Download .conf
        </v-btn>
        <v-spacer />
        <v-btn @click="qrDialog = false">Close</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- ===== Main Page ===== -->
  <div class="amnezia-page pa-4">
    <!-- Header -->
    <div class="d-flex align-center flex-wrap gap-3 mb-5">
      <div class="d-flex align-center gap-2">
        <v-icon size="32" color="primary">mdi-shield-key</v-icon>
        <div>
          <div class="text-h5 font-weight-bold">AmneziaWG</div>
          <div class="text-caption text-medium-emphasis">Standalone WireGuard VPN Engine</div>
        </div>
      </div>

      <v-spacer />

      <!-- Status chip -->
      <v-chip
        :color="status.running ? 'success' : 'error'"
        variant="tonal"
        size="small"
        :prepend-icon="status.running ? 'mdi-check-circle' : 'mdi-close-circle'"
      >
        {{ status.running ? 'Running' : 'Stopped' }}
      </v-chip>

      <!-- Interface controls -->
      <v-btn
        v-if="!status.running"
        color="success"
        variant="tonal"
        size="small"
        :loading="ifaceLoading"
        prepend-icon="mdi-play"
        @click="startInterface"
      >Start</v-btn>
      <v-btn
        v-else
        color="error"
        variant="tonal"
        size="small"
        :loading="ifaceLoading"
        prepend-icon="mdi-stop"
        @click="stopInterface"
      >Stop</v-btn>

      <v-btn
        variant="outlined"
        size="small"
        prepend-icon="mdi-cog"
        @click="openConfig"
      >Config</v-btn>

      <v-btn
        color="primary"
        variant="tonal"
        size="small"
        prepend-icon="mdi-plus"
        @click="openAddPeer"
      >Add Peer</v-btn>
    </div>

    <!-- Summary cards -->
    <v-row dense class="mb-4">
      <v-col cols="6" sm="3">
        <v-card class="rounded-xl text-center pa-3" variant="tonal" color="primary">
          <div class="text-h4 font-weight-bold">{{ peers.length }}</div>
          <div class="text-caption">Total Peers</div>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card class="rounded-xl text-center pa-3" variant="tonal" color="success">
          <div class="text-h4 font-weight-bold">{{ peers.filter(p => p.enable).length }}</div>
          <div class="text-caption">Active</div>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card class="rounded-xl text-center pa-3" variant="tonal" color="info">
          <div class="text-h6 font-weight-bold">{{ formatBytes(totalRx) }}</div>
          <div class="text-caption">Total Download</div>
        </v-card>
      </v-col>
      <v-col cols="6" sm="3">
        <v-card class="rounded-xl text-center pa-3" variant="tonal" color="warning">
          <div class="text-h6 font-weight-bold">{{ formatBytes(totalTx) }}</div>
          <div class="text-caption">Total Upload</div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Peers table -->
    <v-card class="rounded-xl" elevation="2">
      <v-card-title class="d-flex align-center pa-4">
        <v-icon class="me-2">mdi-account-multiple</v-icon>
        Peers
        <v-spacer />
        <v-text-field
          v-model="search"
          placeholder="Search peers..."
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
          density="compact"
          hide-details
          style="max-width: 240px"
          clearable
        />
      </v-card-title>
      <v-divider />

      <v-data-table
        :headers="headers"
        :items="filteredPeers"
        :loading="tableLoading"
        item-value="id"
        hover
        class="amnezia-table"
      >
        <!-- Name -->
        <template v-slot:item.name="{ item }">
          <div class="d-flex align-center gap-2">
            <v-avatar size="28" :color="item.enable ? 'primary' : 'grey'" variant="tonal">
              <v-icon size="16">mdi-account</v-icon>
            </v-avatar>
            <div>
              <div class="font-weight-medium">{{ item.name }}</div>
              <div class="text-caption text-medium-emphasis">{{ item.remark }}</div>
            </div>
          </div>
        </template>

        <!-- Status -->
        <template v-slot:item.enable="{ item }">
          <v-chip
            :color="item.enable ? 'success' : 'error'"
            size="x-small"
            variant="tonal"
          >{{ item.enable ? 'Active' : 'Disabled' }}</v-chip>
        </template>

        <!-- Traffic -->
        <template v-slot:item.traffic="{ item }">
          <div class="text-caption">
            <span class="text-success">↓ {{ formatBytes(item.down) }}</span>
            &nbsp;
            <span class="text-warning">↑ {{ formatBytes(item.up) }}</span>
          </div>
          <!-- Volume progress -->
          <v-progress-linear
            v-if="item.volume > 0"
            :model-value="Math.min(100, ((item.up + item.down) / item.volume) * 100)"
            color="primary"
            height="3"
            rounded
            class="mt-1"
          />
        </template>

        <!-- Expiry -->
        <template v-slot:item.expiry="{ item }">
          <template v-if="item.expiry > 0">
            <v-chip
              :color="isExpired(item.expiry) ? 'error' : 'info'"
              size="x-small"
              variant="tonal"
            >
              {{ formatExpiry(item.expiry) }}
            </v-chip>
          </template>
          <span v-else class="text-medium-emphasis">∞</span>
        </template>

        <!-- Actions -->
        <template v-slot:item.actions="{ item }">
          <div class="d-flex gap-1">
            <v-btn icon size="x-small" variant="text" @click="togglePeer(item)" :color="item.enable ? 'warning' : 'success'">
              <v-icon>{{ item.enable ? 'mdi-pause' : 'mdi-play' }}</v-icon>
              <v-tooltip activator="parent">{{ item.enable ? 'Disable' : 'Enable' }}</v-tooltip>
            </v-btn>
            <v-btn icon size="x-small" variant="text" color="primary" @click="openEditPeer(item)">
              <v-icon>mdi-pencil</v-icon>
              <v-tooltip activator="parent">Edit</v-tooltip>
            </v-btn>
            <v-btn icon size="x-small" variant="text" color="info" @click="showQR(item)">
              <v-icon>mdi-qrcode</v-icon>
              <v-tooltip activator="parent">Download Config / QR</v-tooltip>
            </v-btn>
            <v-btn icon size="x-small" variant="text" color="error" @click="askDelete(item)">
              <v-icon>mdi-delete</v-icon>
              <v-tooltip activator="parent">Delete</v-tooltip>
            </v-btn>
          </div>
        </template>

        <!-- Empty state -->
        <template v-slot:no-data>
          <div class="text-center pa-8">
            <v-icon size="48" color="medium-emphasis" class="mb-2">mdi-shield-off</v-icon>
            <div class="text-body-1 text-medium-emphasis">No peers yet.</div>
            <v-btn color="primary" variant="tonal" class="mt-3" @click="openAddPeer" prepend-icon="mdi-plus">
              Add First Peer
            </v-btn>
          </div>
        </template>
      </v-data-table>
    </v-card>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { AmneziaPeer, AmneziaConfig, createEmptyPeer, formatBytes, formatExpiry } from '@/types/amnezia'
import AmneziaPeerModal from '@/layouts/modals/AmneziaPeer.vue'
import HttpUtils from '@/plugins/httputil'
import { push } from 'notivue'

// ── State ─────────────────────────────────────────────────────────────────────
const peers        = ref<AmneziaPeer[]>([])
const status       = ref({ running: false })
const tableLoading = ref(false)
const ifaceLoading = ref(false)
const cfgLoading   = ref(false)
const delLoading   = ref(false)
const search       = ref('')

const peerModal   = ref(false)
const editingPeer = ref<AmneziaPeer | null>(null)

const showConfig  = ref(false)
const cfg         = ref<AmneziaConfig>({
  interface: 'awg0', address: '10.8.0.1/24', listenPort: 51820,
  privateKey: 'auto', publicKey: '', dns: '1.1.1.1', mtu: 1420,
  postUp: '', postDown: '',
  jc: 0, jmin: 0, jmax: 0, s1: 0, s2: 0, h1: 0, h2: 0, h3: 0, h4: 0,
})

const deleteDialog  = ref(false)
const peerToDelete  = ref<AmneziaPeer | null>(null)

const qrDialog  = ref(false)
const qrPeer    = ref<AmneziaPeer | null>(null)
const qrCode    = ref<string | null>(null)

// ── Computed ──────────────────────────────────────────────────────────────────
const filteredPeers = computed(() => {
  if (!search.value) return peers.value
  const s = search.value.toLowerCase()
  return peers.value.filter(p =>
    p.name.toLowerCase().includes(s) ||
    p.allowedIPs.toLowerCase().includes(s) ||
    p.remark?.toLowerCase().includes(s)
  )
})

const totalRx = computed(() => peers.value.reduce((a, p) => a + (p.down || 0), 0))
const totalTx = computed(() => peers.value.reduce((a, p) => a + (p.up || 0), 0))

// ── Table headers ─────────────────────────────────────────────────────────────
const headers = [
  { title: 'Peer', key: 'name', sortable: true },
  { title: 'Allowed IPs', key: 'allowedIPs', sortable: false },
  { title: 'Status', key: 'enable', sortable: true, align: 'center' as const },
  { title: 'Traffic', key: 'traffic', sortable: false },
  { title: 'Expiry', key: 'expiry', sortable: true },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
]

// ── Helpers ───────────────────────────────────────────────────────────────────
const isExpired = (ts: number) => ts > 0 && ts < Date.now() / 1000

// ── Data loading ──────────────────────────────────────────────────────────────
const loadStatus = async () => {
  const msg = await HttpUtils.get('api/amnezia/status')
  if (msg.success && msg.obj) status.value = msg.obj
}

const loadPeers = async () => {
  tableLoading.value = true
  const msg = await HttpUtils.get('api/amnezia/peers')
  tableLoading.value = false
  if (msg.success && msg.obj) peers.value = msg.obj
}

// ── Interface controls ────────────────────────────────────────────────────────
const startInterface = async () => {
  ifaceLoading.value = true
  const msg = await HttpUtils.post('api/amnezia/start', null)
  ifaceLoading.value = false
  if (msg.success) await loadStatus()
}

const stopInterface = async () => {
  ifaceLoading.value = true
  const msg = await HttpUtils.post('api/amnezia/stop', null)
  ifaceLoading.value = false
  if (msg.success) await loadStatus()
}

// ── Config ────────────────────────────────────────────────────────────────────
const openConfig = async () => {
  const msg = await HttpUtils.get('api/amnezia/config')
  if (msg.success && msg.obj) {
    cfg.value = { ...cfg.value, ...msg.obj, privateKey: 'auto' }
  }
  showConfig.value = true
}

const genServerKey = async () => {
  cfgLoading.value = true
  const msg = await HttpUtils.get('api/amnezia/keypair')
  cfgLoading.value = false
  if (msg.success && msg.obj) {
    cfg.value.privateKey = msg.obj.privateKey
    cfg.value.publicKey  = msg.obj.publicKey
  }
}

const saveConfig = async () => {
  cfgLoading.value = true
  const msg = await HttpUtils.post('api/amnezia/config', cfg.value)
  cfgLoading.value = false
  if (msg.success) showConfig.value = false
}

// ── Peer CRUD ─────────────────────────────────────────────────────────────────
const openAddPeer = () => {
  editingPeer.value = null
  peerModal.value = true
}

const openEditPeer = (p: AmneziaPeer) => {
  editingPeer.value = { ...p }
  peerModal.value = true
}

const togglePeer = async (p: AmneziaPeer) => {
  const msg = await HttpUtils.post(`api/amnezia/peers/${p.id}/toggle`, null)
  if (msg.success) await loadPeers()
}

const askDelete = (p: AmneziaPeer) => {
  peerToDelete.value = p
  deleteDialog.value = true
}

const confirmDelete = async () => {
  if (!peerToDelete.value) return
  delLoading.value = true
  // HttpUtils doesn't have delete, use fetch directly
  try {
    const base = (window as any).BASE_URL || '/'
    const resp = await fetch(`${base}api/amnezia/peers/${peerToDelete.value.id}`, { method: 'DELETE' })
    const data = await resp.json()
    if (data.success) {
      push.success({ message: 'Peer deleted' })
      await loadPeers()
    } else {
      push.error({ message: data.msg || 'Delete failed' })
    }
  } catch (e: any) {
    push.error({ message: e.toString() })
  }
  delLoading.value = false
  deleteDialog.value = false
}

// ── QR / Config download ──────────────────────────────────────────────────────
const showQR = async (p: AmneziaPeer) => {
  qrPeer.value = p
  qrCode.value = null
  qrDialog.value = true
  // Load config text (also used for download)
  const serverIP = window.location.hostname
  const msg = await HttpUtils.get(`api/amnezia/peers/${p.id}/config`, { server: serverIP })
  if (msg.success) {
    // msg.obj will be the raw .conf text (returned as plain text, so check)
    try {
      // Generate QR using a data URL approach (inline SVG/Canvas not needed — show text)
      qrCode.value = `<pre style="text-align:left;font-size:11px;white-space:pre-wrap;word-break:break-all">${msg.obj}</pre>`
    } catch { qrCode.value = '<p>Config loaded</p>' }
  }
}

const downloadConf = async () => {
  if (!qrPeer.value) return
  const serverIP = window.location.hostname
  const url = `${(window as any).BASE_URL || '/'}api/amnezia/peers/${qrPeer.value.id}/config?server=${serverIP}`
  const a = document.createElement('a')
  a.href = url
  a.download = `${qrPeer.value.name}.conf`
  a.click()
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(async () => {
  await Promise.all([loadStatus(), loadPeers()])
})
</script>

<style scoped>
.amnezia-page {
  max-width: 1200px;
  margin: 0 auto;
}
.amnezia-table :deep(tbody tr:hover) {
  background: rgba(var(--v-theme-primary), 0.04);
}
</style>
