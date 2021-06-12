SOURCEFOLDER=./cmd/neoray
DEBUGEXEC=bin/debug/neoray_debug.exe
RELEASEEXEC=bin/release/neoray_release.exe

build:
	go build -o ${DEBUGEXEC} -race ${SOURCEFOLDER}

run: build
	./${DEBUGEXEC}

release:
	go build -tags release -ldflags -H=windowsgui -o ${RELEASEEXEC} ${SOURCEFOLDER}

test:
	go test -race ${SOURCEFOLDER}

bench:
	go test -run=XXX -bench=. -race ${SOURCEFOLDER}

debug:
	dlv debug ${SOURCEFOLDER}

clean:
	-rm ${DEBUGEXEC}
