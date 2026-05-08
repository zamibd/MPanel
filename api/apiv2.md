# MPanel API Documentation (v2)

This documentation covers the **REST API with Token** (`/apiv2`), designed for external applications requiring programmatic access to the MPanel system without relying on browser sessions.

## Authentication

To interact with the REST API, authentication via an API token is required. 

### Generating an API Token
1. Navigate to the Admin page in the MPanel web interface.
2. Create a new API Token.
3. For security purposes, it is recommended to set an expiration date and rotate tokens periodically.
4. Copy and securely store the generated token, as it will not be shown again.

### Using the API Token
Include your token in the request header `Token`, as shown in the example below:

```bash
curl -H "Token: <Your Token Key>" "http://localhost:2095/app/apiv2/inbounds?id=2"
```

---

## Response Structure

Every API response follows a consistent JSON format:

```json
{
  "success": true,
  "msg": "",
  "obj": {}
}
```

- **`success`**: A boolean indicating whether the request was successful (`true`) or not (`false`).
- **`msg`**: A message describing the result of the operation. If successful, this is typically an empty string or the name of the action. If failed, it provides an error message.
- **`obj`**: The object containing the requested data or the result of the action. This varies depending on the endpoint but will always return relevant data (or `null` if there is no data to return).

### Example Success Response (`/apiv2/save`)
```json
{
  "success": true,
  "msg": "save",
  "obj": {
    "id": 1,
    "tag": "my-inbound"
  }
}
```

### Example Failure Response
```json
{
  "success": false,
  "msg": "Invalid token",
  "obj": null
}
```

---

## API Endpoints

### POST Endpoints

| Endpoint | Description | Request Parameters / Body |
|----------|-------------|---------------------------|
| `POST /apiv2/save` | Save configuration data (e.g., inbounds, clients) | `object` (string), `action` (string), `data` (JSON), `initUsers` (optional string) |
| `POST /apiv2/restartApp` | Restart the MPanel application | None |
| `POST /apiv2/restartSb` | Restart the Sing-Box Core | None |
| `POST /apiv2/linkConvert` | Convert a generic node link | `link` (string) |
| `POST /apiv2/subConvert` | Convert a subscription link to JSON | `link` (string) |
| `POST /apiv2/importdb` | Import a database file | FormData `db` (file) |

### Real-World Examples (POST)

**1. Create a new Inbound (`/apiv2/save`)**
This example demonstrates how to create a simple `vless` inbound listening on port `443` without TLS.

```bash
curl -X POST "http://localhost:2095/app/apiv2/save" \
  -H "Token: <Your Token Key>" \
  -H "Content-Type: application/json" \
  -d '{
    "object": "inbounds",
    "action": "new",
    "data": {
      "type": "vless",
      "tag": "my-vless-in",
      "tls_id": 0,
      "addrs": ["0.0.0.0"],
      "out_json": {},
      "listen_port": 443
    }
  }'
```

**3. Create a WireGuard Endpoint with Auto Key Generation (`/apiv2/save`)**
You can set `private_key` to `"auto"` and MPanel will automatically generate and assign a valid keypair to the endpoint for you.

```bash
curl -X POST "http://localhost:2095/app/apiv2/save" \
  -H "Token: <Your Token Key>" \
  -H "Content-Type: application/json" \
  -d '{
    "object": "endpoints",
    "action": "new",
    "data": {
      "type": "wireguard",
      "tag": "my-wg-endpoint",
      "server": "198.51.100.1",
      "server_port": 51820,
      "local_address": ["10.0.0.2/32"],
      "private_key": "auto"
    }
  }'
```

**4. Update an Endpoint to Add Peers Automatically (`/apiv2/save`)**
You can pass `"auto"` in the peer's `public_key` field. MPanel will automatically generate a private/public keypair, assign the `public_key` for the connection, and save the `private_key` in the options for you to retrieve later!

```bash
curl -X POST "http://localhost:2095/app/apiv2/save" \
  -H "Token: <Your Token Key>" \
  -H "Content-Type: application/json" \
  -d '{
    "object": "endpoints",
    "action": "edit",
    "data": {
      "id": 1,
      "type": "wireguard",
      "tag": "my-wg-endpoint",
      "local_address": ["10.0.0.2/32"],
      "private_key": "YOUR_PRIVATE_KEY",
      "peers": [
        {
          "server": "198.51.100.1",
          "server_port": 51820,
          "public_key": "auto",
          "allowed_ips": ["192.168.1.0/24"]
        }
      ]
    }
  }'
```

**5. Restart the Sing-Box Core (`/apiv2/restartSb`)**
To apply changes dynamically, restart the core.

