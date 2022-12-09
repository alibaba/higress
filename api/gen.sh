#!/bin/bash

set -eu

# Generate all protos
buf generate \
  --path networking \

# Generate CRDs 
cue-gen -verbose -f=./cue.yaml -crd=true
