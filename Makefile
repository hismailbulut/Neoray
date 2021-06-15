SOURCEFOLDER=./cmd/neoray
DEBUGEXE=bin/neoray_debug.exe
RELEASEEXE=bin/neoray.exe

build:
	go build -o $(DEBUGEXE) -race $(SOURCEFOLDER)

run: build
	./$(DEBUGEXE) $(ARGS)

release:
	go build -tags release -ldflags -H=windowsgui -o $(RELEASEEXE) $(SOURCEFOLDER)

runr: release
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
