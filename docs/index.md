---
page_title: "Provider: openimages"
description: |-
  Read-only data sources over the openimages.cloud catalog of signed, boot-tested cloud images for OpenStack and Proxmox VE.
---

# openimages Provider

Data sources over the [openimages.cloud](https://openimages.cloud) catalog: the versioned URL,
checksums, sizes and Glance properties of signed, boot-tested cloud images (Debian, Ubuntu,
Rocky, Alma, Alpine, FreeBSD, Flatcar, Talos, OPNsense, OpenStack service images and more), to
chain into `openstack_images_image_v2`, a Proxmox VE download or any resource that takes an
image URL and a checksum.

The provider only reads the public catalog index
([`index.json`](https://images.openimages.cloud/index.json)): it needs no credentials and
creates nothing. The [catalog](https://openimages.cloud/images/) lists the image ids.

## Example Usage

```terraform
terraform {
  required_providers {
    openimages = {
      source  = "open-img-cloud/openimages"
      version = "~> 0.2"
    }
    openstack = {
      source  = "terraform-provider-openstack/openstack"
      version = "~> 3.0"
    }
  }
}

provider "openimages" {}

data "openimages_image" "debian" {
  id = "debian-13"
}

# Glance downloads the image itself (web-download) and gets the catalog properties
# (os_distro, hw_qemu_guest_agent, hw_disk_bus, SCS-0102 metadata...).
resource "openstack_images_image_v2" "debian" {
  name             = "${data.openimages_image.debian.name} (${data.openimages_image.debian.version})"
  image_source_url = data.openimages_image.debian.url
  web_download     = true
  container_format = "bare"
  disk_format      = data.openimages_image.debian.disk_format
  min_disk_gb      = data.openimages_image.debian.min_disk_gb
  min_ram_mb       = data.openimages_image.debian.min_ram_mb
  properties       = data.openimages_image.debian.glance_properties

  # Glance computes the SHA-512 of what it downloaded: it must be the catalog's.
  lifecycle {
    postcondition {
      condition     = lookup(self.properties, "os_hash_value", "") == data.openimages_image.debian.sha512
      error_message = "The image Glance downloaded does not match the catalog SHA-512."
    }
  }
}
```

### Proxmox VE

With the [bpg/proxmox](https://registry.terraform.io/providers/bpg/proxmox/latest) provider, the
node downloads the versioned file and checks its SHA-256:

```terraform
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
```

### Several images, like oic-sync

`openimages_images` lists the catalog, `openimages_image` gives the stable name oic-sync uses
(`name`) and the upstream support status (`end_of_life`):

```terraform
data "openimages_image" "this" {
  for_each = toset(["debian-13", "ubuntu-24.04", "rocky-linux-9"])
  id       = each.value
}

resource "openstack_images_image_v2" "this" {
  for_each = { for id, img in data.openimages_image.this : id => img if !img.end_of_life }

  name             = each.value.name
  image_source_url = each.value.url
  web_download     = true
  container_format = "bare"
  disk_format      = each.value.disk_format
  min_disk_gb      = each.value.min_disk_gb
  min_ram_mb       = each.value.min_ram_mb
  properties       = each.value.glance_properties
}
```

## Verifying the signature

Glance `web_download` and Proxmox `download_file` check the checksum. Each image is also signed
with [Sigstore cosign](https://openimages.cloud/verify/); the data source gives what
`cosign verify-blob` needs:

```shell
cosign verify-blob <file> --bundle <bundle_url> \
  --certificate-identity-regexp '<cosign_certificate_identity_regexp>' \
  --certificate-oidc-issuer '<cosign_certificate_oidc_issuer>'
```

## Internal mirror

For clouds without Internet access, point the provider at a mirror of the catalog
([oic-sync mirror](https://openimages.cloud/oic-sync/), Ansible role
`openimages.cloud.oic_mirror`):

```terraform
provider "openimages" {
  index_url = "https://images.example.internal/index.json"
}
```

## Schema

### Optional

- `index_url` (String) Catalog index URL (`oic-index/v1`). Defaults to `$OPENIMAGES_INDEX_URL`, else
  `https://images.openimages.cloud/index.json`.
