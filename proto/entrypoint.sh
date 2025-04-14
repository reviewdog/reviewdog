#!/bin/bash
protoc --proto_path=./rdf --go_out=./rdf --go_opt=paths=source_relative --jsonschema_out=./rdf/jsonschema ./rdf/reviewdog.proto
protoc --proto_path=./metacomment --go_out=./metacomment --go_opt=paths=source_relative ./metacomment/metacomment.proto
