package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &imageDataSource{}

type imageDataSource struct{ client *Client }

type imageModel struct {
	ID       types.String `tfsdk:"id"`
	Variant  types.String `tfsdk:"variant"`
	Firmware types.String `tfsdk:"firmware"`
	Version  types.String `tfsdk:"version"`

	Title            types.String `tfsdk:"title"`
	Status           types.String `tfsdk:"status"`
	LoginUser        types.String `tfsdk:"login_user"`
	Arch             types.String `tfsdk:"arch"`
	DiskFormat       types.String `tfsdk:"disk_format"`
	URL              types.String `tfsdk:"url"`
	SHA256           types.String `tfsdk:"sha256"`
	SHA512           types.String `tfsdk:"sha512"`
	Size             types.Int64  `tfsdk:"size"`
	VirtualSize      types.Int64  `tfsdk:"virtual_size"`
	MinDiskGB        types.Int64  `tfsdk:"min_disk_gb"`
	BundleURL        types.String `tfsdk:"bundle_url"`
	ManifestURL      types.String `tfsdk:"manifest_url"`
	BuiltAt          types.String `tfsdk:"built_at"`
	Name             types.String `tfsdk:"name"`
	GlanceProperties types.Map    `tfsdk:"glance_properties"`
	OptionalProps    types.Map    `tfsdk:"optional_glance_properties"`
	EndOfLife        types.Bool   `tfsdk:"end_of_life"`
	EndOfLifeSoon    types.Bool   `tfsdk:"end_of_life_soon"`
	SecurityUntil    types.String `tfsdk:"security_until"`
	SupportURL       types.String `tfsdk:"support_url"`
	CosignIdentity   types.String `tfsdk:"cosign_certificate_identity_regexp"`
	CosignIssuer     types.String `tfsdk:"cosign_certificate_oidc_issuer"`
}

func NewImageDataSource() datasource.DataSource { return &imageDataSource{} }

func (d *imageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *imageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computed := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: desc}
	}
	resp.Schema = schema.Schema{
		Description: "One image of the openimages.cloud catalog: download URL, checksums and Glance properties of the latest version (or of `version`).",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Required: true, Description: "Catalog id, e.g. `debian-13`, `rocky-linux-9`, `talos-linux`."},
			"variant":  schema.StringAttribute{Optional: true, Computed: true, Description: "Variant for images that have several (e.g. `bios`/`uefi` for Alpine, `glibc`/`musl` for Alpaquita). Default: the catalog's first artifact."},
			"firmware": schema.StringAttribute{Optional: true, Computed: true, Description: "`bios` or `uefi`, to pick by firmware instead of variant."},
			"version":  schema.StringAttribute{Optional: true, Computed: true, Description: "Version to use; default: the latest. Older versions come from the image's versions.json."},

			"title":        computed("Human name, e.g. `Debian 13`."),
			"status":       computed("`stable`, `archive` or `preview`."),
			"login_user":   computed("Default login user (empty for images without one, e.g. Talos)."),
			"arch":         computed("Architecture, e.g. `x86_64`."),
			"disk_format":  computed("Disk format for Glance (`qcow2`)."),
			"url":          computed("Versioned, immutable download URL (`web_download` / `image_source_url` / Proxmox `download_file`)."),
			"sha256":       computed("SHA-256 of the file."),
			"sha512":       computed("SHA-512 of the file (Glance os_hash_value)."),
			"size":         schema.Int64Attribute{Computed: true, Description: "File size in bytes."},
			"virtual_size": schema.Int64Attribute{Computed: true, Description: "Disk size seen by the guest, in bytes."},
			"min_disk_gb":  schema.Int64Attribute{Computed: true, Description: "Virtual size rounded up to GiB, for Glance `min_disk_gb`."},
			"bundle_url":   computed("Sigstore bundle of the file (cosign verify-blob --bundle)."),
			"manifest_url": computed("MANIFEST.json of this version (commit, CI run, builder digest)."),
			"built_at":     computed("Build time, RFC 3339."),
			"name":         computed("Stable Glance name, the same oic-sync gives: the title, plus the variant in capitals when the image has several (`Alpine Linux (UEFI)`)."),
			"optional_glance_properties": schema.MapAttribute{
				Computed: true, ElementType: types.StringType,
				Description: "Properties the catalog recommends where the cloud supports them (e.g. hw_rng_model); merge them into `properties` if yours does.",
			},
			"end_of_life":      schema.BoolAttribute{Computed: true, Description: "The upstream release is past its end of life (endoflife.date dates published in the index)."},
			"end_of_life_soon": schema.BoolAttribute{Computed: true, Description: "The upstream release reaches its end of life soon."},
			"security_until":   computed("End of upstream security support, YYYY-MM-DD (empty when unknown)."),
			"support_url":      computed("Source of the support dates (endoflife.date page)."),
			"glance_properties": schema.MapAttribute{
				Computed: true, ElementType: types.StringType,
				Description: "Glance properties of the catalog (os_distro, hw_qemu_guest_agent, hw_disk_bus...) plus the oic_* keys oic-sync sets, for `properties` of openstack_images_image_v2.",
			},
			"cosign_certificate_identity_regexp": computed("Identity to pass to cosign verify-blob."),
			"cosign_certificate_oidc_issuer":     computed("OIDC issuer to pass to cosign verify-blob."),
		},
	}
}

