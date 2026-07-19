# Host your VSmart API on AWS Lightsail (~12 users)
#
# Stack: Ubuntu + Docker Compose (API, workers, Postgres, Nginx)
# Frontend stays on Vercel.
#
# No domain? Use Part A (HTTP via public IP).
# Have a domain later? Do Part B (HTTPS with Caddy).

## 0. What you will build

```text
Users → Vercel frontend
            │
            ▼
   http(s)://YOUR_API
            │
            ▼
   Lightsail Ubuntu (~$7/mo)
     Nginx → Go API + workers + Postgres
```

**Never open Postgres port 5432 to the internet.**

---

## 1. Create Lightsail instance

1. Open https://lightsail.aws.amazon.com
2. **Create instance**
   - Region: Mumbai `ap-south-1` (if users are in India)
   - OS: **Ubuntu 22.04 LTS**
   - Plan: **$7** (1 GB RAM recommended)
   - Name: `vsmart-api`
3. Wait until **Running**
4. **Networking → Firewall** allow only:
   - SSH `22`
   - HTTP `80`
   - HTTPS `443`
5. Copy the **Public IP**

> Do **not** add 5432.

---

## 2. Connect with SSH (browser is easiest)

Instance → **Connect using SSH**

---

## 3. Server hardening (first login)

### 3.1 Ubuntu firewall

```bash
sudo apt update
sudo apt install -y ufw fail2ban unattended-upgrades htop ncdu

sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
# Temporary if you have NO domain yet and will hit IP:8081 from outside:
# sudo ufw allow 8081/tcp
sudo ufw --force enable
sudo ufw status
```

### 3.2 Automatic security updates

```bash
sudo dpkg-reconfigure -plow unattended-upgrades
# Choose Yes
```

### 3.3 fail2ban (SSH brute-force protection)

```bash
sudo systemctl enable --now fail2ban
sudo systemctl status fail2ban --no-pager
```

---

## 4. Install Docker (official apt repo — not get.docker.com)

```bash
sudo apt update
sudo apt install -y ca-certificates curl gnupg

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo usermod -aG docker ubuntu
```

**Log out of SSH and reconnect**, then:

```bash
docker --version
docker compose version
```

---

## 5. Folder layout

```bash
mkdir -p ~/app ~/backups ~/app/logs ~/app/scripts
```

---

## 6. GitHub Deploy Key (no password on server)

### On the server

```bash
ssh-keygen -t ed25519 -C "vsmart-lightsail-deploy" -f ~/.ssh/vsmart_deploy -N ""
cat ~/.ssh/vsmart_deploy.pub
```

### On GitHub

1. Repo → **Settings → Deploy keys → Add deploy key**
2. Paste the public key
3. Allow **read-only**

### SSH config on server

```bash
cat >> ~/.ssh/config <<'EOF'
Host github.com
  HostName github.com
  User git
  IdentityFile ~/.ssh/vsmart_deploy
  IdentitiesOnly yes
EOF
chmod 600 ~/.ssh/config
ssh -T git@github.com || true
```

### Clone

```bash
cd ~/app
git clone git@github.com:sarveshsusi/vsmarttech-api.git .
git checkout feature/modular-quality
```

(If clone into `~/app` complains about non-empty dir, clone into a temp folder then move.)

Alternate clean clone:

```bash
cd ~
git clone git@github.com:sarveshsusi/vsmarttech-api.git app-src
mv app-src/* app-src/.[!.]* ~/app/ 2>/dev/null || true
# Preferred structure:
# ~/app/backend  (compose lives here)
ln -sfn ~/app/backend/scripts ~/app/scripts 2>/dev/null || cp -r ~/app/backend/scripts/* ~/app/scripts/
```

Recommended final layout:

```text
/home/ubuntu
  ├── app
  │    └── backend/     # docker compose, .env, Dockerfile
  ├── backups/
  └── scripts/          # backup.sh, update.sh copies
```

```bash
cp ~/app/backend/scripts/*.sh ~/scripts/ 2>/dev/null || cp ~/app/backend/scripts/*.sh ~/app/scripts/
chmod +x ~/app/backend/scripts/*.sh
mkdir -p ~/scripts
cp ~/app/backend/scripts/*.sh ~/scripts/
chmod +x ~/scripts/*.sh
```

---

## 7. Create `.env`

```bash
cd ~/app/backend
cp .env.production.example .env
nano .env
```

Generate secrets:

```bash
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 24
```

Minimum production values:

```text
APP_ENV=production
SERVER_PORT=8080
RUN_INPROCESS_CRONS=false

POSTGRES_USER=vsmart
POSTGRES_PASSWORD=LONG_RANDOM_PASSWORD
POSTGRES_DB=vsmartcrm_db
DATABASE_URL=postgresql://vsmart:LONG_RANDOM_PASSWORD@postgres:5432/vsmartcrm_db

JWT_ACCESS_SECRET=...
JWT_REFRESH_SECRET=...
REMEMBER_DEVICE_SECRET=...

FRONTEND_URL=https://YOUR_VERCEL_OR_CRM_URL

STORAGE_TYPE=s3
AWS_REGION=ap-south-1
AWS_ACCESS_KEY_ID=...          # see IAM note below
AWS_SECRET_ACCESS_KEY=...
AWS_S3_BUCKET=...
AWS_S3_FOLDER=uploads

MAIL_HOST=...
MAIL_PORT=587
MAIL_USERNAME=...
MAIL_PASSWORD=...
MAIL_FROM=no-reply@example.com

# optional for backup.sh S3 upload
# BACKUP_S3_URI=s3://your-backup-bucket/vsmart
```

### IAM note (Lightsail)

