resource "ffxf_database_user" "example" {
  database_id = 42
  username    = "app"
  read_only   = true
}
