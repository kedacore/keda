// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse
// +build cse

package mongocrypt

// #cgo linux solaris darwin pkg-config: libmongocrypt
// #cgo windows CFLAGS: -I"c:/libmongocrypt/include"
// #cgo windows LDFLAGS: -lmongocrypt -Lc:/libmongocrypt/bin
// #include <mongocrypt.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/mongocrypt/options"
)

type MongoCrypt struct {
	wrapped      *C.mongocrypt_t
	kmsProviders bsoncore.Document
}

// Version returns the version string for the loaded libmongocrypt, or an empty string
// if libmongocrypt was not loaded.
func Version() string {
	str := C.GoString(C.mongocrypt_version(nil))
	return str
}

// NewMongoCrypt constructs a new MongoCrypt instance configured using the provided MongoCryptOptions.
func NewMongoCrypt(opts *options.MongoCryptOptions) (*MongoCrypt, error) {
	// create mongocrypt_t handle
	wrapped := C.mongocrypt_new()
	if wrapped == nil {
		return nil, errors.New("could not create new mongocrypt object")
	}
	crypt := &MongoCrypt{
		wrapped:      wrapped,
		kmsProviders: opts.KmsProviders,
	}

	// set options in mongocrypt
	if err := crypt.setProviderOptions(opts.KmsProviders); err != nil {
		return nil, err
	}
	if err := crypt.setLocalSchemaMap(opts.LocalSchemaMap); err != nil {
		return nil, err
	}
	if err := crypt.setEncryptedFieldsMap(opts.EncryptedFieldsMap); err != nil {
		return nil, err
	}

	if opts.BypassQueryAnalysis {
		C.mongocrypt_setopt_bypass_query_analysis(wrapped)
	}

	// If loading the crypt_shared library isn't disabled, set the default library search path "$SYSTEM"
	// and set a library override path if one was provided.
	if !opts.CryptSharedLibDisabled {
		systemStr := C.CString("$SYSTEM")
		defer C.free(unsafe.Pointer(systemStr))
		C.mongocrypt_setopt_append_crypt_shared_lib_search_path(crypt.wrapped, systemStr)

		if opts.CryptSharedLibOverridePath != "" {
			cryptSharedLibOverridePathStr := C.CString(opts.CryptSharedLibOverridePath)
			defer C.free(unsafe.Pointer(cryptSharedLibOverridePathStr))
			C.mongocrypt_setopt_set_crypt_shared_lib_path_override(crypt.wrapped, cryptSharedLibOverridePathStr)
		}
	}

	C.mongocrypt_setopt_use_need_kms_credentials_state(crypt.wrapped)

	// initialize handle
	if !C.mongocrypt_init(crypt.wrapped) {
		return nil, crypt.createErrorFromStatus()
	}

	return crypt, nil
}

// CreateEncryptionContext creates a Context to use for encryption.
func (m *MongoCrypt) CreateEncryptionContext(db string, cmd bsoncore.Document) (*Context, error) {
	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	cmdBinary := newBinaryFromBytes(cmd)
	defer cmdBinary.close()
	dbStr := C.CString(db)
	defer C.free(unsafe.Pointer(dbStr))

	if ok := C.mongocrypt_ctx_encrypt_init(ctx.wrapped, dbStr, C.int32_t(-1), cmdBinary.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}
	return ctx, nil
}

// CreateDecryptionContext creates a Context to use for decryption.
func (m *MongoCrypt) CreateDecryptionContext(cmd bsoncore.Document) (*Context, error) {
	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	cmdBinary := newBinaryFromBytes(cmd)
	defer cmdBinary.close()

	if ok := C.mongocrypt_ctx_decrypt_init(ctx.wrapped, cmdBinary.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}
	return ctx, nil
}

// lookupString returns a string for the value corresponding to the given key in the document.
// if the key does not exist or the value is not a string, the empty string is returned.
func lookupString(doc bsoncore.Document, key string) string {
	strVal, _ := doc.Lookup(key).StringValueOK()
	return strVal
}

func setAltName(ctx *Context, altName string) error {
	// create document {"keyAltName": keyAltName}
	idx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendStringElement(doc, "keyAltName", altName)
	doc, _ = bsoncore.AppendDocumentEnd(doc, idx)

	keyAltBinary := newBinaryFromBytes(doc)
	defer keyAltBinary.close()

	if ok := C.mongocrypt_ctx_setopt_key_alt_name(ctx.wrapped, keyAltBinary.wrapped); !ok {
		return ctx.createErrorFromStatus()
	}
	return nil
}

func setKeyMaterial(ctx *Context, keyMaterial []byte) error {
	// Create document {"keyMaterial": keyMaterial} using the generic binary sybtype 0x00.
	idx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendBinaryElement(doc, "keyMaterial", 0x00, keyMaterial)
	doc, err := bsoncore.AppendDocumentEnd(doc, idx)
	if err != nil {
		return err
	}

	keyMaterialBinary := newBinaryFromBytes(doc)
	defer keyMaterialBinary.close()

	if ok := C.mongocrypt_ctx_setopt_key_material(ctx.wrapped, keyMaterialBinary.wrapped); !ok {
		return ctx.createErrorFromStatus()
	}
	return nil
}

