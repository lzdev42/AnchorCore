//go:build !(darwin || linux)

package anchorcore

import "os"

func getTunnelName(fd int32) (string, error) {
	return "", os.ErrInvalid
}