func (d *imageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *imageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m imageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idx, err := d.client.Index(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the openimages.cloud index", err.Error())
		return
	}
	im, err := idx.Find(m.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(pathOf("id"), "Unknown image", err.Error())
		return
	}
	a, err := d.client.ArtifactAt(ctx, idx, im, m.Version.ValueString(), m.Variant.ValueString(), m.Firmware.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("No matching artifact", err.Error())
		return
	}
	props, diags := types.MapValueFrom(ctx, types.StringType, Properties(a))
	resp.Diagnostics.Append(diags...)
	optional := im.OptionalGlanceProperties
	if optional == nil {
		optional = map[string]string{}
	}
	opt, diags := types.MapValueFrom(ctx, types.StringType, optional)
	resp.Diagnostics.Append(diags...)
	m.OptionalProps = opt
	m.Name = types.StringValue(im.Name(a))
	eol, soon, until, src := false, false, "", ""
	if im.Support != nil {
		eol, soon = im.Support.IsEOL, im.Support.EOLSoon && !im.Support.IsEOL
		if im.Support.SecurityUntil != nil {
			until = *im.Support.SecurityUntil
		}
		if im.Support.Source != nil {
			src = *im.Support.Source
		}
	}
	m.EndOfLife = types.BoolValue(eol)
	m.EndOfLifeSoon = types.BoolValue(soon)
	m.SecurityUntil = types.StringValue(until)
	m.SupportURL = types.StringValue(src)

	m.Version = types.StringValue(a.Version)
	if a.Variant != nil {
		m.Variant = types.StringValue(*a.Variant)
	}
	m.Firmware = types.StringValue(a.Firmware)
	m.Title = types.StringValue(im.Title)
	m.Status = types.StringValue(im.Status)
	login := ""
	if im.LoginUser != nil {
		login = *im.LoginUser
	}
	m.LoginUser = types.StringValue(login)
	m.Arch = types.StringValue(a.Arch)
	m.DiskFormat = types.StringValue(a.Format)
	m.URL = types.StringValue(a.URL)
	m.SHA256 = types.StringValue(a.SHA256)
	m.SHA512 = types.StringValue(a.SHA512)
	m.Size = types.Int64Value(a.Size)
	m.VirtualSize = types.Int64Value(a.VirtualSize)
	m.MinDiskGB = types.Int64Value(MinDiskGB(a.VirtualSize))
	m.BundleURL = types.StringValue(a.BundleURL)
	m.ManifestURL = types.StringValue(a.ManifestURL)
	m.BuiltAt = types.StringValue(a.BuiltAt)
	m.GlanceProperties = props
	m.CosignIdentity = types.StringValue(idx.Signing.CertificateIdentityRegexp)
	m.CosignIssuer = types.StringValue(idx.Signing.CertificateOIDCIssuer)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
