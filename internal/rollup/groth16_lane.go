package rollup

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/solidity"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

const (
	fixedGroth16SetupSeed   = "funny-rollup-fixed-vk-groth16-bn254-2x128-shadow-state-root-gate-v1/setup"
	fixedGroth16ProveDomain = "funny-rollup-fixed-vk-groth16-bn254-2x128-shadow-state-root-gate-v1/prove"

	groth16BackendWrapperContractSource = `

contract FunnyRollupGroth16Backend is Verifier {
    function verifyTupleProof(
        uint256[2] calldata a,
        uint256[2][2] calldata b,
        uint256[2] calldata c,
        uint256[6] calldata input
    ) external view returns (bool) {
        uint256[8] memory proof;
        proof[0] = a[0];
        proof[1] = a[1];
        proof[2] = b[0][0];
        proof[3] = b[0][1];
        proof[4] = b[1][0];
        proof[5] = b[1][1];
        proof[6] = c[0];
        proof[7] = c[1];

        try this.verifyProof(proof, input) {
            return true;
        } catch {
            return false;
        }
    }
}
`
)

var (
	fixedGroth16LaneAssets fixedGroth16Lane
	deterministicRandMu    sync.Mutex
)

type fixedGroth16OuterSignalCircuit struct {
	PublicInputs  [6]frontend.Variable `gnark:",public"`
	WitnessInputs [6]frontend.Variable
}

func (c *fixedGroth16OuterSignalCircuit) Define(api frontend.API) error {
	for i := range c.PublicInputs {
		api.AssertIsEqual(c.PublicInputs[i], c.WitnessInputs[i])
	}
	return nil
}

type fixedGroth16Lane struct {
	once           sync.Once
	ccs            constraint.ConstraintSystem
	pk             groth16.ProvingKey
	vk             groth16.VerifyingKey
	contractSource string
	err            error
}

type FixedGroth16Artifact struct {
	ProofBytes   string                    `json:"proof_bytes"`
	PublicInputs []string                  `json:"public_inputs"`
	ProofTuple   VerifierGroth16ProofTuple `json:"proof_tuple"`
}

type deterministicReader struct {
	seed    []byte
	counter uint64
	buffer  []byte
	offset  int
}

func newDeterministicReader(seed string) *deterministicReader {
	return &deterministicReader{
		seed: append([]byte(nil), []byte(seed)...),
	}
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	written := 0
	for written < len(p) {
		if r.offset == len(r.buffer) {
			var counter [8]byte
			binary.BigEndian.PutUint64(counter[:], r.counter)
			sum := sha256.Sum256(append(append([]byte(nil), r.seed...), counter[:]...))
			r.buffer = sum[:]
			r.offset = 0
			r.counter++
		}
		n := copy(p[written:], r.buffer[r.offset:])
		written += n
		r.offset += n
	}
	return written, nil
}

func withDeterministicRand(seed string, fn func() error) error {
	deterministicRandMu.Lock()
	defer deterministicRandMu.Unlock()

	previous := crand.Reader
	crand.Reader = newDeterministicReader(seed)
	defer func() {
		crand.Reader = previous
	}()

	return fn()
}

func loadFixedGroth16Lane() (*fixedGroth16Lane, error) {
	fixedGroth16LaneAssets.once.Do(func() {
		fixedGroth16LaneAssets.err = fixedGroth16LaneAssets.initialize()
	})
	if fixedGroth16LaneAssets.err != nil {
		return nil, fixedGroth16LaneAssets.err
	}
	return &fixedGroth16LaneAssets, nil
}

func (l *fixedGroth16Lane) initialize() error {
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &fixedGroth16OuterSignalCircuit{})
	if err != nil {
		return fmt.Errorf("compile fixed groth16 circuit: %w", err)
	}

	var pk groth16.ProvingKey
	var vk groth16.VerifyingKey
	if err := withDeterministicRand(fixedGroth16SetupSeed, func() error {
		var setupErr error
		pk, vk, setupErr = groth16.Setup(ccs)
		return setupErr
	}); err != nil {
		return fmt.Errorf("setup fixed groth16 lane: %w", err)
	}

	var contract bytes.Buffer
	if err := vk.ExportSolidity(&contract, solidity.WithPragmaVersion("0.8.24")); err != nil {
		return fmt.Errorf("export fixed groth16 solidity verifier: %w", err)
	}

	l.ccs = ccs
	l.pk = pk
	l.vk = vk
	l.contractSource = strings.TrimRight(contract.String(), "\n") + groth16BackendWrapperContractSource
	return nil
}

