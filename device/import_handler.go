package device

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/httpkit"
)

// importTemplate is the file that the template link gives: a UTF-8 BOM so
// that Excel reads the diacritics, a `;` delimiter as Excel writes in the
// Romanian and Russian locales, the header, and one example row.
const importTemplate = "\xef\xbb\xbf" +
	"dev_eui;name;vendor;model;type;address;note;groups\r\n" +
	"70B3D57ED0001234;Bloc 3, scara 1;Milesight;EM300-DI;water;str. Ștefan cel Mare 1;;\"Botanica;Bloc 3\"\r\n"

type importHandler struct {
	imports *importService
	guard   guard
}

func newImportHandler(imports *importService, g *core.Guard) *importHandler {
	return &importHandler{imports: imports, guard: g}
}

func registerImportHandler(mux *http.ServeMux, h *importHandler) {
	mux.Handle("POST /api/v1/devices/import/preview", member(h.guard, h.preview))
	mux.Handle("POST /api/v1/devices/import", member(h.guard, h.commit))
	mux.Handle("GET /api/v1/devices/import/template", member(h.guard, h.template))
}

// importRowResponse is one row of a preview. The frontend mirrors it in
// vue/src/device/use-device-import.ts.
type importRowResponse struct {
	Line      int      `json:"line"`
	Status    string   `json:"status"`
	Code      string   `json:"code,omitempty"`
	DevEUI    string   `json:"dev_eui"`
	Name      string   `json:"name"`
	Vendor    string   `json:"vendor,omitempty"`
	Model     string   `json:"model,omitempty"`
	Groups    []string `json:"groups"`
	NewModel  bool     `json:"new_model"`
	NewGroups []string `json:"new_groups"`
}

type importTotalsResponse struct {
	New       int `json:"new"`
	Exists    int `json:"exists"`
	Error     int `json:"error"`
	NewModels int `json:"new_models"`
	NewGroups int `json:"new_groups"`
}

type importPreviewResponse struct {
	Rows   []importRowResponse  `json:"rows"`
	Totals importTotalsResponse `json:"totals"`
}

func (h *importHandler) preview(w http.ResponseWriter, req *http.Request, orgID int64) error {
	rows, err := readCSV(w, req)
	if err != nil {
		return err
	}
	p, err := h.imports.preview(req.Context(), orgID, rows)
	if err != nil {
		return err
	}
	resp := importPreviewResponse{
		Rows:   make([]importRowResponse, len(p.rows)),
		Totals: importTotalsResponse(p.totals),
	}
	for i := range p.rows {
		r := &p.rows[i]
		resp.Rows[i] = importRowResponse{
			Line: r.Line, Status: r.Status, Code: r.Code, DevEUI: r.DevEUI, Name: r.Name,
			Vendor: r.Vendor, Model: r.Model, Groups: nonNil(r.Groups), NewModel: r.NewModel,
			NewGroups: nonNil(r.NewGroups),
		}
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

func (h *importHandler) commit(w http.ResponseWriter, req *http.Request, orgID int64) error {
	rows, err := readCSV(w, req)
	if err != nil {
		return err
	}
	totals, err := h.imports.commit(req.Context(), orgID, rows)
	if err != nil {
		return err
	}
	httpkit.JSON(w, http.StatusOK, importTotalsResponse(totals))
	return nil
}

func (h *importHandler) template(w http.ResponseWriter, _ *http.Request, _ int64) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="lora-devices.csv"`)
	_, _ = io.WriteString(w, importTemplate)
	return nil
}

// readCSV reads a text/csv body of at most maxImportBytes and parses it. A
// text/csv body is not one that an HTML form can send, so a cross-site form
// cannot import into an organization.
func readCSV(w http.ResponseWriter, req *http.Request) ([]importRow, error) {
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/csv" {
		return nil, errImportContentType
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, req.Body, maxImportBytes))
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		return nil, errImportTooLarge
	}
	if err != nil {
		return nil, err
	}
	return parseCSV(data)
}

// nonNil returns an empty slice for nil, so the JSON holds [] and not null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
