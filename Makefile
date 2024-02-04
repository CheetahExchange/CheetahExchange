GOCC=go
GOFLAGS=-ldflags '-w -s'


all: spot-core spot-rest spot-pushing
.PHONY: all

clean: spot-core-clean spot-rest-clean spot-pushing-clean
.PHONY: clean

# spot-core
spot-core: spot-core-clean
	$(GOCC) build $(GOFLAGS) ./cmd/spot-core
.PHONY: spot-core

spot-core-clean:
	rm -f spot-core
.PHONY: spot-core-clean

# spot-rest
spot-rest: spot-rest-clean
	$(GOCC) build $(GOFLAGS) ./cmd/spot-rest
.PHONY: spot-rest

spot-rest-clean:
	rm -f spot-rest
.PHONY: spot-rest-clean

# spot-pushing
spot-pushing: spot-pushing-clean
	$(GOCC) build $(GOFLAGS) ./cmd/spot-pushing
.PHONY: spot-pushing

spot-pushing-clean:
	rm -f spot-pushing
.PHONY: spot-pushing-clean


