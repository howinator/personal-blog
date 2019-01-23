#!/usr/bin/env bash

kubectl -n prod set image deploy personal-blog="howinator/personal-blog:v${1}"
