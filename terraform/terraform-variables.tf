# /terraform/variables.tf

variable "aws_region" {
  description = "The AWS region to deploy into"
  type        = string
  default     = "us-east-1"
}

variable "app_name" {
  description = "Name of the application"
  type        = string
  default     = "solana-client"
}

variable "app_count" {
  description = "Number of application instances to run"
  type        = number
  default     = 2
}

variable "task_cpu" {
  description = "CPU units for Fargate task"
  type        = number
  default     = 256
  validation {
    condition     = contains([256, 512, 1024, 2048, 4096], var.task_cpu)
    error_message = "Valid CPU values: 256, 512, 1024, 2048, 4096."
  }
}

variable "task_memory" {
  description = "Memory in MiB for Fargate task"
  type        = number
  default     = 512
  validation {
    condition     = contains([512, 1024, 2048, 3072, 4096, 8192, 16384], var.task_memory)
    error_message = "Valid memory values: 512, 1024, 2048, 3072, 4096, 8192, 16384."
  }
}

