# Copyright (c) Microsoft Corporation.

export GO111MODULE=on

ifeq ($(OS),Windows_NT)
	EXEEXT=.exe
else
	EXEEXT=
endif

all: mockcloudagent

.PHONY: mockcloudagent
mockcloudagent:
	go build -o bin/mockcloudagent$(EXEEXT) github.com/microsoft/cluster-api-provider-azurestackhci/test/mockcloudagent/cmd
