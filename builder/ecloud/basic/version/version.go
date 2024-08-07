package version

import (
	"github.com/hashicorp/packer/packer-plugin-sdk/version"
	packerVersion "github.com/hashicorp/packer/version"
)

var EcloudPluginVersion *version.PluginVersion

func init() {
	EcloudPluginVersion = version.InitializePluginVersion(
		packerVersion.Version, packerVersion.VersionPrerelease)
}
