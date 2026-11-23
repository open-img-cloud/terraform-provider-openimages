---
page_title: "openimages_image Data Source"
description: |-
  One image of the openimages.cloud catalog: URL, checksums and Glance properties.
---

# openimages_image (Data Source)

One image of the catalog, at its latest version or a given one: the versioned download URL,
the checksums, the sizes and the Glance properties, ready for `openstack_images_image_v2` or a
Proxmox VE download. Images with several variants (Alpine BIOS and UEFI, Alpaquita glibc and
musl) are selected with `variant` or `firmware`; without them, the catalog's first artifact.

## Example Usage

```terraform
# Latest Debian 13, imported by Glance itself
data "openimages_image" "debian" {
  id = "debian-13"
}

resource "openstack_images_image_v2" "debian" {
  name             = "${data.openimages_image.debian.name} (${data.openimages_image.debian.version})"
  image_source_url = data.openimages_image.debian.url
  web_download     = true
  container_format = "bare"
  disk_format      = data.openimages_image.debian.disk_format
  min_disk_gb      = data.openimages_image.debian.min_disk_gb
  min_ram_mb       = data.openimages_image.debian.min_ram_mb
  properties       = data.openimages_image.debian.glance_properties
}

# A variant, by firmware
data "openimages_image" "alpine_uefi" {
  id       = "alpine-linux"
  firmware = "uefi"
}

# A given version (from the image's versions.json)
data "openimages_image" "debian_pinned" {
  id      = "debian-13"
  version = "13-20261103-2651"
}
```

## Schema

### Required

- `id` (String) Catalog id, e.g. `debian-13`, `rocky-linux-9`, `talos-linux` (list them with `openimages_images`).

### Optional

- `variant` (String) Variant for images that have several (`bios`/`uefi` for Alpine, `glibc`/`musl` for Alpaquita). Default: the catalog's first artifact.
- `firmware` (String) `bios` or `uefi`, to pick by firmware instead of variant.
- `version` (String) Version to use; default: the latest. Older versions come from the image's `versions.json`; their Glance properties are the current ones with version-specific `oic_version` and `oic_sha256`.

### Read-Only

- `url` (String) Versioned, immutable download URL.
- `sha256`, `sha512` (String) Checksums of the file (`sha512` is what Glance stores as `os_hash_value`).
- `size`, `virtual_size` (Number) File and guest disk sizes, in bytes; `min_disk_gb` (Number) the virtual size rounded up to GiB.
- `min_ram_mb` (Number) Smallest RAM the image boots with, in MiB (Glance `min_ram_mb`); 0 when the index does not publish it.
- `disk_format` (String) `qcow2`. `arch` (String) e.g. `x86_64`.
- `name` (String) Stable Glance name, as oic-sync gives it: the title, plus the variant in capitals when the image has several (`Alpine Linux (UEFI)`).
- `glance_properties` (Map of String) The catalog's Glance properties plus the `oic_*` keys oic-sync sets (`oic_id`, `oic_version`, `oic_sha256`, `oic_variant`).
- `optional_glance_properties` (Map of String) Properties the catalog recommends where the cloud supports them (`hw_rng_model`).
- `end_of_life`, `end_of_life_soon` (Boolean) Upstream support status (endoflife.date dates published in the index); `security_until` (String) end of security support, `support_url` (String) its source.
- `title`, `status`, `login_user`, `built_at` (String).
- `bundle_url` (String) Sigstore bundle; verify with `cosign verify-blob --bundle` and the `cosign_certificate_identity_regexp` and `cosign_certificate_oidc_issuer` attributes.
- `manifest_url` (String) `MANIFEST.json` of the version (commit, CI run, builder digest).
