package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kivik/couchdb/v3/chttp"
	kivik "github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivik/v3/driver"
)

type db struct {
	*client
	dbName string
}

var (
	_ driver.DB                   = &db{}
	_ driver.OptsFinder           = &db{}
	_ driver.MetaGetter           = &db{}
	_ driver.AttachmentMetaGetter = &db{}
	_ driver.PartitionedDB        = &db{}
)

func (d *db) path(path string) string {
	url, err := url.Parse(d.dbName + "/" + strings.TrimPrefix(path, "/"))
	if err != nil {
		panic("THIS IS A BUG: d.path failed: " + err.Error())
	}
	return url.String()
}

func optionsToParams(opts ...map[string]interface{}) (url.Values, error) {
	params := url.Values{}
	for _, optsSet := range opts {
		if err := encodeKeys(optsSet); err != nil {
			return nil, err
		}
		for key, i := range optsSet {
			var values []string
			switch v := i.(type) {
			case string:
				values = []string{v}
			case []string:
				values = v
			case bool:
				values = []string{fmt.Sprintf("%t", v)}
			case int, uint, uint8, uint16, uint32, uint64, int8, int16, int32, int64:
				values = []string{fmt.Sprintf("%d", v)}
			default:
				return nil, &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("kivik: invalid type %T for options", i)}
			}
			for _, value := range values {
				params.Add(key, value)
			}
		}
	}
	return params, nil
}

// rowsQuery performs a query that returns a rows iterator.
func (d *db) rowsQuery(ctx context.Context, path string, opts map[string]interface{}) (driver.Rows, error) {
	payload := make(map[string]interface{})
	if keys := opts["keys"]; keys != nil {
		delete(opts, "keys")
		payload["keys"] = keys
	}
	rowsInit := newRows
	if queries := opts["queries"]; queries != nil {
		rowsInit = func(ctx context.Context, r io.ReadCloser) driver.Rows {
			return newMultiQueriesRows(ctx, r)
		}
		delete(opts, "queries")
		payload["queries"] = queries
		// Funny that this works even in CouchDB 1.x. It seems 1.x just ignores
		// extra path elements beyond the view name. So yay for accidental
		// backward compatibility!
		path = filepath.Join(path, "queries")
	}
	query, err := optionsToParams(opts)
	if err != nil {
		return nil, err
	}
	options := &chttp.Options{Query: query}
	method := http.MethodGet
	if len(payload) > 0 {
		method = http.MethodPost
		options.GetBody = chttp.BodyEncoder(payload)
		options.Header = http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		}
	}
	resp, err := d.Client.DoReq(ctx, method, d.path(path), options)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return rowsInit(ctx, resp.Body), nil
}

// AllDocs returns all of the documents in the database.
func (d *db) AllDocs(ctx context.Context, opts map[string]interface{}) (driver.Rows, error) {
	reqPath := "_all_docs"
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	return d.rowsQuery(ctx, reqPath, opts)
}

// DesignDocs returns all of the documents in the database.
func (d *db) DesignDocs(ctx context.Context, opts map[string]interface{}) (driver.Rows, error) {
	return d.rowsQuery(ctx, "_design_docs", opts)
}

// LocalDocs returns all of the documents in the database.
func (d *db) LocalDocs(ctx context.Context, opts map[string]interface{}) (driver.Rows, error) {
	return d.rowsQuery(ctx, "_local_docs", opts)
}

// Query queries a view.
func (d *db) Query(ctx context.Context, ddoc, view string, opts map[string]interface{}) (driver.Rows, error) {
	reqPath := fmt.Sprintf("_design/%s/_view/%s", chttp.EncodeDocID(ddoc), chttp.EncodeDocID(view))
	if part, ok := opts[OptionPartition].(string); ok {
		delete(opts, OptionPartition)
		reqPath = path.Join("_partition", part, reqPath)
	}
	return d.rowsQuery(ctx, reqPath, opts)
}

