AMATL_VERSION := 0.30.0
AMATL_BIN ?= tools/amatl-$(AMATL_VERSION)/bin/amatl

misc/communication/booklet/%.pdf: $(AMATL_BIN)
	mkdir -p dist/communication
	$(MAKE) run-with-env CMD='$(AMATL_BIN) render --config misc/communication/amatl/booklet.yml  pdf --template-left-delimiter '{%' --template-right-delimiter '%}' -o dist/communication/booklet-$*.pdf misc/communication/booklet/$*.md'

tools/amatl-$(AMATL_VERSION)/bin/amatl:
	mkdir -p tools/amatl-$(AMATL_VERSION)/bin
	curl -kL --output tools/amatl-$(AMATL_VERSION)/amatl.tar.gz https://github.com/Bornholm/amatl/releases/download/v$(AMATL_VERSION)/amatl_$(AMATL_VERSION)_linux_amd64.tar.gz
	( cd tools/amatl-$(AMATL_VERSION) && tar -xzf amatl.tar.gz amatl )
	mv tools/amatl-$(AMATL_VERSION)/amatl tools/amatl-$(AMATL_VERSION)/bin/
	rm -f tools/amatl-$(AMATL_VERSION)/amatl.tar.gz