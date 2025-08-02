# 🚀 Quick Start Guide

เนื่องจากมีปัญหา Docker network connectivity ให้ใช้วิธีเหล่านี้:

## วิธีที่ 1: ใช้ External Database Services (แนะนำ)

### 1. ใช้ Online Database Services
```bash
# แก้ไข backend/.env
DB_HOST=your-postgres-host
DB_PORT=5432
DB_USER=your-username
DB_PASSWORD=your-password
DB_NAME=tru_activity_dev

REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
```

**Free Database Services:**
- **PostgreSQL**: Supabase, Neon, ElephantSQL
- **Redis**: Redis Cloud, Upstash

### 2. เริ่ม Development
```bash
./scripts/dev-local.sh start
```

## วิธีที่ 2: Local Database Setup

### PostgreSQL
```bash
# Ubuntu/Debian
sudo apt install postgresql postgresql-contrib
sudo service postgresql start

# สร้าง database
sudo -u postgres createdb tru_activity_dev

# แก้ไข backend/.env
DB_PASSWORD=   # ไม่ต้องใส่ password สำหรับ local
```

### Redis
```bash
# Ubuntu/Debian  
sudo apt install redis-server
sudo service redis-server start
```

## วิธีที่ 3: ใช้ SQLite (ง่ายที่สุด)

แก้ไข backend code ให้ใช้ SQLite แทน PostgreSQL:

```go
// internal/database/connection.go
// เปลี่ยนจาก PostgreSQL เป็น SQLite
dsn := "tru_activity.db"
db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
```

## การทดสอบ

```bash
# ทดสอบ backend
curl http://localhost:8080/health

# ทดสอบ frontend  
curl http://localhost:5173
```

## URLs ที่ใช้งาน

- **Frontend**: http://localhost:5173
- **Backend API**: http://localhost:8080
- **GraphQL Playground**: http://localhost:8080/

## หยุดการทำงาน

```bash
./scripts/dev-local.sh stop
```

---

**หมายเหตุ**: เมื่อ Docker network ใช้งานได้ปกติ สามารถกลับไปใช้ `./scripts/dev.sh` ได้ตามปกติ