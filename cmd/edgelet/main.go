package main

import (
	"edge/internal/edgelet"
)

const (
	runAddress = ":10250"
)

func main() {
	edgelet.Run(runAddress)
}
