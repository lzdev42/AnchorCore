//go:build !android

package tailscale

import "github.com/sagernet/sing-box/experimental/anchorcore/platform"

func setAndroidProtectFunc(platformInterface platform.Interface) {
}
