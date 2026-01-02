terraform {
  required_version = ">= 1.0"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

variable "worker_id" {
  type        = string
  default     = "worker-1"
  description = "Worker ID for concurrent testing"
}

# Resource with delay to test locking
resource "null_resource" "concurrent_test" {
  triggers = {
    worker_id = var.worker_id
    timestamp = timestamp()
  }

  provisioner "local-exec" {
    command = "sleep 2"  # Short delay to test lock holding
  }
}

output "worker_id" {
  value = var.worker_id
}

output "completed_at" {
  value = timestamp()
}
