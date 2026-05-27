output "vm_ip" {
  value = yandex_vpc_address.pipeline.external_ipv4_address[0].address
}

output "ssh_command" {
  value = "ssh ubuntu@${yandex_vpc_address.pipeline.external_ipv4_address[0].address}"
}

output "temporal_ui_url" {
  value = "http://${yandex_vpc_address.pipeline.external_ipv4_address[0].address}:8080"
}
