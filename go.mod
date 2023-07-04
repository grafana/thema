module github.com/grafana/thema

go 1.19

// contains openapi encoder fixes. commented out because impact for
// most thema CLI uses is minimal, and others (e.g. grafana/grafana)
// can hoist this themselves if they need it
replace cuelang.org/go => github.com/sdboyer/cue v0.5.0-beta.2.0.20221218111347-341999f48bdb

require (
	cuelang.org/go v0.5.0
	github.com/cockroachdb/errors v1.9.1
	github.com/dave/dst v0.27.2
	github.com/getkin/kin-openapi v0.115.0
	github.com/golangci/lint-1 v0.0.0-20181222135242-d2cdd8c08219
	github.com/google/go-cmp v0.5.8
	github.com/grafana/cuetsy v0.1.8
	github.com/labstack/echo/v4 v4.9.1
	github.com/matryer/moq v0.2.7
	github.com/spf13/cobra v1.4.0
	github.com/stretchr/testify v1.8.1
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yalue/merged_fs v1.2.2
	golang.org/x/mod v0.7.0
	golang.org/x/text v0.4.0
	golang.org/x/tools v0.3.0
)

require (
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/proto v1.10.0 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/invopop/yaml v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/gommon v0.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/perimeterx/marshmallow v1.1.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20220428173112-74888fd59c2b // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
