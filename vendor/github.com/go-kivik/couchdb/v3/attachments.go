package couchdb

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kivik/couchdb/v3/chttp"
	kivik "github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivik/v3/driver"
)

func (d *db) PutAttachment(ctx context.Context, docID, rev string, att *driver.Attachment, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if att == nil {
		return "", missingArg("att")
	}
	if att.Filename == "" {
		return "", missingArg("att.Filename")
	}
	if att.Content == nil {
		return "", missingArg("att.Content")
	}

	fullCommit, err := fullCommit(options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	if rev != "" {
		query.Set("rev", rev)
	}
	var response struct {
		Rev string `json:"rev"`
	}
	opts := &chttp.Options{
		Body:        att.Content,
		ContentType: att.ContentType,
		FullCommit:  fullCommit,
		Query:       query,
	}
	_, err = d.Client.DoJSON(ctx, http.MethodPut, d.path(chttp.EncodeDocID(docID)+"/"+att.Filename), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}

func (d *db) GetAttachmentMeta(ctx context.Context, docID, filename string, options map[string]interface{}) (*driver.Attachment, error) {
	resp, err := d.fetchAttachment(ctx, http.MethodHead, docID, filename, options)
	if err != nil {
		return nil, err
	}
	att, err := decodeAttachment(resp)
	return att, err
}

func (d *db) GetAttachment(ctx context.Context, docID, filename string, options map[string]interface{}) (*driver.Attachment, error) {
	resp, err := d.fetchAttachment(ctx, http.MethodGet, docID, filename, options)
	if err != nil {
		return nil, err
	}
	return decodeAttachment(resp)
}

func (d *db) fetchAttachment(ctx context.Context, method, docID, filename string, options map[string]interface{}) (*http.Response, error) {
	if method == "" {
		return nil, errors.New("method required")
	}
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}

	inm, err := ifNoneMatch(options)
	if err != nil {
		return nil, err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	opts := &chttp.Options{
		IfNoneMatch: inm,
		Query:       query,
	}
	resp, err := d.Client.DoReq(ctx, method, d.path(chttp.EncodeDocID(docID)+"/"+filename), opts)
	if err != nil {
		return nil, err
	}
	return resp, chttp.ResponseError(resp)
}

func decodeAttachment(resp *http.Response) (*driver.Attachment, error) {
	cType, err := getContentType(resp)
	if err != nil {
		return nil, err
	}
	digest, err := getDigest(resp)
	if err != nil {
		return nil, err
	}

	return &driver.Attachment{
		ContentType: cType,
		Digest:      digest,
		Size:        resp.ContentLength,
		Content:     resp.Body,
	}, nil
}

func getContentType(resp *http.Response) (string, error) {
	ctype := resp.Header.Get("Content-Type")
	if _, ok := resp.Header["Content-Type"]; !ok {
		return "", &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: errors.New("no Content-Type in response")}
	}
	return ctype, nil
}

func getDigest(resp *http.Response) (string, error) {
	etag, ok := chttp.ETag(resp)
	if !ok {
		return "", &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: errors.New("ETag header not found")}
	}
	return etag, nil
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if rev == "" {
		return "", missingArg("rev")
	}
	if filename == "" {
		return "", missingArg("filename")
	}

	fullCommit, err := fullCommit(options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	query.Set("rev", rev)
	var response struct {
		Rev string `json:"rev"`
	}

	opts := &chttp.Options{
		FullCommit: fullCommit,
		Query:      query,
	}
	_, err = d.Client.DoJSON(ctx, http.MethodDelete, d.path(chttp.EncodeDocID(docID)+"/"+filename), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}
