isWindows :=$(filter Windows_NT, $(OS))

ifeq ($(EXECNAME),)
	EXECNAME=neoray
endif

ifeq ($(OUTDIR),)
	OUTDIR=bin
endif

VERSION=0.2.4

SOURCEFOLDER=./cmd/neoray
SOURCETESTFOLDER=./cmd/neoray/...
PACKAGETESTFOLDER=./pkg/...
GLOWDIR=./pkg/opengl/glow

ifeq ($(OS),Windows_NT)
	EXECPATHDEBUG=.\$(OUTDIR)\$(EXECNAME)_debug.exe
	EXECPATHRELEASE=.\$(OUTDIR)\$(EXECNAME).exe
	RELEASEFLAGS=-ldflags="-H=windowsgui -s -w"
	DELETECOMMAND=del
else
	EXECPATHDEBUG=./$(OUTDIR)/$(EXECNAME)_debug
	EXECPATHRELEASE=./$(OUTDIR)/$(EXECNAME)
	RELEASEFLAGS=-ldflags="-s -w"
	DELETECOMMAND=rm
endif

build:
	go build -tags debug -race -o $(EXECPATHDEBUG) $(SOURCEFOLDER)

run: build
	$(EXECPATHDEBUG) $(ARGS)

generate:
	$(if $(isWindows), cd cmd/neoray/assets && go-winres make --product-version=$(VERSION) --file-version=$(VERSION))
	glow generate -out=./pkg/opengl/gl -api=gl -version=3.3 -profile=core -restrict=$(GLOWDIR)/glfunclist.json -tmpl=$(GLOWDIR)/tmpl -xml=$(GLOWDIR)/xml

release:
	go build $(RELEASEFLAGS) -o $(EXECPATHRELEASE) $(SOURCEFOLDER)

test:
	go test -race $(SOURCETESTFOLDER)
	go test -race $(PACKAGETESTFOLDER)

bench:
	go test -run=XXX -bench=. -benchmem -race $(SOURCETESTFOLDER)
	go test -run=XXX -bench=. -benchmem -race $(PACKAGETESTFOLDER)

debug:
	dlv debug $(SOURCEFOLDER)

clean:
	$(DELETECOMMAND) $(EXECPATHDEBUG)
	$(DELETECOMMAND) $(EXECPATHRELEASE)
