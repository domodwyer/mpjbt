SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY=			mpjbt
BINARY_LINUX=	$(BINARY)-linux
BINARY_FREEBSD=	$(BINARY)-freebsd

VERSION_DATE=	$(shell date -u '+%Y-%m-%d %H:%M:%S')
VERSION_TAG=	$(shell git describe --tag || echo "unknown-version")

LDFLAGS=-ldflags="-s -w -X \"main.versionTag=$(VERSION_TAG)\" -X \"main.versionDate=$(VERSION_DATE)\""

.DEFAULT_GOAL: $(BINARY)

$(BINARY) osx: $(SOURCES)
	@echo ""
	@echo "	Tagging as $(VERSION_TAG) ($(VERSION_DATE))"
	@echo ""
	go build ${LDFLAGS} -o $(BINARY)

$(BINARY_LINUX) linux: $(SOURCES)
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o $(BINARY_LINUX)

$(BINARY_FREEBSD) freebsd: $(SOURCES)
	GOOS=freebsd GOARCH=amd64 go build ${LDFLAGS} -o $(BINARY_FREEBSD)

.PHONY: all
all: $(BINARY) $(BINARY_LINUX) $(BINARY_FREEBSD)

.PHONY: pack
pack:
	@if [ -a $(BINARY) ] ; \
	then \
	     upx $(BINARY) || echo "$(BINARY) already packed" ; \
	fi;

	@if [ -a $(BINARY_LINUX) ] ; \
	then \
	     upx $(BINARY_LINUX) ; \
	fi;

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -f ${BINARY}-linux ] ; then rm ${BINARY}-linux ; fi
	if [ -f ${BINARY}-freebsd ] ; then rm ${BINARY}-freebsd ; fi
	if [ -f ${BINARY}.tar.bz2 ] ; then rm ${BINARY}.tar.bz2 ; fi

tar: $(SOURCES) all
	if [ -f ${BINARY}.tar.bz2 ] ; then rm ${BINARY}.tar.bz2 ; fi
	tar cfj $(BINARY).tar.bz2 $(BINARY) $(BINARY_LINUX) $(BINARY_FREEBSD) || exit