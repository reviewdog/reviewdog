#!/bin/bash
protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --jsonschema_out=./jsonschema ./reviewdog.proto 
