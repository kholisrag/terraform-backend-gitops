terraform {
  required_version = ">= 1.0"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

variable "iteration" {
  type        = number
  default     = 1
  description = "Iteration number for testing stateful operations"
}

# Use terraform_data for stateful operations
resource "terraform_data" "counter" {
  input = var.iteration
}

# Create multiple resources to generate realistic state size
resource "null_resource" "items" {
  count = 5

  triggers = {
    iteration = var.iteration
    index     = count.index
  }
}

output "counter_value" {
  value = terraform_data.counter.output
}

output "items_count" {
  value = length(null_resource.items)
}

output "iteration" {
  value = var.iteration
}
