# Updating this fork of deepmap/oapi-codegen

The contents of the `oapi-codegen` are copied from the latest `deepmap/oapi-codegen`, with the following changes:
* All packages except `pkg/codegen` and `pkg/util` have been removed, as they are not needed for thema.
* Package refs have been changed from `github.com/deepmap/oapi-codegen` to `github.com/grafana/thema/internal/deepmap/oapi-codegen`
* `TestExamplePetStoreCodeGeneration` and `TestExamplePetStoreCodeGenerationWithUserTemplates` have been removed from `pkg/codegen/codegen_test.go`
  * (this is to remove any need to reference the swagger packages, and said tests do not impact the code being used by thema)
* The contents of [this PR](https://github.com/deepmap/oapi-codegen/pull/717) in deepmap/oapi-codegen have been played onto this fork

A full diff of changes can be found at [diff.txt].

When updating this fork, please add whatever changes you make (if different from the main oapi-codegen branch) to the above list and update [diff.txt] accordingly. 
This can be done with:
```shell
$ diff <path_to_deepmap>/oapi-codegen/pkg/codegen <path_to_grafana>/thema/internal/deepmap/oapi-codegen/pkg/codegen > <path_to_grafana>/thema/internal/deepmap/diff.txt
```
If you make changes to other packages, please also include them in the diff.