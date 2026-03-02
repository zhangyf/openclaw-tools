package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("./war-briefing-detailed")
	output, _ := cmd.Output()
	
	lines := strings.Split(string(output), "\n")
	
	for i, line := range lines {
		fmt.Printf("%3d: %s\n", i, line)
		if strings.Contains(line, "cat ") || strings.Contains(line, "文件") {
			fmt.Printf("     ^ 匹配\n")
		}
	}
}