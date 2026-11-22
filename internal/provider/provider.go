// Package provider implements the openimages Terraform/OpenTofu provider:
// read-only data sources over the openimages.cloud catalog index.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &openimagesProvider{}

type openimagesProvider struct {
	version string
}

type providerModel struct {
	IndexURL types.String `tfsdk:"index_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &openimagesProvider{version: version} }
}

func (p *openimagesProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openimages"
	resp.Version = p.version
}

func (p *openimagesProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read-only data sources over the openimages.cloud catalog: the URL, checksums and Glance properties of signed, boot-tested cloud images, to chain into openstack_images_image_v2 or a Proxmox download.",
		Attributes: map[string]schema.Attribute{
			"index_url": schema.StringAttribute{
				Optional:    true,
				Description: "Catalog index URL (oic-index/v1). Defaults to $OPENIMAGES_INDEX_URL, else " + DefaultIndexURL + ". Point it at an internal mirror (oic-sync mirror, Ansible role oic_mirror) when the cloud cannot reach the Internet.",
			},
		},
	}
}

func (p *openimagesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	url := DefaultIndexURL
	if v := os.Getenv("OPENIMAGES_INDEX_URL"); v != "" {
		url = v
	}
	if !cfg.IndexURL.IsNull() && !cfg.IndexURL.IsUnknown() && cfg.IndexURL.ValueString() != "" {
		url = cfg.IndexURL.ValueString()
	}
	client := NewClient(url, "terraform-provider-openimages/"+p.version)
	resp.DataSourceData = client
}

func (p *openimagesProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{NewImageDataSource, NewImagesDataSource}
}

func (p *openimagesProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
