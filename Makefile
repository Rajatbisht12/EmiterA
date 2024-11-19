.PHONY: test clean

build:
	mkdir -p functions
	go get ./...
	go build -o ./functions/ ./server/...

netlify:
	mkdir -p functions
	go get ./...
	go install ./...

clean:
	rm -f functions/*