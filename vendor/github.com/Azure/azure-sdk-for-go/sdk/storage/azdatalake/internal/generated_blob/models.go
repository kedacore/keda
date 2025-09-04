//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package generated_blob

import (
	"encoding/xml"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"net/url"
)

type TransactionalContentSetter interface {
	SetCRC64([]byte)
	SetMD5([]byte)
}

func (a *AppendBlobClientAppendBlockOptions) SetCRC64(v []byte) {
	a.TransactionalContentCRC64 = v
}

func (a *AppendBlobClientAppendBlockOptions) SetMD5(v []byte) {
	a.TransactionalContentMD5 = v
}

func (b *BlockBlobClientStageBlockOptions) SetCRC64(v []byte) {
	b.TransactionalContentCRC64 = v
}

func (b *BlockBlobClientStageBlockOptions) SetMD5(v []byte) {
	b.TransactionalContentMD5 = v
}

func (p *PageBlobClientUploadPagesOptions) SetCRC64(v []byte) {
	p.TransactionalContentCRC64 = v
}

func (p *PageBlobClientUploadPagesOptions) SetMD5(v []byte) {
	p.TransactionalContentMD5 = v
}

type SourceContentSetter interface {
	SetSourceContentCRC64(v []byte)
	SetSourceContentMD5(v []byte)
}

func (a *AppendBlobClientAppendBlockFromURLOptions) SetSourceContentCRC64(v []byte) {
	a.SourceContentcrc64 = v
}

func (a *AppendBlobClientAppendBlockFromURLOptions) SetSourceContentMD5(v []byte) {
	a.SourceContentMD5 = v
}

func (b *BlockBlobClientStageBlockFromURLOptions) SetSourceContentCRC64(v []byte) {
	b.SourceContentcrc64 = v
}

func (b *BlockBlobClientStageBlockFromURLOptions) SetSourceContentMD5(v []byte) {
	b.SourceContentMD5 = v
}

func (p *PageBlobClientUploadPagesFromURLOptions) SetSourceContentCRC64(v []byte) {
	p.SourceContentcrc64 = v
}

func (p *PageBlobClientUploadPagesFromURLOptions) SetSourceContentMD5(v []byte) {
	p.SourceContentMD5 = v
}

// Custom UnmarshalXML functions for types that need special handling.

// UnmarshalXML implements the xml.Unmarshaller interface for type PathPrefix.
func (p *PathPrefix) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	type alias PathPrefix
	aux := &struct {
		*alias
		PathName *PathName `xml:"Name"`
	}{
		alias: (*alias)(p),
	}
	if err := dec.DecodeElement(aux, &start); err != nil {
		return err
	}
	if aux.PathName != nil {
		if aux.PathName.Encoded != nil && *aux.PathName.Encoded {
			name, err := url.QueryUnescape(*aux.PathName.Content)

			if err != nil {
				return err
			}
			p.Name = to.Ptr(string(name))
		} else {
			p.Name = aux.PathName.Content
		}
	}
	return nil
}

// UnmarshalXML implements the xml.Unmarshaller interface for type PathItem.
func (p *PathItem) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	type alias PathItem
	aux := &struct {
		*alias
		PathName   *PathName            `xml:"Name"`
		Metadata   additionalProperties `xml:"Metadata"`
		OrMetadata additionalProperties `xml:"OrMetadata"`
	}{
		alias: (*alias)(p),
	}
	if err := dec.DecodeElement(aux, &start); err != nil {
		return err
	}
	p.Metadata = (map[string]*string)(aux.Metadata)
	p.OrMetadata = (map[string]*string)(aux.OrMetadata)
	if aux.PathName != nil {
		if aux.PathName.Encoded != nil && *aux.PathName.Encoded {
			name, err := url.QueryUnescape(*aux.PathName.Content)

			// name, err := base64.StdEncoding.DecodeString(*aux.PathName.Content)
			if err != nil {
				return err
			}
			p.Name = to.Ptr(string(name))
		} else {
			p.Name = aux.PathName.Content
		}
	}
	return nil
}
