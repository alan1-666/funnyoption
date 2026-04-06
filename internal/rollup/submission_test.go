package rollup

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildShadowBatchSubmissionBundleReady(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)

	bundle, err := BuildShadowBatchSubmissionBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildShadowBatchSubmissionBundle returned error: %v", err)
	}
	if bundle.Status != SubmissionStatusReady {
		t.Fatalf("status = %s, want %s", bundle.Status, SubmissionStatusReady)
	}
	if !bundle.ReadyForAcceptance {
		t.Fatalf("expected bundle to be ready for acceptance")
	}
	if len(bundle.Blockers) != 0 {
		t.Fatalf("expected no blockers, got %+v", bundle.Blockers)
	}
	if bundle.RecordBatchMetadataCall.FunctionName != "recordBatchMetadata" {
		t.Fatalf("unexpected record call function: %s", bundle.RecordBatchMetadataCall.FunctionName)
	}
	if bundle.AcceptVerifiedBatchCall.FunctionName != FunnyRollupCoreAcceptVerifiedBatchMethod {
		t.Fatalf("unexpected accept call function: %s", bundle.AcceptVerifiedBatchCall.FunctionName)
	}
	if bundle.Batch.BatchID != targetBatch.BatchID {
		t.Fatalf("batch_id = %d, want %d", bundle.Batch.BatchID, targetBatch.BatchID)
	}

	recordValues, err := unpackCallValues(bundle.RecordBatchMetadataCall.Calldata, "recordBatchMetadata")
	if err != nil {
		t.Fatalf("unpackCallValues(record) returned error: %v", err)
	}
	if len(recordValues) != 4 {
		t.Fatalf("expected 4 record arguments, got %d", len(recordValues))
	}
	if got, ok := recordValues[0].(uint64); !ok || got != uint64(targetBatch.BatchID) {
		t.Fatalf("record batchId = %#v, want %d", recordValues[0], targetBatch.BatchID)
	}

	acceptValues, err := unpackCallValues(bundle.AcceptVerifiedBatchCall.Calldata, FunnyRollupCoreAcceptVerifiedBatchMethod)
	if err != nil {
		t.Fatalf("unpackCallValues(accept) returned error: %v", err)
	}
	if len(acceptValues) != 4 {
		t.Fatalf("expected 4 accept arguments, got %d", len(acceptValues))
	}
	authStatuses, ok := acceptValues[2].([]uint8)
	if !ok {
		t.Fatalf("accept auth statuses type = %T, want []uint8", acceptValues[2])
	}
	if len(authStatuses) != 1 || authStatuses[0] != uint8(SolidityAuthJoinStatusJoined) {
		t.Fatalf("unexpected auth statuses: %+v", authStatuses)
	}
	proofBytes, ok := acceptValues[3].([]byte)
	if !ok {
		t.Fatalf("accept verifierProof type = %T, want []byte", acceptValues[3])
	}
	if !bytes.Equal(proofBytes, common.FromHex(bundle.VerifierArtifactBundle.VerifierInterface.Calldata.Proof)) {
		t.Fatalf("accept verifierProof bytes do not match exported proof")
	}
}

func TestBuildShadowBatchSubmissionBundleBlockedAuth(t *testing.T) {
	batch := legacyCompatVerifierGateBatch(t)

	bundle, err := BuildShadowBatchSubmissionBundle(nil, batch)
	if err != nil {
		t.Fatalf("BuildShadowBatchSubmissionBundle returned error: %v", err)
	}
	if bundle.Status != SubmissionStatusBlockedAuth {
		t.Fatalf("status = %s, want %s", bundle.Status, SubmissionStatusBlockedAuth)
	}
	if bundle.ReadyForAcceptance {
		t.Fatalf("expected blocked auth bundle to stay not ready for acceptance")
	}
	if len(bundle.Blockers) == 0 {
		t.Fatalf("expected blocked auth bundle to surface blockers")
	}
	if bundle.VerifierArtifactBundle.AcceptanceContract.ReadyForAcceptance {
		t.Fatalf("expected acceptance contract to stay non-ready")
	}
}

func TestBuildSubmissionHashIsDeterministic(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)
	bundle, err := BuildShadowBatchSubmissionBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildShadowBatchSubmissionBundle returned error: %v", err)
	}

	firstJSON, firstHash, err := buildSubmissionHash(bundle)
	if err != nil {
		t.Fatalf("buildSubmissionHash first run returned error: %v", err)
	}
	secondJSON, secondHash, err := buildSubmissionHash(bundle)
	if err != nil {
		t.Fatalf("buildSubmissionHash second run returned error: %v", err)
	}
	if firstJSON != secondJSON {
		t.Fatalf("submission JSON changed across runs")
	}
	if firstHash != secondHash {
		t.Fatalf("submission hash changed across runs: %s vs %s", firstHash, secondHash)
	}
}

func unpackCallValues(calldataHex, methodName string) ([]interface{}, error) {
	method, ok := funnyRollupCoreSubmissionABI.Methods[methodName]
	if !ok {
		return nil, nil
	}
	raw := common.FromHex(calldataHex)
	return method.Inputs.UnpackValues(raw[4:])
}
