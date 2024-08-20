#!/bin/bash
set -euxo pipefail
pwd
protoc --go_out=. --go_opt=paths=source_relative proto/pdpb/playdough.proto
go test ./pkg/...
go build github.com/steinarvk/playdough/cmd/playdough
