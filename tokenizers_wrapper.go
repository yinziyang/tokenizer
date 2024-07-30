package tokenizer

// TODO packaging: how do we build the rust lib for distribution?

/*
#cgo LDFLAGS: -ltokenizers -ldl -lm -lstdc++
#include <stdlib.h>
#include "tokenizers.h"
*/
import "C"

// NOTE: There should be NO space between the comments and the `import "C"` line.
import (
	"io"
	"unsafe"
)

type Tokenizer struct {
	tokenizer unsafe.Pointer
}

type tokenizerOpts struct {
	encodeSpecialTokens C.bool
}

type TokenizerOption func(to *tokenizerOpts)

func WithEncodeSpecialTokens() TokenizerOption {
	return func(to *tokenizerOpts) {
		to.encodeSpecialTokens = C.bool(true)
	}
}

type TruncationDirection int

const (
	TruncationDirectionLeft TruncationDirection = iota
	TruncationDirectionRight
)

var _ io.Closer = (*Tokenizer)(nil)

func FromBytes(data []byte, opts ...TokenizerOption) (*Tokenizer, error) {
	allOpts := &tokenizerOpts{
		// by default, we do not encode special tokens
		encodeSpecialTokens: C.bool(false),
	}
	for _, opt := range opts {
		opt(allOpts)
	}
	tokenizer := C.from_bytes((*C.uchar)(unsafe.Pointer(&data[0])), C.uint(len(data)), (*C.struct_TokenizerOptions)(unsafe.Pointer(allOpts)))
	return &Tokenizer{tokenizer: tokenizer}, nil
}

func FromBytesWithTruncation(data []byte, maxLen uint32, dir TruncationDirection) (*Tokenizer, error) {
	tokenizer := C.from_bytes_with_truncation((*C.uchar)(unsafe.Pointer(&data[0])), C.uint(len(data)), C.uint(maxLen), C.uchar(dir))
	return &Tokenizer{tokenizer: tokenizer}, nil
}

func FromFile(path string) (*Tokenizer, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	tokenizer, err := C.from_file(cPath)
	if err != nil {
		return nil, err
	}
	return &Tokenizer{tokenizer: tokenizer}, nil
}

func (t *Tokenizer) Close() error {
	C.free_tokenizer(t.tokenizer)
	t.tokenizer = nil
	return nil
}

type Encoding struct {
	IDs               []uint32
	TypeIDs           []uint32
	SpecialTokensMask []uint32
	AttentionMask     []uint32
	Tokens            []string
}

type encodeOpts struct {
	AddSpecialTokens C.bool

	ReturnTypeIDs           C.bool
	ReturnTokens            C.bool
	ReturnSpecialTokensMask C.bool
	ReturnAttentionMask     C.bool
}

type EncodeOption func(eo *encodeOpts)

func uintVecToSlice(arrPtr *C.uint, len int) []uint32 {
	arr := unsafe.Slice(arrPtr, len)
	slice := make([]uint32, len)
	for i, v := range arr {
		slice[i] = uint32(v)
	}
	return slice
}

func (t *Tokenizer) Encode(str string, maxLen int, needPad bool, addSpecialTokens bool) ([]uint32, []string) {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	options := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
		ReturnTokens:     C.bool(true),
	}
	res := C.encode(t.tokenizer, cStr, (*C.struct_EncodeOptions)(unsafe.Pointer(&options)))
	resLen := int(res.len)
	if resLen == 0 {
		return nil, nil
	}
	defer C.free_buffer(res)

	ids := uintVecToSlice(res.ids, resLen)

	var tokens []string
	if res.tokens != nil {
		tokens = make([]string, resLen)
		for i, s := range (*[1 << 30]*C.char)(unsafe.Pointer(res.tokens))[:resLen:resLen] {
			tokens[i] = C.GoString(s)
		}
	}
	var finalIds []uint32
	var finalTokens []string
	var tokenLen = len(tokens)
	if tokenLen > maxLen {
		if addSpecialTokens {
			finalIds = append(finalIds, ids[0])
			finalIds = append(finalIds, ids[1:maxLen-1]...)
			finalIds = append(finalIds, ids[tokenLen-1])

			finalTokens = append(finalTokens, tokens[0])
			finalTokens = append(finalTokens, tokens[1:maxLen-1]...)
			finalTokens = append(finalTokens, tokens[tokenLen-1])
		} else {
			finalIds = append(finalIds, ids[:maxLen]...)
			finalTokens = append(finalTokens, tokens[:maxLen]...)
		}
	} else if tokenLen < maxLen && needPad {
		finalIds = append(finalIds, ids...)
		finalTokens = append(finalTokens, tokens...)
		for len(finalTokens) < maxLen {
			finalIds = append(finalIds, 0)
			finalTokens = append(finalTokens, "[PAD]")
		}
	} else {
		finalIds = append(finalIds, ids...)
		finalTokens = append(finalTokens, tokens...)
	}

	return finalIds, finalTokens
}

func WithReturnAllAttributes() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTypeIDs = C.bool(true)
		eo.ReturnSpecialTokensMask = C.bool(true)
		eo.ReturnAttentionMask = C.bool(true)
		eo.ReturnTokens = C.bool(true)
	}
}

