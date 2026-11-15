package cross_win

import (
	"fmt"
	"os/exec"
)

func ClearFirewallRules(exePath string) {
	cmds := [][]string{
		{"advfirewall", "firewall", "delete", "rule", "name=all", "program=" + exePath},
		{"advfirewall", "firewall", "delete", "rule", "name=wire-pod"},
	}
	for _, args := range cmds {
		out, err := exec.Command("netsh", args...).CombinedOutput()
		if err != nil {
			fmt.Printf("netsh %v: %v (probably just no matching rules): %s\n", args, err, out)
		}
	}
}
