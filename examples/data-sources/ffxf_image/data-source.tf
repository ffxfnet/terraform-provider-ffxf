data "ffxf_image" "debian" {
  slug = "debian-13"
}

output "image_default_user" {
  value = data.ffxf_image.debian.default_user
}

output "image_supports_ssh_keys" {
  value = data.ffxf_image.debian.supports_ssh_keys
}