```bash
curl -X POST "http://localhost:2095/app/apiv2/restartSb" \
  -H "Token: <Your Token Key>"
```

### GET Endpoints

| Endpoint | Description | Query Parameters |
|----------|-------------|------------------|
| `GET /apiv2/load` | Load full MPanel data | `lu` (string): Last update timestamp |
| `GET /apiv2/inbounds` | Get inbound object(s) | `id` (string, optional): Specific inbound ID |
| `GET /apiv2/outbounds` | Get outbound objects | None |
| `GET /apiv2/endpoints` | Get endpoint objects | None |
| `GET /apiv2/services` | Get service objects | None |
| `GET /apiv2/tls` | Get TLS objects | None |
| `GET /apiv2/clients` | Get client objects | `id` (string, optional): Specific client ID |
| `GET /apiv2/config` | Get config objects | None |
| `GET /apiv2/users` | Retrieve user list | None |
| `GET /apiv2/settings` | Get app settings | None |
| `GET /apiv2/stats` | Get statistical data | `resource` (string), `tag` (string), `limit` (integer, default: 100) |
| `GET /apiv2/status` | Get server status | `r` (string): Status request types (comma separated) |
| `GET /apiv2/onlines` | Get online lists | None |
| `GET /apiv2/logs` | Retrieve server logs | `c` (integer): Number of logs, `l` (string): Log level |
| `GET /apiv2/changes` | Get user changes/audit logs | `a` (string, optional): actor name, `k` (string, optional): key, `c` (integer) limit |
| `GET /apiv2/keypairs` | Get cryptographic keypairs | `k` (string): ech |
| `GET /apiv2/getdb` | Download the database file | `exclude` (string): Fields to exclude |
| `GET /apiv2/checkOutbound` | Test outbound connectivity | None |

### AmneziaWG Endpoints

The AmneziaWG integration provides direct management of a standalone WireGuard/AmneziaWG interface.

| Endpoint | Method | Description | Request Body / Query Params |
|----------|--------|-------------|-----------------------------|
| `/apiv2/amnezia/config` | GET | Get the server configuration | None |
| `/apiv2/amnezia/config` | POST | Save the server configuration | `AmneziaConfig` JSON object (set `privateKey: "auto"` to generate) |
| `/apiv2/amnezia/start` | POST | Start the AmneziaWG interface | None |
| `/apiv2/amnezia/stop` | POST | Stop the AmneziaWG interface | None |
| `/apiv2/amnezia/status` | GET | Get interface running status | None |
| `/apiv2/amnezia/peers` | GET | List all peers | None |
| `/apiv2/amnezia/peers/:id` | GET | Get a specific peer by ID | None |
| `/apiv2/amnezia/peers` | POST | Add a new peer | `AmneziaPeer` JSON object |
| `/apiv2/amnezia/peers/:id` | PUT | Edit an existing peer | `AmneziaPeer` JSON object |
| `/apiv2/amnezia/peers/:id` | DELETE | Delete a specific peer | None |
| `/apiv2/amnezia/peers/:id/toggle` | POST | Enable/Disable a peer | None |
| `/apiv2/amnezia/peers/:id/config` | GET | Download peer `.conf` file | `server` (string, optional): Server IP/Hostname |
| `/apiv2/amnezia/keypair` | GET | Generate a new keypair | None |

### Real-World Examples (GET)

**1. Retrieve a specific Inbound by ID (`/apiv2/inbounds`)**
```bash
curl -H "Token: <Your Token Key>" \
  "http://localhost:2095/app/apiv2/inbounds?id=1"
```

**2. Check Server Status & Resources (`/apiv2/status`)**
```bash
curl -H "Token: <Your Token Key>" \
  "http://localhost:2095/app/apiv2/status?r=cpu,mem,net,sys,sbd,dsk,swp,dio"
```

**3. Generate a WireGuard Keypair (`/apiv2/keypairs`)**
You can automatically generate a valid private and public key pair for use in WireGuard endpoints.
```bash
curl -H "Token: <Your Token Key>" \
  "http://localhost:2095/app/apiv2/keypairs?k=wireguard"
```

#### Status Request Options (`r` query parameter)
You can choose which information you need by passing a comma-separated list of the following options:

| Option | Return Description |
|--------|--------------------|
| `cpu` | CPU usage and load status |
| `mem` | RAM/Memory status |
| `net` | Network statistics (TX/RX) |
| `sys` | General system information |
| `sbd` | Sing-box core information |
| `dsk` | General disk status |
| `dio` | Disk I/O usage statistics |
| `swp` | Swap memory status |
| `db`  | Database statistics and usages |
