.PHONY: build
build:
	$(MAKE) -C serverless build

.PHONY: clean
clean:
	$(MAKE) -C serverless clean

.PHONY: test
test:
	$(MAKE) -C serverless test

.PHONY: test-one
test-one:
	$(MAKE) -C serverless test-one

.PHONY: next-steps
next-steps:
	$(MAKE) -C serverless next-steps

.PHONY: copy
copy:
	$(MAKE) -C serverless copy
