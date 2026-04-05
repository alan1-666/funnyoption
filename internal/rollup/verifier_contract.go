package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sharedauth "funnyoption/internal/shared/auth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	solidityBytes32ABIType = mustABISolidityType("bytes32")
	solidityBytesABIType   = mustABISolidityType("bytes")
	solidityUint64ABIType  = mustABISolidityType("uint64")
	solidityUint8ArrayType = mustABISolidityType("uint8[]")
)

func BuildVerifierGateBatchContract(history []StoredBatch, batch StoredBatch) (VerifierGateBatchContract, error) {
	authProof, err := BuildVerifierAuthProofContract(history, batch)
	if err != nil {
		return VerifierGateBatchContract{}, err
	}
	return VerifierGateBatchContract{
		PublicInputs:    BuildShadowBatchPublicInputs(batch),
		L1BatchMetadata: BuildL1BatchMetadata(batch),
		AuthProof:       authProof,
		Limitations: []string{
			"auth_proof only binds canonical V2 authorization_ref joins onto the stable shadow-batch-v1 batch/public-input surface; it does not change the public-input shape.",
			"ready_for_verifier only means each verifier-eligible NONCE_ADVANCED in the target batch can be joined to canonical TRADING_KEY_AUTHORIZED witness material; it is not a proof, verifier verdict, or production Mode B claim.",
			"the later FunnyRollupCore acceptance hook consumes this same boundary in Foundry-only form, but the repo still does not have a full prover, full verifier, or production Mode B state-root truth.",
		},
	}, nil
}

func BuildVerifierArtifactBundle(history []StoredBatch, batch StoredBatch) (VerifierArtifactBundle, error) {
	acceptanceContract, err := BuildVerifierStateRootAcceptanceContract(history, batch)
	if err != nil {
		return VerifierArtifactBundle{}, err
	}

	authProofDigest, err := buildVerifierAuthProofDigestContract(acceptanceContract.SolidityExport)
	if err != nil {
		return VerifierArtifactBundle{}, err
	}
	verifierGateDigest, err := buildVerifierGateDigestContract(
		acceptanceContract.SolidityExport.Calldata.PublicInputs,
		authProofDigest.AuthProofHash,
	)
	if err != nil {
		return VerifierArtifactBundle{}, err
	}
	verifierInterface, err := buildVerifierInterfaceSolidityExport(
		acceptanceContract.SolidityExport.Calldata.PublicInputs,
		verifierGateDigest,
	)
	if err != nil {
		return VerifierArtifactBundle{}, err
	}

	return VerifierArtifactBundle{
		AcceptanceContract: acceptanceContract,
		AuthProofDigest:    authProofDigest,
		VerifierGateDigest: verifierGateDigest,
		VerifierInterface:  verifierInterface,
		Limitations: []string{
			"the verifier artifact bundle reuses BuildVerifierStateRootAcceptanceContract(...).solidity_export and adds only the deterministic authProofHash/verifierGateHash contract needed by a later prover/verifier worker.",
			"the verifier export now pins one explicit proof/public-signal schema plus one explicit inner proofData-v1 payload carrying one batch-specific fixed-vk Groth16/BN254 proof artifact derived from the actual outer signals for the current FunnyRollupVerifier contract, but it is still not a general prover pipeline or production Mode B truth switch.",
			fmt.Sprintf(
				"proofTypeHash identifies one full verifier-facing proof contract, not just a proving-family label: proving system + curve, bytes32 public-signal lifting rule, exact circuit/verifying-key lane, and proofBytes ABI codec; the first real lane is keccak256(%q) and keeps proof bytes inside proofData-v1 as abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c).",
				FunnyRollupBatchVerifierFirstGroth16ProofType,
			),
			"proofData-v2 is not required for the first fixed-vk Groth16 lane; it only becomes necessary if verifier-relevant metadata such as vk/circuit/aggregation selection must travel separately from proofTypeHash + proofBytes.",
			"shadow-batch-v1 public_inputs stay unchanged; auth witness rows remain compressed into authProofHash for this narrow tranche, and public signals stay limited to batchEncodingHash/authProofHash/verifierGateHash.",
		},
	}, nil
}