func ExportFunnyRollupGroth16BackendSolidity() (string, error) {
	lane, err := loadFixedGroth16Lane()
	if err != nil {
		return "", err
	}
	return lane.contractSource, nil
}

func BuildFixedGroth16Artifact(batchEncodingHash, authProofHash, verifierGateHash string) (FixedGroth16Artifact, error) {
	proofBytes, publicInputs, proofTuple, err := buildBatchSpecificGroth16Proof(batchEncodingHash, authProofHash, verifierGateHash)
	if err != nil {
		return FixedGroth16Artifact{}, err
	}
	return FixedGroth16Artifact{
		ProofBytes:   "0x" + hex.EncodeToString(proofBytes),
		PublicInputs: publicInputs,
		ProofTuple:   proofTuple,
	}, nil
}

func buildBatchSpecificGroth16Proof(batchEncodingHash, authProofHash, verifierGateHash string) ([]byte, []string, VerifierGroth16ProofTuple, error) {
	lane, err := loadFixedGroth16Lane()
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}

	publicInputs, err := buildGroth16PublicInputsHex(batchEncodingHash, authProofHash, verifierGateHash)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}

	assignment, err := groth16CircuitAssignmentFromHex(publicInputs)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	fullWitness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("build fixed groth16 witness: %w", err)
	}

	var proof groth16.Proof
	if err := withDeterministicRand(groth16ProofSeed(batchEncodingHash, authProofHash, verifierGateHash), func() error {
		var proveErr error
		proof, proveErr = groth16.Prove(
			lane.ccs,
			lane.pk,
			fullWitness,
			solidity.WithProverTargetSolidityVerifier(backend.GROTH16),
		)
		return proveErr
	}); err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("prove fixed groth16 batch artifact: %w", err)
	}

	publicWitness, err := fullWitness.Public()
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("build fixed groth16 public witness: %w", err)
	}
	if err := groth16.Verify(
		proof,
		lane.vk,
		publicWitness,
		solidity.WithVerifierTargetSolidityVerifier(backend.GROTH16),
	); err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("verify fixed groth16 batch artifact: %w", err)
	}

	proofBytes, err := marshalGroth16SolidityProof(proof)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	proofTuple, err := buildGroth16ProofTuple(proofBytes)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	return proofBytes, publicInputs, proofTuple, nil
}

func groth16CircuitAssignmentFromHex(publicInputs []string) (fixedGroth16OuterSignalCircuit, error) {
	if len(publicInputs) != 6 {
		return fixedGroth16OuterSignalCircuit{}, fmt.Errorf("expected 6 Groth16 public inputs, got %d", len(publicInputs))
	}

	var assignment fixedGroth16OuterSignalCircuit
	for i, value := range publicInputs {
		scalar, err := groth16BigIntFromHex(value)
		if err != nil {
			return fixedGroth16OuterSignalCircuit{}, fmt.Errorf("parse groth16 public input %d: %w", i, err)
		}
		assignment.PublicInputs[i] = scalar
		assignment.WitnessInputs[i] = new(big.Int).Set(scalar)
	}
	return assignment, nil
}

func groth16BigIntFromHex(value string) (*big.Int, error) {
	normalized := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "0x"))
	if normalized == "" {
		return nil, fmt.Errorf("hex value is required")
	}
	scalar, ok := new(big.Int).SetString(normalized, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex value %q", value)
	}
	return scalar, nil
}

func groth16ProofSeed(batchEncodingHash, authProofHash, verifierGateHash string) string {
	return strings.Join([]string{
		fixedGroth16ProveDomain,
		strings.ToLower(strings.TrimSpace(batchEncodingHash)),
		strings.ToLower(strings.TrimSpace(authProofHash)),
		strings.ToLower(strings.TrimSpace(verifierGateHash)),
	}, "|")
}

func marshalGroth16SolidityProof(proof groth16.Proof) ([]byte, error) {
	if marshaler, ok := proof.(interface{ MarshalSolidity() []byte }); ok {
		return append([]byte(nil), marshaler.MarshalSolidity()...), nil
	}

	var raw bytes.Buffer
	if _, err := proof.WriteRawTo(&raw); err != nil {
		return nil, fmt.Errorf("marshal fixed groth16 proof: %w", err)
	}
	if raw.Len() < 32*8 {
		return nil, fmt.Errorf("expected at least 256 raw Groth16 proof bytes, got %d", raw.Len())
	}
	return append([]byte(nil), raw.Bytes()[:32*8]...), nil
}
