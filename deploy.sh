#!/usr/bin/env bash

k -n prod set image deploy personal-blog="howinator/personal-blog:v${1}"
