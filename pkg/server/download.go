package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/mime"
	"github.com/ethpandaops/tracoor/pkg/proto/tracoor/indexer"
	tStore "github.com/ethpandaops/tracoor/pkg/store"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ObjectDownloader struct {
	log        logrus.FieldLogger
	store      tStore.Store
	mux        *runtime.ServeMux
	indexer    indexer.IndexerClient
	grpcConn   string
	grpcOpts   []grpc.DialOption
	compressor *compression.Compressor
}

func NewObjectDownloader(log logrus.FieldLogger, store tStore.Store, mux *runtime.ServeMux, grpcConn string, grpcOpts []grpc.DialOption) *ObjectDownloader {
	return &ObjectDownloader{
		log:        log,
		store:      store,
		mux:        mux,
		grpcConn:   grpcConn,
		grpcOpts:   grpcOpts,
		compressor: compression.NewCompressor(),
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

	if err := d.mux.HandlePath("GET", "/download/beacon_block/{id}", d.beaconBlockHandler); err != nil {
		return fmt.Errorf("failed to register beacon block download handler: %v", err)
	}

	if err := d.mux.HandlePath("GET", "/download/beacon_bad_block/{id}", d.beaconBadBlockHandler); err != nil {
		return fmt.Errorf("failed to register beacon bad block download handler: %v", err)
	}

	if err := d.mux.HandlePath("GET", "/download/beacon_bad_blob/{id}", d.beaconBadBlobHandler); err != nil {
		return fmt.Errorf("failed to register beacon bad blob download handler: %v", err)
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

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetBeaconStateURL(ctx, state.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for beacon state ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetBeaconState(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon state from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get beacon state", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(state.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeOctet))

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) beaconBlockHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListBeaconBlock(ctx, &indexer.ListBeaconBlockRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list beacon blocks for ID %s", id)

		d.writeJSONError(w, "Failed to list beacon blocks", http.StatusInternalServerError)

		return
	}

	if len(resp.BeaconBlocks) == 0 {
		d.writeJSONError(w, "No beacon blocks found", http.StatusNotFound)

		return
	}

	if len(resp.BeaconBlocks) > 1 {
		d.writeJSONError(w, "More than one beacon block found", http.StatusInternalServerError)

		return
	}

	block := resp.BeaconBlocks[0]

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetBeaconBlockURL(ctx, block.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for beacon block ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetBeaconBlock(ctx, block.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon block from store for ID %s from %s", id, block.Location.Value)

		d.writeJSONError(w, "Failed to get beacon block", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(block.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeOctet))

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) beaconBadBlockHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListBeaconBadBlock(ctx, &indexer.ListBeaconBadBlockRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list beacon bad blocks for ID %s", id)

		d.writeJSONError(w, "Failed to list beacon bad blocks", http.StatusInternalServerError)

		return
	}

	if len(resp.BeaconBadBlocks) == 0 {
		d.writeJSONError(w, "No beacon bad blocks found", http.StatusNotFound)

		return
	}

	if len(resp.BeaconBadBlocks) > 1 {
		d.writeJSONError(w, "More than one beacon bad block found", http.StatusInternalServerError)

		return
	}

	block := resp.BeaconBadBlocks[0]

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetBeaconBadBlockURL(ctx, block.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for beacon bad block ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetBeaconBadBlock(ctx, block.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon bad block from store for ID %s from %s", id, block.Location.Value)

		d.writeJSONError(w, "Failed to get beacon bad block", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(block.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeOctet))

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) beaconBadBlobHandler(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	ctx := r.Context()

	id := pathParams["id"]
	if id == "" {
		d.writeJSONError(w, "No ID provided", http.StatusBadRequest)

		return
	}

	resp, err := d.indexer.ListBeaconBadBlob(ctx, &indexer.ListBeaconBadBlobRequest{
		Id: id,
		Pagination: &indexer.PaginationCursor{
			Limit: 1,
		},
	})
	if err != nil {
		d.log.WithError(err).Errorf("Failed to list beacon bad blobs for ID %s", id)

		d.writeJSONError(w, "Failed to list beacon bad blobs", http.StatusInternalServerError)

		return
	}

	if len(resp.BeaconBadBlobs) == 0 {
		d.writeJSONError(w, "No beacon bad blobs found", http.StatusNotFound)

		return
	}

	if len(resp.BeaconBadBlobs) > 1 {
		d.writeJSONError(w, "More than one beacon bad blob found", http.StatusInternalServerError)

		return
	}

	blob := resp.BeaconBadBlobs[0]

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetBeaconBadBlobURL(ctx, blob.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for beacon bad blob ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetBeaconBadBlob(ctx, blob.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get beacon bad blob from store for ID %s from %s", id, blob.Location.Value)

		d.writeJSONError(w, "Failed to get beacon bad blob", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(blob.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeOctet))

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

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetExecutionBlockTraceURL(ctx, state.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for block trace ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetExecutionBlockTrace(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get execution block trace from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get execution block trace", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(state.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeJSON))

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

	if d.store.PreferURLs() {
		var itemURL string

		itemURL, err = d.store.GetExecutionBadBlockURL(ctx, state.Location.Value, 3600)
		if err != nil {
			d.log.WithError(err).Errorf("Failed to get URL for bad block ID %s", id)
			d.writeJSONError(w, "Failed to get URL for item", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, itemURL, http.StatusTemporaryRedirect)

		return
	}

	data, err := d.store.GetExecutionBadBlock(ctx, state.Location.Value)
	if err != nil {
		d.log.WithError(err).Errorf("Failed to get execution bad block from store for ID %s from %s", id, state.Location.Value)

		d.writeJSONError(w, "Failed to get execution bad block", http.StatusInternalServerError)

		return
	}

	algo, err := compression.GetCompressionAlgorithm(state.Location.Value)
	if err == nil {
		w.Header().Set("Content-Encoding", algo.ContentEncoding)
	}

	w.Header().Set("Content-Type", string(mime.ContentTypeJSON))

	_, err = w.Write(*data)
	if err != nil {
		d.writeJSONError(w, "Failed to write response", http.StatusInternalServerError)
	}
}

func (d *ObjectDownloader) writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", string(mime.ContentTypeJSON))

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		d.log.WithError(err).Error("Failed to write error response")
	}
}
