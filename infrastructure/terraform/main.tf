terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "cloudbuild.googleapis.com",
    "secretmanager.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
    "cloudtrace.googleapis.com",
    "vpcaccess.googleapis.com",
    "servicenetworking.googleapis.com",
    "compute.googleapis.com"
  ])

  service            = each.value
  disable_on_destroy = false
}

# VPC Network
resource "google_compute_network" "vpc" {
  name                    = "tru-activity-vpc"
  auto_create_subnetworks = false
  depends_on              = [google_project_service.apis]
}

resource "google_compute_subnetwork" "subnet" {
  name          = "tru-activity-subnet"
  ip_cidr_range = "10.0.0.0/24"
  region        = var.region
  network       = google_compute_network.vpc.id

  private_ip_google_access = true
}

# Private service connection for Cloud SQL
resource "google_compute_global_address" "private_ip_address" {
  name          = "tru-activity-private-ip"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

# VPC Access Connector for Cloud Run
resource "google_vpc_access_connector" "connector" {
  name           = "tru-activity-connector"
  region         = var.region
  ip_cidr_range  = "10.8.0.0/28"
  network        = google_compute_network.vpc.name
  machine_type   = "e2-micro"
  min_instances  = 2
  max_instances  = 10
  depends_on     = [google_project_service.apis]
}

# Cloud SQL PostgreSQL Instance
resource "google_sql_database_instance" "postgres" {
  name             = "tru-activity-postgres"
  database_version = "POSTGRES_15"
  region           = var.region
  deletion_protection = false

  settings {
    tier              = "db-custom-2-4096"
    availability_type = "REGIONAL"
    disk_type         = "PD_SSD"
    disk_size         = 20
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      location                       = var.region
      point_in_time_recovery_enabled = true
      backup_retention_settings {
        retained_backups = 7
        retention_unit   = "COUNT"
      }
    }

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.vpc.id
      enable_private_path_for_google_cloud_services = true
    }

    database_flags {
      name  = "max_connections"
      value = "100"
    }

    database_flags {
      name  = "shared_preload_libraries"
      value = "pg_stat_statements"
    }

    insights_config {
      query_insights_enabled  = true
      record_application_tags = true
      record_client_address   = true
    }
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

resource "google_sql_database" "database" {
  name     = "tru_activity"
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_user" "app_user" {
  name     = "tru_activity_user"
  instance = google_sql_database_instance.postgres.name
  password = var.db_password
}

# Cloud Memorystore Redis
resource "google_redis_instance" "redis" {
  name           = "tru-activity-redis"
  tier           = "STANDARD_HA"
  memory_size_gb = 1
  region         = var.region

  authorized_network = google_compute_network.vpc.id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"
  redis_version      = "REDIS_7_0"

  redis_configs = {
    maxmemory-policy = "allkeys-lru"
  }

  depends_on = [google_project_service.apis]
}

# Service Accounts
resource "google_service_account" "backend_sa" {
  account_id   = "tru-activity-backend"
  display_name = "TRU Activity Backend Service Account"
  description  = "Service account for TRU Activity backend Cloud Run service"
}

resource "google_service_account" "migration_sa" {
  account_id   = "tru-activity-migration"
  display_name = "TRU Activity Migration Service Account"
  description  = "Service account for database migrations"
}

resource "google_service_account" "cloudbuild_sa" {
  account_id   = "tru-activity-cloudbuild"
  display_name = "TRU Activity Cloud Build Service Account"
  description  = "Service account for Cloud Build deployments"
}

# IAM Bindings
resource "google_project_iam_member" "backend_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

resource "google_project_iam_member" "backend_redis_editor" {
  project = var.project_id
  role    = "roles/redis.editor"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

resource "google_project_iam_member" "backend_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

resource "google_project_iam_member" "backend_monitoring_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

resource "google_project_iam_member" "backend_logging_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

resource "google_project_iam_member" "backend_trace_writer" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.backend_sa.email}"
}

# Migration Service Account permissions
resource "google_project_iam_member" "migration_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.migration_sa.email}"
}

resource "google_project_iam_member" "migration_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.migration_sa.email}"
}

resource "google_project_iam_member" "migration_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.migration_sa.email}"
}

# Cloud Build Service Account permissions
resource "google_project_iam_member" "cloudbuild_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_sa.email}"
}

resource "google_project_iam_member" "cloudbuild_storage_admin" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.cloudbuild_sa.email}"
}

resource "google_project_iam_member" "cloudbuild_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.cloudbuild_sa.email}"
}

resource "google_project_iam_member" "cloudbuild_sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.cloudbuild_sa.email}"
}

# Secrets in Secret Manager
resource "google_secret_manager_secret" "db_config" {
  secret_id = "db-config"
  
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "db_config" {
  secret = google_secret_manager_secret.db_config.id
  
  secret_data = jsonencode({
    host     = google_sql_database_instance.postgres.private_ip_address
    port     = "5432"
    name     = google_sql_database.database.name
    user     = google_sql_user.app_user.name
    password = var.db_password
  })
}

resource "google_secret_manager_secret" "redis_config" {
  secret_id = "redis-config"
  
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "redis_config" {
  secret = google_secret_manager_secret.redis_config.id
  
  secret_data = jsonencode({
    url = "redis://${google_redis_instance.redis.host}:${google_redis_instance.redis.port}"
  })
}

resource "google_secret_manager_secret" "jwt_config" {
  secret_id = "jwt-config"
  
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "jwt_config" {
  secret = google_secret_manager_secret.jwt_config.id
  
  secret_data = jsonencode({
    secret = var.jwt_secret
  })
}

resource "google_secret_manager_secret" "email_config" {
  secret_id = "email-config"
  
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "email_config" {
  secret = google_secret_manager_secret.email_config.id
  
  secret_data = jsonencode({
    from            = var.email_from
    sendgrid_api_key = var.sendgrid_api_key
  })
}

resource "google_secret_manager_secret" "qr_config" {
  secret_id = "qr-config"
  
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "qr_config" {
  secret = google_secret_manager_secret.qr_config.id
  
  secret_data = jsonencode({
    secret = var.qr_secret
  })
}

# Cloud Run service will be deployed via Cloud Build