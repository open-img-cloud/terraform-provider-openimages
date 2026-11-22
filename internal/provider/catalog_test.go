package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// fixture serves testdata/index.json (base_url rewritten to the test server)
// and testdata/debian-13-versions.json.
func fixture(t *testing.T) (*httptest.Server, *Client) {
	t.Helper()
	idx, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	vers, err := os.ReadFile("testdata/debian-13-versions.json")
	if err != nil {
		t.Fatal(err)
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "terraform-provider-openimages/") {
			t.Errorf("unexpected User-Agent %q", ua)
		}
		switch r.URL.Path {
		case "/index.json":
			body := strings.ReplaceAll(string(idx), "https://images.openimages.cloud", srv.URL)
			w.Write([]byte(body))
		case "/debian-13/versions.json":
			w.Write(vers)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, NewClient(srv.URL+"/index.json", "terraform-provider-openimages/test")
}

func TestFindAndSelect(t *testing.T) {
	_, c := fixture(t)
	idx, err := c.Index(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Find("nope"); err == nil || !strings.Contains(err.Error(), "debian-13") {
		t.Fatalf("unknown id should list the available ones, got %v", err)
	}
	alpine, _ := idx.Find("alpine-linux")
	a, err := SelectArtifact(alpine, "", "")
	if err != nil || variantName(a.Variant) != "bios" {
		t.Fatalf("default artifact: %v %v", a, err)
	}
	a, err = SelectArtifact(alpine, "", "uefi")
	if err != nil || a.Firmware != "uefi" {
		t.Fatalf("by firmware: %v %v", a, err)
	}
	if _, err := SelectArtifact(alpine, "musl", ""); err == nil || !strings.Contains(err.Error(), "bios/bios") {
		t.Fatalf("unknown variant should list the available ones, got %v", err)
	}
	if a.GlanceProperties["oic_id"] != "alpine-linux" || a.SHA256 == "" || !strings.HasSuffix(a.URL, ".qcow2") {
		t.Fatalf("artifact fields: %+v", a)
	}
}

func TestOlderVersion(t *testing.T) {
	srv, c := fixture(t)
	ctx := context.Background()
	idx, _ := c.Index(ctx)
	deb, _ := idx.Find("debian-13")
	var vs Versions
	raw, _ := os.ReadFile("testdata/debian-13-versions.json")
	if err := json.Unmarshal(raw, &vs); err != nil || len(vs.Versions) < 2 {
		t.Skip("fixture has a single version")
	}
	old := vs.Versions[1]
	a, err := c.ArtifactAt(ctx, idx, deb, old.Version, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Version != old.Version || a.SHA256 != old.Artifacts[0].SHA256 {
		t.Fatalf("older version fields: %+v", a)
	}
	if !strings.HasPrefix(a.URL, srv.URL+"/debian-13/"+old.Version+"/") || !strings.HasSuffix(a.BundleURL, ".qcow2.bundle") {
		t.Fatalf("older version URLs: %s %s", a.URL, a.BundleURL)
	}
	if a.GlanceProperties["oic_version"] != old.Version || a.GlanceProperties["oic_sha256"] != a.SHA256 {
		t.Fatalf("oic_* keys must follow the version: %v", a.GlanceProperties)
	}
	cur, _ := SelectArtifact(deb, "", "")
	if cur.GlanceProperties["oic_version"] == old.Version {
		t.Fatal("selecting an older version must not modify the cached index")
	}
	if _, err := c.ArtifactAt(ctx, idx, deb, "0.0", "", ""); err == nil || !strings.Contains(err.Error(), old.Version) {
		t.Fatalf("unknown version should list the published ones, got %v", err)
	}
}

func TestFilterAndMinDisk(t *testing.T) {
	_, c := fixture(t)
	idx, _ := c.Index(context.Background())
	ids := func(ims []Image) string {
		var s []string
		for _, im := range ims {
			s = append(s, im.ID)
		}
		return strings.Join(s, ",")
	}
	if got := ids(FilterImages(idx.Images, "", "", false)); strings.Contains(got, "amazon-linux-2,") || strings.HasSuffix(got, "amazon-linux-2") {
		t.Fatalf("archived images are hidden by default: %s", got)
	}
	if got := ids(FilterImages(idx.Images, "", "", true)); !strings.Contains(got, "amazon-linux-2") {
		t.Fatalf("include_archived: %s", got)
	}
	if got := ids(FilterImages(idx.Images, "kubernetes", "", false)); got != "talos-linux" {
		t.Fatalf("by tag: %s", got)
	}
	if got := ids(FilterImages(idx.Images, "", "debian", false)); got != "debian-13" {
		t.Fatalf("by os_distro: %s", got)
	}
	if MinDiskGB(3221225472) != 3 || MinDiskGB(3221225473) != 4 || MinDiskGB(0) != 0 {
		t.Fatal("MinDiskGB rounding")
	}
}

func TestBadSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"schema":"other/v9","images":[]}`))
	}))
	defer srv.Close()
	_, err := NewClient(srv.URL, "terraform-provider-openimages/test").Index(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unsupported schema") {
		t.Fatalf("want unsupported schema, got %v", err)
	}
}
