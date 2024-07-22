.PHONY: test 
test:
	go test -race ./... -v
	# go test -race ./...
