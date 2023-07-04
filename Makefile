.PHONY: gen-release

gen-release:
	./scripts/check_semantic_versioning.sh $(tag)
	git tag $(tag)
	git push origin $(tag)
	goreleaser --rm-dist
