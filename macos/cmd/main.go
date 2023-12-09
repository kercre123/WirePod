package main

import (
	cross_mac "github.com/kercre123/WirePod/cross/mac"
	"github.com/kercre123/WirePod/cross/podapp"
)

func main() {
	podapp.StartWirePod(cross_mac.NewMacOS())
}