// Get fetches the requested document.
func (d *db) Get(ctx context.Context, docID string, options map[string]interface{}) (*driver.Document, error) {
	resp, rev, err := d.get(ctx, http.MethodGet, docID, options)
	if err != nil {
		return nil, err
	}
	ct, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	switch ct {
	case typeJSON:
		return &driver.Document{
			Rev:           rev,
			ContentLength: resp.ContentLength,
			Body:          resp.Body,
		}, nil
	case typeMPRelated:
		boundary := strings.Trim(params["boundary"], "\"")
		if boundary == "" {
			return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: errors.New("kivik: boundary missing for multipart/related response")}
		}
		mpReader := multipart.NewReader(resp.Body, boundary)
		body, err := mpReader.NextPart()
		if err != nil {
			return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
		length := int64(-1)
		if cl, e := strconv.ParseInt(body.Header.Get("Content-Length"), 10, 64); e == nil {
			length = cl
		}

		// TODO: Use a TeeReader here, to avoid slurping the entire body into memory at once
		content, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
		var metaDoc struct {
			Attachments map[string]attMeta `json:"_attachments"`
		}
		if err := json.Unmarshal(content, &metaDoc); err != nil {
			return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}

		return &driver.Document{
			ContentLength: length,
			Rev:           rev,
			Body:          ioutil.NopCloser(bytes.NewBuffer(content)),
			Attachments: &multipartAttachments{
				content:  resp.Body,
				mpReader: mpReader,
				meta:     metaDoc.Attachments,
			},
		}, nil
	default:
		return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("kivik: invalid content type in response: %s", ct)}
	}
}

type attMeta struct {
	ContentType string `json:"content_type"`
	Size        *int64 `json:"length"`
	Follows     bool   `json:"follows"`
}

type multipartAttachments struct {
	content  io.ReadCloser
	mpReader *multipart.Reader
	meta     map[string]attMeta
}

var _ driver.Attachments = &multipartAttachments{}

func (a *multipartAttachments) Next(att *driver.Attachment) error {
	part, err := a.mpReader.NextPart()
	switch err {
	case io.EOF:
		return err
	case nil:
		// fall through
	default:
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}

	disp, dispositionParams, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Content-Disposition: %s", err)}
	}
	if disp != "attachment" {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("Unexpected Content-Disposition: %s", disp)}
	}
	filename := dispositionParams["filename"]

	meta := a.meta[filename]
	if !meta.Follows {
		return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("File '%s' not in manifest", filename)}
	}

	size := int64(-1)
	if meta.Size != nil {
		size = *meta.Size
	} else if cl, e := strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64); e == nil {
		size = cl
	}

	var cType string
	if ctHeader, ok := part.Header["Content-Type"]; ok {
		cType, _, err = mime.ParseMediaType(ctHeader[0])
		if err != nil {
			return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
		}
	} else {
		cType = meta.ContentType
	}

	*att = driver.Attachment{
		Filename:        filename,
		Size:            size,
		ContentType:     cType,
		Content:         part,
		ContentEncoding: part.Header.Get("Content-Encoding"),
	}
	return nil
}

func (a *multipartAttachments) Close() error {
	return a.content.Close()
}

// Rev returns the most current rev of the requested document.
func (d *db) GetMeta(ctx context.Context, docID string, options map[string]interface{}) (size int64, rev string, err error) {
	resp, rev, err := d.get(ctx, http.MethodHead, docID, options)
	if err != nil {
		return 0, "", err
	}
	return resp.ContentLength, rev, err
}

func (d *db) get(ctx context.Context, method, docID string, options map[string]interface{}) (*http.Response, string, error) {
	if docID == "" {
		return nil, "", missingArg("docID")
	}

	inm, err := ifNoneMatch(options)
	if err != nil {
		return nil, "", err
	}

	params, err := optionsToParams(options)
	if err != nil {
		return nil, "", err
	}
	opts := &chttp.Options{
		Accept:      typeMPRelated + "," + typeJSON,
		IfNoneMatch: inm,
		Query:       params,
	}
	if _, ok := options[NoMultipartGet]; ok {
		opts.Accept = typeJSON
	}
	resp, err := d.Client.DoReq(ctx, method, d.path(chttp.EncodeDocID(docID)), opts)
	if err != nil {
		return nil, "", err
	}
	if respErr := chttp.ResponseError(resp); respErr != nil {
		return nil, "", respErr
	}
	rev, err := chttp.GetRev(resp)
	return resp, rev, err
}

func (d *db) CreateDoc(ctx context.Context, doc interface{}, options map[string]interface{}) (docID, rev string, err error) {
	result := struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}{}

	fullCommit, err := fullCommit(options)
	if err != nil {
		return "", "", err
	}

	path := d.dbName
	if len(options) > 0 {
		params, e := optionsToParams(options)
		if e != nil {
			return "", "", e
		}
		path += "?" + params.Encode()
	}

	opts := &chttp.Options{
		Body:       chttp.EncodeBody(doc),
		FullCommit: fullCommit,
	}
	_, err = d.Client.DoJSON(ctx, http.MethodPost, path, opts, &result)
	return result.ID, result.Rev, err
}