Lightsail does **not** work like EC2 instance roles for arbitrary S3 buckets in most setups.

Do this instead:

1. IAM → create user `vsmart-s3-uploader`
2. Attach a policy that only allows `s3:PutObject`, `s3:GetObject` on your bucket ARN
3. Create access keys → put in `.env`

Rotate keys if they ever leak.

---

## 8. Start the stack

```bash
cd ~/app/backend
docker compose up -d --build
```

### No domain yet (HTTP via public IP)

1. Lightsail Networking → add **Custom TCP 8081**
2. UFW: `sudo ufw allow 8081/tcp`
3. Test:

```bash
curl http://YOUR_PUBLIC_IP:8081/healthz
curl http://YOUR_PUBLIC_IP:8081/readyz
```

4. Vercel: `VITE_API_URL=http://YOUR_PUBLIC_IP:8081`

> Fine for early testing. Add a domain + HTTPS soon.

### With Caddy later (HTTPS)

Bind Nginx to localhost only so 8081 is not public:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod-caddy.yml up -d
```

Remove **8081** from Lightsail firewall and UFW once Caddy is working.

---

## 9. Verify containers

```bash
docker compose ps
docker stats --no-stream
df -h
curl -fsS http://127.0.0.1:8081/healthz
curl -fsS http://127.0.0.1:8081/readyz
```

Compose already includes:

- `restart: unless-stopped`
- healthchecks
- log rotation (`max-size: 10m`, `max-file: 5`)

---

## 10. Create admin user

On the server:

```bash
cd /tmp
cat > hash.go <<'EOF'
package main
import ("fmt"; "golang.org/x/crypto/bcrypt")
func main() {
  h, err := bcrypt.GenerateFromPassword([]byte("Admin@12345"), 12)
  if err != nil { panic(err) }
  fmt.Print(string(h))
}
EOF

HASH=$(docker run --rm -v "$PWD":/work -w /work golang:1.25-alpine \
  sh -c 'go mod init t >/dev/null && go get golang.org/x/crypto/bcrypt >/dev/null && go run hash.go')

cd ~/app/backend
source .env
docker compose exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "INSERT INTO users (id,name,email,password,role,is_active,must_reset_password,two_fa_enabled,last_password_reset_at,created_at,updated_at) VALUES (gen_random_uuid(),'Admin','admin@vsmart.local','$HASH','admin',true,false,false,NOW(),NOW(),NOW()) ON CONFLICT (email) DO UPDATE SET password=EXCLUDED.password, role='admin', is_active=true, updated_at=NOW();"
```

Login with `admin@vsmart.local` / `Admin@12345` and change the password.

---

## 11. Daily database backups

```bash
chmod +x ~/scripts/backup.sh ~/scripts/update.sh
# Fix paths inside scripts if needed: APP_DIR=$HOME/app
```

Edit crontab:

```bash
crontab -e
```

Add:

```cron
15 2 * * * APP_DIR=/home/ubuntu/app /home/ubuntu/scripts/backup.sh >> /home/ubuntu/app/logs/backup.log 2>&1
```

Test once:

```bash
APP_DIR=/home/ubuntu/app /home/ubuntu/scripts/backup.sh
ls -lh ~/backups
```

Optional S3: install AWS CLI, set `BACKUP_S3_URI=s3://bucket/vsmart` in `.env` or environment.

---

## 12. One-command updates

```bash
~/scripts/update.sh
```

---

## 13. When you buy a domain (Part B — HTTPS)

DNS **A record**: `api` → Lightsail public IP

Install Caddy:

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
```

Bind Nginx to localhost only:

```bash
cd ~/app/backend
docker compose -f docker-compose.yml -f docker-compose.prod-caddy.yml up -d
```

`/etc/caddy/Caddyfile`:

```caddy
api.yourdomain.com {
    encode gzip

    reverse_proxy 127.0.0.1:8081

    header {
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
        Referrer-Policy strict-origin-when-cross-origin
    }
}
```

```bash
sudo systemctl reload caddy
curl https://api.yourdomain.com/healthz
```

Then switch Vercel to:

```text
VITE_API_URL=https://api.yourdomain.com
```

Remove public `8081` from Lightsail firewall + UFW once Caddy is working.

---

## 14. Optional monitoring (Netdata)

```bash
bash <(curl -Ss https://my-netdata.io/kickstart.sh)
```

Open `http://YOUR_IP:19999` (only after locking Netdata down / using SSH tunnel — do not leave it public forever).

Safer:

```bash
ssh -L 19999:127.0.0.1:19999 ubuntu@YOUR_IP
```

Then open http://127.0.0.1:19999 on your laptop.

---

## 15. Connect Vercel

1. Vercel → Environment Variables  
2. `VITE_API_URL=http://YOUR_IP:8081` (no domain) **or** `https://api.yourdomain.com`  
3. Redeploy frontend  
4. Backend `FRONTEND_URL` must match the browser origin exactly  

---

## Checklist

- [ ] Lightsail $7 Ubuntu running  
- [ ] Firewall: 22/80/443 only (5432 closed)  
- [ ] UFW + fail2ban + unattended-upgrades  
- [ ] Docker from official apt repo  
- [ ] Deploy key clone  
- [ ] `.env` filled (strong secrets)  
- [ ] `docker compose ps` healthy  
- [ ] `/healthz` and `/readyz` OK  
- [ ] Daily `backup.sh` cron  
- [ ] `update.sh` works  
- [ ] Vercel pointed at API  
- [ ] Admin can log in  

---

## Is $7 enough for 12 users?

Yes, for a modest Go API + Postgres + Docker with light traffic. Watch:

```bash
htop
docker stats
df -h
```

Upgrade the Lightsail plan if RAM is constantly maxed.
