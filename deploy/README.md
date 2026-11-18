# Deploying Cagie (VPS + Vercel)

```
Customer → https://shop.<domain>  Vercel (Next.js web app, apps/web)
         → https://api.<domain>   your VPS: Caddy → gateway → services, Postgres, RabbitMQ
```

Replace `<domain>` with your domain everywhere below, for example `shop.example.com`.
Do the steps in order: the API must be live before the web app is built.

## 1. DNS for the API

At your domain registrar, add:

| Type | Name | Value |
|---|---|---|
| A | `api` | your VPS's public IPv4 address |

Check it from your laptop (it can take a few minutes): `dig +short api.<domain>` must print the VPS IP.
Caddy needs this before it starts, to get the HTTPS certificate.

## 2. Prepare the server (Ubuntu 24.04, once)

Log in: `ssh <user>@<vps ip>`. Every command below runs on the VPS.

**Updates**
```bash
sudo apt update && sudo apt upgrade -y
```

**Swap (2 GB of disk used as spare memory, so a busy moment does not crash anything)**
```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
free -h   # "Swap: 2.0Gi" means it worked
```

**Firewall (allow SSH first, or you lock yourself out)**
```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 443/udp
sudo ufw enable
sudo ufw status
```
Only Caddy publishes ports (80, 443). Postgres, RabbitMQ and the services have no published ports, so they are unreachable from the internet whatever the firewall says.

**Docker** (from the official guide, https://docs.docker.com/engine/install/ubuntu/)
```bash
sudo apt install -y ca-certificates curl git
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
sudo tee /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo docker run --rm hello-world
```
Optional, to run docker without `sudo`: `sudo usermod -aG docker $USER`, then log out and back in.

## 3. Get the code and make the secrets (once)

```bash
sudo git clone https://github.com/hogie12/cagie-shop.git /opt/cagie-shop
sudo chown -R $USER:$USER /opt/cagie-shop
cd /opt/cagie-shop/deploy
sudo ./setup.sh shop.<domain> api.<domain> <your email>
```
`setup.sh` writes `deploy/.env` (random passwords) and `deploy/keys/` (JWT keys). It prints the **admin password once**: save it in a password manager. These files stay on the server and are gitignored.

## 4. Start it

```bash
cd /opt/cagie-shop/deploy
sudo ./deploy.sh
```
The first start pulls the images (a few minutes) and Caddy fetches the certificate. Then, from your laptop:
```bash
curl -s https://api.<domain>/products | head -c 300
```

## 5. The web app on Vercel

1. Vercel → **Add New → Project** → import `hogie12/cagie-shop`.
2. **Root Directory**: `apps/web` (framework: Next.js, detected).
3. **Environment Variables** (both):
   - `NEXT_PUBLIC_API_URL` = `https://api.<domain>`
   - `API_URL` = `https://api.<domain>`
4. **Deploy**.
5. Project → **Settings → Domains** → add `shop.<domain>` and create the DNS record Vercel shows you (usually a CNAME for `shop`).

The shop and the API must be on the same registered domain (`shop.` and `api.`), or the sign-in cookie will not be sent.

## 6. Daily backups

```bash
crontab -e
# add this line: every day at 03:00
0 3 * * * /opt/cagie-shop/deploy/backup.sh >> /opt/cagie-shop/deploy/backups/backup.log 2>&1
```
Copy `deploy/backups/` off the server now and then (for example with `scp`).

## Everyday operations

| Task | Command (in `/opt/cagie-shop/deploy`) |
|---|---|
| Update to the newest images | `git pull && sudo ./deploy.sh latest` |
| Roll back to one exact build | `sudo ./deploy.sh <full commit sha>` |
| See what runs | `sudo docker compose -f docker-compose.prod.yml ps` |
| Follow a service's logs | `sudo docker compose -f docker-compose.prod.yml logs -f order` |
| Memory use | `sudo docker stats --no-stream` |
| Restore one database | `sudo docker compose -f docker-compose.prod.yml exec -T postgres pg_restore -U cagie -d orders --clean < backups/orders-YYYY-MM-DD.dump` |
