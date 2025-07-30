# TRU Activity - Production Deployment Guide

## 📖 Overview

คู่มือการ deploy TRU Activity system ไปยัง production environments รองรับ 3 วิธี:
1. **Google Cloud Platform (แนะนำ)** - Cloud Run + Cloud SQL + Firebase
2. **Docker + VPS** - สำหรับ self-hosted server
3. **Kubernetes** - สำหรับ enterprise deployment

---

## 🚀 วิธีที่ 1: Google Cloud Platform (แนะนำ)

### Prerequisites
- Google Cloud account พร้อม billing enabled
- Domain name สำหรับ custom domain (optional)
- Google Cloud CLI, Firebase CLI, Terraform installed

### ขั้นตอนที่ 1: เตรียม Environment Variables

สร้างไฟล์ `.env.deploy`:

```bash
# Project Configuration
PROJECT_ID=your-gcp-project-id
REGION=asia-southeast1
FRONTEND_PROJECT_ID=your-firebase-project-id

# Database Configuration  
DB_PASSWORD=your-secure-database-password-32-chars-minimum
DB_INSTANCE_NAME=tru-activity-db-prod

# Authentication
JWT_SECRET=your-jwt-secret-key-64-chars-minimum
JWT_EXPIRE_HOURS=24

# Email Configuration
EMAIL_FROM=noreply@yourdomain.com
SENDGRID_API_KEY=your-sendgrid-api-key

# QR System
QR_SECRET=your-qr-secret-key-32-chars-minimum

# Domain (optional)
CUSTOM_DOMAIN=yourdomain.com
```

### ขั้นตอนที่ 2: Automated Deployment

```bash
# 1. Clone repository
git clone https://github.com/your-org/tru-activity.git
cd tru-activity

# 2. Make deployment script executable
chmod +x scripts/deploy.sh

# 3. Run full deployment (แนะนำ)
./scripts/deploy.sh deploy

# หรือ deploy ทีละส่วน
./scripts/deploy.sh infrastructure  # Infrastructure เท่านั้น
./scripts/deploy.sh backend        # Backend เท่านั้น
./scripts/deploy.sh frontend       # Frontend เท่านั้น
```

### ขั้นตอนที่ 3: Manual Configuration (ถ้าต้องการ custom setup)

#### 3.1 Setup Infrastructure

```bash
cd infrastructure/terraform

# สร้าง terraform.tfvars
cat > terraform.tfvars << EOF
project_id = "your-project-id"
region = "asia-southeast1"
db_password = "your-secure-password"
jwt_secret = "your-jwt-secret"
email_from = "noreply@yourdomain.com"
sendgrid_api_key = "your-sendgrid-key"
qr_secret = "your-qr-secret"
EOF

# Deploy infrastructure
terraform init
terraform plan
terraform apply
```

#### 3.2 Deploy Backend

```bash
# Build และ deploy ด้วย Cloud Build
gcloud builds submit \
  --config=cloudbuild.yaml \
  --substitutions=_REGION=asia-southeast1 \
  --project=your-project-id
```

#### 3.3 Deploy Frontend

```bash
cd frontend

# Update environment variables
export PUBLIC_API_URL=https://your-backend-url
export PUBLIC_GRAPHQL_URL=https://your-backend-url/query
export PUBLIC_ENV=production

# Build และ deploy
npm ci
npm run build
firebase deploy --only hosting
```

### ขั้นตอนที่ 4: Custom Domain Setup (Optional)

```bash
# Map custom domain to Cloud Run
gcloud run domain-mappings create \
  --service=tru-activity-backend \
  --domain=api.yourdomain.com \
  --region=asia-southeast1

# Map custom domain to Firebase
firebase hosting:channel:deploy production \
  --project=your-project-id \
  --expires=never
```

---

## 🐳 วิธีที่ 2: Docker + VPS Deployment

### Prerequisites
- Ubuntu 20.04+ server
- Docker และ Docker Compose installed
- Nginx installed
- Domain name with DNS pointing to server

