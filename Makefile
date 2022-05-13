PROJECT_NAME := "edge"

.PHONY: all build clean 

all: build

build:
	@chmod +x build.sh && ./build.sh

clean:
	@rm api/pb/* bin/*
