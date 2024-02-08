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
	"github.com/klauspost/compress/snappy"
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

	if err := d.mux.HandlePath("GET", "/download/beacon_state/{id}", d.beaconStateHandler); err != nil {
		return fmt.Errorf("failed to register beacon state download handler: %v", err)
	}

	if err := d.mux.HandlePath("GET", "/download/execution_block_trace/{id}", d.executionBlockTraceHandler); err != nil {
		return fmt.Errorf("failed to register execution block trace download handler: %v", err)
	}

	if err := d.mux.HandlePath("GET", "/download/execution_bad_block/{id}", d.executionBadBlock); err != nil {
		return fmt.Errorf("failed to register execution block trace download handler: %v", err)
	}

	return nil
}

func (d *ObjectDownloader) beaconStateHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

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

		d.writeJSONError(w, "Failed to list beacon states", http.StatusInternalServerError)

		return
	}

	if len(resp.BeaconStates) == 0 {
		d.writeJSONError(w, "No beacon states found", http.StatusNotFound)

		return
	}

	if len(resp.BeaconStates) > 1 {
		d.writeJSONError(w, "More than one beacon state found", http.StatusInternalServerError)

		return
	}

	state := resp.BeaconStates[0]

	data, err := d.store.GetBeaconState(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon state from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get beacon state", http.StatusInternalServerError)

		return
	}

	if err := setResponseCompression(w, r, data); err != nil {
		d.writeJSONError(w, err.Error(), http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) executionBlockTraceHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListExecutionBlockTrace(ctx, &indexer.ListExecutionBlockTraceRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list execution block trace for ID %s", id)

		d.writeJSONError(w, "Failed to list execution block trace", http.StatusInternalServerError)

		return
	}

	if len(resp.ExecutionBlockTraces) == 0 {
		d.writeJSONError(w, "No execution block trace found", http.StatusNotFound)

		return
	}

	if len(resp.ExecutionBlockTraces) > 1 {
		d.writeJSONError(w, "More than one execution block trace found", http.StatusInternalServerError)

		return
	}

	state := resp.ExecutionBlockTraces[0]

	data, err := d.store.GetExecutionBlockTrace(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get execution block trace from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get execution block trace", http.StatusInternalServerError)

		return
	}

	if err := setResponseCompression(w, r, data); err != nil {
		d.writeJSONError(w, err.Error(), http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) executionBadBlock(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListExecutionBadBlock(ctx, &indexer.ListExecutionBadBlockRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list execution bad block for ID %s", id)

		d.writeJSONError(w, "Failed to list execution bad block", http.StatusInternalServerError)

		return
	}

	if len(resp.ExecutionBadBlocks) == 0 {
		d.writeJSONError(w, "No execution bad block found", http.StatusNotFound)

		return
	}

	if len(resp.ExecutionBadBlocks) > 1 {
		d.writeJSONError(w, "More than one execution bad block found", http.StatusInternalServerError)

		return
	}

	state := resp.ExecutionBadBlocks[0]

	data, err := d.store.GetExecutionBadBlock(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get execution bad block from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get execution bad block", http.StatusInternalServerError)

		return
	}

	if err := setResponseCompression(w, r, data); err != nil {
		d.writeJSONError(w, err.Error(), http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func setResponseCompression(w http.ResponseWriter, r *http.Request, data *[]byte) error {
	compression := r.Header.Get("Accept-Encoding")
	if compression == "" {
		// If no compression is specified, directly write the data without compression.
		return nil
	}

	if strings.Contains(compression, "gzip") {
		w.Header().Set("Content-Encoding", "gzip")

		var b bytes.Buffer

		gz := gzip.NewWriter(&b)

		if _, err := gz.Write(*data); err != nil {
			return err
		}

		if err := gz.Close(); err != nil {
			return err
		}

		*data = b.Bytes()

		return nil
	}

	if strings.Contains(compression, "snappy") {
		w.Header().Set("Content-Encoding", "snappy")

		var b bytes.Buffer

		df := snappy.NewWriter(&b)

		if _, err := df.Write(*data); err != nil {
			return err
		}

		if err := df.Close(); err != nil {
			return err
		}

		*data = b.Bytes()

		return nil
	}

	return nil
}

func (d *ObjectDownloader) writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		d.log.WithError(err).Error("Failed to write error response")
	}
}
