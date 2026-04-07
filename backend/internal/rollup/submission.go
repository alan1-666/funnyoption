package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const funnyRollupCoreSubmissionABIJSON = `[
  {
    "type":"function",
    "name":"recordBatchMetadata",
    "inputs":[
      {"name":"batchId","type":"uint64"},
      {"name":"batchDataHash","type":"bytes32"},
      {"name":"prevStateRoot","type":"bytes32"},
      {"name":"nextStateRoot","type":"bytes32"}
    ]
  },
  {
    "type":"function",
    "name":"publishBatchData",
    "inputs":[
      {"name":"batchId","type":"uint64"},
      {"name":"batchData","type":"bytes"}
    ]
  },
  {
    "type":"function",
    "name":"acceptVerifiedBatch",
    "inputs":[
      {
        "name":"publicInputs",
        "type":"tuple",
        "components":[
          {"name":"batchId","type":"uint64"},
          {"name":"firstSequenceNo","type":"uint64"},
          {"name":"lastSequenceNo","type":"uint64"},
          {"name":"entryCount","type":"uint64"},
          {"name":"batchDataHash","type":"bytes32"},
          {"name":"prevStateRoot","type":"bytes32"},
          {"name":"balancesRoot","type":"bytes32"},
          {"name":"ordersRoot","type":"bytes32"},
          {"name":"positionsFundingRoot","type":"bytes32"},
          {"name":"withdrawalsRoot","type":"bytes32"},
          {"name":"nextStateRoot","type":"bytes32"},
          {"name":"conservationHash","type":"bytes32"}
        ]
      },
      {
        "name":"metadataSubset",
        "type":"tuple",
        "components":[
          {"name":"batchId","type":"uint64"},
          {"name":"batchDataHash","type":"bytes32"},
          {"name":"prevStateRoot","type":"bytes32"},
          {"name":"nextStateRoot","type":"bytes32"}
        ]
      },
      {"name":"authStatuses","type":"uint8[]"},
      {"name":"verifierProof","type":"bytes"}
    ]
  }
]`

var funnyRollupCoreSubmissionABI = mustContractABI(funnyRollupCoreSubmissionABIJSON)

type rollupCorePublicInputsCall struct {
	BatchID              uint64      `abi:"batchId"`
	FirstSequenceNo      uint64      `abi:"firstSequenceNo"`
	LastSequenceNo       uint64      `abi:"lastSequenceNo"`
	EntryCount           uint64      `abi:"entryCount"`
	BatchDataHash        common.Hash `abi:"batchDataHash"`
	PrevStateRoot        common.Hash `abi:"prevStateRoot"`
	BalancesRoot         common.Hash `abi:"balancesRoot"`
	OrdersRoot           common.Hash `abi:"ordersRoot"`
	PositionsFundingRoot common.Hash `abi:"positionsFundingRoot"`
	WithdrawalsRoot      common.Hash `abi:"withdrawalsRoot"`
	NextStateRoot        common.Hash `abi:"nextStateRoot"`
	ConservationHash     common.Hash `abi:"conservationHash"`
}

type rollupCoreMetadataSubsetCall struct {
	BatchID       uint64      `abi:"batchId"`
	BatchDataHash common.Hash `abi:"batchDataHash"`
	PrevStateRoot common.Hash `abi:"prevStateRoot"`
	NextStateRoot common.Hash `abi:"nextStateRoot"`
}

