PROJECT_NAME := "edge"

.PHONY: all gen build image clean 

all: build

gen: 
	@chmod +x api/edge-proto/* && ./api/edge-proto/make_pb.sh api/edge-proto/pb api/edge-proto/proto

build:
	@chmod +x build.sh && ./build.sh

image:
	@chmod +x image_build.sh && ./image_build.sh

ci_image:
	@chmod +x image_build.sh && ./image_build.sh cicd

clean:
	@rm -rf api/edge-proto/pb/* bin/*
