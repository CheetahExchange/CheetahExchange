GOCC=go
GOFLAGS=-ldflags '-w -s'

###############################

all: billing_processor binlog_processor matching_engine ordering_processor trading_processor pushing_server rest_server
.PHONY: all

clean: billing_processor_clean binlog_processor_clean matching_engine_clean ordering_processor_clean \
	trading_processor_clean pushing_server_clean rest_server_clean
.PHONY: clean

legacy: gitbitex_spot
.PHONY: legacy

legacy_clean: gitbitex_spot_clean
.PHONY: legacy_clean

###############################

billing_processor: billing_processor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/billing_processor
.PHONY: billing_processor

billing_processor_clean:
	rm -f billing_processor
.PHONY: billing_processor_clean

binlog_processor: binlog_processor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/binlog_processor
.PHONY: binlog_processor

binlog_processor_clean:
	rm -f binlog_processor
.PHONY: binlog_processor_clean

matching_engine: matching_engine_clean
	$(GOCC) build $(GOFLAGS) ./cmd/matching_engine
.PHONY: matching_engine

matching_engine_clean:
	rm -f matching_engine
.PHONY: matching_engine_clean

ordering_processor: ordering_processor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/ordering_processor
.PHONY: ordering_processor

ordering_processor_clean:
	rm -f ordering_processor
.PHONY: ordering_processor_clean

trading_processor: trading_processor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/trading_processor
.PHONY: trading_processor

trading_processor_clean:
	rm -f trading_processor
.PHONY: trading_processor_clean

pushing_server: pushing_server_clean
	$(GOCC) build $(GOFLAGS) ./cmd/pushing_server
.PHONY: pushing_server

pushing_server_clean:
	rm -f pushing_server
.PHONY: pushing_server_clean

rest_server: rest_server_clean
	$(GOCC) build $(GOFLAGS) ./cmd/rest_server
.PHONY: rest_server

rest_server_clean:
	rm -f rest_server
.PHONY: rest_server_clean

###############################

gitbitex_spot: gitbitex_spot_clean
	$(GOCC) build $(GOFLAGS) ./cmd/gitbitex_spot
.PHONY: gitbitex_spot

gitbitex_spot_clean:
	rm -f gitbitex_spot
.PHONY: gitbitex_spot_clean