func BuildShadowBatchSubmissionBundle(history []StoredBatch, batch StoredBatch) (ShadowBatchSubmissionBundle, error) {
	shadowBatchContract, err := BuildShadowBatchContract(batch)
	if err != nil {
		return ShadowBatchSubmissionBundle{}, err
	}
	artifactBundle, err := BuildVerifierArtifactBundle(history, batch)
	if err != nil {
		return ShadowBatchSubmissionBundle{}, err
	}
	recordCall, err := buildRecordBatchMetadataCall(artifactBundle.AcceptanceContract.L1BatchMetadata)
	if err != nil {
		return ShadowBatchSubmissionBundle{}, err
	}
	publishCall, err := buildPublishBatchDataCall(batch)
	if err != nil {
		return ShadowBatchSubmissionBundle{}, err
	}
	acceptCall, err := buildAcceptVerifiedBatchCall(
		artifactBundle.AcceptanceContract.SolidityExport.Calldata,
		artifactBundle.VerifierInterface.Calldata.Proof,
	)
	if err != nil {
		return ShadowBatchSubmissionBundle{}, err
	}
	status, blockers := buildSubmissionStatus(artifactBundle.AcceptanceContract.AuthStatuses)

	return ShadowBatchSubmissionBundle{
		SubmissionVersion:       SubmissionEncodingVersion,
		Status:                  status,
		ReadyForAcceptance:      artifactBundle.AcceptanceContract.ReadyForAcceptance,
		Batch:                   buildSubmissionBatchSummary(batch),
		ShadowBatchContract:     shadowBatchContract,
		VerifierArtifactBundle:  artifactBundle,
		RecordBatchMetadataCall: recordCall,
		PublishBatchDataCall:    publishCall,
		AcceptVerifiedBatchCall: acceptCall,
		Blockers:                blockers,
		Limitations: []string{
			"this submission bundle is a deterministic shadow-to-onchain export lane; it does not switch production truth away from SQL/Kafka settlement or direct-vault claim.",
			"recordBatchMetadata(...), publishBatchData(...), and acceptVerifiedBatch(...) payloads are chain-ready calldata exports, but this tranche does not broadcast live transactions.",
			"READY only means auth statuses are fully JOINED and the bundle is eligible for the current verifier-gated acceptance path; it is not yet a production Mode B finality claim.",
		},
	}, nil
}

func DecodeSubmissionBundle(data string) (ShadowBatchSubmissionBundle, error) {
	if strings.TrimSpace(data) == "" {
		return ShadowBatchSubmissionBundle{}, fmt.Errorf("submission data is required")
	}
	var bundle ShadowBatchSubmissionBundle
	if err := json.Unmarshal([]byte(data), &bundle); err != nil {
		return ShadowBatchSubmissionBundle{}, fmt.Errorf("decode submission bundle: %w", err)
	}
	return bundle, nil
}

func DecodeStoredSubmissionBundle(submission StoredSubmission) (ShadowBatchSubmissionBundle, error) {
	return DecodeSubmissionBundle(submission.SubmissionData)
}

func buildSubmissionBatchSummary(batch StoredBatch) SubmissionBatchSummary {
	return SubmissionBatchSummary{
		BatchID:              batch.BatchID,
		EncodingVersion:      strings.TrimSpace(batch.EncodingVersion),
		FirstSequence:        batch.FirstSequence,
		LastSequence:         batch.LastSequence,
		EntryCount:           batch.EntryCount,
		InputHash:            strings.TrimSpace(batch.InputHash),
		BatchDataHash:        canonicalBatchDataHash(batch),
		PrevStateRoot:        defaultStateRoot(batch.PrevStateRoot),
		BalancesRoot:         defaultComponentRoot(batch.BalancesRoot, ZeroBalancesRoot()),
		OrdersRoot:           defaultComponentRoot(batch.OrdersRoot, hashStrings("shadow", "orders", ZeroNonceRoot(), ZeroOpenOrdersRoot())),
		PositionsFundingRoot: defaultComponentRoot(batch.PositionsFundingRoot, hashStrings("shadow", "positions_funding", hashStrings("shadow", "positions", "leafs", "empty"), ZeroMarketFundingRoot(), ZeroInsuranceRoot())),
		WithdrawalsRoot:      defaultComponentRoot(batch.WithdrawalsRoot, ZeroWithdrawalsRoot()),
		NextStateRoot:        defaultStateRoot(batch.StateRoot),
		ConservationHash:     buildBatchConservationHash(batch),
	}
}