### ขั้นตอนที่ 1: Server Setup

```bash
# 1. Update system
sudo apt update && sudo apt upgrade -y

# 2. Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
sudo usermod -aG docker $USER

# 3. Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 4. Install Nginx
sudo apt install nginx certbot python3-certbot-nginx -y
```

### ขั้นตอนที่ 2: Application Deployment

```bash
# 1. Clone repository
git clone https://github.com/your-org/tru-activity.git
cd tru-activity

# 2. สร้าง production environment file
cat > .env.prod << EOF
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=truactivity
DB_PASSWORD=your-secure-password
DB_NAME=tru_activity_prod
DB_SSLMODE=require

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password

# JWT
JWT_SECRET=your-jwt-secret-64-chars
JWT_EXPIRE_HOURS=24

# Email
EMAIL_FROM=noreply@yourdomain.com
SENDGRID_API_KEY=your-sendgrid-key

# QR System
QR_SECRET=your-qr-secret-32-chars

# Server
PORT=8080
ENV=production
CORS_ORIGINS=https://yourdomain.com
EOF

# 3. สร้าง production docker-compose file
cat > docker-compose.prod.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: tru-activity-db-prod
    environment:
      POSTGRES_USER: truactivity
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: tru_activity_prod
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backend/migrations:/docker-entrypoint-initdb.d
    ports:
      - "127.0.0.1:5432:5432"
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U truactivity"]
      interval: 30s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: tru-activity-redis-prod
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "127.0.0.1:6379:6379"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "auth", "${REDIS_PASSWORD}", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: tru-activity-backend-prod
    env_file:
      - .env.prod
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 5

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - PUBLIC_API_URL=https://yourdomain.com/api
        - PUBLIC_GRAPHQL_URL=https://yourdomain.com/api/query
        - PUBLIC_ENV=production
    container_name: tru-activity-frontend-prod
    ports:
      - "127.0.0.1:3000:3000"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
EOF
```

### ขั้นตอนที่ 3: Production Frontend Dockerfile

```bash
# สร้าง production Dockerfile สำหรับ frontend
cat > frontend/Dockerfile << 'EOF'
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .

# Build arguments
ARG PUBLIC_API_URL
ARG PUBLIC_GRAPHQL_URL
ARG PUBLIC_ENV

ENV PUBLIC_API_URL=$PUBLIC_API_URL
ENV PUBLIC_GRAPHQL_URL=$PUBLIC_GRAPHQL_URL  
ENV PUBLIC_ENV=$PUBLIC_ENV

RUN npm run build

# Production stage
FROM node:20-alpine AS production

WORKDIR /app
COPY --from=builder /app/build build/
COPY --from=builder /app/node_modules node_modules/
COPY package.json .

EXPOSE 3000
CMD ["node", "build"]
EOF
```

### ขั้นตอนที่ 4: Nginx Configuration

```bash
# สร้าง Nginx configuration
sudo tee /etc/nginx/sites-available/tru-activity << 'EOF'
upstream backend {
    server 127.0.0.1:8080;
}

upstream frontend {
    server 127.0.0.1:3000;
}

server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;

    # SSL Configuration (จะถูกเพิ่มโดย Certbot)
    
    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # API Routes
    location /api/ {
        proxy_pass http://backend/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    # Frontend Routes
    location / {
        proxy_pass http://frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Static Files Caching
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        add_header X-Content-Type-Options nosniff;
    }

    # Security
    location ~ /\. {
        deny all;
    }
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/tru-activity /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### ขั้นตอนที่ 5: SSL Certificate

```bash
# Get SSL certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Test auto-renewal
sudo certbot renew --dry-run
```

### ขั้นตอนที่ 6: Deploy Application

```bash
# Deploy application
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

---

## ⚙️ วิธีที่ 3: Kubernetes Deployment

### Prerequisites
- Kubernetes cluster (GKE, EKS, AKS, หรือ self-managed)
- kubectl configured
- Helm 3.x installed

