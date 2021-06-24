ifeq ($(OS),Windows_NT)
	DEBUGEXE=bin/neoray_debug.exe
	RELEASEEXE=bin/neoray.exe
	RELEASEFLAGS=-ldflags -H=windowsgui
else
	DEBUGEXE=bin/neoray_debug
	RELEASEEXE=bin/neoray
	RELEASEFLAGS=
endif

build:
	go build -tags debug -o $(DEBUGEXE) -race

run: build
	./$(DEBUGEXE) $(ARGS)

release:
	go build $(RELEASEFLAGS) -o $(RELEASEEXE)

release-run: release
	./$(RELEASEEXE) $(ARGS)

test:
	go test -race

bench:
	go test -run=XXX -bench=. -race

debug:
	dlv debug

clean:
	-rm $(DEBUGEXE)
	-rm $(RELEASEEXE)
