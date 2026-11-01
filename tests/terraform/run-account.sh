#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
tmp_dir=$(mktemp -d /tmp/wt-b-account-tests.XXXXXX)
mkdir "$tmp_dir/bin"
cp "$repo_root/internal/provider/provider.go" "$tmp_dir/provider.go"
restore_provider() {
  cp "$tmp_dir/provider.go" "$repo_root/internal/provider/provider.go"
}
cleanup() {
  restore_provider
  rm -f "$tmp_dir/provider.go" "$tmp_dir/tfrc" "$tmp_dir/bin/terraform-provider-ffxf"
  rmdir "$tmp_dir/bin" "$tmp_dir"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

provider_file="$repo_root/internal/provider/provider.go"
sed -i '/NewLoadBalancerListenerResource,/a\		NewAccountAreaSSHKeyResource,\
\		NewAccountAreaVPCMemberResource,\
\		NewAccountAreaBudgetResource,' "$provider_file"
sed -i '/NewLoadBalancerMetricsDataSource,/a\		NewAccountAreaPlansDataSource,\
\		NewAccountAreaImagesDataSource,\
\		NewAccountAreaVPCsDataSource,\
\		NewAccountAreaSSHKeysDataSource,\
\		NewAccountAreaAccountDataSource,\
\		NewAccountAreaUsageDataSource,\
\		NewAccountAreaRunnersDataSource,\
\		NewAccountAreaRunnerJobsDataSource,' "$provider_file"
gofmt -w "$provider_file"
(cd "$repo_root" && go build -o "$tmp_dir/bin/terraform-provider-ffxf" .)
restore_provider
cat > "$tmp_dir/tfrc" <<EOF_TFRC
provider_installation {
  dev_overrides {
    "registry.opentofu.org/ffxfnet/ffxf" = "$tmp_dir/bin"
  }
  direct {}
}
EOF_TFRC
for test_dir in "$repo_root"/tests/terraform/*; do
  [ -d "$test_dir" ] || continue
  (cd "$test_dir" && TF_CLI_CONFIG_FILE="$tmp_dir/tfrc" tofu test)
done
