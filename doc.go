package goskema

// Package goskema provides:
//
// - Type-safe validation and transformation based on Schema/Codec (Parse/Validate/Decode/Encode)
// - A stable error model via Issues (JSON Pointer, code, message)
// - Metadata for Presence collection and preserve-encoding through WithMeta APIs
// - Streaming validation via Source/Stream with duplicate-key/depth/size enforcement
//
// Design policy:
// - Keep only public APIs in the root package; put detailed implementations under internal/.
// - Place DSL under dsl/, codecs under codec/, and the CLI under cmd/goskema.
// - Prefer black-box testing against public APIs.
//
// Typical usage:
//
//  s := buildSchema()
//  v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data))
//  dm, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(data))
//
//  wire, err := someCodec.Encode(ctx, domain)
//  wire2, err := someCodec.EncodePreserving(ctx, dm)
//
