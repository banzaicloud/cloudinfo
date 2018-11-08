#!/usr/bin/env bash

sed -i -E 's/<base href=.*>/<base href="\'$PRODUCTINFO_BASEPATH'\/">/' /web/dist/ui/index.html

echo "Set basepath to $PRODUCTINFO_BASEPATH"

exec /bin/productinfo $@