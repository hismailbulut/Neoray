SOURCEFOLDER=./cmd/neoray

ifeq ($(OS),Windows_NT)
	DEBUGEXE=bin/neoray_debug.exe
	RELEASEEXE=bin/neoray.exe
else
	DEBUGEXE=bin/neoray_debug
	RELEASEEXE=bin/neoray
endif

build:
	go build -o $(DEBUGEXE) -race $(SOURCEFOLDER)

run: build
	./$(DEBUGEXE) $(ARGS)

release:
	ifeq ($(OS),Windows_NT)
		go build -tags release -ldflags -H=windowsgui -o $(RELEASEEXE) $(SOURCEFOLDER)
	else
		go build -tags release -o $(RELEASEEXE) $(SOURCEFOLDER)
	endif

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
