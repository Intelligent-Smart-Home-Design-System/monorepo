terraform {
  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
    }
  }
  required_version = ">= 1.3"
}

provider "yandex" {
  zone      = "ru-central1-a"
}

data "yandex_vpc_network" "pipeline" {
  name = "default"
}

resource "yandex_vpc_subnet" "pipeline" {
  name           = "pipeline-subnet"
  zone           = var.yc_zone
  network_id     = data.yandex_vpc_network.pipeline.id
  v4_cidr_blocks = ["10.0.0.0/24"]
}

resource "yandex_vpc_security_group" "pipeline" {
  name       = "pipeline-sg"
  network_id = data.yandex_vpc_network.pipeline.id

  ingress {
    protocol       = "TCP"
    port           = 22
    v4_cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    protocol       = "TCP"
    port           = 8080
    v4_cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol       = "ANY"
    from_port      = 0
    to_port        = 65535
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "yandex_vpc_address" "pipeline" {
  name = "pipeline-static-ip"
  external_ipv4_address {
    zone_id = var.yc_zone
  }
}

data "yandex_compute_image" "ubuntu" {
  family = "ubuntu-2204-lts"
}

resource "yandex_compute_instance" "pipeline" {
  name        = var.vm_name
  platform_id = "standard-v3"
  zone        = var.yc_zone

  resources {
    cores         = var.vm_cores
    memory        = var.vm_memory_gb
    core_fraction = 100
  }

  boot_disk {
    initialize_params {
      image_id = data.yandex_compute_image.ubuntu.id
      size     = var.vm_disk_gb
      type     = "network-ssd"
    }
  }

  network_interface {
    subnet_id          = yandex_vpc_subnet.pipeline.id
    security_group_ids = [yandex_vpc_security_group.pipeline.id]
    nat                = true
    nat_ip_address     = yandex_vpc_address.pipeline.external_ipv4_address[0].address
  }

  metadata = {
    user-data = templatefile("${path.module}/../../infra/scripts/bootstrap.sh.tpl", {
      nginx_username = var.nginx_username
      nginx_password = var.nginx_password
    })
    ssh-keys = "ubuntu:${file(var.ssh_public_key_path)}"
  }
}
