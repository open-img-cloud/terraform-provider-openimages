---
page_title: "openimages_image Data Source"
description: |-
  One image of the openimages.cloud catalog: URL, checksums and Glance properties.
---

# openimages_image (Data Source)

One image of the catalog, at its latest version or a given one.

## Example Usage

```terraform
data "openimages_image" "alpine" {
  id       = "alpine-linux"
  firmware = "uefi"
}

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
- `disk_format` (String) `qcow2`. `arch` (String) e.g. `x86_64`.
- `glance_properties` (Map of String) The catalog's Glance properties plus the `oic_*` keys oic-sync sets.
- `title`, `status`, `login_user`, `built_at` (String).
- `bundle_url` (String) Sigstore bundle; verify with `cosign verify-blob --bundle` and the `cosign_certificate_identity_regexp` and `cosign_certificate_oidc_issuer` attributes.
- `manifest_url` (String) `MANIFEST.json` of the version (commit, CI run, builder digest).
