module "fail" {
  source        = "../../"
  condition     = false
  error_message = "sample error"
}
