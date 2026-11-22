package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func pathOf(name string) path.Path { return path.Root(name) }

var _ datasource.DataSourceWithConfigure = &imagesDataSource{}

type imagesDataSource struct{ client *Client }

type imagesModel struct {
	Tag             types.String   `tfsdk:"tag"`
	OSDistro        types.String   `tfsdk:"os_distro"`
	IncludeArchived types.Bool     `tfsdk:"include_archived"`
	IDs             types.List     `tfsdk:"ids"`
	Images          []imageSummary `tfsdk:"images"`
}

type imageSummary struct {
	ID            types.String `tfsdk:"id"`
	Title         types.String `tfsdk:"title"`
	LatestVersion types.String `tfsdk:"latest_version"`
	Status        types.String `tfsdk:"status"`
	OSDistro      types.String `tfsdk:"os_distro"`
	LoginUser     types.String `tfsdk:"login_user"`
	Tags          types.List   `tfsdk:"tags"`
}

func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: desc}
	}
	resp.Schema = schema.Schema{
		Description: "The images of the openimages.cloud catalog, optionally filtered; use the ids with openimages_image.",
		Attributes: map[string]schema.Attribute{
			"tag":              schema.StringAttribute{Optional: true, Description: "Keep images carrying this tag (e.g. `kubernetes`, `cloud-init`, `octavia`)."},
			"os_distro":        schema.StringAttribute{Optional: true, Description: "Keep images with this OpenStack os_distro (e.g. `ubuntu`, `rocky`)."},
			"include_archived": schema.BoolAttribute{Optional: true, Description: "Also list archived images (end of life upstream). Default false."},
			"ids":              schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Catalog ids of the images kept."},
			"images": schema.ListNestedAttribute{
				Computed: true, Description: "The images kept.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":             s("Catalog id."),
					"title":          s("Human name."),
					"latest_version": s("Latest published version."),
					"status":         s("`stable`, `archive` or `preview`."),
					"os_distro":      s("OpenStack os_distro (empty when none)."),
					"login_user":     s("Default login user (empty when none)."),
					"tags":           schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Tags."},
				}},
			},
		},
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// FilterImages applies the data source filters (exported for tests).
func FilterImages(images []Image, tag, osDistro string, includeArchived bool) []Image {
	var out []Image
	for _, im := range images {
		if !includeArchived && im.Status == "archive" {
			continue
		}
		if tag != "" && !slices.Contains(im.Tags, tag) {
			continue
		}
		if osDistro != "" && (im.OSDistro == nil || *im.OSDistro != osDistro) {
			continue
		}
		out = append(out, im)
	}
	return out
}

func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m imagesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idx, err := d.client.Index(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the openimages.cloud index", err.Error())
		return
	}
	kept := FilterImages(idx.Images, m.Tag.ValueString(), m.OSDistro.ValueString(), m.IncludeArchived.ValueBool())
	ids := make([]string, 0, len(kept))
	m.Images = make([]imageSummary, 0, len(kept))
	for _, im := range kept {
		ids = append(ids, im.ID)
		tags, diags := types.ListValueFrom(ctx, types.StringType, im.Tags)
		resp.Diagnostics.Append(diags...)
		distro, login := "", ""
		if im.OSDistro != nil {
			distro = *im.OSDistro
		}
		if im.LoginUser != nil {
			login = *im.LoginUser
		}
		m.Images = append(m.Images, imageSummary{
			ID: types.StringValue(im.ID), Title: types.StringValue(im.Title),
			LatestVersion: types.StringValue(im.LatestVersion), Status: types.StringValue(im.Status),
			OSDistro: types.StringValue(distro), LoginUser: types.StringValue(login), Tags: tags,
		})
	}
	idList, diags := types.ListValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	m.IDs = idList
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
