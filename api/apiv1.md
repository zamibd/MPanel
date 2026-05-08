# MPanel API Documentation (v1)

This documentation covers the **Frontend API** (`/api`), designed primarily for the MPanel frontend web application. Unlike the token-based REST API (`/apiv2`), this API relies on **Cookie-based Session Authentication**.

## Authentication

To interact with the v1 API, you must first authenticate by logging in and obtaining a session cookie.

### 1. Login
Send a POST request to the `/api/login` endpoint with your username and password.

```bash
curl -X POST -c cookies.txt -d "username=admin&password=yourpassword" "http://localhost:2095/app/api/login"
```

If successful, the server will return a standard JSON response and set a session cookie. You must include this cookie in all subsequent requests.

### 2. Using the Session Cookie
For subsequent requests, attach the session cookie received during login.

```bash
curl -b cookies.txt "http://localhost:2095/app/api/inbounds"
```

### 3. Logout
To invalidate your current session, send a GET request to `/api/logout`.

```bash
curl -b cookies.txt "http://localhost:2095/app/api/logout"
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
- **`msg`**: A message describing the result of the operation.
- **`obj`**: The object containing the requested data or the result of the action.

---

## API Endpoints

The v1 API inherits most of its functionality from the core `ApiService` (similar to `/apiv2`), but includes additional session and system-management endpoints.

### POST Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /api/login` | Authenticate and obtain a session cookie |
| `POST /api/changePass` | Change the administrator password |
| `POST /api/save` | Save configuration data (e.g., inbounds, clients) |
| `POST /api/restartApp` | Restart the MPanel application |
| `POST /api/restartSb` | Restart the Sing-Box Core |
| `POST /api/linkConvert` | Convert a generic node link |
| `POST /api/subConvert` | Convert a subscription link to JSON |
| `POST /api/importdb` | Import a database file (FormData) |
| `POST /api/addToken` | Generate a new API token for `/apiv2` access |
| `POST /api/deleteToken` | Delete an existing API token |

### GET Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/logout` | Invalidate the current session |
| `GET /api/load` | Load full MPanel data |
| `GET /api/inbounds` | Get inbound object(s) |
| `GET /api/outbounds` | Get outbound objects |
| `GET /api/endpoints` | Get endpoint objects |
| `GET /api/services` | Get service objects |
| `GET /api/tls` | Get TLS objects |
| `GET /api/clients` | Get client objects |
| `GET /api/config` | Get config objects |
| `GET /api/users` | Retrieve user list |
| `GET /api/settings` | Get app settings |
| `GET /api/stats` | Get statistical data |
| `GET /api/status` | Get server status (cpu, mem, net, sys, etc.) |
| `GET /api/onlines` | Get online connection lists |
| `GET /api/logs` | Retrieve server logs |
| `GET /api/changes` | Get user changes/audit logs |
| `GET /api/keypairs` | Get cryptographic keypairs |
| `GET /api/getdb` | Download the database file |
| `GET /api/tokens` | Retrieve the list of active API tokens |
| `GET /api/singbox-config` | Download the generated Sing-Box `config.json` |
| `GET /api/checkOutbound` | Test outbound connectivity |

### AmneziaWG Endpoints

The AmneziaWG integration provides direct management of a standalone WireGuard/AmneziaWG interface.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/amnezia/config` | GET | Get the server configuration |
| `/api/amnezia/config` | POST | Save the server configuration (set `privateKey: "auto"` to generate) |
| `/api/amnezia/start` | POST | Start the AmneziaWG interface |
| `/api/amnezia/stop` | POST | Stop the AmneziaWG interface |
| `/api/amnezia/status` | GET | Get interface running status |
| `/api/amnezia/peers` | GET | List all peers |
| `/api/amnezia/peers/:id` | GET | Get a specific peer by ID |
| `/api/amnezia/peers` | POST | Add a new peer (`AmneziaPeer` JSON body) |
| `/api/amnezia/peers/:id` | PUT | Edit an existing peer (`AmneziaPeer` JSON body) |
| `/api/amnezia/peers/:id` | DELETE | Delete a specific peer |
| `/api/amnezia/peers/:id/toggle` | POST | Enable/Disable a peer |
| `/api/amnezia/peers/:id/config` | GET | Download peer `.conf` file (query: `server` IP/domain) |
| `/api/amnezia/keypair` | GET | Generate a new WireGuard keypair |

> **Note:** For specific query parameters and JSON payload structures for data endpoints (like `/status`, `/save`, `/stats`), please refer to the `/apiv2` documentation, as the parameters are identical between `/api` and `/apiv2`.

### Real-World Examples (Using Cookies)

**1. Generate a new API Token (`/api/addToken`)**
This requires you to have a valid session cookie.

```bash
curl -X POST "http://localhost:2095/app/api/addToken" \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "expiry": 1735689600
  }'
```

**2. List active tokens (`/api/tokens`)**
```bash
curl -b cookies.txt "http://localhost:2095/app/api/tokens"
```

**3. Generate a WireGuard Keypair (`/api/keypairs`)**
You can automatically generate a valid private and public key pair for use in WireGuard endpoints.
```bash
curl -b cookies.txt "http://localhost:2095/app/api/keypairs?k=wireguard"
```

**4. Test Outbound Connectivity (`/api/checkOutbound`)**
```bash
curl -b cookies.txt "http://localhost:2095/app/api/checkOutbound"
```
