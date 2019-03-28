#!/usr/bin/env bash

go generate ./... && go tool vet -all -shadow=true ./ && go build