func BuildVerifierStateRootAcceptanceContract(history []StoredBatch, batch StoredBatch) (VerifierStateRootAcceptanceContract, error) {
	gateBatch, err := BuildVerifierGateBatchContract(history, batch)
	if err != nil {
		return VerifierStateRootAcceptanceContract{}, err
	}

	authStatuses := make([]VerifierAcceptanceAuthStatus, 0, len(gateBatch.AuthProof.NonceAuthorizations))
	readyForAcceptance := gateBatch.AuthProof.ReadyForVerifier
	for _, nonceAuthorization := range gateBatch.AuthProof.NonceAuthorizations {
		authStatuses = append(authStatuses, VerifierAcceptanceAuthStatus{
			Sequence:   nonceAuthorization.Sequence,
			SourceRef:  nonceAuthorization.SourceRef,
			JoinStatus: nonceAuthorization.JoinStatus,
		})
		if nonceAuthorization.JoinStatus != VerifierAuthJoinSatisfied {
			readyForAcceptance = false
		}
	}

	solidityExport, err := buildVerifierAcceptanceSolidityExport(
		gateBatch.PublicInputs,
		gateBatch.L1BatchMetadata,
		authStatuses,
	)
	if err != nil {
		return VerifierStateRootAcceptanceContract{}, err
	}

	return VerifierStateRootAcceptanceContract{
		PublicInputs:       gateBatch.PublicInputs,
		L1BatchMetadata:    gateBatch.L1BatchMetadata,
		ReadyForAcceptance: readyForAcceptance,
		AuthStatuses:       authStatuses,
		SolidityExport:     solidityExport,
		Limitations: []string{
			"ready_for_acceptance only means every target-batch nonce auth row currently materialized for the verifier gate is JOINED; it is not itself a proof or verifier verdict.",
			"the acceptance projection keeps the stable shadow-batch-v1 public-input shape and now also freezes the Go -> Solidity acceptVerifiedBatch calldata boundary, including enum ordinals for auth_statuses.",
			"wallet EIP-712 verification, Ed25519 order-signature verification, and production withdrawal truth remain follow-up prover/verifier work.",
		},
	}, nil
}

func buildVerifierAuthProofDigestContract(solidityExport VerifierAcceptanceSolidityExport) (VerifierAuthProofDigestContract, error) {
	encodedStatuses := make([]uint8, 0, len(solidityExport.Calldata.AuthStatuses))
	for _, authStatus := range solidityExport.Calldata.AuthStatuses {
		encodedStatuses = append(encodedStatuses, uint8(authStatus))
	}
	packed, err := abi.Arguments{{Type: solidityUint8ArrayType}}.Pack(encodedStatuses)
	if err != nil {
		return VerifierAuthProofDigestContract{}, err
	}
	return VerifierAuthProofDigestContract{
		HashFunction:  "keccak256(abi.encode(authStatuses))",
		ArgumentType:  "uint8[]",
		AuthStatuses:  append([]SolidityAuthJoinStatus(nil), solidityExport.Calldata.AuthStatuses...),
		AuthProofHash: crypto.Keccak256Hash(packed).Hex(),
	}, nil
}

