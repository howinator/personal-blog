#!/usr/bin/env bash
set -e
hugo

git fetch --tags
new_version=v$((1 + $(git tags | ggrep -P "v\d+" | sed 's/v//g' | sort -n | tail -n 1)))
git add -A
git commit --allow-empty -S -m "Version to $new_version"
git tag "$new_version"
git push origin
git push --tags


docker build -t howinator/personal-blog .
docker tag howinator/personal-blog:latest "howinator/personal-blog:$new_version"
docker push "howinator/personal-blog:$new_version"

echo "New version at $new_version"

./deploy.sh "$new_version"