func putOpts(doc interface{}, options map[string]interface{}) (*chttp.Options, error) {
	fullCommit, err := fullCommit(options)
	if err != nil {
		return nil, err
	}
	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	if _, ok := options[NoMultipartPut]; !ok {
		if atts, ok := extractAttachments(doc); ok {
			boundary, size, multipartBody, e := newMultipartAttachments(chttp.EncodeBody(doc), atts)
			if e != nil {
				return nil, e
			}
			return &chttp.Options{
				Body:          multipartBody,
				FullCommit:    fullCommit,
				Query:         params,
				ContentLength: size,
				ContentType:   fmt.Sprintf(typeMPRelated+"; boundary=%q", boundary),
			}, nil
		}
	}
	return &chttp.Options{
		Body:       chttp.EncodeBody(doc),
		FullCommit: fullCommit,
		Query:      params,
	}, nil
}

func (d *db) Put(ctx context.Context, docID string, doc interface{}, options map[string]interface{}) (rev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	opts, err := putOpts(doc, options)
	if err != nil {
		return "", err
	}
	var result struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}
	_, err = d.Client.DoJSON(ctx, http.MethodPut, d.path(chttp.EncodeDocID(docID)), opts, &result)
	if err != nil {
		return "", err
	}
	return result.Rev, nil
}

const attachmentsKey = "_attachments"

func extractAttachments(doc interface{}) (*kivik.Attachments, bool) {
	if doc == nil {
		return nil, false
	}
	v := reflect.ValueOf(doc)
	if v.Type().Kind() == reflect.Ptr {
		return extractAttachments(v.Elem().Interface())
	}
	if stdMap, ok := doc.(map[string]interface{}); ok {
		return interfaceToAttachments(stdMap[attachmentsKey])
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("json") == attachmentsKey {
			return interfaceToAttachments(v.Field(i).Interface())
		}
	}
	return nil, false
}

func interfaceToAttachments(i interface{}) (*kivik.Attachments, bool) {
	switch t := i.(type) {
	case kivik.Attachments:
		atts := make(kivik.Attachments, len(t))
		for k, v := range t {
			atts[k] = v
			delete(t, k)
		}
		return &atts, true
	case *kivik.Attachments:
		atts := new(kivik.Attachments)
		*atts = *t
		*t = nil
		return atts, true
	}
	return nil, false
}

// newMultipartAttachments reads a json stream on in, and produces a
// multipart/related output suitable for a PUT request.
func newMultipartAttachments(in io.ReadCloser, att *kivik.Attachments) (boundary string, size int64, content io.ReadCloser, err error) {
	tmp, err := ioutil.TempFile("", "kivik-multipart-*")
	if err != nil {
		return "", 0, nil, err
	}
	body := multipart.NewWriter(tmp)
	w := sync.WaitGroup{}
	w.Add(1)
	go func() {
		err = createMultipart(body, in, att)
		e := in.Close()
		if err == nil {
			err = e
		}
		w.Done()
	}()
	w.Wait()
	if e := tmp.Sync(); err == nil {
		err = e
	}
	if info, e := tmp.Stat(); e == nil {
		size = info.Size()
	} else {
		if err == nil {
			err = e
		}
	}
	if _, e := tmp.Seek(0, 0); e != nil && err == nil {
		err = e
	}
	return body.Boundary(),
		size,
		tmp,
		err
}

func createMultipart(w *multipart.Writer, r io.ReadCloser, atts *kivik.Attachments) error {
	doc, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type": {typeJSON},
	})
	if err != nil {
		return err
	}
	attJSON := replaceAttachments(r, atts)
	if _, e := io.Copy(doc, attJSON); e != nil {
		return e
	}

	// Sort the filenames to ensure order consistent with json.Marshal's ordering
	// of the stubs in the body
	filenames := make([]string, 0, len(*atts))
	for filename := range *atts {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		att := (*atts)[filename]
		file, err := w.CreatePart(textproto.MIMEHeader{
			// "Content-Type":        {att.ContentType},
			// "Content-Disposition": {fmt.Sprintf(`attachment; filename=%q`, filename)},
			// "Content-Length":      {strconv.FormatInt(att.Size, 10)},
		})
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, att.Content); err != nil {
			return err
		}
		_ = att.Content.Close()
	}

	return w.Close()
}

type lener interface {
	Len() int
}

type stater interface {
	Stat() (os.FileInfo, error)
}

