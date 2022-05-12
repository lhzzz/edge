#! /bin/bash

go build -o bin/edgelet cmd/edgelet/main.go
go build -o bin/edgectl cmd/edgectl/main.go