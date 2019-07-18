tools:
	go build -mod vendor -o bin/wof-s3-sync cmd/wof-s3-sync/main.go
	go build -mod vendor -o bin/wof-s3-delete cmd/wof-s3-delete/main.go

lambda-delete:
	@make self
	if test -f main; then rm -f main; fi
	if test -f s3-delete.zip; then rm -f s3-delete.zip; fi
	GOOS=linux go build -mod vendor -o main cmd/wof-s3-delete.go
	zip s3-delete.zip main
	rm -f main
