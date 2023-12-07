.PHONY: build clean test

BINARY="docker-scan"

build:
	@GOARCH=amd64 go build -o ${BINARY}

clean:
	@if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

rebuild:
	@make clean
	@make build

test:
	@make build
	@./${BINARY}
	@make clean