func buildBatchConservationHash(batch StoredBatch) string {
	if input, err := DecodeBatchInput(batch.InputData); err == nil {
		if record, err := BuildConservationRecord(batch.BatchID, input.Entries); err == nil {
			return record.ConservationHash
		}
	}
	return ZeroConservationHash()
}

func buildSubmissionStatus(authStatuses []VerifierAcceptanceAuthStatus) (string, []string) {
	blockers := make([]string, 0)
	for _, authStatus := range authStatuses {
		if authStatus.JoinStatus == VerifierAuthJoinSatisfied {
			continue
		}
		blockers = append(blockers, fmt.Sprintf("sequence=%d source_ref=%s join_status=%s", authStatus.Sequence, authStatus.SourceRef, authStatus.JoinStatus))
	}
	if len(blockers) > 0 {
		return SubmissionStatusBlockedAuth, blockers
	}
	return SubmissionStatusReady, nil
}

func buildRecordBatchMetadataCall(metadata L1BatchMetadata) (RollupContractCall, error) {
	method := funnyRollupCoreSubmissionABI.Methods["recordBatchMetadata"]
	batchID, err := solidityUint64FromInt64(metadata.BatchID, "l1_batch_metadata.batch_id")
	if err != nil {
		return RollupContractCall{}, err
	}
	batchDataHash, err := solidityHashFromBytes32(metadata.BatchDataHash, "l1_batch_metadata.batch_data_hash")
	if err != nil {
		return RollupContractCall{}, err
	}
	prevStateRoot, err := solidityHashFromBytes32(metadata.PrevStateRoot, "l1_batch_metadata.prev_state_root")
	if err != nil {
		return RollupContractCall{}, err
	}
	nextStateRoot, err := solidityHashFromBytes32(metadata.NextStateRoot, "l1_batch_metadata.next_state_root")
	if err != nil {
		return RollupContractCall{}, err
	}
	data, err := method.Inputs.Pack(batchID, batchDataHash, prevStateRoot, nextStateRoot)
	if err != nil {
		return RollupContractCall{}, fmt.Errorf("pack recordBatchMetadata calldata: %w", err)
	}
	return RollupContractCall{
		ContractName: FunnyRollupCoreContractName,
		ContractPath: FunnyRollupCoreContractPath,
		FunctionName: "recordBatchMetadata",
		Selector:     "0x" + hex.EncodeToString(method.ID),
		Calldata:     "0x" + hex.EncodeToString(append(method.ID, data...)),
	}, nil
}

func buildPublishBatchDataCall(batch StoredBatch) (RollupContractCall, error) {
	method := funnyRollupCoreSubmissionABI.Methods["publishBatchData"]
	batchID, err := solidityUint64FromInt64(batch.BatchID, "batch.batch_id")
	if err != nil {
		return RollupContractCall{}, err
	}
	batchData := []byte(batch.InputData)
	data, err := method.Inputs.Pack(batchID, batchData)
	if err != nil {
		return RollupContractCall{}, fmt.Errorf("pack publishBatchData calldata: %w", err)
	}
	return RollupContractCall{
		ContractName: FunnyRollupCoreContractName,
		ContractPath: FunnyRollupCoreContractPath,
		FunctionName: "publishBatchData",
		Selector:     "0x" + hex.EncodeToString(method.ID),
		Calldata:     "0x" + hex.EncodeToString(append(method.ID, data...)),
	}, nil
}