func rewrapDataKey(ctx *Context, filter []byte) error {
	filterBinary := newBinaryFromBytes(filter)
	defer filterBinary.close()

	if ok := C.mongocrypt_ctx_rewrap_many_datakey_init(ctx.wrapped, filterBinary.wrapped); !ok {
		return ctx.createErrorFromStatus()
	}
	return nil
}

// CreateDataKeyContext creates a Context to use for creating a data key.
func (m *MongoCrypt) CreateDataKeyContext(kmsProvider string, opts *options.DataKeyOptions) (*Context, error) {
	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	// Create a masterKey document of the form { "provider": <provider string>, other options... }.
	var masterKey bsoncore.Document
	switch {
	case opts.MasterKey != nil:
		// The original key passed into the top-level API was already transformed into a raw BSON document and passed
		// down to here, so we can modify it without copying. Remove the terminating byte to add the "provider" field.
		masterKey = opts.MasterKey[:len(opts.MasterKey)-1]
		masterKey = bsoncore.AppendStringElement(masterKey, "provider", kmsProvider)
		masterKey, _ = bsoncore.AppendDocumentEnd(masterKey, 0)
	default:
		masterKey = bsoncore.NewDocumentBuilder().AppendString("provider", kmsProvider).Build()
	}

	masterKeyBinary := newBinaryFromBytes(masterKey)
	defer masterKeyBinary.close()

	if ok := C.mongocrypt_ctx_setopt_key_encryption_key(ctx.wrapped, masterKeyBinary.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}

	for _, altName := range opts.KeyAltNames {
		if err := setAltName(ctx, altName); err != nil {
			return nil, err
		}
	}

	if opts.KeyMaterial != nil {
		if err := setKeyMaterial(ctx, opts.KeyMaterial); err != nil {
			return nil, err
		}
	}

	if ok := C.mongocrypt_ctx_datakey_init(ctx.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}
	return ctx, nil
}

const (
	IndexTypeUnindexed = 1
	IndexTypeIndexed   = 2
)

// CreateExplicitEncryptionContext creates a Context to use for explicit encryption.
func (m *MongoCrypt) CreateExplicitEncryptionContext(doc bsoncore.Document, opts *options.ExplicitEncryptionOptions) (*Context, error) {

	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	if opts.KeyID != nil {
		keyIDBinary := newBinaryFromBytes(opts.KeyID.Data)
		defer keyIDBinary.close()

		if ok := C.mongocrypt_ctx_setopt_key_id(ctx.wrapped, keyIDBinary.wrapped); !ok {
			return nil, ctx.createErrorFromStatus()
		}
	}
	if opts.KeyAltName != nil {
		if err := setAltName(ctx, *opts.KeyAltName); err != nil {
			return nil, err
		}
	}

	algoStr := C.CString(opts.Algorithm)
	defer C.free(unsafe.Pointer(algoStr))

	if ok := C.mongocrypt_ctx_setopt_algorithm(ctx.wrapped, algoStr, -1); !ok {
		return nil, ctx.createErrorFromStatus()
	}

	if opts.QueryType != "" {
		queryStr := C.CString(opts.QueryType)
		defer C.free(unsafe.Pointer(queryStr))
		if ok := C.mongocrypt_ctx_setopt_query_type(ctx.wrapped, queryStr, -1); !ok {
			return nil, ctx.createErrorFromStatus()
		}
	}

	if opts.ContentionFactor != nil {
		if ok := C.mongocrypt_ctx_setopt_contention_factor(ctx.wrapped, C.int64_t(*opts.ContentionFactor)); !ok {
			return nil, ctx.createErrorFromStatus()
		}
	}

	docBinary := newBinaryFromBytes(doc)
	defer docBinary.close()
	if ok := C.mongocrypt_ctx_explicit_encrypt_init(ctx.wrapped, docBinary.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}

	return ctx, nil
}

// CreateExplicitDecryptionContext creates a Context to use for explicit decryption.
func (m *MongoCrypt) CreateExplicitDecryptionContext(doc bsoncore.Document) (*Context, error) {
	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	docBinary := newBinaryFromBytes(doc)
	defer docBinary.close()

	if ok := C.mongocrypt_ctx_explicit_decrypt_init(ctx.wrapped, docBinary.wrapped); !ok {
		return nil, ctx.createErrorFromStatus()
	}
	return ctx, nil
}

// CryptSharedLibVersion returns the version number for the loaded crypt_shared library, or 0 if the
// crypt_shared library was not loaded.
func (m *MongoCrypt) CryptSharedLibVersion() uint64 {
	return uint64(C.mongocrypt_crypt_shared_lib_version(m.wrapped))
}

