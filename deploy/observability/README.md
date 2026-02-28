# Observability Stack Setup

All services run as systemd units on the same VPS as the Go API.

## 1. Prometheus

```bash
# Download
wget https://github.com/prometheus/prometheus/releases/download/v2.53.0/prometheus-2.53.0.linux-amd64.tar.gz
tar xzf prometheus-2.53.0.linux-amd64.tar.gz
sudo mv prometheus-2.53.0.linux-amd64/prometheus /usr/local/bin/
sudo mv prometheus-2.53.0.linux-amd64/promtool /usr/local/bin/

# Config
sudo mkdir -p /etc/prometheus
sudo cp prometheus.yml /etc/prometheus/prometheus.yml
sudo mkdir -p /var/lib/prometheus

# Systemd unit
sudo tee /etc/systemd/system/prometheus.service > /dev/null <<'EOF'
[Unit]
Description=Prometheus
After=network.target

[Service]
Type=simple
User=prometheus
ExecStart=/usr/local/bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/var/lib/prometheus \
  --storage.tsdb.retention.time=30d
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo useradd --no-create-home --shell /bin/false prometheus || true
sudo chown -R prometheus:prometheus /var/lib/prometheus /etc/prometheus
sudo systemctl daemon-reload
sudo systemctl enable --now prometheus
```

## 2. Grafana Tempo

```bash
# Download
wget https://github.com/grafana/tempo/releases/download/v2.6.1/tempo_2.6.1_linux_amd64.deb
sudo dpkg -i tempo_2.6.1_linux_amd64.deb

# Config
sudo cp tempo.yml /etc/tempo/tempo.yml
sudo mkdir -p /var/lib/tempo/{traces,wal,metrics}
sudo chown -R tempo:tempo /var/lib/tempo

sudo systemctl daemon-reload
sudo systemctl enable --now tempo
```

## 3. Grafana Loki

```bash
# Download
wget https://github.com/grafana/loki/releases/download/v3.2.1/loki_3.2.1_amd64.deb
sudo dpkg -i loki_3.2.1_amd64.deb

# Config
sudo cp loki.yml /etc/loki/loki.yml
sudo mkdir -p /var/lib/loki/{chunks,rules}
sudo chown -R loki:loki /var/lib/loki

sudo systemctl daemon-reload
sudo systemctl enable --now loki
```

## 4. Promtail

```bash
# Download
wget https://github.com/grafana/loki/releases/download/v3.2.1/promtail_3.2.1_amd64.deb
sudo dpkg -i promtail_3.2.1_amd64.deb

# Config
sudo cp promtail.yml /etc/promtail/promtail.yml
sudo mkdir -p /var/lib/promtail
sudo chown -R promtail:promtail /var/lib/promtail

# Promtail needs access to systemd journal
sudo usermod -aG systemd-journal promtail

sudo systemctl daemon-reload
sudo systemctl enable --now promtail
```

## 5. Grafana

```bash
# Install via apt
sudo apt-get install -y apt-transport-https software-properties-common
sudo mkdir -p /etc/apt/keyrings/
wget -q -O - https://apt.grafana.com/gpg.key | gpg --dearmor | sudo tee /etc/apt/keyrings/grafana.gpg > /dev/null
echo "deb [signed-by=/etc/apt/keyrings/grafana.gpg] https://apt.grafana.com stable main" | sudo tee /etc/apt/sources.list.d/grafana.list
sudo apt-get update
sudo apt-get install -y grafana

# Provision datasources
sudo cp -r grafana/provisioning/* /etc/grafana/provisioning/

sudo systemctl daemon-reload
sudo systemctl enable --now grafana-server
```

Grafana is available at `http://your-vps:3000` (default login: admin/admin).

## 6. Enable telemetry in the Go API

Set in your `.env`:

```
OTEL_ENABLED=true
OTEL_SERVICE_NAME=parsa-api
OTEL_ENVIRONMENT=production
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_METRICS_PORT=9464
```

Restart the API service:

```bash
sudo systemctl restart parsa-api
```

## Ports Summary

| Service    | Port  | Purpose                    |
|------------|-------|----------------------------|
| Parsa API  | 9464  | /metrics (Prometheus pull) |
| Prometheus | 9090  | Metrics storage + query    |
| Tempo      | 3200  | Trace query HTTP           |
| Tempo      | 4317  | OTLP gRPC receiver         |
| Loki       | 3100  | Log storage + query        |
| Promtail   | 9080  | Promtail status            |
| Grafana    | 3000  | Dashboards UI              |

## Verify

```bash
# Check all services are running
sudo systemctl status prometheus tempo loki promtail grafana-server

# Check metrics endpoint
curl http://localhost:9464/metrics

# Check Prometheus targets
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[].health'
```
