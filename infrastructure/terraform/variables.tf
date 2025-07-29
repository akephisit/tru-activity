variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
  default     = "asia-southeast1"
}

variable "db_password" {
  description = "Password for the PostgreSQL database user"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT secret key for authentication"
  type        = string
  sensitive   = true
}

variable "email_from" {
  description = "From email address for notifications"
  type        = string
}

variable "sendgrid_api_key" {
  description = "SendGrid API key for email notifications"
  type        = string
  sensitive   = true
}

variable "qr_secret" {
  description = "Secret key for QR code generation and validation"
  type        = string
  sensitive   = true
}