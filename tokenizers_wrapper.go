package tokenizer

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L./ -ltokenizers -ldl -lm -lstdc++ -O3
#include <stdlib.h>
#include "tokenizers.h"
*/
import "C"

// NOTE: There should be NO space between the comments and the `import "C"` line.
import (
	"io"
	"unsafe"
)

var maxKeepLength = 20 * 512

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

func (e *Encoding) Reset() {
	e.AttentionMask = e.AttentionMask[:0]
	e.IDs = e.IDs[:0]
	e.SpecialTokensMask = e.SpecialTokensMask[:0]
	e.Tokens = e.Tokens[:0]
	e.TypeIDs = e.TypeIDs[:0]
}

type encodeOpts struct {
	AddSpecialTokens C.bool

	ReturnTypeIDs           C.bool
	ReturnTokens            C.bool
	ReturnSpecialTokensMask C.bool
	ReturnAttentionMask     C.bool
}

type EncodeOption func(eo *encodeOpts)

func uintVecToSlice(arrPtr *C.uint, length int) []uint32 {
	if arrPtr == nil || length <= 0 {
		return nil
	}
	arr := unsafe.Slice(arrPtr, length)
	slice := make([]uint32, length)
	for i, v := range arr {
		slice[i] = uint32(v)
	}
	return slice
}

func stringVecToSlice(arrPtr *C.char, length int) []string {
	if arrPtr == nil || length <= 0 {
		return nil
	}
	arr := unsafe.Slice((**C.char)(unsafe.Pointer(arrPtr)), length)
	slice := make([]string, length)
	for i, v := range arr {
		if v != nil {
			slice[i] = C.GoString(v)
		} else {
			slice[i] = ""
		}
	}

	return slice
}

func (t *Tokenizer) Encode(str string, addSpecialTokens bool) ([]uint32, []string) {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	options := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
		ReturnTokens:     C.bool(true),
	}
	res := C.encode(t.tokenizer, cStr, (*C.struct_EncodeOptions)(unsafe.Pointer(&options)))
	defer C.free_buffer(res)
	length := int(res.len)
	if length == 0 {
		return nil, nil
	}

	ids := uintVecToSlice(res.ids, length)

	var tokens []string
	if res.tokens != nil {
		tokens = stringVecToSlice(res.tokens, length)
	}
	return ids, tokens
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

func (t *Tokenizer) EncodeWithOptions(str string, addSpecialTokens bool, opts ...EncodeOption) *Encoding {
	if len(str) > maxKeepLength {
		str = t.TruncateString(str, uint(maxKeepLength))
	}
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))

	encOptions := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
	}
	for _, opt := range opts {
		opt(&encOptions)
	}

	res := C.encode(t.tokenizer, cStr, (*C.struct_EncodeOptions)(unsafe.Pointer(&encOptions)))
	defer C.free_buffer(res)
	length := int(res.len)
	if length == 0 {
		return &Encoding{}
	}

	encoding := Encoding{}
	encoding.IDs = uintVecToSlice(res.ids, length)

	if encOptions.ReturnTypeIDs && res.type_ids != nil {
		encoding.TypeIDs = uintVecToSlice(res.type_ids, length)
	}

	if encOptions.ReturnTokens && res.tokens != nil {
		encoding.Tokens = stringVecToSlice(res.tokens, length)
	}

	if encOptions.ReturnSpecialTokensMask && res.special_tokens_mask != nil {
		encoding.SpecialTokensMask = uintVecToSlice(res.special_tokens_mask, length)
	}

	if encOptions.ReturnAttentionMask && res.attention_mask != nil {
		encoding.AttentionMask = uintVecToSlice(res.attention_mask, length)
	}

	return &encoding
}

func (t *Tokenizer) Decode(tokenIDs []uint32, skipSpecialTokens bool) string {
	if len(tokenIDs) == 0 {
		return ""
	}
	length := C.uint(len(tokenIDs))
	res := C.decode(t.tokenizer, (*C.uint)(unsafe.Pointer(&tokenIDs[0])), length, C.bool(skipSpecialTokens))
	defer C.free_string(res)
	return C.GoString(res)
}

func (t *Tokenizer) VocabSize() uint32 {
	return uint32(C.vocab_size(t.tokenizer))
}

func (t *Tokenizer) TruncateString(message string, keepsize uint) string {
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	cResult := C.truncate_string(cMessage, C.uint(keepsize))
	defer C.free_string(cResult)
	return C.GoString(cResult)
}
