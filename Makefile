SOURCEFOLDER=./cmd/neoray
EXECUTABLE=bin/neoray.exe

build:
	go build -o ${EXECUTABLE} -race ${SOURCEFOLDER}

run: build
	./${EXECUTABLE}

test:
	go test -race ${SOURCEFOLDER}

debug:
	dlv debug ${SOURCEFOLDER}

clean:
	-rm ${EXECUTABLE}
