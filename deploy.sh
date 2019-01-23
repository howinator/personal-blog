#!/usr/bin/env bash

kubectl -n prod set image deploy/personal-blog nginx="howinator/personal-blog:${1}"
