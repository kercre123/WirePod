package main

import (
	"github.com/kercre123/WirePod/cross/podapp"
	cross_win "github.com/kercre123/WirePod/cross/win"
)

func main() {
	podapp.StartWirePod(cross_win.NewWindows())
}