func buildAcceptVerifiedBatchCall(calldata VerifierAcceptanceSolidityCalldata, verifierProof string) (RollupContractCall, error) {
	method := funnyRollupCoreSubmissionABI.Methods[FunnyRollupCoreAcceptVerifiedBatchMethod]
	publicInputs, err := buildSubmissionPublicInputsCall(calldata.PublicInputs)
	if err != nil {
		return RollupContractCall{}, err
	}
	metadataSubset, err := buildSubmissionMetadataSubsetCall(calldata.MetadataSubset)
	if err != nil {
		return RollupContractCall{}, err
	}
	authStatuses := make([]uint8, 0, len(calldata.AuthStatuses))
	for _, authStatus := range calldata.AuthStatuses {
		authStatuses = append(authStatuses, uint8(authStatus))
	}
	proofBytes := common.FromHex(strings.TrimSpace(verifierProof))
	data, err := method.Inputs.Pack(publicInputs, metadataSubset, authStatuses, proofBytes)
	if err != nil {
		return RollupContractCall{}, fmt.Errorf("pack acceptVerifiedBatch calldata: %w", err)
	}
	return RollupContractCall{
		ContractName: FunnyRollupCoreContractName,
		ContractPath: FunnyRollupCoreContractPath,
		FunctionName: FunnyRollupCoreAcceptVerifiedBatchMethod,
		Selector:     "0x" + hex.EncodeToString(method.ID),
		Calldata:     "0x" + hex.EncodeToString(append(method.ID, data...)),
	}, nil
}

func buildSubmissionPublicInputsCall(publicInputs SolidityVerifierPublicInputs) (rollupCorePublicInputsCall, error) {
	batchDataHash, err := solidityHashFromBytes32(publicInputs.BatchDataHash, "public_inputs.batch_data_hash")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	prevStateRoot, err := solidityHashFromBytes32(publicInputs.PrevStateRoot, "public_inputs.prev_state_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	balancesRoot, err := solidityHashFromBytes32(publicInputs.BalancesRoot, "public_inputs.balances_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	ordersRoot, err := solidityHashFromBytes32(publicInputs.OrdersRoot, "public_inputs.orders_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	positionsFundingRoot, err := solidityHashFromBytes32(publicInputs.PositionsFundingRoot, "public_inputs.positions_funding_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	withdrawalsRoot, err := solidityHashFromBytes32(publicInputs.WithdrawalsRoot, "public_inputs.withdrawals_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	nextStateRoot, err := solidityHashFromBytes32(publicInputs.NextStateRoot, "public_inputs.next_state_root")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	conservationHash, err := solidityHashFromBytes32(publicInputs.ConservationHash, "public_inputs.conservation_hash")
	if err != nil {
		return rollupCorePublicInputsCall{}, err
	}
	return rollupCorePublicInputsCall{
		BatchID:              publicInputs.BatchID,
		FirstSequenceNo:      publicInputs.FirstSequence,
		LastSequenceNo:       publicInputs.LastSequence,
		EntryCount:           publicInputs.EntryCount,
		BatchDataHash:        batchDataHash,
		PrevStateRoot:        prevStateRoot,
		BalancesRoot:         balancesRoot,
		OrdersRoot:           ordersRoot,
		PositionsFundingRoot: positionsFundingRoot,
		WithdrawalsRoot:      withdrawalsRoot,
		NextStateRoot:        nextStateRoot,
		ConservationHash:     conservationHash,
	}, nil
}

func buildSubmissionMetadataSubsetCall(metadata SolidityL1BatchMetadata) (rollupCoreMetadataSubsetCall, error) {
	batchDataHash, err := solidityHashFromBytes32(metadata.BatchDataHash, "metadata_subset.batch_data_hash")
	if err != nil {
		return rollupCoreMetadataSubsetCall{}, err
	}
	prevStateRoot, err := solidityHashFromBytes32(metadata.PrevStateRoot, "metadata_subset.prev_state_root")
	if err != nil {
		return rollupCoreMetadataSubsetCall{}, err
	}
	nextStateRoot, err := solidityHashFromBytes32(metadata.NextStateRoot, "metadata_subset.next_state_root")
	if err != nil {
		return rollupCoreMetadataSubsetCall{}, err
	}
	return rollupCoreMetadataSubsetCall{
		BatchID:       metadata.BatchID,
		BatchDataHash: batchDataHash,
		PrevStateRoot: prevStateRoot,
		NextStateRoot: nextStateRoot,
	}, nil
}

func buildSubmissionHash(bundle ShadowBatchSubmissionBundle) (string, string, error) {
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return "", "", err
	}
	return string(encoded), hashStrings("shadow", "submission_bundle", string(encoded)), nil
}

func mustContractABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}
