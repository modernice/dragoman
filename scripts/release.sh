#!/bin/sh

ROOT=$(git rev-parse --show-toplevel)
VERSION=$1

if [ -z "$VERSION" ]; then
	echo "Usage: $0 <version>"
	exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
	echo "Tag '$VERSION' already exists"
	exit 1
fi

echo "$VERSION" > "$ROOT/version.txt"

git add "$ROOT/version.txt"
git commit -m "$VERSION"
git tag "$VERSION"
git push origin main
git push origin "$VERSION"
