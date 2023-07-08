#!/bin/sh

ROOT=$(git rev-parse --show-toplevel)
VERSION=$1

if [ -z "$VERSION" ]; then
	echo "Usage: $0 <version>"
	exit 1
fi

echo "$VERSION" > "$ROOT/cmd/dragoman/version.txt"

git add "$ROOT/cmd/dragoman/version.txt"
git commit -m "$VERSION"
git tag "$VERSION"
git push "$VERSION"
