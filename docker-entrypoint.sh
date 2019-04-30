#!/usr/bin/env bash

set -e

sed -i -E 's/<base href=.*>/<base href="\'$CLOUDINFO_BASEPATH'\/">/' /web/dist/ui/index.html

echo "Set basepath to $CLOUDINFO_BASEPATH"

exec $@