### ขั้นตอนที่ 1: Create Namespace และ Secrets

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tru-activity
---
apiVersion: v1
kind: Secret
metadata:
  name: tru-activity-secrets
  namespace: tru-activity
type: Opaque
stringData:
  db-password: "your-secure-password"
  jwt-secret: "your-jwt-secret"
  redis-password: "your-redis-password"
  sendgrid-api-key: "your-sendgrid-key"
  qr-secret: "your-qr-secret"
```

### ขั้นตอนที่ 2: Database และ Redis Deployment

```yaml
# k8s/postgres.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: tru-activity
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_USER
          value: "truactivity"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tru-activity-secrets
              key: db-password
        - name: POSTGRES_DB
          value: "tru_activity_prod"
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 20Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: tru-activity
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

### ขั้นตอนที่ 3: Backend Deployment

```yaml
# k8s/backend.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: tru-activity
spec:
  replicas: 3
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: gcr.io/your-project/tru-activity-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "postgres"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tru-activity-secrets
              key: db-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: tru-activity-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: backend
  namespace: tru-activity
spec:
  selector:
    app: backend
  ports:
  - port: 80
    targetPort: 8080
```

### ขั้นตอนที่ 4: Deploy to Kubernetes

```bash
# Apply configurations
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/backend.yaml
kubectl apply -f k8s/frontend.yaml
kubectl apply -f k8s/ingress.yaml

# Check deployment status
kubectl get pods -n tru-activity
kubectl get services -n tru-activity
```

---

## 🔧 Production Maintenance

### Backup Strategy

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/tru-activity"

mkdir -p $BACKUP_DIR

# Database backup
if command -v docker &> /dev/null; then
    # Docker deployment
    docker exec tru-activity-db-prod pg_dump -U truactivity tru_activity_prod > $BACKUP_DIR/db_backup_$DATE.sql
    docker exec tru-activity-redis-prod redis-cli BGSAVE
    docker cp tru-activity-redis-prod:/data/dump.rdb $BACKUP_DIR/redis_backup_$DATE.rdb
elif command -v kubectl &> /dev/null; then
    # Kubernetes deployment
    kubectl exec -n tru-activity postgres-0 -- pg_dump -U truactivity tru_activity_prod > $BACKUP_DIR/db_backup_$DATE.sql
fi

