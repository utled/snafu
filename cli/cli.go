package cli

import (
	"bufio"
	"fmt"
	"os"
	"icu/initial"
	"icu/maintain"
	"icu/setup"
	"strings"
)

func Main() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		arguments := strings.Split(strings.TrimSpace(input), " ")
		switch arguments[0] {
		case "test":
			//test.Main()
		case "setup":
			err := setup.Main()
			if err != nil {
				fmt.Println(err)
			}
		case "fullscan":
			initial.StartInitialScan()
		case "sync":
			maintain.Start()
		default:
			fmt.Println(arguments)
		}
	}

}
