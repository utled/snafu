package maintain

import (
	"fmt"
	"time"
)

const (
	longScope  = "/home/utled/GolandProjects/"
	shortScope = "/home/utled/GolandProjects/snafu"
)

func Start() error {
	scanCount := 10
	for scanCount > 0 {
		var startPath string
		var err error

		if scanCount%5 == 0 {
			startPath = longScope
		} else {
			startPath = shortScope
		}

		fmt.Printf("Starting scan of: %s\n", startPath)
		startTime := time.Now()
		err = orchestrateScan(startPath)
		if err != nil {
			return err
		}
		fmt.Println("Scan completed")
		elapsed := time.Since(startTime)
		fmt.Printf("Scan took %s\n", elapsed)

		time.Sleep(1 * time.Second)

		if scanCount == 1 {
			scanCount = 10
		} else {
			scanCount--
		}

	}

	return nil
}
