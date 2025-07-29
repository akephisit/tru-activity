output "postgres_connection_name" {
  description = "Cloud SQL PostgreSQL connection name"
  value       = google_sql_database_instance.postgres.connection_name
}

output "postgres_private_ip" {
  description = "Cloud SQL PostgreSQL private IP address"
  value       = google_sql_database_instance.postgres.private_ip_address
}

output "redis_host" {
  description = "Redis instance host"
  value       = google_redis_instance.redis.host
}

output "redis_port" {
  description = "Redis instance port"
  value       = google_redis_instance.redis.port
}

output "vpc_connector_name" {
  description = "VPC Access Connector name"
  value       = google_vpc_access_connector.connector.name
}

output "backend_service_account_email" {
  description = "Backend service account email"
  value       = google_service_account.backend_sa.email
}

output "migration_service_account_email" {
  description = "Migration service account email"
  value       = google_service_account.migration_sa.email
}

output "cloudbuild_service_account_email" {
  description = "Cloud Build service account email"
  value       = google_service_account.cloudbuild_sa.email
}

output "database_name" {
  description = "PostgreSQL database name"
  value       = google_sql_database.database.name
}

output "database_user" {
  description = "PostgreSQL database user"
  value       = google_sql_user.app_user.name
}