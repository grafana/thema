# Release a new version
This process uses [GoReleaser](https://goreleaser.com/install/) to generate the release.

1. Execute `make gen-release tag=your_tag`.
2. To be consistent, all tags should include `v` prefix (ex. v0.1.1). Otherwise, the process couldn't start.
3. When the tag is created, GoReleaser will do the rest of the work for you.
