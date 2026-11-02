#!/bin/sh
# Runs every OpenTofu native test in tests/terraform against a freshly built provider.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

mkdir "$tmp_dir/bin"
(cd "$repo_root" && go build -o "$tmp_dir/bin/terraform-provider-ffxf" .)
cat > "$tmp_dir/tfrc" <<EOF
provider_installation {
  dev_overrides {
    "registry.opentofu.org/ffxfnet/ffxf" = "$tmp_dir/bin"
  }
  direct {}
}
EOF

for test_dir in "$repo_root"/tests/terraform/*/; do
  (cd "$test_dir" && TF_CLI_CONFIG_FILE="$tmp_dir/tfrc" tofu test)
done
