#!/usr/bin/env bash
set -e

git fetch --tags
new_version=$((1 + $(git tags | ggrep -P "v\d+" | sed 's/v//g' | sort -n | tail -n 1)))
git add -A
git commit --allow-empty -S -m "Version to v$new_version"
git tag "v$new_version"
git push origin
git push --tags


docker build -t howinator/personal-blog .
docker tag howinator/personal-blog:latest "howinator/personal-blog:v$new_version"
docker push "howinator/personal-blog:v$new_version"

echo "New version at v$new_version"
