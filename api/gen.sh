#!/bin/bash

set -eu

# Generate all protos
buf generate \
    --path networking \
    --path extensions

# Generate CRDs 
cue-gen -verbose -f=./cue.yaml -crd=true
