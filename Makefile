build:
	mkdir -p functions
	mkdir -p ui/build
	go mod download
	go build -o ./functions/ ./server/...
	cd ui && CI=false npm install && CI=false npm run build

netlify:
	mkdir -p functions
	mkdir -p ui/build
	go mod download
	go install ./...
	cd ui && CI=false npm install && CI=false npm run build

clean:
	rm -f functions/*
	rm -rf ui/build