package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"funnyoption/internal/rollup"

	gnarklogger "github.com/consensys/gnark/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type artifactEnvelope struct {
	ProofSchemaHash     string                      `json:"proof_schema_hash"`
	PublicSignalsHash   string                      `json:"public_signals_hash"`
	ProofDataSchemaHash string                      `json:"proof_data_schema_hash"`
	ProofVersionHash    string                      `json:"proof_version_hash"`
	BatchEncodingHash   string                      `json:"batch_encoding_hash"`
	AuthProofHash       string                      `json:"auth_proof_hash"`
	VerifierGateHash    string                      `json:"verifier_gate_hash"`
	Artifact            rollup.FixedGroth16Artifact `json:"artifact"`
	ProofData           string                      `json:"proof_data"`
	Proof               string                      `json:"proof"`
}

func main() {
	gnarklogger.Disable()

	var (
		printContract     = flag.Bool("contract", false, "print the deterministic Solidity backend source")
		batchEncodingHash = flag.String("batch-encoding-hash", "", "bytes32 batch encoding hash")
		authProofHash     = flag.String("auth-proof-hash", "", "bytes32 auth proof hash")
		verifierGateHash  = flag.String("verifier-gate-hash", "", "bytes32 verifier gate hash")
	)
	flag.Parse()

	if *printContract {
		contractSource, err := rollup.ExportFunnyRollupGroth16BackendSolidity()
		if err != nil {
			fatal(err)
		}
		fmt.Print(contractSource)
		return
	}

	if strings.TrimSpace(*batchEncodingHash) == "" || strings.TrimSpace(*authProofHash) == "" || strings.TrimSpace(*verifierGateHash) == "" {
		fatal(fmt.Errorf("batch-encoding-hash, auth-proof-hash, and verifier-gate-hash are required unless -contract is set"))
	}

	context := rollup.SolidityVerifierGateContext{
		BatchEncodingHash: *batchEncodingHash,
		AuthProofHash:     *authProofHash,
		VerifierGateHash:  *verifierGateHash,
	}
	artifact, err := rollup.BuildFixedGroth16Artifact(context, rollup.BuildDeterministicStateTransitionWitnessMaterial(context))
	if err != nil {
		fatal(err)
	}

	proofSchemaHash := crypto.Keccak256Hash([]byte(rollup.FunnyRollupBatchVerifierProofSchemaVersion))
	publicSignalsHash := crypto.Keccak256Hash([]byte(rollup.FunnyRollupBatchVerifierPublicSignalsV1))
	proofDataSchemaHash := crypto.Keccak256Hash([]byte(rollup.FunnyRollupBatchVerifierProofDataVersion))
	proofVersionHash := crypto.Keccak256Hash([]byte(rollup.FunnyRollupBatchVerifierProofVersion))
	normalizedBatchEncodingHash := normalizeHash(*batchEncodingHash)
	normalizedAuthProofHash := normalizeHash(*authProofHash)
	normalizedVerifierGateHash := normalizeHash(*verifierGateHash)

	proofData, err := abi.Arguments{
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes")},
	}.Pack(
		proofDataSchemaHash,
		proofVersionHash,
		normalizedBatchEncodingHash,
		normalizedAuthProofHash,
		normalizedVerifierGateHash,
		common.FromHex(artifact.ProofBytes),
	)
	if err != nil {
		fatal(err)
	}

	proof, err := abi.Arguments{
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes32")},
		{Type: mustABISolidityType("bytes")},
	}.Pack(
		proofSchemaHash,
		publicSignalsHash,
		normalizedBatchEncodingHash,
		normalizedAuthProofHash,
		normalizedVerifierGateHash,
		proofData,
	)
	if err != nil {
		fatal(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(artifactEnvelope{
		ProofSchemaHash:     proofSchemaHash.Hex(),
		PublicSignalsHash:   publicSignalsHash.Hex(),
		ProofDataSchemaHash: proofDataSchemaHash.Hex(),
		ProofVersionHash:    proofVersionHash.Hex(),
		BatchEncodingHash:   normalizedBatchEncodingHash.Hex(),
		AuthProofHash:       normalizedAuthProofHash.Hex(),
		VerifierGateHash:    normalizedVerifierGateHash.Hex(),
		Artifact:            artifact,
		ProofData:           "0x" + common.Bytes2Hex(proofData),
		Proof:               "0x" + common.Bytes2Hex(proof),
	}); err != nil {
		fatal(err)
	}
}

func mustABISolidityType(typeName string) abi.Type {
	typ, err := abi.NewType(typeName, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

func normalizeHash(value string) common.Hash {
	return common.HexToHash(value)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