func buildVerifierGateDigestContract(publicInputs SolidityVerifierPublicInputs, authProofHash string) (VerifierGateDigestContract, error) {
	batchEncodingHash := crypto.Keccak256Hash([]byte(BatchEncodingVersion)).Hex()
	batchDataHash, err := solidityHashFromBytes32(publicInputs.BatchDataHash, "public_inputs.batch_data_hash")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	prevStateRoot, err := solidityHashFromBytes32(publicInputs.PrevStateRoot, "public_inputs.prev_state_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	balancesRoot, err := solidityHashFromBytes32(publicInputs.BalancesRoot, "public_inputs.balances_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	ordersRoot, err := solidityHashFromBytes32(publicInputs.OrdersRoot, "public_inputs.orders_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	positionsFundingRoot, err := solidityHashFromBytes32(publicInputs.PositionsFundingRoot, "public_inputs.positions_funding_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	withdrawalsRoot, err := solidityHashFromBytes32(publicInputs.WithdrawalsRoot, "public_inputs.withdrawals_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	nextStateRoot, err := solidityHashFromBytes32(publicInputs.NextStateRoot, "public_inputs.next_state_root")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}
	normalizedAuthProofHash, err := solidityHashFromBytes32(authProofHash, "auth_proof_hash")
	if err != nil {
		return VerifierGateDigestContract{}, err
	}

	packed, err := abi.Arguments{
		{Type: solidityBytes32ABIType},
		{Type: solidityUint64ABIType},
		{Type: solidityUint64ABIType},
		{Type: solidityUint64ABIType},
		{Type: solidityUint64ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
	}.Pack(
		crypto.Keccak256Hash([]byte(BatchEncodingVersion)),
		publicInputs.BatchID,
		publicInputs.FirstSequence,
		publicInputs.LastSequence,
		publicInputs.EntryCount,
		batchDataHash,
		prevStateRoot,
		balancesRoot,
		ordersRoot,
		positionsFundingRoot,
		withdrawalsRoot,
		nextStateRoot,
		normalizedAuthProofHash,
	)
	if err != nil {
		return VerifierGateDigestContract{}, err
	}

	return VerifierGateDigestContract{
		EncodingVersion:     BatchEncodingVersion,
		EncodingVersionHash: batchEncodingHash,
		HashFunction:        "keccak256(abi.encode(batchEncodingHash, batchId, firstSequenceNo, lastSequenceNo, entryCount, batchDataHash, prevStateRoot, balancesRoot, ordersRoot, positionsFundingRoot, withdrawalsRoot, nextStateRoot, authProofHash))",
		FieldOrder: []VerifierGateDigestField{
			{Name: "batchEncodingHash", Type: "bytes32"},
			{Name: "batchId", Type: "uint64"},
			{Name: "firstSequenceNo", Type: "uint64"},
			{Name: "lastSequenceNo", Type: "uint64"},
			{Name: "entryCount", Type: "uint64"},
			{Name: "batchDataHash", Type: "bytes32"},
			{Name: "prevStateRoot", Type: "bytes32"},
			{Name: "balancesRoot", Type: "bytes32"},
			{Name: "ordersRoot", Type: "bytes32"},
			{Name: "positionsFundingRoot", Type: "bytes32"},
			{Name: "withdrawalsRoot", Type: "bytes32"},
			{Name: "nextStateRoot", Type: "bytes32"},
			{Name: "authProofHash", Type: "bytes32"},
		},
		PublicInputs:     publicInputs,
		AuthProofHash:    authProofHash,
		VerifierGateHash: crypto.Keccak256Hash(packed).Hex(),
	}, nil
}

func buildVerifierInterfaceSolidityExport(publicInputs SolidityVerifierPublicInputs, verifierGateDigest VerifierGateDigestContract) (VerifierInterfaceSolidityExport, error) {
	proofSchemaHash := crypto.Keccak256Hash([]byte(FunnyRollupBatchVerifierProofSchemaVersion))
	publicSignalsVersionHash := crypto.Keccak256Hash([]byte(FunnyRollupBatchVerifierPublicSignalsV1))
	proofDataSchemaHash := crypto.Keccak256Hash([]byte(FunnyRollupBatchVerifierProofDataVersion))
	proofTypeHash := crypto.Keccak256Hash([]byte(FunnyRollupBatchVerifierProofVersion))
	normalizedBatchEncodingHash, err := solidityHashFromBytes32(verifierGateDigest.EncodingVersionHash, "batch_encoding_hash")
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	normalizedAuthProofHash, err := solidityHashFromBytes32(verifierGateDigest.AuthProofHash, "auth_proof_hash")
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	normalizedVerifierGateHash, err := solidityHashFromBytes32(verifierGateDigest.VerifierGateHash, "verifier_gate_hash")
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	groth16ProofBytes, groth16PublicInputs, groth16ProofTuple, err := buildBatchSpecificGroth16Proof(
		verifierGateDigest.EncodingVersionHash,
		verifierGateDigest.AuthProofHash,
		verifierGateDigest.VerifierGateHash,
	)
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	proofDataBytes, err := abi.Arguments{
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytesABIType},
	}.Pack(
		proofDataSchemaHash,
		proofTypeHash,
		normalizedBatchEncodingHash,
		normalizedAuthProofHash,
		normalizedVerifierGateHash,
		groth16ProofBytes,
	)
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	proofBytes, err := abi.Arguments{
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytes32ABIType},
		{Type: solidityBytesABIType},
	}.Pack(
		proofSchemaHash,
		publicSignalsVersionHash,
		normalizedBatchEncodingHash,
		normalizedAuthProofHash,
		normalizedVerifierGateHash,
		proofDataBytes,
	)
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}
	groth16Fixture, err := buildGroth16Fixture(groth16PublicInputs, groth16ProofTuple)
	if err != nil {
		return VerifierInterfaceSolidityExport{}, err
	}

	return VerifierInterfaceSolidityExport{
		ContractName:             FunnyRollupBatchVerifierContractName,
		ImplementationName:       FunnyRollupBatchVerifierImplementationName,
		ContractPath:             FunnyRollupBatchVerifierContractPath,
		FunctionName:             FunnyRollupBatchVerifierMethod,
		ContextType:              "FunnyRollupVerifierTypes.VerifierContext",
		PublicInputsType:         "FunnyRollupVerifierTypes.VerifierPublicInputs",
		PublicSignalsType:        "FunnyRollupVerifierTypes.ProofPublicSignals",
		PublicSignalsVersion:     FunnyRollupBatchVerifierPublicSignalsV1,
		PublicSignalsVersionHash: publicSignalsVersionHash.Hex(),
		PublicSignalsFieldOrder: []VerifierProofSchemaField{
			{Name: "batchEncodingHash", Type: "bytes32"},
			{Name: "authProofHash", Type: "bytes32"},
			{Name: "verifierGateHash", Type: "bytes32"},
		},
		ProofType:          "bytes",
		ProofSchemaVersion: FunnyRollupBatchVerifierProofSchemaVersion,
		ProofSchemaHash:    proofSchemaHash.Hex(),
		ProofFieldOrder: []VerifierProofSchemaField{
			{Name: "proofSchemaHash", Type: "bytes32"},
			{Name: "publicSignalsSchemaHash", Type: "bytes32"},
			{Name: "publicSignals.batchEncodingHash", Type: "bytes32"},
			{Name: "publicSignals.authProofHash", Type: "bytes32"},
			{Name: "publicSignals.verifierGateHash", Type: "bytes32"},
			{Name: "proofData", Type: "bytes"},
		},
		ProofDataSchemaVersion: FunnyRollupBatchVerifierProofDataVersion,
		ProofDataSchemaHash:    proofDataSchemaHash.Hex(),
		ProofDataFieldOrder: []VerifierProofSchemaField{
			{Name: "proofDataSchemaHash", Type: "bytes32"},
			{Name: "proofTypeHash", Type: "bytes32"},
			{Name: "batchEncodingHash", Type: "bytes32"},
			{Name: "authProofHash", Type: "bytes32"},
			{Name: "verifierGateHash", Type: "bytes32"},
			{Name: "proofBytes", Type: "bytes"},
		},
		ProofEncoding:     "abi.encode(proofSchemaHash, publicSignalsSchemaHash, publicSignals.batchEncodingHash, publicSignals.authProofHash, publicSignals.verifierGateHash, proofData)",
		ProofDataEncoding: "abi.encode(proofDataSchemaHash, proofTypeHash, publicSignals.batchEncodingHash, publicSignals.authProofHash, publicSignals.verifierGateHash, proofBytes)",
		ProofVersion:      FunnyRollupBatchVerifierProofVersion,
		ProofVersionHash:  proofTypeHash.Hex(),
		Groth16Fixture:    groth16Fixture,
		Calldata: VerifierInterfaceSolidityCalldata{
			Context: SolidityVerifierGateContext{
				BatchEncodingHash: verifierGateDigest.EncodingVersionHash,
				PublicInputs:      publicInputs,
				AuthProofHash:     verifierGateDigest.AuthProofHash,
				VerifierGateHash:  verifierGateDigest.VerifierGateHash,
			},
			PublicSignals: VerifierProofPublicSignalsCalldata{
				BatchEncodingHash: verifierGateDigest.EncodingVersionHash,
				AuthProofHash:     verifierGateDigest.AuthProofHash,
				VerifierGateHash:  verifierGateDigest.VerifierGateHash,
			},
			ProofDataFields: VerifierProofDataCalldata{
				ProofDataSchemaHash: proofDataSchemaHash.Hex(),
				ProofTypeHash:       proofTypeHash.Hex(),
				BatchEncodingHash:   verifierGateDigest.EncodingVersionHash,
				AuthProofHash:       verifierGateDigest.AuthProofHash,
				VerifierGateHash:    verifierGateDigest.VerifierGateHash,
				ProofBytes:          "0x" + hex.EncodeToString(groth16ProofBytes),
			},
			ProofData: "0x" + hex.EncodeToString(proofDataBytes),
			Proof:     "0x" + hex.EncodeToString(proofBytes),
		},
	}, nil
}

func buildGroth16Fixture(publicInputs []string, proofTuple VerifierGroth16ProofTuple) (VerifierGroth16Fixture, error) {
	if len(publicInputs) != 6 {
		return VerifierGroth16Fixture{}, fmt.Errorf("expected 6 Groth16 public inputs, got %d", len(publicInputs))
	}
	return VerifierGroth16Fixture{
		ProofBytesEncoding: "abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)",
		PublicInputFieldOrder: []VerifierProofSchemaField{
			{Name: "batchEncodingHashHi", Type: "uint256"},
			{Name: "batchEncodingHashLo", Type: "uint256"},
			{Name: "authProofHashHi", Type: "uint256"},
			{Name: "authProofHashLo", Type: "uint256"},
			{Name: "verifierGateHashHi", Type: "uint256"},
			{Name: "verifierGateHashLo", Type: "uint256"},
		},
		PublicInputs:    publicInputs,
		ProofTuple:      proofTuple,
		ExpectedVerdict: true,
	}, nil
}

func buildGroth16PublicInputsHex(batchEncodingHash, authProofHash, verifierGateHash string) ([]string, error) {
	batchEncoding, err := solidityHashFromBytes32(batchEncodingHash, "groth16.batch_encoding_hash")
	if err != nil {
		return nil, err
	}
	authProof, err := solidityHashFromBytes32(authProofHash, "groth16.auth_proof_hash")
	if err != nil {
		return nil, err
	}
	verifierGate, err := solidityHashFromBytes32(verifierGateHash, "groth16.verifier_gate_hash")
	if err != nil {
		return nil, err
	}
	return []string{
		uint128Hex(batchEncoding[:16]),
		uint128Hex(batchEncoding[16:]),
		uint128Hex(authProof[:16]),
		uint128Hex(authProof[16:]),
		uint128Hex(verifierGate[:16]),
		uint128Hex(verifierGate[16:]),
	}, nil
}

func buildGroth16ProofTuple(proofBytes []byte) (VerifierGroth16ProofTuple, error) {
	if len(proofBytes) != 32*8 {
		return VerifierGroth16ProofTuple{}, fmt.Errorf("expected 256-byte Groth16 proof tuple, got %d bytes", len(proofBytes))
	}
	words := make([]string, 8)
	for i := 0; i < 8; i++ {
		words[i] = "0x" + hex.EncodeToString(proofBytes[i*32:(i+1)*32])
	}
	return VerifierGroth16ProofTuple{
		A: [2]string{words[0], words[1]},
		B: [2][2]string{{words[2], words[3]}, {words[4], words[5]}},
		C: [2]string{words[6], words[7]},
	}, nil
}

func uint128Hex(raw []byte) string {
	return "0x" + hex.EncodeToString(raw)
}

func buildVerifierAcceptanceSolidityExport(
	publicInputs ShadowBatchPublicInputs,
	metadata L1BatchMetadata,
	authStatuses []VerifierAcceptanceAuthStatus,
) (VerifierAcceptanceSolidityExport, error) {
	exportedPublicInputs, err := buildSolidityVerifierPublicInputs(publicInputs)
	if err != nil {
		return VerifierAcceptanceSolidityExport{}, err
	}
	exportedMetadata, err := buildSolidityL1BatchMetadata(metadata)
	if err != nil {
		return VerifierAcceptanceSolidityExport{}, err
	}
	exportedAuthStatuses, err := buildSolidityAuthStatuses(authStatuses)
	if err != nil {
		return VerifierAcceptanceSolidityExport{}, err
	}
	return VerifierAcceptanceSolidityExport{
		Schema: verifierAcceptanceSoliditySchema(),
		Calldata: VerifierAcceptanceSolidityCalldata{
			PublicInputs:   exportedPublicInputs,
			MetadataSubset: exportedMetadata,
			AuthStatuses:   exportedAuthStatuses,
		},
	}, nil
}

func verifierAcceptanceSoliditySchema() VerifierAcceptanceSoliditySchema {
	return VerifierAcceptanceSoliditySchema{
		ContractName: FunnyRollupCoreContractName,
		ContractPath: FunnyRollupCoreContractPath,
		FunctionName: FunnyRollupCoreAcceptVerifiedBatchMethod,
		Arguments: []VerifierAcceptanceSolidityArgument{
			{
				Name:     "publicInputs",
				Type:     "FunnyRollupCore.VerifierPublicInputs",
				Provided: true,
				Components: []VerifierAcceptanceSolidityComponent{
					{Name: "batchId", Type: "uint64"},
					{Name: "firstSequenceNo", Type: "uint64"},
					{Name: "lastSequenceNo", Type: "uint64"},
					{Name: "entryCount", Type: "uint64"},
					{Name: "batchDataHash", Type: "bytes32"},
					{Name: "prevStateRoot", Type: "bytes32"},
					{Name: "balancesRoot", Type: "bytes32"},
					{Name: "ordersRoot", Type: "bytes32"},
					{Name: "positionsFundingRoot", Type: "bytes32"},
					{Name: "withdrawalsRoot", Type: "bytes32"},
					{Name: "nextStateRoot", Type: "bytes32"},
				},
			},
			{
				Name:     "metadataSubset",
				Type:     "FunnyRollupCore.L1BatchMetadata",
				Provided: true,
				Components: []VerifierAcceptanceSolidityComponent{
					{Name: "batchId", Type: "uint64"},
					{Name: "batchDataHash", Type: "bytes32"},
					{Name: "prevStateRoot", Type: "bytes32"},
					{Name: "nextStateRoot", Type: "bytes32"},
				},
			},
			{
				Name:     "authStatuses",
				Type:     "FunnyRollupCore.AuthJoinStatus[]",
				Provided: true,
			},
			{
				Name:     "verifierProof",
				Type:     "bytes",
				Provided: false,
			},
		},
		AuthStatusEnumValues: []VerifierAcceptanceSolidityEnumValue{
			{Name: "UNSPECIFIED", Value: SolidityAuthJoinStatusUnspecified},
			{Name: VerifierAuthJoinSatisfied, Value: SolidityAuthJoinStatusJoined},
			{Name: VerifierAuthJoinMissing, Value: SolidityAuthJoinStatusMissingTradingKeyAuthorized},
			{Name: VerifierAuthJoinIneligible, Value: SolidityAuthJoinStatusNonVerifierEligible},
		},
	}
}

func buildSolidityVerifierPublicInputs(publicInputs ShadowBatchPublicInputs) (SolidityVerifierPublicInputs, error) {
	batchID, err := solidityUint64FromInt64(publicInputs.BatchID, "public_inputs.batch_id")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	firstSequence, err := solidityUint64FromInt64(publicInputs.FirstSequence, "public_inputs.first_sequence_no")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	lastSequence, err := solidityUint64FromInt64(publicInputs.LastSequence, "public_inputs.last_sequence_no")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	entryCount, err := solidityUint64FromInt(publicInputs.EntryCount, "public_inputs.entry_count")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	batchDataHash, err := solidityBytes32(publicInputs.BatchDataHash, "public_inputs.batch_data_hash")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	prevStateRoot, err := solidityBytes32(publicInputs.PrevStateRoot, "public_inputs.prev_state_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	balancesRoot, err := solidityBytes32(publicInputs.BalancesRoot, "public_inputs.balances_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	ordersRoot, err := solidityBytes32(publicInputs.OrdersRoot, "public_inputs.orders_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	positionsFundingRoot, err := solidityBytes32(publicInputs.PositionsFundingRoot, "public_inputs.positions_funding_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	withdrawalsRoot, err := solidityBytes32(publicInputs.WithdrawalsRoot, "public_inputs.withdrawals_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	nextStateRoot, err := solidityBytes32(publicInputs.NextStateRoot, "public_inputs.next_state_root")
	if err != nil {
		return SolidityVerifierPublicInputs{}, err
	}
	return SolidityVerifierPublicInputs{
		BatchID:              batchID,
		FirstSequence:        firstSequence,
		LastSequence:         lastSequence,
		EntryCount:           entryCount,
		BatchDataHash:        batchDataHash,
		PrevStateRoot:        prevStateRoot,
		BalancesRoot:         balancesRoot,
		OrdersRoot:           ordersRoot,
		PositionsFundingRoot: positionsFundingRoot,
		WithdrawalsRoot:      withdrawalsRoot,
		NextStateRoot:        nextStateRoot,
	}, nil
}

func buildSolidityL1BatchMetadata(metadata L1BatchMetadata) (SolidityL1BatchMetadata, error) {
	batchID, err := solidityUint64FromInt64(metadata.BatchID, "l1_batch_metadata.batch_id")
	if err != nil {
		return SolidityL1BatchMetadata{}, err
	}
	batchDataHash, err := solidityBytes32(metadata.BatchDataHash, "l1_batch_metadata.batch_data_hash")
	if err != nil {
		return SolidityL1BatchMetadata{}, err
	}
	prevStateRoot, err := solidityBytes32(metadata.PrevStateRoot, "l1_batch_metadata.prev_state_root")
	if err != nil {
		return SolidityL1BatchMetadata{}, err
	}
	nextStateRoot, err := solidityBytes32(metadata.NextStateRoot, "l1_batch_metadata.next_state_root")
	if err != nil {
		return SolidityL1BatchMetadata{}, err
	}
	return SolidityL1BatchMetadata{
		BatchID:       batchID,
		BatchDataHash: batchDataHash,
		PrevStateRoot: prevStateRoot,
		NextStateRoot: nextStateRoot,
	}, nil
}

func buildSolidityAuthStatuses(authStatuses []VerifierAcceptanceAuthStatus) ([]SolidityAuthJoinStatus, error) {
	exported := make([]SolidityAuthJoinStatus, 0, len(authStatuses))
	for _, authStatus := range authStatuses {
		status, err := solidityAuthJoinStatus(authStatus.JoinStatus)
		if err != nil {
			return nil, err
		}
		exported = append(exported, status)
	}
	return exported, nil
}

func solidityAuthJoinStatus(joinStatus string) (SolidityAuthJoinStatus, error) {
	switch strings.TrimSpace(joinStatus) {
	case VerifierAuthJoinSatisfied:
		return SolidityAuthJoinStatusJoined, nil
	case VerifierAuthJoinMissing:
		return SolidityAuthJoinStatusMissingTradingKeyAuthorized, nil
	case VerifierAuthJoinIneligible:
		return SolidityAuthJoinStatusNonVerifierEligible, nil
	default:
		return SolidityAuthJoinStatusUnspecified, fmt.Errorf("unsupported auth join status %q for Solidity export", joinStatus)
	}
}

func solidityUint64FromInt64(value int64, field string) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative for Solidity uint64 export", field)
	}
	return uint64(value), nil
}

func solidityUint64FromInt(value int, field string) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative for Solidity uint64 export", field)
	}
	return uint64(value), nil
}

