// AmneziaWG peer types for the frontend
export interface AmneziaPeer {
  id: number
  name: string
  publicKey: string
  privateKey: string
  allowedIPs: string
  enable: boolean
  expiry: number      // Unix timestamp, 0 = no expiry
  volume: number      // bytes limit, 0 = unlimited
  up: number          // bytes uploaded
  down: number        // bytes downloaded
  remark: string
}

export interface AmneziaConfig {
  interface: string
  publicKey: string
  privateKey: string
  address: string
  listenPort: number
  dns: string
  mtu: number
  postUp: string
  postDown: string
  // obfuscation params
  jc: number
  jmin: number
  jmax: number
  s1: number
  s2: number
  h1: number
  h2: number
  h3: number
  h4: number
}

export function createEmptyPeer(): AmneziaPeer {
  return {
    id: 0,
    name: '',
    publicKey: '',
    privateKey: 'auto',
    allowedIPs: 'auto',
    enable: true,
    expiry: 0,
    volume: 0,
    up: 0,
    down: 0,
    remark: '',
  }
}

export function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

export function formatExpiry(ts: number): string {
  if (!ts || ts === 0) return '∞'
  return new Date(ts * 1000).toLocaleString()
}
