#!/bin/bash
#
# Append linter results at the bottom of commit message to check results on
# commit.
echo -e "# ------------------------ >8 ------------------------" >> $1
echo -e "# reviewdog results" >> $1
golangci-lint run --out-format=line-number --fast ./... | reviewdog -f=golangci-lint -diff="git diff @" >> $1
