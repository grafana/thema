package kernel

// A TypeFactory must emit a pointer to the Go type that a kernel will
// ultimately produce as output.
//
// TODO the function accomplished by this should be trivial to achieve with generics...?
type TypeFactory func() interface{}
