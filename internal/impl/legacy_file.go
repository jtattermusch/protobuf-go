// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"

	// TODO: Avoid reliance on old API. However, there is currently a
	// chicken and egg problem where we need the descriptor protos to implement
	// the new API.
	protoV1 "github.com/golang/protobuf/proto"
	descriptorV1 "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// Every enum and message type generated by protoc-gen-go since commit 2fc053c5
// on February 25th, 2016 has had a method to get the raw descriptor.
// Types that were not generated by protoc-gen-go or were generated prior
// to that version are not supported.
//
// The []byte returned is the encoded form of a FileDescriptorProto message
// compressed using GZIP. The []int is the path from the top-level file
// to the specific message or enum declaration.
type (
	legacyEnum interface {
		EnumDescriptor() ([]byte, []int)
	}
	legacyMessage interface {
		Descriptor() ([]byte, []int)
	}
)

var fileDescCache sync.Map // map[*byte]*descriptorV1.FileDescriptorProto

// loadFileDesc unmarshals b as a compressed FileDescriptorProto message.
//
// This assumes that b is immutable and that b does not refer to part of a
// concatenated series of GZIP files (which would require shenanigans that
// rely on the concatenation properties of both protobufs and GZIP).
// File descriptors generated by protoc-gen-go do not rely on that property.
func loadFileDesc(b []byte) *descriptorV1.FileDescriptorProto {
	// Fast-path: check whether we already have a cached file descriptor.
	if v, ok := fileDescCache.Load(&b[0]); ok {
		return v.(*descriptorV1.FileDescriptorProto)
	}

	// Slow-path: decompress and unmarshal the file descriptor proto.
	m := new(descriptorV1.FileDescriptorProto)
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	b, err = ioutil.ReadAll(zr)
	if err != nil {
		panic(err)
	}
	// TODO: What about extensions?
	// The protoV1 API does not eagerly unmarshal extensions.
	if err := protoV1.Unmarshal(b, m); err != nil {
		panic(err)
	}
	fileDescCache.Store(&b[0], m)
	return m
}