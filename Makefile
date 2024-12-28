all: build

build:
	mkdir -p bin
	go build -o bin/ ./...

release:
	go build -o . -ldflags '-s -w' ./...

clean:
	rm -rf bin
