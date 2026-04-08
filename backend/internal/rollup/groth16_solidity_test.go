package rollup

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	bn254 "github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	bn254groth16 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/solidity"
	"github.com/consensys/gnark/frontend"
)

func TestMarshalGroth16SolidityProofCommitmentPairing(t *testing.T) {
	context := SolidityVerifierGateContext{
		BatchEncodingHash: "0x3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb",
		PublicInputs: SolidityVerifierPublicInputs{
			BatchID:              18,
			FirstSequence:        113,
			LastSequence:         114,
			EntryCount:           2,
			BatchDataHash:        "0x4a3737d55508107d2d266501d671002be51bfa277158cb634e408ee6bd1d03e1",
			PrevStateRoot:        "0x444ca6ff4f1e52736965f9e6fe78d97c5d95d7f92fda9119833f0790a89fa299",
			BalancesRoot:         "0x4a7c4c770b61f5d37888eddb20af9503b6e23cbae7fe504354226bff4b281dae",
			OrdersRoot:           "0xc845cf12fe838cfef7beacc043271ba34609758f63d9333839b2eb745ee99ef8",
			PositionsFundingRoot: "0x04a1db17fdb9bb3abfdeded23c181ca9c05bf38a3564df717b61edc5ee5375f2",
			WithdrawalsRoot:      "0x4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96",
			NextStateRoot:        "0x3c6357723a9e0283928a119c633c7001edbba24f8c0606fac1a555c1c5e29668",
			ConservationHash:     "0x99fa1219e95995f8c396427d8ccd8a1efd8ca98a4d76843333ab6f4f8df6828d",
		},
		AuthProofHash:    "0x569e75fc77c1a856f6daaf9e69d8a9566ca34aa47f9133711ce065a571af0cfd",
		VerifierGateHash: "0x799a8cc5d69b522a0c6430bdf852fe788773c223a8cd6fe02d1ce64d19c134c9",
	}

	material := BuildDeterministicStateTransitionWitnessMaterial(context)

	lane, err := loadFixedGroth16Lane()
	if err != nil {
		t.Fatalf("loadFixedGroth16Lane: %v", err)
	}

	concreteVK, ok := lane.vk.(*bn254groth16.VerifyingKey)
	if !ok {
		t.Fatalf("VK is not *bn254groth16.VerifyingKey, got %T", lane.vk)
	}
	if len(concreteVK.CommitmentKeys) == 0 {
		t.Fatal("no commitment keys in VK")
	}
	t.Logf("VK has %d commitment keys", len(concreteVK.CommitmentKeys))

	transitionWitnessHash, err := buildTransitionWitnessHash(context, material)
	if err != nil {
		t.Fatalf("buildTransitionWitnessHash: %v", err)
	}
	publicInputs, err := buildGroth16PublicInputsHex(context.BatchEncodingHash, context.AuthProofHash, context.VerifierGateHash, transitionWitnessHash)
	if err != nil {
		t.Fatalf("buildGroth16PublicInputsHex: %v", err)
	}

	assignment, err := groth16CircuitAssignmentFromContext(context, material, publicInputs)
	if err != nil {
		t.Fatalf("groth16CircuitAssignmentFromContext: %v", err)
	}

	fullWitness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		t.Fatalf("NewWitness: %v", err)
	}

	var proof groth16.Proof
	if err := withDeterministicRand(groth16ProofSeed(context, material), func() error {
		var proveErr error
		proof, proveErr = groth16.Prove(
			lane.ccs,
			lane.pk,
			fullWitness,
			solidity.WithProverTargetSolidityVerifier(backend.GROTH16),
		)
		return proveErr
	}); err != nil {
		t.Fatalf("prove: %v", err)
	}

	publicWitness, err := fullWitness.Public()
	if err != nil {
		t.Fatalf("Public: %v", err)
	}
	if err := groth16.Verify(
		proof,
		lane.vk,
		publicWitness,
		solidity.WithVerifierTargetSolidityVerifier(backend.GROTH16),
	); err != nil {
		t.Fatalf("Go-side verify FAILED: %v", err)
	}
	t.Log("Go-side verify PASSED")

	// Get raw MarshalSolidity output
	marshaler := proof.(interface{ MarshalSolidity() []byte })
	rawSolidity := marshaler.MarshalSolidity()
	t.Logf("MarshalSolidity raw length: %d bytes", len(rawSolidity))

	// Get our marshalGroth16SolidityProof output
	rawTupleBytes, err := marshalGroth16SolidityProof(proof)
	if err != nil {
		t.Fatalf("marshalGroth16SolidityProof: %v", err)
	}
	t.Logf("marshalGroth16SolidityProof length: %d bytes", len(rawTupleBytes))

	if len(rawTupleBytes) != 384 {
		t.Fatalf("expected 384 bytes from marshalGroth16SolidityProof, got %d", len(rawTupleBytes))
	}

	// Decompose the rawTupleBytes
	commitXBytes := rawTupleBytes[256:288]
	commitYBytes := rawTupleBytes[288:320]
	pokXBytes := rawTupleBytes[320:352]
	pokYBytes := rawTupleBytes[352:384]

	t.Logf("commitment X: 0x%s", hex.EncodeToString(commitXBytes))
	t.Logf("commitment Y: 0x%s", hex.EncodeToString(commitYBytes))
	t.Logf("pok X: 0x%s", hex.EncodeToString(pokXBytes))
	t.Logf("pok Y: 0x%s", hex.EncodeToString(pokYBytes))

	// Reconstruct G1 points
	var commitPoint, pokPoint bn254.G1Affine
	commitPoint.X.SetBigInt(new(big.Int).SetBytes(commitXBytes))
	commitPoint.Y.SetBigInt(new(big.Int).SetBytes(commitYBytes))
	pokPoint.X.SetBigInt(new(big.Int).SetBytes(pokXBytes))
	pokPoint.Y.SetBigInt(new(big.Int).SetBytes(pokYBytes))

	if !commitPoint.IsOnCurve() {
		t.Fatal("commitment point NOT on curve")
	}
	if !pokPoint.IsOnCurve() {
		t.Fatal("pok point NOT on curve")
	}
	t.Log("both points are on the BN254 curve")

	// Verify Pedersen commitment pairing: e(C, GSigmaNeg) * e(Pok, G) == 1
	cmtVk := concreteVK.CommitmentKeys[0]

	pairingOk, err := bn254.PairingCheck(
		[]bn254.G1Affine{commitPoint, pokPoint},
		[]bn254.G2Affine{cmtVk.GSigmaNeg, cmtVk.G},
	)
	if err != nil {
		t.Fatalf("PairingCheck error: %v", err)
	}
	if !pairingOk {
		t.Fatal("PEDERSEN COMMITMENT PAIRING CHECK FAILED - reproduces CommitmentInvalid()!")
	}
	t.Log("Pedersen commitment pairing check PASSED")

	// Compare MarshalSolidity bytes with marshalGroth16SolidityProof output
	if len(rawSolidity) < 260 {
		t.Fatalf("MarshalSolidity too short: %d bytes", len(rawSolidity))
	}

	nbCommitments := int(rawSolidity[256])<<24 | int(rawSolidity[257])<<16 | int(rawSolidity[258])<<8 | int(rawSolidity[259])
	t.Logf("nbCommitments from MarshalSolidity: %d", nbCommitments)

	marshalCommitPok := rawSolidity[260:]
	myCommitPok := rawTupleBytes[256:]
	if hex.EncodeToString(marshalCommitPok) != hex.EncodeToString(myCommitPok) {
		t.Logf("MarshalSolidity commitment+pok (%d bytes): 0x%s", len(marshalCommitPok), hex.EncodeToString(marshalCommitPok))
		t.Logf("marshalGroth16 commitment+pok (%d bytes): 0x%s", len(myCommitPok), hex.EncodeToString(myCommitPok))
		t.Fatal("commitment/pok bytes MISMATCH!")
	}
	t.Log("commitment/pok bytes MATCH between MarshalSolidity and marshalGroth16SolidityProof")

	// Now test that the full BuildFixedGroth16Artifact output also works
	artifact, err := BuildFixedGroth16Artifact(context, material)
	if err != nil {
		t.Fatalf("BuildFixedGroth16Artifact: %v", err)
	}
	t.Logf("artifact.ProofBytes length: %d chars (hex)", len(artifact.ProofBytes))

	t.Log("=== ALL CHECKS PASSED ===")
}
