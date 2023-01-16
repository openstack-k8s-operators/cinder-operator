#!/usr/bin/env bash
LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)
echo Cleaning go.work
echo -e "go 1.18\n\nuse (\n\t.\n\t./api\n)" > $LOCATION/../go.work
