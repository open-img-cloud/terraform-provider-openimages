# End-to-end: the data sources against the live catalog (CI, ci.yml e2e).
terraform {
  required_providers {
    openimages = { source = "open-img-cloud/openimages" }
  }
}

data "openimages_image" "debian" { id = "debian-13" }
data "openimages_image" "alpine_uefi" {
  id       = "alpine-linux"
  firmware = "uefi"
}
data "openimages_images" "all" {}
data "openimages_images" "k8s" { tag = "kubernetes" }

output "checks" {
  value = {
    debian_url      = startswith(data.openimages_image.debian.url, "https://images.openimages.cloud/debian-13/")
    debian_sha512   = length(data.openimages_image.debian.sha512) == 128
    debian_props    = data.openimages_image.debian.glance_properties["os_distro"] == "debian"
    debian_min_disk = data.openimages_image.debian.min_disk_gb > 0
    alpine_uefi     = data.openimages_image.alpine_uefi.firmware == "uefi" && strcontains(data.openimages_image.alpine_uefi.url, "uefi")
    catalog_size    = length(data.openimages_images.all.ids) > 20
    talos_in_k8s    = contains(data.openimages_images.k8s.ids, "talos-linux")
    bundle          = endswith(data.openimages_image.debian.bundle_url, ".bundle")
  }
}
