.PHONY: build
build:
	$(MAKE) -C static-website build

.PHONY: clean
clean:
	$(MAKE) -C static-website clean

.PHONY: test
test:
	$(MAKE) -C static-website test

.PHONY: test-one
test-one:
	$(MAKE) -C static-website test-one

.PHONY: copy
copy:
	$(MAKE) -C static-website copy
