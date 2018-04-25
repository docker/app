// Copyright 2013 Miek Gieben. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs11

/*
#include <stdlib.h>
#include <string.h>
#include "pkcs11go.h"
*/
import "C"
import "unsafe"

// GCMParams represents the parameters for the AES-GCM mechanism.
type GCMParams struct {
	IV      []byte
	AAD     []byte
	TagSize int
}

// NewGCMParams returns a pointer to the AES-GCM parameters.
// This is a convenience function for passing GCM parameters to
// available mechanisms
func NewGCMParams(iv, aad []byte, tagSize int) *GCMParams {
	return &GCMParams{
		IV:      iv,
		AAD:     aad,
		TagSize: tagSize,
	}
}

func cGCMParams(p *GCMParams) (arena, []byte) {
	params := C.CK_GCM_PARAMS{
		ulTagBits: C.CK_ULONG(p.TagSize),
	}
	var arena arena
	if len(p.IV) > 0 {
		iv, ivLen := arena.Allocate(p.IV)
		params.pIv = C.CK_BYTE_PTR(iv)
		params.ulIvLen = ivLen
	}
	if len(p.AAD) > 0 {
		aad, aadLen := arena.Allocate(p.AAD)
		params.pAAD = C.CK_BYTE_PTR(aad)
		params.ulAADLen = aadLen
	}
	return arena, C.GoBytes(unsafe.Pointer(&params), C.int(unsafe.Sizeof(params)))
}
