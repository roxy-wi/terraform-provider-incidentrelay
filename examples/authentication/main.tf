terraform {
  required_providers {
    incidentrelay = {
      source  = "roxy-wi/incidentrelay"
      version = "~> 0.4"
    }
  }
}

variable "incidentrelay_base_url" {
  type        = string
  description = "IncidentRelay base URL."
}

variable "incidentrelay_token" {
  type        = string
  description = "IncidentRelay API token."
  sensitive   = true
  default     = null
}

variable "incidentrelay_username" {
  type        = string
  description = "IncidentRelay username for login-based authentication."
  default     = null
}

variable "incidentrelay_password" {
  type        = string
  description = "IncidentRelay password for login-based authentication."
  sensitive   = true
  default     = null
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url

  # Recommended for CI and automation.
  token = var.incidentrelay_token

  # Alternative for local development. Leave token null when using this mode.
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}

data "incidentrelay_version" "current" {}

output "incidentrelay_version" {
  value = data.incidentrelay_version.current.service_version
}