// attachmentSize determines the size of the `in` stream by reading the entire
// stream first.  This method is a no-op if att.Size is already > , and sets the Size
// parameter accordingly. If Size is already set, this function does nothing.
// It attempts the following methods:
//
//    1. Calls `Len()`, if implemented by `in` (i.e. `*bytes.Buffer`)
//    2. Calls `Stat()`, if implemented by `in` (i.e. `*os.File`) then returns
//       the file's size
//    3. Read the entire stream to determine the size, and replace att.Content
//       to be replayed.
func attachmentSize(att *kivik.Attachment) error {
	if att.Size > 0 {
		return nil
	}
	size, r, err := readerSize(att.Content)
	if err != nil {
		return err
	}
	rc, ok := r.(io.ReadCloser)
	if !ok {
		rc = ioutil.NopCloser(r)
	}

	att.Content = rc
	att.Size = size
	return nil
}

func readerSize(in io.Reader) (int64, io.Reader, error) {
	if ln, ok := in.(lener); ok {
		return int64(ln.Len()), in, nil
	}
	if st, ok := in.(stater); ok {
		info, err := st.Stat()
		if err != nil {
			return 0, nil, err
		}
		return info.Size(), in, nil
	}
	content, err := ioutil.ReadAll(in)
	if err != nil {
		return 0, nil, err
	}
	buf := bytes.NewBuffer(content)
	return int64(buf.Len()), ioutil.NopCloser(buf), nil
}

// NewAttachment is a convenience function, which sets the size of the attachment
// based on content. This is intended for creating attachments to be uploaded
// using multipart/related capabilities of Put().  The attachment size will be
// set to the first of the following found:
//
//    1. `size`, if present. Only the first value is considered
//    2. content.Len(), if implemented (i.e. *bytes.Buffer)
//    3. content.Stat().Size(), if implemented (i.e. *os.File)
//    4. Read the entire content into memory, to determine the size. This can
//       use a lot of memory for large attachments. Please use a file, or
//       specify the size directly instead.
func NewAttachment(filename, contentType string, content io.Reader, size ...int64) (*kivik.Attachment, error) {
	var filesize int64
	if len(size) > 0 {
		filesize = size[0]
	} else {
		var err error
		filesize, content, err = readerSize(content)
		if err != nil {
			return nil, err
		}
	}
	rc, ok := content.(io.ReadCloser)
	if !ok {
		rc = ioutil.NopCloser(content)
	}
	return &kivik.Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     rc,
		Size:        filesize,
	}, nil
}

// replaceAttachments reads a json stream on in, looking for the _attachments
// key, then replaces its value with the marshaled version of att.
func replaceAttachments(in io.ReadCloser, atts *kivik.Attachments) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		stubs, err := attachmentStubs(atts)
		if err != nil {
			_ = w.CloseWithError(err)
			_ = in.Close()
			return
		}
		err = copyWithAttachmentStubs(w, in, stubs)
		e := in.Close()
		if err == nil {
			err = e
		}
		_ = w.CloseWithError(err)
	}()
	return r
}