func solidityBytes32(value, field string) (string, error) {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimPrefix(strings.ToLower(normalized), "0x")
	if len(normalized) != 64 {
		return "", fmt.Errorf("%s must be 32-byte hex for Solidity export", field)
	}
	if _, err := hex.DecodeString(normalized); err != nil {
		return "", fmt.Errorf("%s must be 32-byte hex for Solidity export: %w", field, err)
	}
	return "0x" + normalized, nil
}

func solidityHashFromBytes32(value, field string) (common.Hash, error) {
	normalized, err := solidityBytes32(value, field)
	if err != nil {
		return common.Hash{}, err
	}
	return common.HexToHash(normalized), nil
}

func mustABISolidityType(typeName string) abi.Type {
	typ, err := abi.NewType(typeName, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

func BuildVerifierAuthProofContract(history []StoredBatch, batch StoredBatch) (VerifierAuthProofContract, error) {
	index, err := indexVerifierTradingKeyAuthorizations(history)
	if err != nil {
		return VerifierAuthProofContract{}, err
	}

	input, err := DecodeBatchInput(batch.InputData)
	if err != nil {
		return VerifierAuthProofContract{}, err
	}

	usedRefs := make(map[string]VerifierTradingKeyAuthorization)
	nonceAuthorizations := make([]VerifierNonceAuthorization, 0)
	readyForVerifier := true

	for _, entry := range input.Entries {
		switch entry.EntryType {
		case EntryTypeTradingKeyAuthorized:
			record, err := verifierTradingKeyAuthorizationFromEntry(batch.BatchID, entry)
			if err != nil {
				return VerifierAuthProofContract{}, err
			}
			index[record.Binding.AuthorizationRef] = record
		case EntryTypeNonceAdvanced:
			var payload NonceAdvancedPayload
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return VerifierAuthProofContract{}, err
			}
			nonceAuthorization, usedRef, err := buildVerifierNonceAuthorization(batch.BatchID, entry, payload, index)
			if err != nil {
				return VerifierAuthProofContract{}, err
			}
			if nonceAuthorization.JoinStatus != VerifierAuthJoinSatisfied {
				readyForVerifier = false
			}
			if usedRef != "" {
				usedRefs[usedRef] = index[usedRef]
			}
			nonceAuthorizations = append(nonceAuthorizations, nonceAuthorization)
		}
	}

	tradingKeyAuthorizations := make([]VerifierTradingKeyAuthorization, 0, len(usedRefs))
	for _, record := range usedRefs {
		tradingKeyAuthorizations = append(tradingKeyAuthorizations, record)
	}
	sort.Slice(tradingKeyAuthorizations, func(i, j int) bool {
		if tradingKeyAuthorizations[i].BatchID != tradingKeyAuthorizations[j].BatchID {
			return tradingKeyAuthorizations[i].BatchID < tradingKeyAuthorizations[j].BatchID
		}
		if tradingKeyAuthorizations[i].Sequence != tradingKeyAuthorizations[j].Sequence {
			return tradingKeyAuthorizations[i].Sequence < tradingKeyAuthorizations[j].Sequence
		}
		return tradingKeyAuthorizations[i].SourceRef < tradingKeyAuthorizations[j].SourceRef
	})
	sort.Slice(nonceAuthorizations, func(i, j int) bool {
		if nonceAuthorizations[i].Sequence != nonceAuthorizations[j].Sequence {
			return nonceAuthorizations[i].Sequence < nonceAuthorizations[j].Sequence
		}
		return nonceAuthorizations[i].SourceRef < nonceAuthorizations[j].SourceRef
	})

	return VerifierAuthProofContract{
		JoinKey:                  "authorization_ref",
		ReadyForVerifier:         readyForVerifier,
		TradingKeyAuthorizations: tradingKeyAuthorizations,
		NonceAuthorizations:      nonceAuthorizations,
		Limitations: []string{
			"auth_proof binds verifier-eligible NONCE_ADVANCED witnesses to canonical TRADING_KEY_AUTHORIZED witnesses by normalized authorization_ref/account/key scope metadata only; it does not verify the wallet EIP-712 signature or the Ed25519 order signature yet.",
			"NON_VERIFIER_ELIGIBLE and MISSING_TRADING_KEY_AUTHORIZED rows stay explicit so future verifier/state-root workers can truthfully reject or defer those batches without reopening shadow-batch-v1 public inputs.",
		},
	}, nil
}

func indexVerifierTradingKeyAuthorizations(history []StoredBatch) (map[string]VerifierTradingKeyAuthorization, error) {
	index := make(map[string]VerifierTradingKeyAuthorization)
	for _, batch := range history {
		input, err := DecodeBatchInput(batch.InputData)
		if err != nil {
			return nil, err
		}
		for _, entry := range input.Entries {
			if entry.EntryType != EntryTypeTradingKeyAuthorized {
				continue
			}
			record, err := verifierTradingKeyAuthorizationFromEntry(batch.BatchID, entry)
			if err != nil {
				return nil, err
			}
			index[record.Binding.AuthorizationRef] = record
		}
	}
	return index, nil
}

func verifierTradingKeyAuthorizationFromEntry(batchID int64, entry JournalEntry) (VerifierTradingKeyAuthorization, error) {
	var payload TradingKeyAuthorizedPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return VerifierTradingKeyAuthorization{}, err
	}
	binding, err := payload.AuthorizationWitness.VerifierBinding()
	if err != nil {
		return VerifierTradingKeyAuthorization{}, err
	}
	if strings.TrimSpace(payload.AuthorizationWitness.WalletTypedDataHash) == "" {
		return VerifierTradingKeyAuthorization{}, fmt.Errorf("wallet_typed_data_hash is required for verifier auth witness")
	}
	if strings.TrimSpace(payload.AuthorizationWitness.WalletSignature) == "" {
		return VerifierTradingKeyAuthorization{}, fmt.Errorf("wallet_signature is required for verifier auth witness")
	}
	return VerifierTradingKeyAuthorization{
		BatchID:             batchID,
		Sequence:            entry.Sequence,
		SourceRef:           strings.TrimSpace(entry.SourceRef),
		Binding:             binding,
		WalletTypedDataHash: strings.TrimSpace(payload.AuthorizationWitness.WalletTypedDataHash),
		WalletSignature:     strings.TrimSpace(payload.AuthorizationWitness.WalletSignature),
	}, nil
}

