build:
	$(MAKE) -C static-website build

clean:
	$(MAKE) -C static-website clean

deploy:
	$(MAKE) -C static-website deploy

destroy:
	$(MAKE) -C static-website destroy

test:
	$(MAKE) -C static-website test
