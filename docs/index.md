---
page_title: "Provider: openimages"
description: |-
  Read-only data sources over the openimages.cloud catalog of signed, boot-tested cloud images.
---

# openimages Provider

Data sources over the [openimages.cloud](https://openimages.cloud) catalog: the versioned URL,
checksums, sizes and Glance properties of signed, boot-tested cloud images for OpenStack and
Proxmox VE, to chain into `openstack_images_image_v2`, a Proxmox `download_file` or any
resource that takes an image URL. The provider only reads the public catalog index; it needs no
credentials and creates nothing.

## Example

```terraform
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
  min_ram_mb       = data.openimages_image.debian.min_ram_mb
  properties       = data.openimages_image.debian.glance_properties
}
```

## Schema

### Optional

- `index_url` (String) Catalog index URL (`oic-index/v1`). Defaults to `$OPENIMAGES_INDEX_URL`, else
  `https://images.openimages.cloud/index.json`. Point it at an internal mirror (oic-sync mirror,
  Ansible role `openimages.cloud.oic_mirror`) for clouds without Internet access.