func buildVerifierNonceAuthorization(batchID int64, entry JournalEntry, payload NonceAdvancedPayload, index map[string]VerifierTradingKeyAuthorization) (VerifierNonceAuthorization, string, error) {
	record := VerifierNonceAuthorization{
		BatchID:   batchID,
		Sequence:  entry.Sequence,
		SourceRef: strings.TrimSpace(entry.SourceRef),
	}
	if payload.OrderAuthorization == nil {
		record.JoinStatus = VerifierAuthJoinIneligible
		record.IneligibleReason = "order_authorization is missing"
		return record, "", nil
	}

	witness := payload.OrderAuthorization
	record.AuthVersion = strings.TrimSpace(witness.AuthVersion)
	record.AuthorizationRef = strings.TrimSpace(witness.AuthorizationRef)
	record.IntentMessageHash = strings.TrimSpace(witness.Intent.MessageHash)
	record.IntentSignature = strings.TrimSpace(witness.Intent.Signature)

	if !witness.VerifierEligible {
		record.JoinStatus = VerifierAuthJoinIneligible
		record.IneligibleReason = strings.TrimSpace(witness.IneligibleReason)
		if record.IneligibleReason == "" {
			record.IneligibleReason = "order authorization witness is not verifier-eligible"
		}
		return record, "", nil
	}

	binding, err := witness.VerifierBinding()
	if err != nil {
		return VerifierNonceAuthorization{}, "", err
	}
	if strings.TrimSpace(payload.AuthKeyID) != binding.TradingKeyID {
		return VerifierNonceAuthorization{}, "", fmt.Errorf("nonce advance auth_key_id %s does not match verifier binding trading_key_id %s", strings.TrimSpace(payload.AuthKeyID), binding.TradingKeyID)
	}
	if payload.AccountID != binding.AccountID {
		return VerifierNonceAuthorization{}, "", fmt.Errorf("nonce advance account_id %d does not match verifier binding account_id %d", payload.AccountID, binding.AccountID)
	}
	if normalizeVerifierField(payload.Scope) != binding.Scope {
		return VerifierNonceAuthorization{}, "", fmt.Errorf("nonce advance scope %s does not match verifier binding scope %s", payload.Scope, binding.Scope)
	}
	if normalizeVerifierField(payload.KeyStatus) != binding.KeyStatus {
		return VerifierNonceAuthorization{}, "", fmt.Errorf("nonce advance key_status %s does not match verifier binding key_status %s", payload.KeyStatus, binding.KeyStatus)
	}
	if err := validateVerifierIntent(binding, witness.Intent); err != nil {
		return VerifierNonceAuthorization{}, "", err
	}

	record.Binding = &binding
	authRecord, ok := index[binding.AuthorizationRef]
	if !ok {
		record.JoinStatus = VerifierAuthJoinMissing
		record.IneligibleReason = "missing canonical TRADING_KEY_AUTHORIZED witness for authorization_ref"
		return record, "", nil
	}
	if err := sharedauth.ValidateVerifierBindingMatch(authRecord.Binding, binding); err != nil {
		return VerifierNonceAuthorization{}, "", err
	}

	record.JoinStatus = VerifierAuthJoinSatisfied
	return record, binding.AuthorizationRef, nil
}

