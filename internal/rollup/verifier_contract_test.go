package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	sharedauth "funnyoption/internal/shared/auth"
)

const (
	verifierArtifactTestBatchEncodingHash   = "0x3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb"
	verifierArtifactTestAuthProofHash       = "0x1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795"
	verifierArtifactTestVerifierGateHash    = "0xad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec5"
	verifierArtifactSecondVerifierGateHash  = "0xaded06975aa053c2bcc1d21f42e5b7f293723cfd9c0baaa25a8d49143d3fc9a1"
	verifierArtifactTestProofSchemaHash     = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7"
	verifierArtifactTestPublicSignalsHash   = "0x404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf"
	verifierArtifactTestProofDataHash       = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f7"
	verifierArtifactTestProofVersionHash    = "0x4c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a23"
	verifierArtifactTestGroth16ProofBytes   = "0x2ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9"
	verifierArtifactSecondGroth16ProofBytes = "0x14f1c8679783fc23c9cb13cf16dedb0144fff7e565f40fa101488c1faf22014f2db41a1351cdc25bfb0f574284194c0771c910a12056d2f958bd70a4e89105181a4b416b0638f16b171230fdeea324eecfb5f2915fb81eaec3d0dfb9fa37e7fa1931cbc56620a048c44b51031194d4aa2d0dfa0224517fe2d92e14744a6e127a01b9fb505ae13d4559ccff39af8944fb44911bcfc725cdb68592534b8abbdf8219b7c28f1a414fb3c4be1776612a6439caf330b1a4d8d46481a938263785a31320652243d775a66961fe63706bfda0f21772729871bd3a53074af781fb625b8a26eca88d3b9b55d5eedc9b64343f0721b574bb04242a6dede27f4259c40acec3"
	verifierArtifactTestProofData           = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001002ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9"
	verifierArtifactTestProof               = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001e0627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001002ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9"
)

func TestBuildVerifierGateBatchContractJoinsCanonicalAuthorizationRefAcrossBatches(t *testing.T) {
	authBatch, targetBatch, authRef := verifierGateTestBatches(t)

	contract, err := BuildVerifierGateBatchContract([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierGateBatchContract returned error: %v", err)
	}
	if contract.PublicInputs.BatchDataHash != targetBatch.InputHash {
		t.Fatalf("batch_data_hash = %s, want %s", contract.PublicInputs.BatchDataHash, targetBatch.InputHash)
	}
	if contract.L1BatchMetadata.NextStateRoot != targetBatch.StateRoot {
		t.Fatalf("next_state_root = %s, want %s", contract.L1BatchMetadata.NextStateRoot, targetBatch.StateRoot)
	}
	if !contract.AuthProof.ReadyForVerifier {
		t.Fatalf("expected auth proof to be ready for verifier, got %+v", contract.AuthProof)
	}
	if len(contract.AuthProof.TradingKeyAuthorizations) != 1 {
		t.Fatalf("expected 1 trading key authorization, got %d", len(contract.AuthProof.TradingKeyAuthorizations))
	}
	if len(contract.AuthProof.NonceAuthorizations) != 1 {
		t.Fatalf("expected 1 nonce authorization, got %d", len(contract.AuthProof.NonceAuthorizations))
	}
	if contract.AuthProof.TradingKeyAuthorizations[0].Binding.AuthorizationRef != authRef {
		t.Fatalf("unexpected authorization_ref: %s", contract.AuthProof.TradingKeyAuthorizations[0].Binding.AuthorizationRef)
	}
	nonceAuth := contract.AuthProof.NonceAuthorizations[0]
	if nonceAuth.JoinStatus != VerifierAuthJoinSatisfied {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinSatisfied, nonceAuth.JoinStatus)
	}
	if nonceAuth.AuthorizationRef != authRef {
		t.Fatalf("unexpected nonce authorization_ref: %s", nonceAuth.AuthorizationRef)
	}
	if nonceAuth.Binding == nil {
		t.Fatalf("expected nonce binding")
	}
	if err := sharedauth.ValidateVerifierBindingMatch(contract.AuthProof.TradingKeyAuthorizations[0].Binding, *nonceAuth.Binding); err != nil {
		t.Fatalf("ValidateVerifierBindingMatch returned error: %v", err)
	}
}

