# Using Thema in Programs

In prior articles, we [wrote a `Ship` lineage in CUE](authoring.md), then made it reliably available in Go via a canonical [`LineageFactory`](https://pkg.go.dev/github.com/grafana/thema#LineageFactory) function named `ShipLineage()`. With that done, we're ready to write a program that puts Thema to work doing something useful.

Now, there are lots of kinds of programs that might use Thema. Here are a few:

* Something with a RESTful HTTP API, which needs to schematize the objects it sends and receives
* Something with configuration file(s), which govern program behavior 
* Something with SQL-shaped storage, which needs some kind of DDL/schema to define its tables 
* Something with NoSQL-shaped storage, where the absence of native database schema makes the need for app-level schema even greater
* Something with protobuf endpoints, which are intrinsically schematized but safely evolving them [is hard](https://docs.buf.build/breaking/rules)
* Something that is a backend to a frontend/browser app, and both need a common language for specifying the data they exchange
* Something that acts as a [Kubernetes Operator](https://www.redhat.com/en/topics/containers/what-is-a-kubernetes-operator), where defining evolvable schema ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) is table stakes

Many of these cases have mature solutions. Some are unlikely to ever be reached by Thema, and some uses for Thema aren't represented. But all of these cases share at least one property: whatever the task at hand is, simultaneously juggling schema versions multiplies the task's complexity. Since we'll never get rid of the need to evolve and change our schemas, the best outcome is encapsulating that juggling to a corner of the program, thereby allowing the rest of the program to _pretend_ that only one version exists.

This tutorial will focus on a general approach to encapsulating the problem of receiving input data, validating it, translating it, and make it available for use as a Go struct. We refer to this cluster of behavior as an **Input Kernel**.

## Input Kernel Overview

TODO