package device

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/waneock/app-lora/pkg/httpkit"
)

// maxIngestBytes bounds the body of one uplink request.
const maxIngestBytes = 1 << 20

// ingestHandler receives the uplinks of the network servers. Its routes are
// public: the secret of an integration, not a session, lets a request in
// (device:D2).
type ingestHandler struct {
	ingest *ingestService
	logger *slog.Logger
}

func newIngestHandler(ingest *ingestService, logger *slog.Logger) *ingestHandler {
	return &ingestHandler{ingest: ingest, logger: logger}
}

func registerIngestHandler(mux *http.ServeMux, h *ingestHandler) {
	mux.HandleFunc("POST "+ingestPath(kindChirpStack), h.chirpStack)
	mux.HandleFunc("POST "+ingestPath(kindTTN), h.ttn)
}

func (h *ingestHandler) chirpStack(w http.ResponseWriter, req *http.Request) {
	h.receive(w, req, kindChirpStack, func(body []byte) (*uplinkMessage, bool, *ruleError) {
		return parseChirpStack(req.URL.Query().Get("event"), req.Header.Get("Content-Type"), body)
	})
}

func (h *ingestHandler) ttn(w http.ResponseWriter, req *http.Request) {
	h.receive(w, req, kindTTN, parseTTN)
}

// receive authenticates a request, reads its body with parse, and stores
// the uplink. A request it cannot read is kept as the last error of the
// integration and answers 4xx. A message that is not an uplink answers 204,
// so the network server does not send it again. A failed store answers 500,
// so the network server can send it again.
func (h *ingestHandler) receive(w http.ResponseWriter, req *http.Request, kind string,
	parse func(body []byte) (*uplinkMessage, bool, *ruleError),
) {
	ctx := req.Context()
	ref, err := h.ingest.authenticate(ctx, bearer(req))
	if err != nil {
		h.write(w, req, err)
		return
	}

	msg, skip, rerr := h.read(w, req, ref, kind, parse)
	if rerr != nil {
		if err := h.ingest.fail(ctx, ref, rerr); err != nil {
			h.write(w, req, err)
			return
		}
		h.write(w, req, rerr)
		return
	}
	if !skip {
		if err := h.ingest.store(ctx, ref, msg); err != nil {
			h.write(w, req, err)
			return
		}
	}
	httpkit.NoContent(w)
}

// read checks the endpoint and parses the body. It returns the rule that the
// request breaks, or the uplink, or skip for a message that is not one.
func (h *ingestHandler) read(w http.ResponseWriter, req *http.Request, ref *integrationRef, kind string,
	parse func(body []byte) (*uplinkMessage, bool, *ruleError),
) (*uplinkMessage, bool, *ruleError) {
	if ref.Kind != kind {
		return nil, false, errWrongEndpoint
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, req.Body, maxIngestBytes))
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		return nil, false, errIngestTooLarge
	}
	if err != nil {
		return nil, false, errIngestInvalidJSON
	}
	return parse(body)
}

func (h *ingestHandler) write(w http.ResponseWriter, req *http.Request, err error) {
	httpkit.WriteError(w, req, h.logger, toHTTP(err))
}

// bearer returns the token of an "Authorization: Bearer <token>" header.
func bearer(req *http.Request) string {
	scheme, token, ok := strings.Cut(req.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