func TestBuildVerifierGateBatchContractMarksMissingAuthorizationWitness(t *testing.T) {
	_, targetBatch, authRef := verifierGateTestBatches(t)

	contract, err := BuildVerifierGateBatchContract(nil, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierGateBatchContract returned error: %v", err)
	}
	if contract.AuthProof.ReadyForVerifier {
		t.Fatalf("expected auth proof to stay non-ready when authorization witness is missing")
	}
	if len(contract.AuthProof.TradingKeyAuthorizations) != 0 {
		t.Fatalf("expected no trading key authorizations, got %d", len(contract.AuthProof.TradingKeyAuthorizations))
	}
	if len(contract.AuthProof.NonceAuthorizations) != 1 {
		t.Fatalf("expected 1 nonce authorization, got %d", len(contract.AuthProof.NonceAuthorizations))
	}
	nonceAuth := contract.AuthProof.NonceAuthorizations[0]
	if nonceAuth.JoinStatus != VerifierAuthJoinMissing {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinMissing, nonceAuth.JoinStatus)
	}
	if nonceAuth.AuthorizationRef != authRef {
		t.Fatalf("authorization_ref = %s, want %s", nonceAuth.AuthorizationRef, authRef)
	}
}

func TestBuildVerifierGateBatchContractRejectsMismatchedAuthorizationBinding(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)

	input, err := DecodeBatchInput(targetBatch.InputData)
	if err != nil {
		t.Fatalf("DecodeBatchInput returned error: %v", err)
	}
	var payload NonceAdvancedPayload
	if err := json.Unmarshal(input.Entries[0].Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	payload.OrderAuthorization.TradingPublicKey = "0x1111111111111111111111111111111111111111111111111111111111111111"
	input.Entries[0] = mustEntry(t, input.Entries[0].Sequence, EntryTypeNonceAdvanced, SourceTypeAPIAuth, input.Entries[0].SourceRef, payload)
	mutatedInput, mutatedHash, err := EncodeBatchInput(input.Entries)
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}
	targetBatch.InputData = mutatedInput
	targetBatch.InputHash = mutatedHash

	if _, err := BuildVerifierGateBatchContract([]StoredBatch{authBatch}, targetBatch); err == nil {
		t.Fatalf("expected mismatched auth binding to fail")
	}
}

func TestBuildVerifierGateBatchContractMarksLegacyCompatNonceAsNonVerifierEligible(t *testing.T) {
	batch := legacyCompatVerifierGateBatch(t)

	contract, err := BuildVerifierGateBatchContract(nil, batch)
	if err != nil {
		t.Fatalf("BuildVerifierGateBatchContract returned error: %v", err)
	}
	if contract.AuthProof.ReadyForVerifier {
		t.Fatalf("expected legacy compatibility nonce auth to stay non-verifier-eligible")
	}
	if len(contract.AuthProof.NonceAuthorizations) != 1 {
		t.Fatalf("expected 1 nonce authorization, got %d", len(contract.AuthProof.NonceAuthorizations))
	}
	if contract.AuthProof.NonceAuthorizations[0].JoinStatus != VerifierAuthJoinIneligible {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinIneligible, contract.AuthProof.NonceAuthorizations[0].JoinStatus)
	}
}

