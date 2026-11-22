---
page_title: "openimages_images Data Source"
description: |-
  The images of the openimages.cloud catalog, optionally filtered.
---

# openimages_images (Data Source)

## Example Usage

```terraform
data "openimages_images" "kubernetes" {
  tag = "kubernetes"
}

data "openimages_image" "each" {
  for_each = toset(data.openimages_images.kubernetes.ids)
  id       = each.value
}
```

## Schema

### Optional

- `tag` (String) Keep images carrying this tag (`kubernetes`, `cloud-init`, `octavia`...).
- `os_distro` (String) Keep images with this OpenStack `os_distro` (`ubuntu`, `rocky`...).
- `include_archived` (Boolean) Also list archived images (end of life upstream). Default `false`.

### Read-Only

- `ids` (List of String) Catalog ids of the images kept.
- `images` (List of Object) `id`, `title`, `latest_version`, `status`, `os_distro`, `login_user`, `tags`.
