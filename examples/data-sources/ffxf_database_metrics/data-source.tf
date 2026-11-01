data "ffxf_database_metrics" "current" {
  database_id = 42
  range       = "1h"
}
