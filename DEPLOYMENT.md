# TRU Activity - Google Cloud Platform Deployment Guide

This guide provides comprehensive instructions for deploying the TRU Activity system to Google Cloud Platform using Cloud Run, Cloud SQL, and other GCP services.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Cloud Run     │◄──►│   Cloud Run      │◄───┤   Cloud SQL     │
│   Frontend      │    │   Backend        │    │   PostgreSQL    │
│  (SvelteKit)    │    │  (Go Fiber)      │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              │
                       ┌──────────────────┐
                       │  Cloud Memorystore│
                       │     Redis         │
                       └──────────────────┘
```

## Prerequisites

### Required Tools
- [Google Cloud CLI](https://cloud.google.com/sdk/docs/install)
- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Docker](https://docs.docker.com/get-docker/)
- [Node.js](https://nodejs.org/) >= 24.4.1
- [Go](https://golang.org/doc/install) >= 1.24.5

### Google Cloud Setup
1. Create a new GCP project or select an existing one
2. Enable billing for the project
3. Install and authenticate with Google Cloud CLI:
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

## Quick Deployment

### Automated Deployment
Use the provided deployment script for a complete setup:

```bash
# Make script executable
chmod +x scripts/deploy.sh

# Run full deployment
./scripts/deploy.sh deploy
```

The script will guide you through:
- Authentication verification
- Infrastructure provisioning
- Backend deployment
- Frontend deployment
- Health checks
- Monitoring setup

### Manual Deployment

#### 1. Infrastructure Setup

```bash
# Navigate to Terraform directory
cd infrastructure/terraform

# Create terraform.tfvars file
cat > terraform.tfvars << EOF
project_id = "your-project-id"
region = "asia-southeast1"

# Database configuration
db_password = "your-secure-database-password"

# JWT configuration
jwt_secret = "your-jwt-secret-key"

# Email configuration
email_from = "noreply@yourdomain.com"
sendgrid_api_key = "your-sendgrid-api-key"

# QR code configuration
qr_secret = "your-qr-secret-key"
EOF

# Initialize and apply Terraform
terraform init
terraform plan
terraform apply
```

### 4. Production Docker Compose

สร้างไฟล์ `docker-compose.prod.yml`:

```yaml
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
      target: production
    container_name: tru-activity-backend-prod
    env_file:
      - backend/.env.prod
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
      dockerfile: Dockerfile.prod
    container_name: tru-activity-frontend-prod
    environment:
      - VITE_API_URL=https://yourdomain.com/api
      - VITE_GRAPHQL_URL=https://yourdomain.com/api/query
    ports:
      - "127.0.0.1:3000:3000"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### 5. Frontend Production Dockerfile

สร้างไฟล์ `frontend/Dockerfile.prod`:

```dockerfile
# Build stage
FROM node:24.4.1-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

# Production stage
FROM node:24.4.1-alpine AS production

WORKDIR /app
COPY --from=builder /app/build build/
COPY --from=builder /app/node_modules node_modules/
COPY package.json .

EXPOSE 3000

CMD ["node", "build"]
```

### 6. Nginx Configuration

สร้างไฟล์ `/etc/nginx/sites-available/tru-activity`:

```nginx
upstream backend {
    server 127.0.0.1:8080;
}

upstream frontend {
    server 127.0.0.1:3000;
}

server {
    listen 80;
    server_name yourdomain.com;

    # Redirect all HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # API routes
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
    }

    # Frontend routes
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

    # Static files caching
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### 7. SSL Certificate

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/tru-activity /etc/nginx/sites-enabled/

# Test Nginx configuration
sudo nginx -t

# Get SSL certificate
sudo certbot --nginx -d yourdomain.com

# Reload Nginx
sudo systemctl reload nginx
```

### 8. Deploy Application

```bash
# Build และ start services
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

## 🔧 Production Maintenance

### Backup Strategy

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/tru-activity"

# Create backup directory
mkdir -p $BACKUP_DIR

# Database backup
docker exec tru-activity-db-prod pg_dump -U truactivity tru_activity_prod > $BACKUP_DIR/db_backup_$DATE.sql

# Redis backup
docker exec tru-activity-redis-prod redis-cli --rdb $BACKUP_DIR/redis_backup_$DATE.rdb

# Compress backups
tar -czf $BACKUP_DIR/backup_$DATE.tar.gz $BACKUP_DIR/*_$DATE.*

# Remove old backups (keep 30 days)
find $BACKUP_DIR -name "backup_*.tar.gz" -mtime +30 -delete

echo "Backup completed: backup_$DATE.tar.gz"
```

### Monitoring Scripts

```bash
#!/bin/bash
# monitor.sh

echo "=== TRU Activity Health Check ==="

# Check containers
echo "Container Status:"
docker-compose -f docker-compose.prod.yml ps

# Check service health
echo -e "\nService Health:"
curl -s http://localhost:8080/health | jq '.' || echo "Backend health check failed"

# Check database connection
echo -e "\nDatabase Status:"
docker exec tru-activity-db-prod pg_isready -U truactivity

# Check Redis
echo -e "\nRedis Status:"
docker exec tru-activity-redis-prod redis-cli ping

# Check disk usage
echo -e "\nDisk Usage:"
df -h

# Check memory usage
echo -e "\nMemory Usage:"
free -h
```

### Log Rotation

เพิ่มใน `/etc/logrotate.d/tru-activity`:

```
/var/log/nginx/access.log
/var/log/nginx/error.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 www-data www-data
    postrotate
        systemctl reload nginx
    endscript
}
```

## 🔒 Security Checklist

- [ ] Database passwords เป็น strong passwords
- [ ] JWT secret เป็น random string อย่างน้อย 32 characters
- [ ] SSL certificates ติดตั้งและ auto-renewal
- [ ] Firewall configured (UFW แนะนำ)
- [ ] Docker containers รันด้วย non-root user
- [ ] Regular security updates
- [ ] Backup strategy implemented
- [ ] Monitoring และ alerting setup
- [ ] Log aggregation และ analysis

## 📊 Performance Optimization

### Database Optimization
```sql
-- Create indexes for better performance
CREATE INDEX CONCURRENTLY idx_activities_search ON activities USING gin(to_tsvector('english', title || ' ' || description));
CREATE INDEX CONCURRENTLY idx_participations_user_activity ON participations(user_id, activity_id);
CREATE INDEX CONCURRENTLY idx_users_search ON users(first_name, last_name, student_id);
```

### Redis Configuration
```conf
# /etc/redis/redis.conf optimizations
maxmemory 512mb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
```

### Nginx Optimizations
```nginx
# Add to http block in /etc/nginx/nginx.conf
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;

# Connection optimization
keepalive_timeout 65;
keepalive_requests 100;
```

## 🆘 Troubleshooting

### Common Issues

**1. Database Connection Failed**
```bash
# Check database logs
docker logs tru-activity-db-prod

# Check network connectivity
docker exec tru-activity-backend-prod ping postgres
```

**2. High Memory Usage**
```bash
# Check container resource usage
docker stats

# Restart services if needed
docker-compose -f docker-compose.prod.yml restart
```

**3. SSL Certificate Issues**
```bash
# Check certificate status
sudo certbot certificates

# Renew certificates manually
sudo certbot renew
```

---

สำหรับการสนับสนุนเพิ่มเติม โปรดติดต่อทีมพัฒนา หรือสร้าง issue ใน GitHub repository