func TestBuildVerifierStateRootAcceptanceContractRequiresAllAuthRowsJoined(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)

	contract, err := BuildVerifierStateRootAcceptanceContract([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierStateRootAcceptanceContract returned error: %v", err)
	}
	if !contract.ReadyForAcceptance {
		t.Fatalf("expected acceptance contract to stay ready when every auth row is JOINED")
	}
	if len(contract.AuthStatuses) != 1 {
		t.Fatalf("expected 1 auth status, got %d", len(contract.AuthStatuses))
	}
	if contract.AuthStatuses[0].JoinStatus != VerifierAuthJoinSatisfied {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinSatisfied, contract.AuthStatuses[0].JoinStatus)
	}
	if contract.PublicInputs.NextStateRoot != targetBatch.StateRoot {
		t.Fatalf("next_state_root = %s, want %s", contract.PublicInputs.NextStateRoot, targetBatch.StateRoot)
	}
	if contract.SolidityExport.Schema.ContractName != FunnyRollupCoreContractName {
		t.Fatalf("contract_name = %s, want %s", contract.SolidityExport.Schema.ContractName, FunnyRollupCoreContractName)
	}
	if contract.SolidityExport.Schema.FunctionName != FunnyRollupCoreAcceptVerifiedBatchMethod {
		t.Fatalf("function_name = %s, want %s", contract.SolidityExport.Schema.FunctionName, FunnyRollupCoreAcceptVerifiedBatchMethod)
	}
	if len(contract.SolidityExport.Schema.Arguments) != 4 {
		t.Fatalf("expected 4 Solidity arguments, got %d", len(contract.SolidityExport.Schema.Arguments))
	}
	if !contract.SolidityExport.Schema.Arguments[0].Provided {
		t.Fatalf("expected publicInputs export to be marked provided")
	}
	if contract.SolidityExport.Schema.Arguments[3].Provided {
		t.Fatalf("expected verifierProof export to stay external")
	}
	if len(contract.SolidityExport.Schema.AuthStatusEnumValues) != 4 {
		t.Fatalf("expected 4 auth enum values, got %d", len(contract.SolidityExport.Schema.AuthStatusEnumValues))
	}
	if contract.SolidityExport.Schema.AuthStatusEnumValues[1].Value != SolidityAuthJoinStatusJoined {
		t.Fatalf("JOINED enum value = %d, want %d", contract.SolidityExport.Schema.AuthStatusEnumValues[1].Value, SolidityAuthJoinStatusJoined)
	}
	if contract.SolidityExport.Calldata.PublicInputs.BatchID != uint64(targetBatch.BatchID) {
		t.Fatalf("solidity public_inputs.batch_id = %d, want %d", contract.SolidityExport.Calldata.PublicInputs.BatchID, targetBatch.BatchID)
	}
	if contract.SolidityExport.Calldata.PublicInputs.BatchDataHash != "0x"+targetBatch.InputHash {
		t.Fatalf("solidity public_inputs.batch_data_hash = %s, want %s", contract.SolidityExport.Calldata.PublicInputs.BatchDataHash, "0x"+targetBatch.InputHash)
	}
	if contract.SolidityExport.Calldata.MetadataSubset.NextStateRoot != "0x"+targetBatch.StateRoot {
		t.Fatalf("solidity metadata_subset.next_state_root = %s, want %s", contract.SolidityExport.Calldata.MetadataSubset.NextStateRoot, "0x"+targetBatch.StateRoot)
	}
	if len(contract.SolidityExport.Calldata.AuthStatuses) != 1 {
		t.Fatalf("expected 1 Solidity auth status, got %d", len(contract.SolidityExport.Calldata.AuthStatuses))
	}
	if contract.SolidityExport.Calldata.AuthStatuses[0] != SolidityAuthJoinStatusJoined {
		t.Fatalf("expected Solidity JOINED status %d, got %d", SolidityAuthJoinStatusJoined, contract.SolidityExport.Calldata.AuthStatuses[0])
	}
}

func TestBuildVerifierStateRootAcceptanceContractRejectsMissingAuthorizationWitness(t *testing.T) {
	_, targetBatch, _ := verifierGateTestBatches(t)

	contract, err := BuildVerifierStateRootAcceptanceContract(nil, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierStateRootAcceptanceContract returned error: %v", err)
	}
	if contract.ReadyForAcceptance {
		t.Fatalf("expected acceptance contract to stay non-ready when authorization witness is missing")
	}
	if len(contract.AuthStatuses) != 1 {
		t.Fatalf("expected 1 auth status, got %d", len(contract.AuthStatuses))
	}
	if contract.AuthStatuses[0].JoinStatus != VerifierAuthJoinMissing {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinMissing, contract.AuthStatuses[0].JoinStatus)
	}
	if contract.SolidityExport.Calldata.AuthStatuses[0] != SolidityAuthJoinStatusMissingTradingKeyAuthorized {
		t.Fatalf("expected Solidity missing-auth status %d, got %d", SolidityAuthJoinStatusMissingTradingKeyAuthorized, contract.SolidityExport.Calldata.AuthStatuses[0])
	}
}

