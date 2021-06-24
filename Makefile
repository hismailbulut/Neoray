SOURCEFOLDER=./cmd/neoray

ifeq ($(OS),Windows_NT)
	DEBUGEXE=bin/neoray_debug.exe
	RELEASEEXE=bin/neoray.exe
	LDFLAGS=-ldflags -H=windowsgui
else
	DEBUGEXE=bin/neoray_debug
	RELEASEEXE=bin/neoray
	LDFLAGS=
endif

build:
	go build -tags debug -o $(DEBUGEXE) -race $(SOURCEFOLDER)

run: build
	./$(DEBUGEXE) $(ARGS)

release:
	go build $(LDFLAGS) -o $(RELEASEEXE) $(SOURCEFOLDER)

release-run: release
	./$(RELEASEEXE) $(ARGS)

test:
	go test -race $(SOURCEFOLDER)

bench:
	go test -run=XXX -bench=. -race $(SOURCEFOLDER)

debug:
	dlv debug $(SOURCEFOLDER)

clean:
	-rm $(DEBUGEXE)
	-rm $(RELEASEEXE)
