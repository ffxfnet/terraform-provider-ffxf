#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
temp_dir=$(mktemp -d)
cleanup() {
  rm -f "$temp_dir/terraform-provider-ffxf" "$temp_dir/tofu.tfrc"
  rmdir "$temp_dir"
}
trap cleanup EXIT HUP INT TERM

go -C "$repo_root" build -o "$temp_dir/terraform-provider-ffxf" .
cat > "$temp_dir/tofu.tfrc" <<CONFIG
provider_installation {
  dev_overrides {
    "registry.opentofu.org/ffxfnet/ffxf" = "$temp_dir"
  }
  direct {}
}
CONFIG

for test_name in \
  ffxf_vms ffxf_vm_network ffxf_vm_metrics ffxf_vm_backups ffxf_vm_actions \
  ffxf_vm_ipv6_address ffxf_vm_rdns ffxf_vm_power_state \
  ffxf_vm_reinstall ffxf_vm_backup_restore ffxf_instance_ssh_keys
do
  (cd "$repo_root/tests/terraform/$test_name" && TF_CLI_CONFIG_FILE="$temp_dir/tofu.tfrc" tofu test)
done
