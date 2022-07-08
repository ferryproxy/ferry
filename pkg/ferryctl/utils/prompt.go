package utils

import (
	"fmt"
)

func Prompt(want string, lines ...string) {
	fmt.Printf("# ++++ Please run the following command to %s:\n", want)
	fmt.Printf("# =============================================\n")
	defer fmt.Printf("# =============================================\n")
	for _, line := range lines {
		fmt.Printf("%s\n", line)
	}
}
