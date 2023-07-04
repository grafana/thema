.PHONY: gen-release

gen-release:
	git tag $(tag)
	git push origin $(tag)
	goreleaser --rm-dist
