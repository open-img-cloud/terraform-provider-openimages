# Debian 13 in Glance, from the openimages.cloud catalog: Glance downloads the
# signed, boot-tested image itself (web-download) and gets the catalog's
# properties (os_distro, hw_qemu_guest_agent, hw_disk_bus...).
terraform {
  required_providers {
    openimages = { source = "open-img-cloud/openimages" }
    openstack  = { source = "terraform-provider-openstack/openstack", version = "~> 3.0" }
  }
}

data "openimages_image" "debian" {
  id = "debian-13"
}

resource "openstack_images_image_v2" "debian" {
  name             = "${data.openimages_image.debian.title} ${data.openimages_image.debian.version}"
  image_source_url = data.openimages_image.debian.url
  web_download     = true
  container_format = "bare"
  disk_format      = data.openimages_image.debian.disk_format
  min_disk_gb      = data.openimages_image.debian.min_disk_gb
  properties       = data.openimages_image.debian.glance_properties

  # Glance computes the SHA-512 of what it downloaded: it must be the catalog's.
  lifecycle {
    postcondition {
      condition     = self.properties["os_hash_value"] == data.openimages_image.debian.sha512
      error_message = "The image Glance downloaded does not match the catalog SHA-512."
    }
  }
}
