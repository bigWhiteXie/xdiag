# Makefile for xdiag project
#
# Targets:
#   build    Build the project executable

build:
	go build -o xdiag ./main.go
	chmod +x ./xdiag

.PHONY: build