#!/bin/bash
go mod tidy
go build -o k8sctl main.go
