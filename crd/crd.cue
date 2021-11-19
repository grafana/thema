package crd

import (
    "list"
    "strings"
    "github.com/grafana/scuemata"
)

// CRD transforms a lineage into a Kubernetes custom resource definition, or a series thereof.
#CRD: {
    args: {
        _sv: [<len(lin.seqs), <len(lin.seqs[_sv[0]].schemas)]
        served: [..._sv]
        storage: _sv
        lin: scuemata.#Lineage
    }

    // Additional metadata necessary to convert a scuemata lineage into a
    // Kubernetes Custom Resource Definition (CRD).
    spec: {
		// scope indicates whether the defined custom resource is cluster-
		// or namespace-scoped.
        scope: "Namespaced" | "Cluster"

		// group is the API group of the defined custom resource. The
		// custom resources are served under `/apis/<group>/...`.
        // Ex.: stable.example.com
        group: string

        names: {
            // categories is a list of grouped resources this custom resource
            // belongs to (e.g. 'all'). This is published in API discovery
            // documents, and used by clients to support invocations like
            // `kubectl get all`.
            categories?: [...string]

            // kind is the serialized kind of the resource. It is normally
            // CamelCase and singular. Custom resource instances will use
            // this value as the `kind` attribute in API calls.
            // TODO default this to scuemata name
            kind: string

            // listKind is the serialized kind of the list for this resource.
            listKind: string | *"\(kind)List"

            // plural is the plural name of the resource to serve. The custom
            // resources are served under
            // `/apis/<group>/<version>/.../<plural>`.
            plural: string | =~ #"[a-z]"#

            // shortNames allow shorter string to match your resource on the CLI
            shortNames?: [...string]

            // singular is the singular name of the resource. It must be all
            // lowercase.
            singular: string | *strings.ToLower(kind)
        }
        // Deprecated upstream, so omitted
        // preserveUnknownFields: bool | *false

		// conversion defines conversion settings for the CRD.
        conversion?: {
            // TODO for now, only allow this, because what we really want to do
            // is swap scuemata translation logic in for Scheme
            strategy: "None"
        }
    }

    // The lineage, transformed into a valid CRD.
    crd: {
        apiVersion: "apiextensions.k8s.io/v1"
        kind: "CustomResourceDefinition"
        metadata: {
            name: "\(spec.names.plural).\(spec.group)"
        }
        spec: spec
        spec: versions: {
            for seqv, seq in args.lin.seqs {
                for schv, sch in seq.schemas {
                    served: list.Contains(args.served, [seqv, schv])
                    storage: [seqv, schv] == args.storage
                    name: "v\(seqv).\(schv)" // Not sure if the dot is allowed
                    schema: {
                        openAPIV3Schema: {...} // This is what needs to be filled in by the encoder
                        cueSchema: sch
                    }
                }
            }
        }
    }

    // The lineage, reformed in the shape of a custom resource validator.
    crv: {
        // TODO
    }
}

