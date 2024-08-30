#!/bin/bash
set -euxo pipefail
pwd
./scripts/genproto.sh
#go test ./pkg/...
go build github.com/steinarvk/playdough
