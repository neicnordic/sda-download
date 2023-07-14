#!/bin/bash

for file in ./*
do
    if [[ $file == *.go ]]
    then
        echo "building plugin $file"
        go build -buildmode=plugin -o ${file/go/so} $file
        echo "built plugin ${file/go/so}"
    fi
done
