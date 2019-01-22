#!/usr/bin/env bash
set -e

git fetch --tags
new_version=$((1 + $(git tags | ggrep -P "v\d+" | sed 's/v//g' | sort -n | tail -n 1)))
git add -A
git commit -S -m "Version to v$new_version"
git tag "v$new_version"
docker push origin
docker push --tags


docker build -t howinator/personal-blog .
docker tag howinator/personal-blog "howinator/personal-blog:$new_version"
docker push "howinator/personal-blog:$new_version"
