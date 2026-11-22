# Several catalog images into Glance, as oic-sync names them, with the
# catalog properties and a check of the digest Glance computes. Images past
# their upstream end of life are left out. This replaces the former
# terraform-openstack-openimages module.
terraform {
  required_providers {
    openimages = { source = "open-img-cloud/openimages" }
    openstack  = { source = "terraform-provider-openstack/openstack", version = "~> 3.0" }
  }
}

variable "images" {
  description = "Catalog ids to import (see data.openimages_images)."
  type        = list(string)
  default     = ["debian-13", "ubuntu-24.04", "rocky-linux-9", "alpine-linux"]
}

variable "with_optional_properties" {
  description = "Also set the properties the catalog recommends where the cloud supports them (hw_rng_model)."
  type        = bool
  default     = false
}

data "openimages_image" "this" {
  for_each = toset(var.images)
  id       = each.value
}

locals {
  current = { for id, img in data.openimages_image.this : id => img if !img.end_of_life }
}

resource "openstack_images_image_v2" "this" {
  for_each = local.current

  name             = each.value.name
  image_source_url = each.value.url
  web_download     = true
  container_format = "bare"
  disk_format      = each.value.disk_format
  min_disk_gb      = each.value.min_disk_gb
  properties = merge(
    var.with_optional_properties ? each.value.optional_glance_properties : {},
    each.value.glance_properties,
  )

  lifecycle {
    postcondition {
      condition     = lookup(self.properties, "os_hash_value", "") == each.value.sha512
      error_message = "The image Glance downloaded for ${each.key} does not match the catalog SHA-512."
    }
  }
}

output "images" {
  value = { for id, img in openstack_images_image_v2.this : id => { id = img.id, name = img.name } }
}

output "end_of_life" {
  description = "Requested images left out because their upstream release is past its end of life."
  value       = [for id, img in data.openimages_image.this : "${id} (security support ended ${img.security_until})" if img.end_of_life]
}
