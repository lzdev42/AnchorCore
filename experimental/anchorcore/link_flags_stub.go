//go:build !unix

package anchorcore

import (
	"net"
)

func linkFlags(rawFlags uint32) net.Flags {
	panic("stub!")
}
