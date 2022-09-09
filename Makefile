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

.PHONY: next-steps
next-steps:
	$(MAKE) -C static-website next-steps

.PHONY: copy
copy:
	$(MAKE) -C static-website copy
