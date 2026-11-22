<p align="center">
  <a href="https://openimages.cloud">
    <img src="https://openimages.cloud/img/og.png" alt="openimages.cloud: signed, boot-tested cloud images for OpenStack and Proxmox" width="720">
  </a>
</p>

<h1 align="center">terraform-provider-openimages</h1>

<p align="center">
  <a href="https://github.com/open-img-cloud/terraform-provider-openimages/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/open-img-cloud/terraform-provider-openimages/actions/workflows/ci.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="License Apache-2.0" src="https://img.shields.io/badge/license-Apache--2.0-555"></a>
</p>

<p align="center">
  <a href="https://openimages.cloud"><b>Website</b></a> ·
  <a href="https://openimages.cloud/images/">Catalog</a> ·
  <a href="https://images.openimages.cloud/index.json">index.json</a> ·
  <a href="https://github.com/open-img-cloud/terraform-provider-openimages/issues">Report an issue</a>
</p>

Terraform and OpenTofu provider with **data sources** over the
[openimages.cloud](https://openimages.cloud) catalog: the versioned URL, checksums, sizes and
Glance properties of signed, boot-tested cloud images for OpenStack and Proxmox VE, to chain
into the resource of your choice. It only reads the public catalog index: no credentials,
nothing created.

```hcl
terraform {
  required_providers {
    openimages = { source = "open-img-cloud/openimages" }
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
}
```

Proxmox VE (bpg/proxmox) and a Glance checksum postcondition: [`examples/`](examples).

## Data sources

| | |
|---|---|
| [`openimages_image`](docs/data-sources/image.md) | One image: `url`, `sha256`, `sha512`, `min_disk_gb`, `name`, `glance_properties`, `end_of_life`, `bundle_url`, `login_user`... at the latest version or a pinned `version`, by `variant` or `firmware` |
| [`openimages_images`](docs/data-sources/images.md) | The catalog ids, filtered by `tag` or `os_distro` |

Provider setting: `index_url` (or `$OPENIMAGES_INDEX_URL`) for an internal mirror.

## Several images, like oic-sync

[`examples/openstack-catalog`](examples/openstack-catalog) imports a list of catalog images
into Glance with oic-sync's names (`name`), the catalog properties, a check of the digest
Glance computes, and images past their upstream end of life (`end_of_life`) left out. It
replaces the former `terraform-openstack-openimages` module.

## Verifying the image

Glance's `web_download` and Proxmox's `download_file` check the checksum. To also check the
Sigstore signature before importing:

```sh
cosign verify-blob <file> --bundle <bundle_url> \
  --certificate-identity-regexp '<cosign_certificate_identity_regexp>' \
  --certificate-oidc-issuer '<cosign_certificate_oidc_issuer>'
```

## Development

```sh
go test ./...                    # unit tests on a fixture of the real index
go build -o ~/bin/terraform-provider-openimages .
# ~/.tofurc or ~/.terraformrc:
#   provider_installation { dev_overrides { "open-img-cloud/openimages" = "/home/you/bin" } direct {} }
cd tests/e2e && tofu apply      # the data sources against the live catalog
```

Releases: a `v*` tag runs goreleaser (binaries for Linux, macOS, Windows and FreeBSD, amd64 and
arm64, `SHA256SUMS` signed with the release GPG key) for the Terraform and OpenTofu registries.

## License

[Apache-2.0](LICENSE).