func validateVerifierIntent(binding sharedauth.VerifierAuthBinding, intent sharedauth.OrderIntentWitness) error {
	if strings.TrimSpace(intent.SessionID) != binding.TradingKeyID {
		return fmt.Errorf("order intent session_id %s does not match verifier binding trading_key_id %s", strings.TrimSpace(intent.SessionID), binding.TradingKeyID)
	}
	if sharedauth.NormalizeHex(intent.WalletAddress) != binding.WalletAddress {
		return fmt.Errorf("order intent wallet_address %s does not match verifier binding wallet_address %s", intent.WalletAddress, binding.WalletAddress)
	}
	if intent.UserID != binding.AccountID {
		return fmt.Errorf("order intent user_id %d does not match verifier binding account_id %d", intent.UserID, binding.AccountID)
	}
	if strings.TrimSpace(intent.MessageHash) == "" {
		return fmt.Errorf("order authorization witness intent_message_hash is required")
	}
	if strings.TrimSpace(intent.Signature) == "" {
		return fmt.Errorf("order authorization witness intent_signature is required")
	}
	return nil
}

func normalizeVerifierField(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func mustDecodeHexBytes(value string) []byte {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		panic(fmt.Sprintf("decode Groth16 fixture proof bytes: %v", err))
	}
	return decoded
}
