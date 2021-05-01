
#GCFLAGS=
#LDFLAGS=
#-ldflags=${LDFLAGS} -gcflags=all=${GCFLAGS} 

SOURCEFOLDER=./cmd/neoray
EXECUTABLE=bin/neoray.exe

build:
	go build -o ${EXECUTABLE} -race ${SOURCEFOLDER}

run: build
	./${EXECUTABLE}

debug:
	dlv debug ${SOURCEFOLDER}

clean:
	rm ${EXECUTABLE}