type stub struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"length"`
}

func (s *stub) MarshalJSON() ([]byte, error) {
	type attJSON struct {
		stub
		Follows bool `json:"follows"`
	}
	att := attJSON{
		stub:    *s,
		Follows: true,
	}
	return json.Marshal(att)
}

func attachmentStubs(atts *kivik.Attachments) (map[string]*stub, error) {
	if atts == nil {
		return nil, nil
	}
	result := make(map[string]*stub, len(*atts))
	for filename, att := range *atts {
		if err := attachmentSize(att); err != nil {
			return nil, err
		}
		result[filename] = &stub{
			ContentType: att.ContentType,
			Size:        att.Size,
		}
	}
	return result, nil
}

// copyWithAttachmentStubs copies r to w, replacing the _attachment value with the
// marshaled version of atts.
func copyWithAttachmentStubs(w io.Writer, r io.Reader, atts map[string]*stub) error {
	dec := json.NewDecoder(r)
	t, err := dec.Token()
	if err == nil {
		if t != json.Delim('{') {
			return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("expected '{', found '%v'", t)}
		}
	}
	if err != nil {
		if err != io.EOF {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%v", t); err != nil {
		return err
	}
	first := true
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: err}
		}
		switch tp := t.(type) {
		case string:
			if !first {
				if _, e := w.Write([]byte(",")); e != nil {
					return e
				}
			}
			first = false
			if _, e := fmt.Fprintf(w, `"%s":`, tp); e != nil {
				return e
			}
			var val json.RawMessage
			if e := dec.Decode(&val); e != nil {
				return e
			}
			if tp == attachmentsKey {
				if e := json.NewEncoder(w).Encode(atts); e != nil {
					return e
				}
				// Once we're here, we can just stream the rest of the input
				// unaltered.
				if _, e := io.Copy(w, dec.Buffered()); e != nil {
					return e
				}
				_, e := io.Copy(w, r)
				return e
			}
			if _, e := w.Write(val); e != nil {
				return e
			}
		case json.Delim:
			if tp != json.Delim('}') {
				return fmt.Errorf("expected '}', found '%v'", t)
			}
			if _, err := fmt.Fprintf(w, "%v", t); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *db) Delete(ctx context.Context, docID, rev string, options map[string]interface{}) (string, error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if rev == "" {
		return "", missingArg("rev")
	}

	fullCommit, err := fullCommit(options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	if query.Get("rev") == "" {
		query.Set("rev", rev)
	}
	opts := &chttp.Options{
		FullCommit: fullCommit,
		Query:      query,
	}
	resp, err := d.Client.DoReq(ctx, http.MethodDelete, d.path(chttp.EncodeDocID(docID)), opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // nolint: errcheck
	return chttp.GetRev(resp)
}

func (d *db) Flush(ctx context.Context) error {
	opts := &chttp.Options{
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	_, err := d.Client.DoError(ctx, http.MethodPost, d.path("/_ensure_full_commit"), opts)
	return err
}

func (d *db) Compact(ctx context.Context) error {
	opts := &chttp.Options{
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	res, err := d.Client.DoReq(ctx, http.MethodPost, d.path("/_compact"), opts)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) CompactView(ctx context.Context, ddocID string) error {
	if ddocID == "" {
		return missingArg("ddocID")
	}
	opts := &chttp.Options{
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	res, err := d.Client.DoReq(ctx, http.MethodPost, d.path("/_compact/"+ddocID), opts)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) ViewCleanup(ctx context.Context) error {
	opts := &chttp.Options{
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	res, err := d.Client.DoReq(ctx, http.MethodPost, d.path("/_view_cleanup"), opts)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) Security(ctx context.Context) (*driver.Security, error) {
	var sec *driver.Security
	_, err := d.Client.DoJSON(ctx, http.MethodGet, d.path("/_security"), nil, &sec)
	return sec, err
}

func (d *db) SetSecurity(ctx context.Context, security *driver.Security) error {
	opts := &chttp.Options{
		GetBody: chttp.BodyEncoder(security),
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	res, err := d.Client.DoReq(ctx, http.MethodPut, d.path("/_security"), opts)
	if err != nil {
		return err
	}
	defer res.Body.Close() // nolint: errcheck
	return chttp.ResponseError(res)
}

func (d *db) Copy(ctx context.Context, targetID, sourceID string, options map[string]interface{}) (targetRev string, err error) {
	if sourceID == "" {
		return "", missingArg("sourceID")
	}
	if targetID == "" {
		return "", missingArg("targetID")
	}
	fullCommit, err := fullCommit(options)
	if err != nil {
		return "", err
	}
	params, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	opts := &chttp.Options{
		FullCommit: fullCommit,
		Query:      params,
		Header: http.Header{
			chttp.HeaderDestination: []string{targetID},
		},
	}
	resp, err := d.Client.DoReq(ctx, "COPY", d.path(chttp.EncodeDocID(sourceID)), opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // nolint: errcheck
	return chttp.GetRev(resp)
}

func (d *db) Purge(ctx context.Context, docMap map[string][]string) (*driver.PurgeResult, error) {
	result := &driver.PurgeResult{}
	options := &chttp.Options{
		GetBody: chttp.BodyEncoder(docMap),
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	_, err := d.Client.DoJSON(ctx, http.MethodPost, d.path("_purge"), options, &result)
	return result, err
}

var _ driver.RevsDiffer = &db{}

func (d *db) RevsDiff(ctx context.Context, revMap interface{}) (driver.Rows, error) {
	options := &chttp.Options{
		GetBody: chttp.BodyEncoder(revMap),
		Header: http.Header{
			chttp.HeaderIdempotencyKey: []string{},
		},
	}
	resp, err := d.Client.DoReq(ctx, http.MethodPost, d.path("_revs_diff"), options)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newRevsDiffRows(ctx, resp.Body), nil
}

type revsDiffParser struct{}

func (p *revsDiffParser) decodeItem(i interface{}, dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	row := i.(*driver.Row)
	row.ID = t.(string)
	return dec.Decode(&row.Value)
}

func newRevsDiffRows(ctx context.Context, in io.ReadCloser) driver.Rows {
	iter := newIter(ctx, nil, "", in, &revsDiffParser{})
	iter.objMode = true
	return &rows{iter: iter}
}
