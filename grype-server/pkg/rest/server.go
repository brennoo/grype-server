package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	api "github.com/openclarity/grype-server/api/server"
	scanner "github.com/openclarity/grype-server/grype-server/pkg/scanner/interface"
)

type Server struct {
	httpServer *http.Server
}

type scanHandler struct {
	sc scanner.Scanner
}

func (h *scanHandler) PostScanSBOM(_ context.Context, request api.PostScanSBOMRequestObject) (api.PostScanSBOMResponseObject, error) {
	var sbom []byte
	if request.Body != nil && request.Body.Sbom != nil {
		sbom = *request.Body.Sbom
	}

	log.Infof("Handling Scan request.")
	vulnerabilities, err := h.sc.Scan(sbom)
	if err != nil {
		log.Errorf("Failed to scan SBOM: %v", err)
		return nil, fmt.Errorf("scan failed: %v", err)
	}

	return api.PostScanSBOM200JSONResponse{Vulnerabilities: &vulnerabilities}, nil
}

func newAPI(sc scanner.Scanner) http.Handler {
	return api.Handler(api.NewStrictHandler(&scanHandler{sc: sc}, nil))
}

func CreateRESTServer(port int, sc scanner.Scanner) (*Server, error) {
	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           newAPI(sc),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}, nil
}

func (s *Server) Start(errChan chan struct{}) {
	log.Infof("Starting REST server on %s", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("Failed to serve REST server: %v", err)
			errChan <- struct{}{}
		}
	}()
}

func (s *Server) Stop() {
	log.Infof("Stopping REST server")
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			log.Errorf("Failed to shutdown REST server: %v", err)
		}
	}
}
