# Parsa Go - Systemd Deployment (Debian 13)

## Server Setup

### 1. Install PostgreSQL

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

### 2. Create Dedicated User

```bash
sudo useradd -r -m -d /home/parsa -s /bin/bash parsa
sudo mkdir -p /home/parsa/parsa-go/bin
sudo chown -R parsa:parsa /home/parsa/parsa-go
```

### 3. Setup PostgreSQL Database

```bash
sudo -u postgres psql
```

```sql
CREATE USER parsa_admin WITH PASSWORD 'your_secure_password';
CREATE DATABASE parsadb OWNER parsa_admin;
GRANT ALL PRIVILEGES ON DATABASE parsadb TO parsa_admin;
```

## Deployment (from dev machine)

### 1. Build Binary

```bash
# On your development machine
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/parsa ./cmd/api/
```

### 2. Copy Files to Server

```bash
# Copy binary
scp bin/parsa user@server:/home/parsa/parsa-go/bin/

# Copy production .env
scp .env.production user@server:/home/parsa/parsa-go/.env

# Copy systemd service file
scp systemd/parsa-go.service user@server:/tmp/
```

### 3. Install Service (on server)

```bash
# Move service file
sudo mv /tmp/parsa-go.service /etc/systemd/system/

# Set permissions
sudo chown -R parsa:parsa /home/parsa/parsa-go
sudo chmod 600 /home/parsa/parsa-go/.env
sudo chmod 755 /home/parsa/parsa-go/bin/parsa

# Reload systemd and enable service
sudo systemctl daemon-reload
sudo systemctl enable parsa-go
sudo systemctl start parsa-go
```

## Granting Access to TLS Certificates

To allow the `parsa` service account to read Let's Encrypt TLS certificates:

1. **Grant 'read' and 'execute' permissions (directory traversal) to the certificate directories:**

```bash
sudo setfacl -R -m u:parsa:rx /etc/letsencrypt/live
sudo setfacl -R -m u:parsa:rx /etc/letsencrypt/archive
```

2. **Ensure these permissions persist after certificate renewal:**  
Create a Certbot deploy hook that reapplies access controls after each renewal.

```bash
sudo tee /etc/letsencrypt/renewal-hooks/deploy/01-permit-parsa.sh > /dev/null <<'EOF'
#!/bin/bash
setfacl -R -m u:parsa:rx /etc/letsencrypt/live
setfacl -R -m u:parsa:rx /etc/letsencrypt/archive
EOF
```

3. **Make the hook script executable:**

```bash
sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/01-permit-parsa.sh
```


## Managing the Service

```bash
# Check status
sudo systemctl status parsa-go

# View logs (live)
sudo journalctl -u parsa-go -f

# View recent logs
sudo journalctl -u parsa-go --since "1 hour ago"

# Restart after updates
sudo systemctl restart parsa-go

# Stop service
sudo systemctl stop parsa-go
```

## Updating the Application

```bash
# On dev machine: build and copy new binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/parsa ./cmd/api/
scp bin/parsa user@server:/home/parsa/parsa-go/bin/

# On server: restart
sudo systemctl restart parsa-go
```

## TLS Certificates (Let's Encrypt)

If using TLS, install certbot and get certificates:

```bash
sudo apt install certbot
sudo certbot certonly --standalone -d yourdomain.com

# Update .env with cert paths:
# TLS_CERT_PATH=/etc/letsencrypt/live/yourdomain.com/fullchain.pem
# TLS_KEY_PATH=/etc/letsencrypt/live/yourdomain.com/privkey.pem
```

Grant parsa user read access to certs:

```bash
sudo usermod -aG ssl-cert parsa
sudo chgrp -R ssl-cert /etc/letsencrypt/live /etc/letsencrypt/archive
sudo chmod -R g+rx /etc/letsencrypt/live /etc/letsencrypt/archive
```

## 🛠️ Network Troubleshooting (Magalu Cloud / MTU Issues)

If the application is running but not accessible via browsers (timeouts/hanging), while `curl` requests work fine, you are likely facing an **MTU/MSS Mismatch**. This is common in cloud providers using overlay networks (like VXLAN) where the packet headers exceed the standard 1500 MTU.

To fix this, we enforce **MSS Clamping** via `iptables`.

### 1. Install Persistence Tool

Ensure `iptables` rules survive reboots:

```bash
sudo apt-get update
sudo apt-get install iptables-persistent -y
```

### 2. Apply the Clamping Rule

We force the TCP MSS to 1200 bytes to safely fit inside the tunnel overhead. (Note: Replace `ens3` with your actual network interface if different).

```bash
# Clean existing mangle rules
sudo iptables -t mangle -F

# Apply the MSS clamp
sudo iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -o ens3 -j TCPMSS --set-mss 1200
```

### 3. Save Changes

Persist the rules to `/etc/iptables/rules.v4`:

```bash
sudo netfilter-persistent save
```

### Verification

Check if the rule is active:

```bash
sudo iptables -t mangle -L -v
```

## Troubleshooting

```bash
# Check if service is running
sudo systemctl is-active parsa-go

# Check for errors
sudo journalctl -u parsa-go -e --no-pager

# Verify .env file is readable
sudo -u parsa cat /home/parsa/parsa-go/.env

# Test binary manually
sudo -u parsa /home/parsa/parsa-go/bin/parsa
```
