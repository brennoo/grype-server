#!/bin/bash

set -euo pipefail

OAPI_CODEGEN=$(go env GOPATH)/bin/oapi-codegen

rm -rf server/ client/
mkdir -p server/ client/

$OAPI_CODEGEN --config server-gen.yaml openapi.yaml
$OAPI_CODEGEN --config client-gen.yaml openapi.yaml
