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
CREATE USER parsa WITH PASSWORD 'your_secure_password';
CREATE DATABASE parsa OWNER parsa;
GRANT ALL PRIVILEGES ON DATABASE parsa TO parsa;
\q
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
