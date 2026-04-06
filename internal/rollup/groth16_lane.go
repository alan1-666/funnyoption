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
	"github.com/consensys/gnark/std/hash/sha2"
	"github.com/consensys/gnark/std/math/uints"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	fixedGroth16SetupSeed   = "funny-rollup-fixed-vk-groth16-bn254-sha256-state-transition-gate-v1/setup"
	fixedGroth16ProveDomain = "funny-rollup-fixed-vk-groth16-bn254-sha256-state-transition-gate-v1/prove"

	groth16BackendWrapperContractSource = `

contract FunnyRollupGroth16Backend is Verifier {
    function verifyTupleProof(
        uint256[2] calldata a,
        uint256[2][2] calldata b,
        uint256[2] calldata c,
        uint256[8] calldata input
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

        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        try this.verifyProof(proof, commitments, commitmentPok, input) {
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

type fixedGroth16StateTransitionCircuit struct {
	PublicInputs [8]frontend.Variable `gnark:",public"`

	BatchEncodingHash [32]uints.U8
	AuthProofHash     [32]uints.U8
	VerifierGateHash  [32]uints.U8

	BatchID       frontend.Variable
	FirstSequence frontend.Variable
	LastSequence  frontend.Variable
	EntryCount    frontend.Variable

	BatchDataHash        [32]uints.U8
	PrevStateRoot        [32]uints.U8
	BalancesRoot         [32]uints.U8
	OrdersRoot           [32]uints.U8
	PositionsFundingRoot [32]uints.U8
	WithdrawalsRoot      [32]uints.U8
	NextStateRoot        [32]uints.U8
	ConservationHash     [32]uints.U8
}

func (c *fixedGroth16StateTransitionCircuit) Define(api frontend.API) error {
	byteAPI, err := uints.NewBytes(api)
	if err != nil {
		return fmt.Errorf("new bytes api: %w", err)
	}
	uint64API, err := uints.NewBinaryField[uints.U64](api)
	if err != nil {
		return fmt.Errorf("new uint64 api: %w", err)
	}
	hasher, err := sha2.New(api)
	if err != nil {
		return fmt.Errorf("new sha256: %w", err)
	}

	api.AssertIsEqual(c.PublicInputs[0], packBytes128MSB(api, byteAPI, c.BatchEncodingHash[:16]))
	api.AssertIsEqual(c.PublicInputs[1], packBytes128MSB(api, byteAPI, c.BatchEncodingHash[16:]))
	api.AssertIsEqual(c.PublicInputs[2], packBytes128MSB(api, byteAPI, c.AuthProofHash[:16]))
	api.AssertIsEqual(c.PublicInputs[3], packBytes128MSB(api, byteAPI, c.AuthProofHash[16:]))
	api.AssertIsEqual(c.PublicInputs[4], packBytes128MSB(api, byteAPI, c.VerifierGateHash[:16]))
	api.AssertIsEqual(c.PublicInputs[5], packBytes128MSB(api, byteAPI, c.VerifierGateHash[16:]))

	abiEncoded := make([]uints.U8, 0, 32*14)
	abiEncoded = append(abiEncoded, c.BatchEncodingHash[:]...)
	abiEncoded = append(abiEncoded, abiUint64ToBytes32(uint64API, c.BatchID)...)
	abiEncoded = append(abiEncoded, abiUint64ToBytes32(uint64API, c.FirstSequence)...)
	abiEncoded = append(abiEncoded, abiUint64ToBytes32(uint64API, c.LastSequence)...)
	abiEncoded = append(abiEncoded, abiUint64ToBytes32(uint64API, c.EntryCount)...)
	abiEncoded = append(abiEncoded, c.BatchDataHash[:]...)
	abiEncoded = append(abiEncoded, c.PrevStateRoot[:]...)
	abiEncoded = append(abiEncoded, c.BalancesRoot[:]...)
	abiEncoded = append(abiEncoded, c.OrdersRoot[:]...)
	abiEncoded = append(abiEncoded, c.PositionsFundingRoot[:]...)
	abiEncoded = append(abiEncoded, c.WithdrawalsRoot[:]...)
	abiEncoded = append(abiEncoded, c.NextStateRoot[:]...)
	abiEncoded = append(abiEncoded, c.ConservationHash[:]...)
	abiEncoded = append(abiEncoded, c.AuthProofHash[:]...)

	hasher.Write(abiEncoded)
	digest := hasher.Sum()
	api.AssertIsEqual(c.PublicInputs[6], packBytes128MSB(api, byteAPI, digest[:16]))
	api.AssertIsEqual(c.PublicInputs[7], packBytes128MSB(api, byteAPI, digest[16:]))
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

type FixedGroth16StateTransitionWitness struct {
	BatchEncodingHash   string                       `json:"batch_encoding_hash"`
	PublicInputs        SolidityVerifierPublicInputs `json:"public_inputs"`
	AuthProofHash       string                       `json:"auth_proof_hash"`
	VerifierGateHash    string                       `json:"verifier_gate_hash"`
	TransitionWitnessHash string                     `json:"transition_witness_hash"`
	EncodingDescription string                       `json:"encoding_description"`
}

type FixedGroth16Artifact struct {
	ProofBytes              string                            `json:"proof_bytes"`
	PublicInputs            []string                          `json:"public_inputs"`
	ProofTuple              VerifierGroth16ProofTuple         `json:"proof_tuple"`
	StateTransitionWitness  FixedGroth16StateTransitionWitness `json:"state_transition_witness"`
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
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &fixedGroth16StateTransitionCircuit{})
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

func BuildFixedGroth16Artifact(context SolidityVerifierGateContext) (FixedGroth16Artifact, error) {
	transitionWitnessHash, err := buildTransitionWitnessHashFromContext(context)
	if err != nil {
		return FixedGroth16Artifact{}, err
	}
	proofBytes, publicInputs, proofTuple, err := buildBatchSpecificGroth16Proof(context)
	if err != nil {
		return FixedGroth16Artifact{}, err
	}
	return FixedGroth16Artifact{
		ProofBytes:   "0x" + hex.EncodeToString(proofBytes),
		PublicInputs: publicInputs,
		ProofTuple:   proofTuple,
		StateTransitionWitness: FixedGroth16StateTransitionWitness{
			BatchEncodingHash: context.BatchEncodingHash,
			PublicInputs:      context.PublicInputs,
			AuthProofHash:     context.AuthProofHash,
			VerifierGateHash:  context.VerifierGateHash,
			TransitionWitnessHash: transitionWitnessHash,
			EncodingDescription: "the circuit privately consumes batch/public-input witness material and derives one sha256 state-transition witness hash over the stable batch/public-input contract while still exposing batchEncodingHash, authProofHash, and verifierGateHash as fixed public limbs",
		},
	}, nil
}

func buildBatchSpecificGroth16Proof(context SolidityVerifierGateContext) ([]byte, []string, VerifierGroth16ProofTuple, error) {
	lane, err := loadFixedGroth16Lane()
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}

	transitionWitnessHash, err := buildTransitionWitnessHashFromContext(context)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	publicInputs, err := buildGroth16PublicInputsHex(context.BatchEncodingHash, context.AuthProofHash, context.VerifierGateHash, transitionWitnessHash)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}

	assignment, err := groth16CircuitAssignmentFromContext(context, publicInputs)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	fullWitness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("build fixed groth16 witness: %w", err)
	}

	var proof groth16.Proof
	if err := withDeterministicRand(groth16ProofSeed(context), func() error {
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

	rawTupleBytes, err := marshalGroth16SolidityProof(proof)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	proofTuple, err := buildGroth16ProofTuple(rawTupleBytes)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	transitionWitnessHash, err = buildTransitionWitnessHashFromContext(context)
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, err
	}
	transitionWitnessBytes, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(transitionWitnessHash), "0x"))
	if err != nil {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("decode transition witness hash: %w", err)
	}
	if len(transitionWitnessBytes) != 32 {
		return nil, nil, VerifierGroth16ProofTuple{}, fmt.Errorf("transition witness hash must be 32 bytes, got %d", len(transitionWitnessBytes))
	}
	proofBytes := make([]byte, 0, len(transitionWitnessBytes)+len(rawTupleBytes))
	proofBytes = append(proofBytes, transitionWitnessBytes...)
	proofBytes = append(proofBytes, rawTupleBytes...)
	return proofBytes, publicInputs, proofTuple, nil
}

func buildTransitionWitnessHashFromContext(context SolidityVerifierGateContext) (string, error) {
	batchEncodingHash, err := solidityHashFromBytes32(context.BatchEncodingHash, "context.batch_encoding_hash")
	if err != nil {
		return "", err
	}
	batchDataHash, err := solidityHashFromBytes32(context.PublicInputs.BatchDataHash, "context.public_inputs.batch_data_hash")
	if err != nil {
		return "", err
	}
	prevStateRoot, err := solidityHashFromBytes32(context.PublicInputs.PrevStateRoot, "context.public_inputs.prev_state_root")
	if err != nil {
		return "", err
	}
	balancesRoot, err := solidityHashFromBytes32(context.PublicInputs.BalancesRoot, "context.public_inputs.balances_root")
	if err != nil {
		return "", err
	}
	ordersRoot, err := solidityHashFromBytes32(context.PublicInputs.OrdersRoot, "context.public_inputs.orders_root")
	if err != nil {
		return "", err
	}
	positionsFundingRoot, err := solidityHashFromBytes32(context.PublicInputs.PositionsFundingRoot, "context.public_inputs.positions_funding_root")
	if err != nil {
		return "", err
	}
	withdrawalsRoot, err := solidityHashFromBytes32(context.PublicInputs.WithdrawalsRoot, "context.public_inputs.withdrawals_root")
	if err != nil {
		return "", err
	}
	nextStateRoot, err := solidityHashFromBytes32(context.PublicInputs.NextStateRoot, "context.public_inputs.next_state_root")
	if err != nil {
		return "", err
	}
	conservationHash, err := solidityHashFromBytes32(context.PublicInputs.ConservationHash, "context.public_inputs.conservation_hash")
	if err != nil {
		return "", err
	}
	authProofHash, err := solidityHashFromBytes32(context.AuthProofHash, "context.auth_proof_hash")
	if err != nil {
		return "", err
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
		{Type: solidityBytes32ABIType},
	}.Pack(
		batchEncodingHash,
		context.PublicInputs.BatchID,
		context.PublicInputs.FirstSequence,
		context.PublicInputs.LastSequence,
		context.PublicInputs.EntryCount,
		batchDataHash,
		prevStateRoot,
		balancesRoot,
		ordersRoot,
		positionsFundingRoot,
		withdrawalsRoot,
		nextStateRoot,
		conservationHash,
		authProofHash,
	)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(packed)
	return "0x" + hex.EncodeToString(sum[:]), nil
}

func groth16CircuitAssignmentFromContext(context SolidityVerifierGateContext, publicInputs []string) (fixedGroth16StateTransitionCircuit, error) {
	if len(publicInputs) != 8 {
		return fixedGroth16StateTransitionCircuit{}, fmt.Errorf("expected 8 Groth16 public inputs, got %d", len(publicInputs))
	}

	batchEncodingHash, err := witnessBytes32FromHex(context.BatchEncodingHash, "context.batch_encoding_hash")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	authProofHash, err := witnessBytes32FromHex(context.AuthProofHash, "context.auth_proof_hash")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	verifierGateHash, err := witnessBytes32FromHex(context.VerifierGateHash, "context.verifier_gate_hash")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	batchDataHash, err := witnessBytes32FromHex(context.PublicInputs.BatchDataHash, "context.public_inputs.batch_data_hash")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	prevStateRoot, err := witnessBytes32FromHex(context.PublicInputs.PrevStateRoot, "context.public_inputs.prev_state_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	balancesRoot, err := witnessBytes32FromHex(context.PublicInputs.BalancesRoot, "context.public_inputs.balances_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	ordersRoot, err := witnessBytes32FromHex(context.PublicInputs.OrdersRoot, "context.public_inputs.orders_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	positionsFundingRoot, err := witnessBytes32FromHex(context.PublicInputs.PositionsFundingRoot, "context.public_inputs.positions_funding_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	withdrawalsRoot, err := witnessBytes32FromHex(context.PublicInputs.WithdrawalsRoot, "context.public_inputs.withdrawals_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	nextStateRoot, err := witnessBytes32FromHex(context.PublicInputs.NextStateRoot, "context.public_inputs.next_state_root")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}
	conservationHashWitness, err := witnessBytes32FromHex(context.PublicInputs.ConservationHash, "context.public_inputs.conservation_hash")
	if err != nil {
		return fixedGroth16StateTransitionCircuit{}, err
	}

	var assignment fixedGroth16StateTransitionCircuit
	for i, value := range publicInputs {
		scalar, err := groth16BigIntFromHex(value)
		if err != nil {
			return fixedGroth16StateTransitionCircuit{}, fmt.Errorf("parse groth16 public input %d: %w", i, err)
		}
		assignment.PublicInputs[i] = scalar
	}
	assignment.BatchEncodingHash = batchEncodingHash
	assignment.AuthProofHash = authProofHash
	assignment.VerifierGateHash = verifierGateHash
	assignment.BatchID = new(big.Int).SetUint64(context.PublicInputs.BatchID)
	assignment.FirstSequence = new(big.Int).SetUint64(context.PublicInputs.FirstSequence)
	assignment.LastSequence = new(big.Int).SetUint64(context.PublicInputs.LastSequence)
	assignment.EntryCount = new(big.Int).SetUint64(context.PublicInputs.EntryCount)
	assignment.BatchDataHash = batchDataHash
	assignment.PrevStateRoot = prevStateRoot
	assignment.BalancesRoot = balancesRoot
	assignment.OrdersRoot = ordersRoot
	assignment.PositionsFundingRoot = positionsFundingRoot
	assignment.WithdrawalsRoot = withdrawalsRoot
	assignment.NextStateRoot = nextStateRoot
	assignment.ConservationHash = conservationHashWitness
	return assignment, nil
}

func witnessBytes32FromHex(value, field string) ([32]uints.U8, error) {
	var witness [32]uints.U8
	normalized, err := solidityHashFromBytes32(value, field)
	if err != nil {
		return witness, err
	}
	for i, raw := range normalized.Bytes() {
		witness[i] = uints.NewU8(raw)
	}
	return witness, nil
}

func packBytes128MSB(api frontend.API, byteAPI *uints.Bytes, raw []uints.U8) frontend.Variable {
	if len(raw) != 16 {
		panic(fmt.Sprintf("expected 16 bytes, got %d", len(raw)))
	}
	acc := frontend.Variable(0)
	for _, item := range raw {
		acc = api.Add(api.Mul(acc, 256), byteAPI.Value(item))
	}
	return acc
}

func abiUint64ToBytes32(uint64API *uints.BinaryField[uints.U64], value frontend.Variable) []uints.U8 {
	encoded := make([]uints.U8, 0, 32)
	encoded = append(encoded, uints.NewU8Array(make([]uint8, 24))...)
	encoded = append(encoded, uint64API.UnpackMSB(uint64API.ValueOf(value))...)
	return encoded
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

func groth16ProofSeed(context SolidityVerifierGateContext) string {
	return strings.Join([]string{
		fixedGroth16ProveDomain,
		strings.ToLower(strings.TrimSpace(context.BatchEncodingHash)),
		fmt.Sprintf("%d", context.PublicInputs.BatchID),
		fmt.Sprintf("%d", context.PublicInputs.FirstSequence),
		fmt.Sprintf("%d", context.PublicInputs.LastSequence),
		fmt.Sprintf("%d", context.PublicInputs.EntryCount),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.BatchDataHash)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.PrevStateRoot)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.BalancesRoot)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.OrdersRoot)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.PositionsFundingRoot)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.WithdrawalsRoot)),
		strings.ToLower(strings.TrimSpace(context.PublicInputs.NextStateRoot)),
		strings.ToLower(strings.TrimSpace(context.AuthProofHash)),
		strings.ToLower(strings.TrimSpace(context.VerifierGateHash)),
	}, "|")
}

func marshalGroth16SolidityProof(proof groth16.Proof) ([]byte, error) {
	if marshaler, ok := proof.(interface{ MarshalSolidity() []byte }); ok {
		raw := marshaler.MarshalSolidity()
		if len(raw) < 32*8 {
			return nil, fmt.Errorf("expected at least 256 Solidity proof bytes, got %d", len(raw))
		}
		const proofTupleLen = 32 * 8 // a(2) + b(4) + c(2) = 8 field elements
		tupleBytes := raw[:proofTupleLen]
		remaining := raw[proofTupleLen:]
		if len(remaining) < 4 {
			return append([]byte(nil), tupleBytes...), nil
		}
		nbCommitments := int(remaining[0])<<24 | int(remaining[1])<<16 | int(remaining[2])<<8 | int(remaining[3])
		commitData := remaining[4:]
		expectedCommitLen := nbCommitments * 64 * 2 // each commitment has point (64 bytes) + pok (64 bytes)
		if nbCommitments > 0 && len(commitData) >= expectedCommitLen {
			result := make([]byte, 0, proofTupleLen+expectedCommitLen)
			result = append(result, tupleBytes...)
			result = append(result, commitData[:expectedCommitLen]...)
			return result, nil
		}
		return append([]byte(nil), tupleBytes...), nil
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
