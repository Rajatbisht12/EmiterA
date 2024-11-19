build:
	mkdir -p functions
	go mod download
	go build -o ./functions/ ./server/...

netlify:
	mkdir -p functions
	go mod download
	go install ./...

clean:
	rm -f functions/*