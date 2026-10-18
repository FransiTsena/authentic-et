# authentic-et Production Deployment Guide

authentic-et is designed for ultimate deployment flexibility. You can deploy:
1. **Independent Backend** — as a lightweight Go binary, microservice container, or on PaaS (Railway, Render, Fly.io).
2. **Independent Frontend** — as a static Single Page Application (SPA) on Vercel, Netlify, Cloudflare Pages, S3/CloudFront, or Nginx.
3. **Unified Single-Stack** — via Docker Compose or Kubernetes with reverse-proxy routing on standard port 5151.

---

## 1. Deployment Topology Matrix

| Target Environment | Deployment Mode | Recommended RAM | Startup Time | Artifact |
|---|---|---|---|---|
| **Independent Frontend** | Static Edge CDN (Vercel / Cloudflare Pages / Netlify) | 0 MB (Serverless) | Instant | Static `dist/` |
| **Independent Frontend Container** | Standalone Nginx Container | 16 MB | < 50 ms | `frontend/Dockerfile` |
| **Independent Backend Binary** | VPS / Bare Metal / Systemd | ~15 MB | < 50 ms | Static Go Binary |
| **Independent Backend Container** | Docker / Fly.io / Railway / Render | 64 MB | < 100 ms | `backend/Dockerfile` |
| **Unified Docker Stack** | Docker Compose (`docker-compose.yml`) | 128 MB | < 2 sec | Full Compose |
| **Kubernetes Cluster** | K8s App Pod with PVC | 128 MB | < 100 ms | Native Manifests / Helm |

---

## 2. Independent Backend Deployment

The backend has zero dependencies outside of Go and can operate with embedded SQLite (no CGO required), PostgreSQL, or ephemeral in-memory storage.

### Option A: Standalone Docker Container

Run directly with Docker:
```bash
docker run -d \
  --name authentic-backend \
  -p 5151:5151 \
  -e DATABASE_DRIVER=sqlite \
  -e SQLITE_PATH=/data/authentic.db \
  -e JWT_SECRET=your-32-char-production-secret-key-here \
  -e ALLOW_ORIGINS="https://your-frontend-domain.com" \
  -e PAYMENT_PROVIDERS_ENABLED="cbe,telebirr,boa,dashen,awash,mpesa" \
  -v authentic_data:/data \
  --restart unless-stopped \
  ghcr.io/fransitsena/authentic-backend:latest
```

### Option B: Build & Run from Source (Linux / macOS / Windows)

```bash
cd backend
go build -o authentic-server ./cmd/server
./authentic-server
```

### Option C: Cloud PaaS (Railway, Render, Fly.io)

1. Set the build context to `backend/` and use `backend/Dockerfile`.
2. Configure environment variables in the PaaS dashboard:
   - `PORT=5151`
   - `DATABASE_DRIVER=sqlite` (or `postgres` if attaching a PostgreSQL database)
   - `SQLITE_PATH=/data/authentic.db` (attach a persistent volume mount at `/data`)
   - `JWT_SECRET=your-secret-key-minimum-32-bytes`
   - `ALLOW_ORIGINS=https://your-frontend-domain.com`
3. Expose port `5151`.

### Option D: Linux Systemd Service (VPS / Bare Metal)

```ini
# /etc/systemd/system/authentic-et.service
[Unit]
Description=authentic-et Payment Verification Service
After=network.target

[Service]
Type=simple
User=authentic
Group=authentic
WorkingDirectory=/opt/authentic-et
ExecStart=/opt/authentic-et/authentic-server
Restart=always
RestartSec=5s

# Environment configuration
Environment=PORT=5151
Environment=DATABASE_DRIVER=sqlite
Environment=SQLITE_PATH=/var/lib/authentic-et/authentic.db
Environment=JWT_SECRET=super-secret-jwt-key-min-32-chars-long
Environment=ALLOW_ORIGINS="https://app.yourdomain.com"

# Security sandboxing
LimitNOFILE=65535
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now authentic-et
```

---

## 3. Independent Frontend Deployment

The frontend is a fast, responsive Single Page Application (React + Vite + Tailwind CSS) with HTML5 history routing.

### Option A: Static Hosting (Vercel, Netlify, Cloudflare Pages)

1. Set Root Directory to `frontend`.
2. Build Settings:
   - **Build Command**: `npm run build`
   - **Output Directory**: `dist`
3. Environment Variables:
   - `VITE_API_BASE_URL`: The public HTTPS URL of your deployed backend, e.g. `https://api.yourdomain.com/api/v1`
4. SPA Routing Rewrites:
   - **Vercel** (`vercel.json`):
     ```json
     {
       "rewrites": [{ "source": "/(.*)", "destination": "/index.html" }]
     }
     ```
   - **Netlify** (`_redirects` in `public/`):
     ```text
     /*    /index.html   200
     ```
   - **Cloudflare Pages**: Automatically falls back to `/index.html` for single-page applications.

### Option B: Standalone Frontend Docker Container

Build and run the frontend with its built-in Nginx reverse proxy:
```bash
cd frontend
docker build -t authentic-frontend .
docker run -d -p 80:80 authentic-frontend
```

---

## 4. Unified Full-Stack Deployment (Recommended)

Run both frontend and backend together with a single command on standard port `5151`:

```bash
docker compose up -d
```

- **Web UI & API Proxy**: `http://localhost:5151`
- **System Health**: `http://localhost:5151/health`
- **Interactive Documentation**: `http://localhost:5151/docs`
- **Initial Setup**: Open `http://localhost:5151` in your browser. The setup wizard prompts you to initialize your root administrator account.

---

## 5. Kubernetes & BeeCloud App Pods

Deploy authentic-et as a stateful, lightweight microservice with persistent volume storage:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: authentic-et
  labels:
    app: authentic-et
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: authentic-et
  template:
    metadata:
      labels:
        app: authentic-et
    spec:
      containers:
        - name: authentic-et
          image: ghcr.io/fransitsena/authentic-et:latest
          ports:
            - containerPort: 5151
              name: http
          env:
            - name: DATABASE_DRIVER
              value: "sqlite"
            - name: SQLITE_PATH
              value: "/data/authentic.db"
          volumeMounts:
            - name: data
              mountPath: /data
          readinessProbe:
            httpGet:
              path: /ready
              port: 5151
            initialDelaySeconds: 2
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /health
              port: 5151
            initialDelaySeconds: 5
            periodSeconds: 10
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: authentic-data-pvc
```
