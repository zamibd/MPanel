# MPanel
**An Advanced Web Panel • Built on SagerNet/Sing-Box**

![](https://img.shields.io/github/v/release/alireza0/mpanel.svg)
![MPanel Docker pull](https://img.shields.io/docker/pulls/alireza7/mpanel.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/zamibd/MPanel)](https://goreportcard.com/report/github.com/zamibd/MPanel)
[![Downloads](https://img.shields.io/github/downloads/alireza0/mpanel/total.svg)](https://img.shields.io/github/downloads/alireza0/mpanel/total.svg)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> **Disclaimer:** This project is only for personal learning and communication, please do not use it for illegal purposes, please do not use it in a production environment

**If you think this project is helpful to you, you may wish to give a**:star2:

**Want to contribute?** See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, testing, and the pull request process.

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/alireza7)

<a href="https://nowpayments.io/donation/alireza7" target="_blank" rel="noreferrer noopener">
   <img src="https://nowpayments.io/images/embeds/donation-button-white.svg" alt="Crypto donation button by NOWPayments">
</a>

## Quick Overview
| Features                               |      Enable?       |
| -------------------------------------- | :----------------: |
| Multi-Protocol                         | :heavy_check_mark: |
| Multi-Language                         | :heavy_check_mark: |
| Multi-Client/Inbound                   | :heavy_check_mark: |
| Advanced Traffic Routing Interface     | :heavy_check_mark: |
| Client & Traffic & System Status       | :heavy_check_mark: |
| Subscription Link (link/json/clash + info)| :heavy_check_mark: |
| Dark/Light Theme                       | :heavy_check_mark: |
| API Interface                          | :heavy_check_mark: |

## Supported Platforms
| Platform | Architecture | Status |
|----------|--------------|---------|
| Linux    | amd64, arm64, armv7, armv6, armv5, 386, s390x | ✅ Supported |
| Windows  | amd64, 386, arm64 | ✅ Supported |
| macOS    | amd64, arm64 | 🚧 Experimental |

## Screenshots

!["Main"](https://github.com/zamibd/MPanel-frontend/raw/main/media/main.png)

[Other UI Screenshots](https://github.com/zamibd/MPanel-frontend/blob/main/screenshots.md)

## API Documentation

[API-Documentation Wiki](https://github.com/zamibd/MPanel/wiki/API-Documentation)

## Default Installation Information
- Panel Port: 2095
- Panel Path: /app/
- Subscription Port: 2096
- Subscription Path: /sub/
- User/Password: admin

## Install & Upgrade to Latest Version

### Linux/macOS
```sh
bash <(curl -Ls https://raw.githubusercontent.com/alireza0/mpanel/master/install.sh)
```

### Windows
1. Download the latest Windows release from [GitHub Releases](https://github.com/zamibd/MPanel/releases/latest)
2. Extract the ZIP file
3. Run `install-windows.bat` as Administrator
4. Follow the installation wizard

## Install legacy Version

**Step 1:** To install your desired legacy version, add the version to the end of the installation command. e.g., ver `1.0.0`:

```sh
VERSION=1.0.0 && bash <(curl -Ls https://raw.githubusercontent.com/alireza0/mpanel/$VERSION/install.sh) $VERSION
```

## Manual installation

### Linux/macOS
1. Get the latest version of MPanel based on your OS/Architecture from GitHub: [https://github.com/zamibd/MPanel/releases/latest](https://github.com/zamibd/MPanel/releases/latest)
2. **OPTIONAL** Get the latest version of `mpanel.sh` [https://raw.githubusercontent.com/alireza0/mpanel/master/mpanel.sh](https://raw.githubusercontent.com/alireza0/mpanel/master/mpanel.sh)
3. **OPTIONAL** Copy `mpanel.sh` to /usr/bin/ and run `chmod +x /usr/bin/mpanel`.
4. Extract mpanel tar.gz file to a directory of your choice and navigate to the directory where you extracted the tar.gz file.
5. Copy *.service files to /etc/systemd/system/ and run `systemctl daemon-reload`.
6. Enable autostart and start MPanel service using `systemctl enable mpanel --now`
7. Start sing-box service using `systemctl enable sing-box --now`

### Windows
1. Get the latest Windows version from GitHub: [https://github.com/zamibd/MPanel/releases/latest](https://github.com/zamibd/MPanel/releases/latest)
2. Download the appropriate Windows package (e.g., `mpanel-windows-amd64.zip`)
3. Extract the ZIP file to a directory of your choice
4. Run `install-windows.bat` as Administrator
5. Follow the installation wizard
6. Access the panel at http://localhost:2095/app

## Uninstall MPanel

```sh
sudo -i

systemctl disable mpanel  --now

rm -f /etc/systemd/system/sing-box.service
systemctl daemon-reload

rm -fr /usr/local/mpanel
rm /usr/bin/mpanel
```

## Install using Docker

<details>
   <summary>Click for details</summary>

### Usage

**Step 1:** Install Docker

```shell
curl -fsSL https://get.docker.com | sh
```

**Step 2:** Install MPanel

> Docker compose method

```shell
mkdir mpanel && cd mpanel
wget -q https://raw.githubusercontent.com/alireza0/mpanel/master/docker-compose.yml
docker compose up -d
```

> Use docker

```shell
mkdir mpanel && cd mpanel
docker run -itd \
    -p 2095:2095 -p 2096:2096 -p 443:443 -p 80:80 \
    -v $PWD/db/:/app/db/ \
    -v $PWD/cert/:/root/cert/ \
    --name mpanel --restart=unless-stopped \
    alireza7/mpanel:latest
```

> Build your own image

```shell
git clone https://github.com/zamibd/MPanel
git submodule update --init --recursive
docker build -t mpanel .
```

</details>

## Manual run ( contribution )

<details>
   <summary>Click for details</summary>

### Build and run whole project
```shell
./runMPanel.sh
```

### Clone the repository
```shell
# clone repository
git clone https://github.com/zamibd/MPanel
# clone submodules
git submodule update --init --recursive
```


### - Frontend

Visit [mpanel-frontend](https://github.com/zamibd/MPanel-frontend) for frontend code

### - Backend
> Please build frontend once before!

To build backend:
```shell
# remove old frontend compiled files
rm -fr web/html/*
# apply new frontend compiled files
cp -R frontend/dist/ web/html/
# build
go build -o mpanel main.go
```

To run backend (from root folder of repository):
```shell
./mpanel
```

</details>

## Languages

- English
- Farsi
- Vietnamese
- Chinese (Simplified)
- Chinese (Traditional)
- Russian

## Features

- Supported protocols:
  - General:  Mixed, SOCKS, HTTP, HTTPS, Direct, Redirect, TProxy
  - V2Ray based: VLESS, VMess, Trojan, Shadowsocks
  - Other protocols: ShadowTLS, Hysteria, Hysteria2, Naive, TUIC
- Supports XTLS protocols
- An advanced interface for routing traffic, incorporating PROXY Protocol, External, and Transparent Proxy, SSL Certificate, and Port
- An advanced interface for inbound and outbound configuration
- Clients’ traffic cap and expiration date
- Displays online clients, inbounds and outbounds with traffic statistics, and system status monitoring
- Subscription service with ability to add external links and subscription
- HTTPS for secure access to the web panel and subscription service (self-provided domain + SSL certificate)
- Dark/Light theme

## Environment Variables

<details>
  <summary>Click for details</summary>

### Usage

| Variable       |                      Type                      | Default       |
| -------------- | :--------------------------------------------: | :------------ |
| MPANEL_LOG_LEVEL  | `"debug"` \| `"info"` \| `"warn"` \| `"error"` | `"info"`      |
| MPANEL_DEBUG      |                   `boolean`                    | `false`       |
| MPANEL_BIN_FOLDER |                    `string`                    | `"bin"`       |
| MPANEL_DB_FOLDER  |                    `string`                    | `"db"`        |
| SINGBOX_API    |                    `string`                    | -             |

</details>

## SSL Certificate

<details>
  <summary>Click for details</summary>

### Certbot

```bash
snap install core; snap refresh core
snap install --classic certbot
ln -s /snap/bin/certbot /usr/bin/certbot

certbot certonly --standalone --register-unsafely-without-email --non-interactive --agree-tos -d <Your Domain Name>
```

</details>

## Stargazers over Time
[![Stargazers over time](https://starchart.cc/alireza0/mpanel.svg)](https://starchart.cc/alireza0/mpanel)
