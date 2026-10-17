GO ?= go
PUBLIC = site/public
PORT ?= 8000

.PHONY: site verify serve test clean

## site: generate the static index into site/public (verifies every record on the way)
site:
	$(GO) run ./cmd/index -root . -out $(PUBLIC)

## verify: verify the signature of every record under records/
verify:
	scripts/verify-all.sh

## serve: generate, then preview at http://localhost:$(PORT)/
serve: site
	python3 -m http.server -d $(PUBLIC) $(PORT)

## test: vet and unit tests of the generator
test:
	$(GO) vet ./...
	$(GO) test ./...

clean:
	rm -rf $(PUBLIC)
