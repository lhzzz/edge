package main

import (
	"edge/internal/edgelet"
)

const (
	cloudAddress = "http://150.158.25.86"
	runAddress   = ":50051"
)

func main() {
	edgelet.Run(cloudAddress, runAddress)
}
