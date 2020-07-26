#!/bin/bash
diff -u sample.{old,new}.txt > sample.diff
git diff --no-index sample.{old,new}.txt > sample.git.diff
diff -u nonewline.{old,new}.txt > nonewline.diff
diff -u nonewline2.{old,new}.txt > nonewline2.diff
diff -u nonewline3.{old,new}.txt > nonewline3.diff
diff -u empty.txt empty.txt > empty.diff
git diff --no-index /dev/null empty.txt > empty_new.diff
git diff --no-index empty.txt /dev/null  > empty_deleted.diff
git diff --no-index /dev/null "empty space.txt" > empty_space.diff
git diff --no-index golint.{old,new}.go > golint.diff
git diff --no-index empty.txt /dev/null > newline_and_empty_deleted.diff
git diff --no-index golint.{old,new}.go >> newline_and_empty_deleted.diff
git -c core.quotepath=true diff --no-index 日本語.{old,new}.txt > 日本語.diff
gofmt -d gofmt.go > gofmt.diff
