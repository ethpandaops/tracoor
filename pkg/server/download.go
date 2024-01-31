package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	"github.com/ethpandaops/tracoor/pkg/store"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/klauspost/compress/gzip"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ObjectDownloader struct {
	log      logrus.FieldLogger
	store    store.Store
	mux      *runtime.ServeMux
	indexer  indexer.IndexerClient
	grpcConn string
	grpcOpts []grpc.DialOption
}

func NewObjectDownloader(log logrus.FieldLogger, store store.Store, mux *runtime.ServeMux, grpcConn string, grpcOpts []grpc.DialOption) *ObjectDownloader {
	return &ObjectDownloader{
		log:      log,
		store:    store,
		mux:      mux,
		grpcConn: grpcConn,
		grpcOpts: grpcOpts,
	}
}

func (d *ObjectDownloader) Start() error {
	// Connect to the indexer
	conn, err := grpc.Dial(d.grpcConn, d.grpcOpts...)
	if err != nil {
		return fmt.Errorf("fail to dial: %v", err)
	}

	d.indexer = indexer.NewIndexerClient(conn)

	d.mux.HandlePath("GET", "/download/beacon_state/{id}", d.beaconStateHandler)

	return nil
}

func (d *ObjectDownloader) beaconStateHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListBeaconState(ctx, &indexer.ListBeaconStateRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list beacon states for ID %s", id)

		writeJSONError(w, "Failed to list beacon states", http.StatusInternalServerError)

		return
	}

	if len(resp.BeaconStates) == 0 {
		writeJSONError(w, "No beacon states found", http.StatusNotFound)

		return
	}

	if len(resp.BeaconStates) > 1 {
		writeJSONError(w, "More than one beacon state found", http.StatusInternalServerError)

		return
	}

	state := resp.BeaconStates[0]

	data, err := d.store.GetBeaconState(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon state from store for ID %s from %s", id, state.Location.Value)

		writeJSONError(w, "Failed to get beacon state", http.StatusInternalServerError)

		return
	}

	if err := setResponseCompression(w, r, data); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = w.Write(*data)
	if err != nil {
		writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func setResponseCompression(w http.ResponseWriter, r *http.Request, data *[]byte) error {
	compression := r.URL.Query().Get("compression")
	if compression == "" {
		// If no compression is specified, directly write the data without compression.
		return nil
	}

	switch strings.ToLower(compression) {
	case "gzip":
		w.Header().Set("Content-Encoding", "gzip")

		w.Header().Set("Content-Type", "application/octet-stream")

		var b bytes.Buffer

		gz := gzip.NewWriter(&b)

		if _, err := gz.Write(*data); err != nil {
			return err
		}

		if err := gz.Close(); err != nil {
			return err
		}

		*data = b.Bytes()

	case "none":
		// No action needed for no compression, data is written as is.
	default:
		return fmt.Errorf("Unsupported compression method")
	}
	return nil
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