func TestBuildVerifierStateRootAcceptanceContractRejectsNonVerifierEligibleAuth(t *testing.T) {
	batch := legacyCompatVerifierGateBatch(t)

	contract, err := BuildVerifierStateRootAcceptanceContract(nil, batch)
	if err != nil {
		t.Fatalf("BuildVerifierStateRootAcceptanceContract returned error: %v", err)
	}
	if contract.ReadyForAcceptance {
		t.Fatalf("expected acceptance contract to stay non-ready for non-verifier-eligible auth rows")
	}
	if len(contract.AuthStatuses) != 1 {
		t.Fatalf("expected 1 auth status, got %d", len(contract.AuthStatuses))
	}
	if contract.AuthStatuses[0].JoinStatus != VerifierAuthJoinIneligible {
		t.Fatalf("expected join status %s, got %s", VerifierAuthJoinIneligible, contract.AuthStatuses[0].JoinStatus)
	}
	if contract.SolidityExport.Calldata.AuthStatuses[0] != SolidityAuthJoinStatusNonVerifierEligible {
		t.Fatalf("expected Solidity ineligible status %d, got %d", SolidityAuthJoinStatusNonVerifierEligible, contract.SolidityExport.Calldata.AuthStatuses[0])
	}
}

func TestBuildVerifierStateRootAcceptanceContractRejectsNonHexBytes32Export(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)
	targetBatch.StateRoot = "not_hex"

	if _, err := BuildVerifierStateRootAcceptanceContract([]StoredBatch{authBatch}, targetBatch); err == nil {
		t.Fatalf("expected invalid Solidity bytes32 export to fail")
	}
}

