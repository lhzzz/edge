PROJECT_NAME := "edge"

.PHONY: all gen build clean 

all: build

gen: 
	@chmod +x api/* && ./api/make_pb.sh api/pb api/proto

build:
	@chmod +x build.sh && ./build.sh

clean:
	@rm api/pb/* bin/*
