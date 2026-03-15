# Netdata Deployment

## PostgreSQL collector (postgres.conf)

The `go.d/postgres.conf` file is a **template** that uses `${DB_USER}` and `${DB_PASSWORD}` placeholders. Render it at deploy time with `envsubst` so actual credentials are substituted:

```bash
# Ensure DB_USER and DB_PASSWORD are set (e.g. from .env or export)
export DB_USER=parsa_admin
export DB_PASSWORD=your_secure_password

envsubst '${DB_USER} ${DB_PASSWORD}' < go.d/postgres.conf | sudo tee /etc/netdata/go.d/postgres.conf
```

Or, if loading from your parsa .env:

```bash
set -a && source /home/parsa/parsa-go/.env && set +a
envsubst '${DB_USER} ${DB_PASSWORD}' < go.d/postgres.conf | sudo tee /etc/netdata/go.d/postgres.conf
```

Restart Netdata after updating the config:

```bash
sudo systemctl restart netdata
```