func TestBuildVerifierArtifactBundleDirectlyConsumesAcceptanceSolidityExport(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)

	acceptanceContract, err := BuildVerifierStateRootAcceptanceContract([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierStateRootAcceptanceContract returned error: %v", err)
	}
	artifact, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierArtifactBundle returned error: %v", err)
	}

	if !reflect.DeepEqual(artifact.AcceptanceContract.SolidityExport, acceptanceContract.SolidityExport) {
		t.Fatalf("artifact acceptance export diverged from BuildVerifierStateRootAcceptanceContract(...).SolidityExport")
	}
	if artifact.AuthProofDigest.HashFunction != "keccak256(abi.encode(authStatuses))" {
		t.Fatalf("unexpected auth proof hash function: %s", artifact.AuthProofDigest.HashFunction)
	}
	if len(artifact.AuthProofDigest.AuthStatuses) != 1 {
		t.Fatalf("expected 1 auth status in auth proof digest, got %d", len(artifact.AuthProofDigest.AuthStatuses))
	}
	if artifact.AuthProofDigest.AuthStatuses[0] != SolidityAuthJoinStatusJoined {
		t.Fatalf("expected JOINED auth status %d, got %d", SolidityAuthJoinStatusJoined, artifact.AuthProofDigest.AuthStatuses[0])
	}
	if artifact.VerifierGateDigest.EncodingVersion != BatchEncodingVersion {
		t.Fatalf("encoding_version = %s, want %s", artifact.VerifierGateDigest.EncodingVersion, BatchEncodingVersion)
	}
	if artifact.VerifierGateDigest.EncodingVersionHash != verifierArtifactTestBatchEncodingHash {
		t.Fatalf("encoding_version_hash = %s, want %s", artifact.VerifierGateDigest.EncodingVersionHash, verifierArtifactTestBatchEncodingHash)
	}
	if artifact.VerifierGateDigest.PublicInputs.BatchID != acceptanceContract.SolidityExport.Calldata.PublicInputs.BatchID {
		t.Fatalf("verifier gate batch_id = %d, want %d", artifact.VerifierGateDigest.PublicInputs.BatchID, acceptanceContract.SolidityExport.Calldata.PublicInputs.BatchID)
	}
	if artifact.AuthProofDigest.AuthProofHash != verifierArtifactTestAuthProofHash {
		t.Fatalf("auth_proof_hash = %s, want %s", artifact.AuthProofDigest.AuthProofHash, verifierArtifactTestAuthProofHash)
	}
	if artifact.VerifierGateDigest.VerifierGateHash != verifierArtifactTestVerifierGateHash {
		t.Fatalf("verifier_gate_hash = %s, want %s", artifact.VerifierGateDigest.VerifierGateHash, verifierArtifactTestVerifierGateHash)
	}
	if artifact.VerifierInterface.ContractName != FunnyRollupBatchVerifierContractName {
		t.Fatalf("contract_name = %s, want %s", artifact.VerifierInterface.ContractName, FunnyRollupBatchVerifierContractName)
	}
	if artifact.VerifierInterface.ImplementationName != FunnyRollupBatchVerifierImplementationName {
		t.Fatalf("implementation_name = %s, want %s", artifact.VerifierInterface.ImplementationName, FunnyRollupBatchVerifierImplementationName)
	}
	if artifact.VerifierInterface.FunctionName != FunnyRollupBatchVerifierMethod {
		t.Fatalf("function_name = %s, want %s", artifact.VerifierInterface.FunctionName, FunnyRollupBatchVerifierMethod)
	}
	if artifact.VerifierInterface.ProofSchemaVersion != FunnyRollupBatchVerifierProofSchemaVersion {
		t.Fatalf("proof_schema_version = %s, want %s", artifact.VerifierInterface.ProofSchemaVersion, FunnyRollupBatchVerifierProofSchemaVersion)
	}
	if artifact.VerifierInterface.ProofSchemaHash != verifierArtifactTestProofSchemaHash {
		t.Fatalf("proof_schema_hash = %s, want %s", artifact.VerifierInterface.ProofSchemaHash, verifierArtifactTestProofSchemaHash)
	}
	if artifact.VerifierInterface.PublicSignalsVersion != FunnyRollupBatchVerifierPublicSignalsV1 {
		t.Fatalf("public_signals_version = %s, want %s", artifact.VerifierInterface.PublicSignalsVersion, FunnyRollupBatchVerifierPublicSignalsV1)
	}
	if artifact.VerifierInterface.PublicSignalsVersionHash != verifierArtifactTestPublicSignalsHash {
		t.Fatalf(
			"public_signals_version_hash = %s, want %s",
			artifact.VerifierInterface.PublicSignalsVersionHash,
			verifierArtifactTestPublicSignalsHash,
		)
	}
	if artifact.VerifierInterface.ProofDataSchemaVersion != FunnyRollupBatchVerifierProofDataVersion {
		t.Fatalf(
			"proof_data_schema_version = %s, want %s",
			artifact.VerifierInterface.ProofDataSchemaVersion,
			FunnyRollupBatchVerifierProofDataVersion,
		)
	}
	if artifact.VerifierInterface.ProofDataSchemaHash != verifierArtifactTestProofDataHash {
		t.Fatalf(
			"proof_data_schema_hash = %s, want %s",
			artifact.VerifierInterface.ProofDataSchemaHash,
			verifierArtifactTestProofDataHash,
		)
	}
	if artifact.VerifierInterface.ProofVersion != FunnyRollupBatchVerifierProofVersion {
		t.Fatalf("proof_version = %s, want %s", artifact.VerifierInterface.ProofVersion, FunnyRollupBatchVerifierProofVersion)
	}
	if artifact.VerifierInterface.ProofVersionHash != verifierArtifactTestProofVersionHash {
		t.Fatalf("proof_version_hash = %s, want %s", artifact.VerifierInterface.ProofVersionHash, verifierArtifactTestProofVersionHash)
	}
	if artifact.VerifierInterface.Calldata.Context.AuthProofHash != artifact.AuthProofDigest.AuthProofHash {
		t.Fatalf("verifier context auth_proof_hash = %s, want %s", artifact.VerifierInterface.Calldata.Context.AuthProofHash, artifact.AuthProofDigest.AuthProofHash)
	}
	if artifact.VerifierInterface.Calldata.Context.VerifierGateHash != artifact.VerifierGateDigest.VerifierGateHash {
		t.Fatalf("verifier context verifier_gate_hash = %s, want %s", artifact.VerifierInterface.Calldata.Context.VerifierGateHash, artifact.VerifierGateDigest.VerifierGateHash)
	}
	if artifact.VerifierInterface.Calldata.PublicSignals.BatchEncodingHash != verifierArtifactTestBatchEncodingHash {
		t.Fatalf(
			"public_signals.batch_encoding_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.PublicSignals.BatchEncodingHash,
			verifierArtifactTestBatchEncodingHash,
		)
	}
	if artifact.VerifierInterface.Calldata.PublicSignals.AuthProofHash != verifierArtifactTestAuthProofHash {
		t.Fatalf(
			"public_signals.auth_proof_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.PublicSignals.AuthProofHash,
			verifierArtifactTestAuthProofHash,
		)
	}
	if artifact.VerifierInterface.Calldata.PublicSignals.VerifierGateHash != verifierArtifactTestVerifierGateHash {
		t.Fatalf(
			"public_signals.verifier_gate_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.PublicSignals.VerifierGateHash,
			verifierArtifactTestVerifierGateHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.ProofDataSchemaHash != verifierArtifactTestProofDataHash {
		t.Fatalf(
			"proof_data_fields.proof_data_schema_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.ProofDataSchemaHash,
			verifierArtifactTestProofDataHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.ProofTypeHash != verifierArtifactTestProofVersionHash {
		t.Fatalf(
			"proof_data_fields.proof_type_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.ProofTypeHash,
			verifierArtifactTestProofVersionHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.BatchEncodingHash != verifierArtifactTestBatchEncodingHash {
		t.Fatalf(
			"proof_data_fields.batch_encoding_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.BatchEncodingHash,
			verifierArtifactTestBatchEncodingHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.AuthProofHash != verifierArtifactTestAuthProofHash {
		t.Fatalf(
			"proof_data_fields.auth_proof_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.AuthProofHash,
			verifierArtifactTestAuthProofHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.VerifierGateHash != verifierArtifactTestVerifierGateHash {
		t.Fatalf(
			"proof_data_fields.verifier_gate_hash = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.VerifierGateHash,
			verifierArtifactTestVerifierGateHash,
		)
	}
	if artifact.VerifierInterface.Calldata.ProofDataFields.ProofBytes != verifierArtifactTestGroth16ProofBytes {
		t.Fatalf(
			"proof_data_fields.proof_bytes = %s, want %s",
			artifact.VerifierInterface.Calldata.ProofDataFields.ProofBytes,
			verifierArtifactTestGroth16ProofBytes,
		)
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofBytesEncoding != "abi.encode(uint256[2] a, uint256[2][2] b, uint256[2] c)" {
		t.Fatalf("unexpected groth16 proof bytes encoding: %s", artifact.VerifierInterface.Groth16Fixture.ProofBytesEncoding)
	}
	if !artifact.VerifierInterface.Groth16Fixture.ExpectedVerdict {
		t.Fatalf("expected groth16 fixture verdict to stay true")
	}
	expectedGroth16PublicInputs := []string{
		"0x3b6489209bd528a9779ecc9db44d4d05",
		"0xdceb8faba670a6922ff939d841f202cb",
		"0x1e7c5c1c118b439a090ebf5654651794",
		"0x76e94bae5ba6a5ae0f146ec3866c8795",
		"0xad3c7037b47a17484ee261126e1d047f",
		"0xb971227cbd4d47f7e7cdce7a07da2ec5",
	}
	if !reflect.DeepEqual(artifact.VerifierInterface.Groth16Fixture.PublicInputs, expectedGroth16PublicInputs) {
		t.Fatalf(
			"groth16 public_inputs = %v, want %v",
			artifact.VerifierInterface.Groth16Fixture.PublicInputs,
			expectedGroth16PublicInputs,
		)
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0] != "0x2ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a" {
		t.Fatalf("unexpected groth16 proof tuple a[0]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1] != "0x23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc19" {
		t.Fatalf("unexpected groth16 proof tuple b[1][1]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.C[1] != "0x0b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9" {
		t.Fatalf("unexpected groth16 proof tuple c[1]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.C[1])
	}
	if artifact.VerifierInterface.Calldata.ProofData != verifierArtifactTestProofData {
		t.Fatalf("proof_data = %s, want %s", artifact.VerifierInterface.Calldata.ProofData, verifierArtifactTestProofData)
	}
	if artifact.VerifierInterface.Calldata.Proof != verifierArtifactTestProof {
		t.Fatalf("verifier proof = %s, want %s", artifact.VerifierInterface.Calldata.Proof, verifierArtifactTestProof)
	}
}

func TestBuildVerifierArtifactBundleDeterministic(t *testing.T) {
	authBatch, targetBatch, _ := verifierGateTestBatches(t)

	first, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("first BuildVerifierArtifactBundle returned error: %v", err)
	}
	second, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("second BuildVerifierArtifactBundle returned error: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic artifact bundle, got first=%+v second=%+v", first, second)
	}
}

func TestBuildBatchSpecificGroth16ProofVariesByVerifierGateHash(t *testing.T) {
	firstProofBytes, firstPublicInputs, _, err := buildBatchSpecificGroth16Proof(
		verifierArtifactTestBatchEncodingHash,
		verifierArtifactTestAuthProofHash,
		verifierArtifactTestVerifierGateHash,
	)
	if err != nil {
		t.Fatalf("buildBatchSpecificGroth16Proof(first) returned error: %v", err)
	}
	secondProofBytes, secondPublicInputs, _, err := buildBatchSpecificGroth16Proof(
		verifierArtifactTestBatchEncodingHash,
		verifierArtifactTestAuthProofHash,
		verifierArtifactSecondVerifierGateHash,
	)
	if err != nil {
		t.Fatalf("buildBatchSpecificGroth16Proof(second) returned error: %v", err)
	}

	if "0x"+hex.EncodeToString(firstProofBytes) != verifierArtifactTestGroth16ProofBytes {
		t.Fatalf("unexpected first proof bytes: 0x%s", hex.EncodeToString(firstProofBytes))
	}
	if "0x"+hex.EncodeToString(secondProofBytes) != verifierArtifactSecondGroth16ProofBytes {
		t.Fatalf("unexpected second proof bytes: 0x%s", hex.EncodeToString(secondProofBytes))
	}
	if reflect.DeepEqual(firstPublicInputs, secondPublicInputs) {
		t.Fatalf("expected Groth16 public inputs to differ across batch-specific verifier gate hashes")
	}
	if reflect.DeepEqual(firstProofBytes, secondProofBytes) {
		t.Fatalf("expected batch-specific Groth16 proof bytes to differ across verifier gate hashes")
	}
}

func legacyCompatVerifierGateBatch(t *testing.T) StoredBatch {
	t.Helper()

	key := sharedauth.AuthorizedTradingKey{
		TradingKeyID:       "sess_legacy",
		AccountID:          1001,
		WalletAddress:      "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:   "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:   sharedauth.DefaultTradingKeyScheme,
		Scope:              sharedauth.DefaultSessionScope,
		ChainID:            97,
		Status:             "ACTIVE",
		AuthorizationNonce: "sess_legacy_nonce",
	}
	intent := sharedauth.OrderIntent{
		SessionID:         key.TradingKeyID,
		WalletAddress:     key.WalletAddress,
		UserID:            key.AccountID,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-legacy",
		Nonce:             7,
		RequestedAtMillis: time.Now().UnixMilli(),
	}
	orderWitness := sharedauth.BuildOrderAuthorizationWitness(key.AccountID, key, intent, "0xfeedface")
	entries := []JournalEntry{
		mustEntry(t, 7, EntryTypeNonceAdvanced, SourceTypeAPIAuth, fmt.Sprintf("%s:%d", key.TradingKeyID, intent.Nonce), NonceAdvancedPayload{
			AccountID:          key.AccountID,
			AuthKeyID:          key.TradingKeyID,
			Scope:              key.Scope,
			KeyStatus:          key.Status,
			AcceptedNonce:      intent.Nonce,
			NextNonce:          intent.Nonce + 1,
			OccurredAtMillis:   intent.RequestedAtMillis,
			OrderAuthorization: &orderWitness,
		}),
	}
	inputData, inputHash, err := EncodeBatchInput(entries)
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}

	return StoredBatch{
		BatchID:         2,
		EncodingVersion: BatchEncodingVersion,
		FirstSequence:   7,
		LastSequence:    7,
		EntryCount:      len(entries),
		InputData:       inputData,
		InputHash:       inputHash,
		PrevStateRoot:   testHexRoot("prev_state_root_legacy"),
		StateRoot:       testHexRoot("next_state_root_legacy"),
	}
}

func verifierGateTestBatches(t *testing.T) (StoredBatch, StoredBatch, string) {
	t.Helper()

	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         sharedauth.DefaultTradingKeyScheme,
		Scope:                    sharedauth.DefaultSessionScope,
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: 1775886700000,
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}
	key := sharedauth.AuthorizedTradingKey{
		TradingKeyID:       authz.TradingKeyID(),
		AccountID:          1001,
		WalletAddress:      authz.WalletAddress,
		TradingPublicKey:   authz.TradingPublicKey,
		TradingKeyScheme:   authz.TradingKeyScheme,
		Scope:              authz.Scope,
		ChainID:            authz.ChainID,
		VaultAddress:       authz.VaultAddress,
		Status:             "ACTIVE",
		ExpiresAtMillis:    0,
		AuthorizationNonce: authz.Challenge,
	}
	authWitness, err := sharedauth.BuildTradingKeyAuthorizationWitness(key.AccountID, authz, key, "0xdeadbeef", 1775886400000)
	if err != nil {
		t.Fatalf("BuildTradingKeyAuthorizationWitness returned error: %v", err)
	}

	intent := sharedauth.OrderIntent{
		SessionID:         key.TradingKeyID,
		WalletAddress:     key.WalletAddress,
		UserID:            key.AccountID,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-1",
		Nonce:             7,
		RequestedAtMillis: 1775886400000,
	}
	orderWitness := sharedauth.BuildOrderAuthorizationWitness(key.AccountID, key, intent, "0xfeedface")

	authEntries := []JournalEntry{
		mustEntry(t, 1, EntryTypeTradingKeyAuthorized, SourceTypeAPIAuth, key.AuthorizationRef(), TradingKeyAuthorizedPayload{
			AuthorizationWitness: authWitness,
		}),
	}
	authInput, authHash, err := EncodeBatchInput(authEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(authEntries) returned error: %v", err)
	}
	targetEntries := []JournalEntry{
		mustEntry(t, 7, EntryTypeNonceAdvanced, SourceTypeAPIAuth, fmt.Sprintf("%s:%d", key.TradingKeyID, intent.Nonce), NonceAdvancedPayload{
			AccountID:          key.AccountID,
			AuthKeyID:          key.TradingKeyID,
			Scope:              key.Scope,
			KeyStatus:          key.Status,
			AcceptedNonce:      intent.Nonce,
			NextNonce:          intent.Nonce + 1,
			OccurredAtMillis:   intent.RequestedAtMillis,
			OrderAuthorization: &orderWitness,
		}),
	}
	targetInput, targetHash, err := EncodeBatchInput(targetEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(targetEntries) returned error: %v", err)
	}

	return StoredBatch{
			BatchID:         1,
			EncodingVersion: BatchEncodingVersion,
			FirstSequence:   1,
			LastSequence:    1,
			EntryCount:      len(authEntries),
			InputData:       authInput,
			InputHash:       authHash,
			PrevStateRoot:   testHexRoot("prev_state_root_auth"),
			StateRoot:       testHexRoot("next_state_root_auth"),
		}, StoredBatch{
			BatchID:         2,
			EncodingVersion: BatchEncodingVersion,
			FirstSequence:   7,
			LastSequence:    7,
			EntryCount:      len(targetEntries),
			InputData:       targetInput,
			InputHash:       targetHash,
			PrevStateRoot:   testHexRoot("next_state_root_auth"),
			StateRoot:       testHexRoot("next_state_root_orders"),
		}, key.AuthorizationRef()
}

func testHexRoot(label string) string {
	return hashStrings("verifier_contract_test", label)
}
