CLOUDS 		:= "aws"
LANGUAGES	:= "typescript python go csharp yaml"

.PHONY: build
build: clean
	./scripts/build.sh

.PHONY: clean
clean:
	./scripts/clean.sh

.PHONY: deploy
deploy:
	./scripts/deploy.sh

.PHONY: destroy
destroy:
	./scripts/destroy.sh

.PHONY: test
test:
	./scripts/test.sh
