sync:
	go build -o bin/sync bin/sync.go

fmt:
	go fmt src/com.mapzen/whosonfirst/s3/sync.go 
	go fmt bin/sync.go 
