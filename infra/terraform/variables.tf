variable "yc_zone" {
  type    = string
  default = "ru-central1-a"
}

variable "vm_name" {
  type    = string
  default = "pipeline-vm"
}

variable "vm_cores" {
  type    = number
  default = 4
}

variable "vm_memory_gb" {
  type    = number
  default = 8
}

variable "vm_disk_gb" {
  type    = number
  default = 50
}

variable "ssh_public_key_path" {
  type    = string
  default = "~/.ssh/id_rsa.pub"
}

variable "nginx_username" {
  type    = string
  default = "admin"
}

variable "nginx_password" {
  description = "Basic auth password for nginx"
  type        = string
  sensitive   = true
}
