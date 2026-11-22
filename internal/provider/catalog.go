package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultIndexURL is the public catalog index (schema oic-index/v1).
const DefaultIndexURL = "https://images.openimages.cloud/index.json"

// Index is the subset of oic-index/v1 the provider reads.
type Index struct {
	Schema      string  `json:"schema"`
	BaseURL     string  `json:"base_url"`
	GeneratedAt string  `json:"generated_at"`
	Images      []Image `json:"images"`
	Signing     Signing `json:"signing"`
}

type Signing struct {
	Type                      string `json:"type"`
	CertificateIdentityRegexp string `json:"certificate_identity_regexp"`
	CertificateOIDCIssuer     string `json:"certificate_oidc_issuer"`
}

type Image struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	LatestVersion string     `json:"latest_version"`
	LoginUser     *string    `json:"login_user"`
	OSDistro      *string    `json:"os_distro"`
	Status        string     `json:"status"`
	Tags          []string   `json:"tags"`
	VersionsURL   string     `json:"versions_url"`
	Artifacts     []Artifact `json:"artifacts"`
}

type Artifact struct {
	Variant          *string           `json:"variant"`
	Version          string            `json:"version"`
	Arch             string            `json:"arch"`
	Format           string            `json:"format"`
	Firmware         string            `json:"firmware"`
	URL              string            `json:"url"`
	SHA256           string            `json:"sha256"`
	SHA512           string            `json:"sha512"`
	Size             int64             `json:"size"`
	VirtualSize      int64             `json:"virtual_size"`
	BundleURL        string            `json:"bundle_url"`
	ManifestURL      string            `json:"manifest_url"`
	BuiltAt          string            `json:"built_at"`
	GlanceProperties map[string]string `json:"glance_properties"`
}

// Versions is <base>/<id>/versions.json: every published version.
type Versions struct {
	Versions []struct {
		Version   string `json:"version"`
		BuiltAt   string `json:"built_at"`
		Artifacts []struct {
			Variant     *string `json:"variant"`
			Filename    string  `json:"filename"`
			SHA256      string  `json:"sha256"`
			SHA512      string  `json:"sha512"`
			Size        int64   `json:"size"`
			VirtualSize int64   `json:"virtual_size"`
			Firmware    string  `json:"firmware"`
			Arch        string  `json:"arch"`
			Path        string  `json:"path"`
			Manifest    string  `json:"manifest"`
		} `json:"artifacts"`
	} `json:"versions"`
}

// Client fetches the index once per provider instance.
type Client struct {
	IndexURL  string
	UserAgent string
	HTTP      *http.Client

	once  sync.Once
	index *Index
	err   error
}

func NewClient(indexURL, userAgent string) *Client {
	return &Client{IndexURL: indexURL, UserAgent: userAgent, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("GET %s: invalid JSON: %w", url, err)
	}
	return nil
}

// Index returns the catalog index, fetched once.
func (c *Client) Index(ctx context.Context) (*Index, error) {
	c.once.Do(func() {
		var idx Index
		if err := c.getJSON(ctx, c.IndexURL, &idx); err != nil {
			c.err = err
			return
		}
		if idx.Schema != "oic-index/v1" {
			c.err = fmt.Errorf("%s: unsupported schema %q (this provider reads oic-index/v1)", c.IndexURL, idx.Schema)
			return
		}
		c.index = &idx
	})
	return c.index, c.err
}

func (idx *Index) Find(id string) (*Image, error) {
	for i := range idx.Images {
		if idx.Images[i].ID == id {
			return &idx.Images[i], nil
		}
	}
	ids := make([]string, 0, len(idx.Images))
	for _, im := range idx.Images {
		ids = append(ids, im.ID)
	}
	sort.Strings(ids)
	return nil, fmt.Errorf("no image %q in the catalog; available: %s", id, strings.Join(ids, ", "))
}

func variantName(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// SelectArtifact picks the latest artifact of an image by variant and/or
// firmware; with neither, the first one (the catalog's primary artifact).
func SelectArtifact(im *Image, variant, firmware string) (*Artifact, error) {
	var avail []string
	for i := range im.Artifacts {
		a := &im.Artifacts[i]
		avail = append(avail, fmt.Sprintf("%s/%s", orDash(variantName(a.Variant)), a.Firmware))
		if variant != "" && variantName(a.Variant) != variant {
			continue
		}
		if firmware != "" && a.Firmware != firmware {
			continue
		}
		return a, nil
	}
	return nil, fmt.Errorf("image %q has no artifact for variant=%q firmware=%q; available (variant/firmware): %s",
		im.ID, variant, firmware, strings.Join(avail, ", "))
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ArtifactAt returns the artifact of an older version from versions.json,
// shaped like an index artifact. Glance properties are those of the current
// catalog entry for the same variant, with the version-specific oic_* keys.
func (c *Client) ArtifactAt(ctx context.Context, idx *Index, im *Image, version, variant, firmware string) (*Artifact, error) {
	current, err := SelectArtifact(im, variant, firmware)
	if err != nil {
		return nil, err
	}
	if version == "" || version == current.Version {
		return current, nil
	}
	url := im.VersionsURL
	if url == "" {
		url = strings.TrimRight(idx.BaseURL, "/") + "/" + im.ID + "/versions.json"
	}
	var vs Versions
	if err := c.getJSON(ctx, url, &vs); err != nil {
		return nil, err
	}
	var known []string
	for _, v := range vs.Versions {
		known = append(known, v.Version)
		if v.Version != version {
			continue
		}
		for _, a := range v.Artifacts {
			if variant != "" && variantName(a.Variant) != variant {
				continue
			}
			if firmware != "" && a.Firmware != firmware {
				continue
			}
			base := strings.TrimRight(idx.BaseURL, "/") + "/" + im.ID + "/"
			props := map[string]string{}
			for k, val := range current.GlanceProperties {
				props[k] = val
			}
			props["oic_version"] = version
			props["oic_sha256"] = a.SHA256
			return &Artifact{
				Variant:          a.Variant,
				Version:          version,
				Arch:             a.Arch,
				Format:           current.Format,
				Firmware:         a.Firmware,
				URL:              base + a.Path,
				SHA256:           a.SHA256,
				SHA512:           a.SHA512,
				Size:             a.Size,
				VirtualSize:      a.VirtualSize,
				BundleURL:        base + a.Path + ".bundle",
				ManifestURL:      base + a.Manifest,
				BuiltAt:          v.BuiltAt,
				GlanceProperties: props,
			}, nil
		}
		return nil, fmt.Errorf("version %s of %q has no artifact for variant=%q firmware=%q", version, im.ID, variant, firmware)
	}
	return nil, fmt.Errorf("no version %q of %q; published: %s", version, im.ID, strings.Join(known, ", "))
}

// MinDiskGB is the virtual size rounded up to whole GiB (Glance min_disk).
func MinDiskGB(virtualSize int64) int64 {
	const gib = 1 << 30
	if virtualSize <= 0 {
		return 0
	}
	return (virtualSize + gib - 1) / gib
}
