# MPanel

**An Advanced Web Panel • Built on SagerNet/Sing-Box**

![](https://img.shields.io/github/v/release/zamibd/MPanel.svg)
![MPanel Docker pull](https://img.shields.io/docker/pulls/zamibd/mpanel.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/zamibd/MPanel)](https://goreportcard.com/report/github.com/zamibd/MPanel)
[![Downloads](https://img.shields.io/github/downloads/zamibd/MPanel/total.svg)](https://img.shields.io/github/downloads/zamibd/MPanel/total.svg)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> **Disclaimer:** This project is only for personal learning and communication. Please do not use it for illegal purposes or in a production environment without proper compliance.

**Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, testing, and the pull request process.

---

## ⚡ Quick Overview

| Features                               |      Status        |
| -------------------------------------- | :----------------: |
| Multi-Protocol                         | ✅ Supported        |
| Multi-Language                         | ✅ Supported        |
| Multi-Client/Inbound                   | ✅ Supported        |
| Advanced Traffic Routing Interface     | ✅ Supported        |
| Client & Traffic & System Status       | ✅ Supported        |
| Subscription Link (link/json/clash)    | ✅ Supported        |
| Dark/Light Theme                       | ✅ Supported        |
| API Interface                          | ✅ Supported        |

## 💻 Supported Platforms

| Platform | Architecture                                  | Status          |
|----------|-----------------------------------------------|-----------------|
| **Linux**    | amd64, arm64, armv7, armv6, armv5, 386, s390x | ✅ Supported     |
| **Windows**  | amd64, 386, arm64                             | ✅ Supported     |
| **macOS**    | amd64, arm64                                  | 🚧 Experimental  |

---

## 📸 Screenshots

!["Main"](https://github.com/zamibd/MPanel-frontend/raw/main/media/main.png)

> [View more UI Screenshots](https://github.com/zamibd/MPanel-frontend/blob/main/screenshots.md)

---

## 🚀 Installation & Upgrade

### Default Information
- **Panel Port:** `2095`
- **Panel Path:** `/app/`
- **Subscription Port:** `2096`
- **Subscription Path:** `/sub/`
- **User/Password:** `admin` / `admin`

### Linux & macOS (Automated Script)
The easiest way to install or upgrade to the latest version of MPanel is via our bash script:

```sh
bash <(curl -Ls https://raw.githubusercontent.com/zamibd/MPanel/main/install.sh)
```

### Install using Docker (Recommended)

1. Clone the repository:
   ```shell
   git clone https://github.com/zamibd/MPanel
   cd MPanel
   git submodule update --init --recursive
   ```
2. Initialize your environment variables:
   ```shell
   make env-init
   ```
3. Start the panel and database in the background:
   ```shell
   make docker-up
   # or simply: make up
   ```

**Helpful Makefile Commands:**
- `make up` / `make down` - Start / Stop services
- `make logs` - View live logs from all services
- `make restart` - Restart all services
- `make ps` - View container status

### Windows Installation
1. Download the latest Windows release from [GitHub Releases](https://github.com/zamibd/MPanel/releases/latest).
2. Extract the ZIP file to your preferred directory.
3. Run `install-windows.bat` as an Administrator.
4. Follow the installation wizard.
5. Access the panel at `http://localhost:2095/app`.

---

## 🛠 Manual Build & Run (For Contributors)

To manually build and run the project from source, please ensure you have built the frontend first!

```shell
# 1. Clone repository & submodules
git clone https://github.com/zamibd/MPanel
cd MPanel
git submodule update --init --recursive

# 2. Apply frontend compiled files (assuming you built them in MPanel-frontend)
rm -fr web/html/*
cp -R frontend/dist/ web/html/

# 3. Build the backend binary
go build -o mpanel main.go

# 4. Run it
./mpanel
```

---

## 📖 API Documentation
If you are integrating MPanel programmatically, please refer to our [API-Documentation Wiki](https://github.com/zamibd/MPanel/wiki/API-Documentation).

## ⚙️ Environment Variables

| Variable          | Type                                           | Default       |
| ----------------- | ---------------------------------------------- | ------------- |
| `MPANEL_LOG_LEVEL`| `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"`      |
| `MPANEL_DEBUG`    | `boolean`                                      | `false`       |
| `MPANEL_BIN_FOLDER`| `string`                                       | `"bin"`       |
| `MPANEL_DB_FOLDER`| `string`                                       | `"db"`        |
| `SINGBOX_API`     | `string`                                       | -             |

## 📊 Stargazers over Time
[![Stargazers over time](https://starchart.cc/zamibd/MPanel.svg)](https://starchart.cc/zamibd/MPanel)
