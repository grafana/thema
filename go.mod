module github.com/grafana/thema

go 1.19

// contains openapi encoder fixes. remove ASAP
replace cuelang.org/go => github.com/sdboyer/cue v0.5.0-beta.1.0.20221218101942-2ebe38f6cfb1

require (
	cuelang.org/go v0.5.0-beta.2
	github.com/deepmap/oapi-codegen v1.10.1
	github.com/getkin/kin-openapi v0.103.0
	github.com/google/go-cmp v0.5.8
	github.com/grafana/cuetsy v0.1.1
	github.com/spf13/cobra v1.4.0
	github.com/stretchr/testify v1.7.1
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yalue/merged_fs v1.2.2
	golang.org/x/mod v0.7.0
	golang.org/x/tools v0.1.12
)

require (
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/proto v1.10.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/invopop/yaml v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/echo/v4 v4.7.2 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20220428173112-74888fd59c2b // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	golang.org/x/crypto v0.0.0-20220513210258-46612604a0f9 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
