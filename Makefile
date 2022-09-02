isWindows :=$(filter Windows_NT, $(OS))

ifeq ($(EXECNAME),)
	EXECNAME=neoray
endif

ifeq ($(OUTDIR),)
	OUTDIR=bin
endif

VERSION    :=0.2.5
SRCDIR     :=./cmd/neoray
SRCTESTDIR :=./cmd/neoray/...
PKGTESTDIR :=./pkg/...
GLOWDIR    :=./pkg/opengl/glow
OUTEXE     :=./$(OUTDIR)/$(EXECNAME)
OUTDBG     :=$(OUTEXE)_debug
LDFLAGS    :=-s -w

ifeq ($(OS),Windows_NT)
	OUTDBG  :=$(OUTDBG).exe
	OUTEXE  :=$(OUTEXE).exe
	LDFLAGS :=$(LDFLAGS) -H=windowsgui
endif

build:
	go build -tags debug -race -o $(OUTDBG) $(SRCDIR)

run: build
	$(OUTDBG) $(ARGS)

generate:
	$(if $(isWindows), cd cmd/neoray/assets && go-winres make --product-version=$(VERSION) --file-version=$(VERSION))
	glow generate -out=./pkg/opengl/gl -api=gl -version=3.3 -profile=core -restrict=$(GLOWDIR)/glfunclist.json -tmpl=$(GLOWDIR)/tmpl -xml=$(GLOWDIR)/xml

release:
	go build -ldflags="$(LDFLAGS)" -o $(OUTEXE) $(SRCDIR)

test:
	go test -race $(SRCTESTDIR)
	go test -race $(PKGTESTDIR)

bench:
	go test -run=XXX -bench=. -benchmem -race $(SRCTESTDIR)
	go test -run=XXX -bench=. -benchmem -race $(PKGTESTDIR)

debug:
	dlv debug $(SRCDIR)

all: generate test bench build release
