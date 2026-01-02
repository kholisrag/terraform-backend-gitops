terraform {
  required_version = ">= 1.0"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

# Simple null resource for testing
resource "null_resource" "test" {
  triggers = {
    timestamp = timestamp()
  }
}

output "test_output" {
  value = "test-value-${null_resource.test.id}"
}

output "timestamp" {
  value = null_resource.test.triggers.timestamp
}
