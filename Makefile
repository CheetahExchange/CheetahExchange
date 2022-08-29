GOCC=go
GOFLAGS=-ldflags '-w -s'

###############################

all: bill_executor matching_engine matching_log_maker fill_executor pushing_server rest_server
.PHONY: all

clean: bill_executor_clean matching_engine_clean matching_log_maker_clean \
	fill_executor_clean pushing_server_clean rest_server_clean
.PHONY: clean

legacy: gitbitex_spot
.PHONY: legacy

legacy_clean: gitbitex_spot_clean
.PHONY: legacy_clean

###############################

bill_executor: bill_executor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/bill_executor
.PHONY: bill_executor

bill_executor_clean:
	rm -f bill_executor
.PHONY: bill_executor_clean

matching_engine: matching_engine_clean
	$(GOCC) build $(GOFLAGS) ./cmd/matching_engine
.PHONY: matching_engine

matching_engine_clean:
	rm -f matching_engine
.PHONY: matching_engine_clean

matching_log_maker: matching_log_maker_clean
	$(GOCC) build $(GOFLAGS) ./cmd/matching_log_maker
.PHONY: matching_log_maker

matching_log_maker_clean:
	rm -f matching_log_maker
.PHONY: matching_log_maker_clean

fill_executor: fill_executor_clean
	$(GOCC) build $(GOFLAGS) ./cmd/fill_executor
.PHONY: fill_executor

fill_executor_clean:
	rm -f fill_executor
.PHONY: fill_executor_clean

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




