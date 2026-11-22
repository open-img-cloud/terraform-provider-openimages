# Rocky Linux 9 on Proxmox VE 9 with the bpg/proxmox provider: the node
# downloads the versioned file and checks its SHA-256 from the catalog.
terraform {
  required_providers {
    openimages = { source = "open-img-cloud/openimages" }
    proxmox    = { source = "bpg/proxmox" }
  }
}

data "openimages_image" "rocky" {
  id = "rocky-linux-9"
}

resource "proxmox_download_file" "rocky" {
  node_name          = "pve"
  datastore_id       = "local"
  content_type       = "import"
  url                = data.openimages_image.rocky.url
  file_name          = basename(data.openimages_image.rocky.url)
  checksum           = data.openimages_image.rocky.sha256
  checksum_algorithm = "sha256"
}
