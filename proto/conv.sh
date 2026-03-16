#!/bin/bash

protoc -I . --go_out=../model --go-grpc_out=../model ./*.proto