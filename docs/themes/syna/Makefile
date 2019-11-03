SHELL := /bin/bash

TARGET := $(shell echo $${PWD\#\#*/})
.DEFAULT_GOAL: $(TARGET)

VERSION := $(shell cat ./VERSION)
BUILD := `git rev-parse HEAD`


.PHONY: build dep dev

all: dep build

build:
	@yarn run build

dev:
	@yarn run dev

dep:
	@yarn install

version:
	@echo $(VERSION)