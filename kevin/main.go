package main

import (
	"fmt"
	"time"
)

func main() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop() // Don't forget to stop the ticker when done

	for {
		select {
		case <-ticker.C:
			fmt.Println("Tick at", time.Now())
		}
	}
}
