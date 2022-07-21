
ifeq ($(OS),Windows_NT)
	SOURCEFOLDER=.\cmd\neoray
	SOURCETESTFOLDER=.\cmd\neoray\...
	PACKAGETESTFOLDER=.\pkg\...
	DEBUGEXE=.\bin\neoray_debug.exe
	RELEASEEXE=.\bin\neoray.exe
	DELETECOMMAND=del
	RELEASEFLAGS=-ldflags -H=windowsgui
	PRECOMMANDS=cd cmd\neoray\internal\assets && go-winres make && cd ..\..\..\..
else
	SOURCEFOLDER=./cmd/neoray
	SOURCETESTFOLDER=./cmd/neoray/...
	PACKAGETESTFOLDER=./pkg/...
	DEBUGEXE=./bin/neoray_debug
	RELEASEEXE=./bin/neoray
	DELETECOMMAND=rm
	RELEASEFLAGS=
	PRECOMMANDS=
endif

build:
	go build -tags debug -race -o $(DEBUGEXE) $(SOURCEFOLDER)

run: build
	$(DEBUGEXE) $(ARGS)

precommands:
	$(PRECOMMANDS)

release: precommands
	go build $(RELEASEFLAGS) -o $(RELEASEEXE) $(SOURCEFOLDER)

test:
	go test -race $(SOURCETESTFOLDER)
	go test -race $(PACKAGETESTFOLDER)

bench:
	go test -run=XXX -bench=. -benchmem -race $(SOURCETESTFOLDER)
	go test -run=XXX -bench=. -benchmem -race $(PACKAGETESTFOLDER)

debug:
	dlv debug $(SOURCEFOLDER)

clean:
	$(DELETECOMMAND) $(DEBUGEXE)
	$(DELETECOMMAND) $(RELEASEEXE)
