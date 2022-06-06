PROJECT_NAME := "edge"

.PHONY: all gen build image clean 

all: build

gen: 
	@chmod +x api/* && ./api/make_pb.sh api/pb api/proto

build:
	@chmod +x build.sh && ./build.sh

image:
	@chmod +x image_build.sh && ./image_build.sh

clean:
	@rm -rf api/pb/* bin/*