// CryptSharedLibVersionString returns the version string for the loaded crypt_shared library, or an
// empty string if the crypt_shared library was not loaded.
func (m *MongoCrypt) CryptSharedLibVersionString() string {
	// Pass in a pointer for "len", but ignore the value because C.GoString can determine the string
	// length without it.
	len := C.uint(0)
	str := C.GoString(C.mongocrypt_crypt_shared_lib_version_string(m.wrapped, &len))
	return str
}

// Close cleans up any resources associated with the given MongoCrypt instance.
func (m *MongoCrypt) Close() {
	C.mongocrypt_destroy(m.wrapped)
}

// RewrapDataKeyContext create a Context to use for rewrapping a data key.
func (m *MongoCrypt) RewrapDataKeyContext(filter []byte, opts *options.RewrapManyDataKeyOptions) (*Context, error) {
	const masterKey = "masterKey"
	const providerKey = "provider"

	ctx := newContext(C.mongocrypt_ctx_new(m.wrapped))
	if ctx.wrapped == nil {
		return nil, m.createErrorFromStatus()
	}

	if opts.Provider != nil {
		// If a provider has been specified, create an encryption key document for creating a data key or for rewrapping
		// datakeys. If a new provider is not specified, then the filter portion of this logic returns the data as it
		// exists in the collection.
		idx, mongocryptDoc := bsoncore.AppendDocumentStart(nil)
		mongocryptDoc = bsoncore.AppendStringElement(mongocryptDoc, providerKey, *opts.Provider)

		if opts.MasterKey != nil {
			mongocryptDoc = opts.MasterKey[:len(opts.MasterKey)-1]
			mongocryptDoc = bsoncore.AppendStringElement(mongocryptDoc, providerKey, *opts.Provider)
		}

		mongocryptDoc, err := bsoncore.AppendDocumentEnd(mongocryptDoc, idx)
		if err != nil {
			return nil, err
		}

		mongocryptBinary := newBinaryFromBytes(mongocryptDoc)
		defer mongocryptBinary.close()

		// Add new masterKey to the mongocrypt context.
		if ok := C.mongocrypt_ctx_setopt_key_encryption_key(ctx.wrapped, mongocryptBinary.wrapped); !ok {
			return nil, ctx.createErrorFromStatus()
		}
	}

	return ctx, rewrapDataKey(ctx, filter)
}

func (m *MongoCrypt) setProviderOptions(kmsProviders bsoncore.Document) error {
	providersBinary := newBinaryFromBytes(kmsProviders)
	defer providersBinary.close()

	if ok := C.mongocrypt_setopt_kms_providers(m.wrapped, providersBinary.wrapped); !ok {
		return m.createErrorFromStatus()
	}
	return nil
}

// setLocalSchemaMap sets the local schema map in mongocrypt.
func (m *MongoCrypt) setLocalSchemaMap(schemaMap map[string]bsoncore.Document) error {
	if len(schemaMap) == 0 {
		return nil
	}

	// convert schema map to BSON document
	schemaMapBSON, err := bson.Marshal(schemaMap)
	if err != nil {
		return fmt.Errorf("error marshalling SchemaMap: %v", err)
	}

	schemaMapBinary := newBinaryFromBytes(schemaMapBSON)
	defer schemaMapBinary.close()

	if ok := C.mongocrypt_setopt_schema_map(m.wrapped, schemaMapBinary.wrapped); !ok {
		return m.createErrorFromStatus()
	}
	return nil
}

// setEncryptedFieldsMap sets the encryptedfields map in mongocrypt.
func (m *MongoCrypt) setEncryptedFieldsMap(encryptedfieldsMap map[string]bsoncore.Document) error {
	if len(encryptedfieldsMap) == 0 {
		return nil
	}

	// convert encryptedfields map to BSON document
	encryptedfieldsMapBSON, err := bson.Marshal(encryptedfieldsMap)
	if err != nil {
		return fmt.Errorf("error marshalling EncryptedFieldsMap: %v", err)
	}

	encryptedfieldsMapBinary := newBinaryFromBytes(encryptedfieldsMapBSON)
	defer encryptedfieldsMapBinary.close()

	if ok := C.mongocrypt_setopt_encrypted_field_config_map(m.wrapped, encryptedfieldsMapBinary.wrapped); !ok {
		return m.createErrorFromStatus()
	}
	return nil
}

// createErrorFromStatus creates a new Error based on the status of the MongoCrypt instance.
func (m *MongoCrypt) createErrorFromStatus() error {
	status := C.mongocrypt_status_new()
	defer C.mongocrypt_status_destroy(status)
	C.mongocrypt_status(m.wrapped, status)
	return errorFromStatus(status)
}

// GetKmsProviders returns the originally configured KMS providers.
func (m *MongoCrypt) GetKmsProviders() bsoncore.Document {
	return m.kmsProviders
}