func WithReturnTypeIDs() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTypeIDs = C.bool(true)
	}
}

func WithReturnSpecialTokensMask() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnSpecialTokensMask = C.bool(true)
	}
}

func WithReturnTokens() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTokens = C.bool(true)
	}
}

func WithReturnAttentionMask() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnAttentionMask = C.bool(true)
	}
}

func (t *Tokenizer) EncodeWithOptions(str string, maxLen int, needPad bool, addSpecialTokens bool, opts ...EncodeOption) Encoding {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))

	encOptions := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
	}
	for _, opt := range opts {
		opt(&encOptions)
	}

	res := C.encode(t.tokenizer, cStr, (*C.struct_EncodeOptions)(unsafe.Pointer(&encOptions)))
	resLen := int(res.len)
	if resLen == 0 {
		return Encoding{}
	}
	defer C.free_buffer(res)

	encoding := Encoding{}
	encoding.IDs = uintVecToSlice(res.ids, resLen)

	if encOptions.ReturnTypeIDs && res.type_ids != nil {
		encoding.TypeIDs = uintVecToSlice(res.type_ids, resLen)
	}

	if encOptions.ReturnTokens && res.tokens != nil {
		tokens := make([]string, resLen)
		for i, s := range (*[1 << 30]*C.char)(unsafe.Pointer(res.tokens))[:resLen:resLen] {
			tokens[i] = C.GoString(s)
		}
		encoding.Tokens = tokens
	}

	if encOptions.ReturnSpecialTokensMask && res.special_tokens_mask != nil {
		encoding.SpecialTokensMask = uintVecToSlice(res.special_tokens_mask, resLen)
	}

	if encOptions.ReturnAttentionMask && res.attention_mask != nil {
		encoding.AttentionMask = uintVecToSlice(res.attention_mask, resLen)
	}

	var finalIds []uint32
	var finalTokens []string
	var finalSpecialTokensMask []uint32
	var finalAttentionMask []uint32
	var finalTypeIds []uint32

	var ids = encoding.IDs
	var tokens = encoding.Tokens
	var specialTokensMask = encoding.SpecialTokensMask
	var attentionMask = encoding.AttentionMask
	var typeIds = encoding.TypeIDs

	var tokenLen = len(tokens)
	if tokenLen > maxLen {
		if addSpecialTokens {
			finalIds = append(finalIds, ids[0])
			finalIds = append(finalIds, ids[1:maxLen-1]...)
			finalIds = append(finalIds, ids[tokenLen-1])

			finalTokens = append(finalTokens, tokens[0])
			finalTokens = append(finalTokens, tokens[1:maxLen-1]...)
			finalTokens = append(finalTokens, tokens[tokenLen-1])

			finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask[0])
			finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask[1:maxLen-1]...)
			finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask[tokenLen-1])

			finalAttentionMask = append(finalAttentionMask, attentionMask[:maxLen]...)

			finalTypeIds = append(finalTypeIds, typeIds[:maxLen]...)
		} else {
			finalIds = append(finalIds, ids[:maxLen]...)
			finalTokens = append(finalTokens, tokens[:maxLen]...)
			finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask[:maxLen]...)
			finalAttentionMask = append(finalAttentionMask, attentionMask[:maxLen]...)
			finalTypeIds = append(finalTypeIds, typeIds[:maxLen]...)
		}

	} else if tokenLen < maxLen && needPad {
		finalIds = append(finalIds, ids...)
		finalTokens = append(finalTokens, tokens...)
		finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask...)
		finalAttentionMask = append(finalAttentionMask, attentionMask...)
		finalTypeIds = append(finalTypeIds, typeIds...)
		for len(finalTokens) < maxLen {
			finalIds = append(finalIds, 0)
			finalTokens = append(finalTokens, "[PAD]")
			finalSpecialTokensMask = append(finalSpecialTokensMask, 1)
			finalAttentionMask = append(finalAttentionMask, 0)
			finalTypeIds = append(finalTypeIds, 0)
		}
	} else {
		finalIds = append(finalIds, ids...)
		finalTokens = append(finalTokens, tokens...)
		finalSpecialTokensMask = append(finalSpecialTokensMask, specialTokensMask...)
		finalAttentionMask = append(finalAttentionMask, attentionMask...)
		finalTypeIds = append(finalTypeIds, typeIds...)
	}

	encoding.IDs = finalIds
	encoding.Tokens = finalTokens
	encoding.SpecialTokensMask = finalSpecialTokensMask
	encoding.AttentionMask = finalAttentionMask
	encoding.TypeIDs = finalTypeIds

	return encoding
}

func (t *Tokenizer) Decode(tokenIDs []uint32, skipSpecialTokens bool) string {
	if len(tokenIDs) == 0 {
		return ""
	}
	len := C.uint(len(tokenIDs))
	res := C.decode(t.tokenizer, (*C.uint)(unsafe.Pointer(&tokenIDs[0])), len, C.bool(skipSpecialTokens))
	defer C.free_string(res)
	return C.GoString(res)
}

func (t *Tokenizer) VocabSize() uint32 {
	return uint32(C.vocab_size(t.tokenizer))
}