# Compress backups
tar -czf $BACKUP_DIR/backup_$DATE.tar.gz $BACKUP_DIR/*_$DATE.*

# Remove old backups (keep 30 days)
find $BACKUP_DIR -name "backup_*.tar.gz" -mtime +30 -delete

echo "Backup completed: backup_$DATE.tar.gz"
```

### Monitoring และ Health Checks

```bash
#!/bin/bash
# monitor.sh

echo "=== TRU Activity Health Check ==="

# Check application health
if command -v curl &> /dev/null; then
    BACKEND_URL="https://yourdomain.com/api"
    
    # Backend health
    if curl -f "$BACKEND_URL/health" &> /dev/null; then
        echo "✅ Backend: Healthy"
    else
        echo "❌ Backend: Unhealthy"
    fi
    
    # Database connectivity
    if curl -f "$BACKEND_URL/ready" &> /dev/null; then
        echo "✅ Database: Connected"
    else
        echo "❌ Database: Connection failed"
    fi
fi

# System resources
echo -e "\n📊 System Resources:"
echo "Memory: $(free -h | awk '/^Mem:/ {print $3 "/" $2}')"
echo "Disk: $(df -h / | awk 'NR==2 {print $3 "/" $2 " (" $5 " used)"}')"
echo "Load: $(uptime | awk -F'load average:' '{print $2}')"
```

### Auto-update Script

```bash
#!/bin/bash
# update.sh

set -e

echo "🔄 Starting TRU Activity update..."

# Backup before update
./backup.sh

# Git pull latest changes
git fetch origin
git checkout main
git pull origin main

# Update dependencies
cd backend && go mod tidy && cd ..
cd frontend && npm ci && cd ..

if command -v docker-compose &> /dev/null; then
    # Docker deployment
    docker-compose -f docker-compose.prod.yml pull
    docker-compose -f docker-compose.prod.yml up -d --build
elif command -v kubectl &> /dev/null; then
    # Kubernetes deployment
    kubectl set image deployment/backend backend=gcr.io/your-project/tru-activity-backend:latest -n tru-activity
    kubectl set image deployment/frontend frontend=gcr.io/your-project/tru-activity-frontend:latest -n tru-activity
fi

# Health check after update
sleep 30
./monitor.sh

echo "✅ Update completed successfully!"
```

---

## 🔒 Security Checklist

### Pre-deployment Security
- [ ] All secrets use strong, randomly generated values
- [ ] Database passwords ≥ 32 characters
- [ ] JWT secrets ≥ 64 characters
- [ ] Environment variables properly configured
- [ ] No hardcoded secrets in code
- [ ] SSL/TLS certificates configured
- [ ] Security headers implemented
- [ ] CORS properly configured

### Post-deployment Security
- [ ] Firewall rules configured (only allow 80, 443, SSH)
- [ ] SSH key-based authentication only
- [ ] Regular security updates scheduled
- [ ] Backup verification working
- [ ] Monitoring and alerting active
- [ ] Log aggregation configured
- [ ] Access controls verified
- [ ] Rate limiting implemented

### Ongoing Security Maintenance
- [ ] Weekly security updates
- [ ] Monthly backup testing
- [ ] Quarterly security audit
- [ ] Annual penetration testing
- [ ] SSL certificate auto-renewal
- [ ] Dependency vulnerability scanning

---

## 📈 Performance Optimization

### Database Optimization
```sql
-- Create performance indexes
CREATE INDEX CONCURRENTLY idx_activities_search ON activities USING gin(to_tsvector('english', title || ' ' || description));
CREATE INDEX CONCURRENTLY idx_participations_user_activity ON participations(user_id, activity_id);
CREATE INDEX CONCURRENTLY idx_users_search ON users(first_name, last_name, student_id);
CREATE INDEX CONCURRENTLY idx_activities_faculty_date ON activities(faculty_id, start_date);
```

### Application Optimization
```bash
# Backend optimizations
export GOGC=100
export GOMAXPROCS=4

# Frontend optimizations - ใน frontend/vite.config.ts
build: {
  rollupOptions: {
    output: {
      manualChunks: {
        vendor: ['svelte', '@apollo/client'],
        ui: ['lucide-svelte', 'bits-ui']
      }
    }
  }
}
```

---

## 🆘 Troubleshooting

### Common Issues และ Solutions

**Database Connection Issues:**
```bash
# Check database status
docker logs tru-activity-db-prod
kubectl logs postgres-0 -n tru-activity

# Test connectivity
docker exec -it tru-activity-backend-prod nc -zv postgres 5432
kubectl exec -it deployment/backend -n tru-activity -- nc -zv postgres 5432
```

**High Memory Usage:**
```bash
# Check container resources
docker stats
kubectl top pods -n tru-activity

# Restart services if needed
docker-compose -f docker-compose.prod.yml restart
kubectl rollout restart deployment/backend -n tru-activity
```

**SSL Certificate Issues:**
```bash
# Check certificate status
sudo certbot certificates
openssl x509 -in /etc/letsencrypt/live/yourdomain.com/fullchain.pem -text -noout

# Force renewal
sudo certbot renew --force-renewal
```

**Application Performance Issues:**
```bash
# Check application logs
docker-compose -f docker-compose.prod.yml logs -f backend
kubectl logs -f deployment/backend -n tru-activity

# Monitor resource usage
htop
kubectl top nodes
```

---

## 📞 Support

สำหรับการสนับสนุนเพิ่มเติม:
- 📧 Email: support@truactivity.com
- 📱 GitHub Issues: https://github.com/your-org/tru-activity/issues
- 📚 Documentation: https://docs.truactivity.com
- 💬 Discord: https://discord.gg/truactivity

---

*เอกสารนี้อัปเดตล่าสุด: $(date)*