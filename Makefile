build:
	mkdir -p functions
	mkdir -p ui/build
	go mod download
	go build -o ./functions/ ./server/...
	cd ui && npm install && npm run build

netlify:
	mkdir -p functions
	mkdir -p ui/build
	go mod download
	go install ./...
	cd ui && npm install && npm run build

clean:
	rm -f functions/*
	rm -rf ui/build