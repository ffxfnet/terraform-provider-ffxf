#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temp_dir="$(mktemp -d /tmp/ffxf-database-tests.XXXXXX)"
cleanup() {
  rm -f "$temp_dir/provider/terraform-provider-ffxf" "$temp_dir/terraform.rc"
  rmdir "$temp_dir/provider"
  rmdir "$temp_dir"
}
trap cleanup EXIT

mkdir -p "$temp_dir/provider"
go -C "$repo_root" build -o "$temp_dir/provider/terraform-provider-ffxf" .
cat > "$temp_dir/terraform.rc" <<EOF
provider_installation {
  dev_overrides {
    "registry.opentofu.org/ffxfnet/ffxf" = "$temp_dir/provider"
  }
  direct {}
}
EOF

for test_dir in \
  "ffxf_database" \
  "ffxf_database_user" \
  "ffxf_database_logical_database" \
  "ffxf_database_extension" \
  "ffxf_database_public_access" \
  "ffxf_database_backup" \
  "ffxf_database_fork" \
  "ffxf_database_restore" \
  "ffxf_database_detach" \
  "ffxf_databases" \
  "ffxf_database_options" \
  "ffxf_database_credentials" \
  "ffxf_database_backups" \
  "ffxf_database_metrics" \
  "ffxf_database_queries" \
; do
  echo "Running OpenTofu tests: $test_dir"
  (cd "$repo_root/tests/terraform/$test_dir" && TF_CLI_CONFIG_FILE="$temp_dir/terraform.rc" tofu test)
done
