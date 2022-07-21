SOURCEFOLDER=./cmd/neoray
DEBUGEXE=bin/neoray_debug
RELEASEEXE=bin/neoray
DELETECOMMAND=rm
RELEASEFLAGS=
PRECOMMANDS=

ifeq ($(OS),Windows_NT)
	DEBUGEXE=bin\neoray_debug.exe
	RELEASEEXE=bin\neoray.exe
	DELETECOMMAND=del
	RELEASEFLAGS=-ldflags -H=windowsgui
	PRECOMMANDS=cd src\assets && go-winres make && cd ..\..
endif

build:
	go build -tags debug -race -o $(DEBUGEXE) $(SOURCEFOLDER)

run: build
	./$(DEBUGEXE) $(ARGS)

precommands:
	$(PRECOMMANDS)

release: precommands
	go build $(RELEASEFLAGS) -o $(RELEASEEXE) $(SOURCEFOLDER)

test:
	go test -race $(SOURCEFOLDER)/...

bench:
	go test -run=XXX -bench=. -benchmem -race $(SOURCEFOLDER)/...

debug:
	dlv debug $(SOURCEFOLDER)

clean:
	$(DELETECOMMAND) $(DEBUGEXE)
	$(DELETECOMMAND) $(RELEASEEXE)
