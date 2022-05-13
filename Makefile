PROJECT_NAME := "edge"

.PHONY: all gen build clean 

all: gen build

gen: 
	@cd api/ && chmod +x make_pb.sh bin/* && ./make_pb.sh && cd ..

build:
	@chmod +x build.sh && ./build.sh

clean:
	@rm api/pb/* bin/*
