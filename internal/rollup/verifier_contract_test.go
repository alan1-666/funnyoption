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
	verifierArtifactTestVerifierGateHash    = "0xbdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e8128626"
	verifierArtifactSecondVerifierGateHash  = "0x795a355fc2c2e98cbac5561fa98476a65d079471a264f5999e37158d9440e026"
	verifierArtifactTestProofSchemaHash     = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7"
	verifierArtifactTestPublicSignalsHash   = "0x404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf"
	verifierArtifactTestProofDataHash       = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f7"
	verifierArtifactTestProofVersionHash    = "0x4c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a23"
	verifierArtifactTestGroth16ProofBytes   = "0x89976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3"
	verifierArtifactSecondGroth16ProofBytes = "0x22781bd0ce3ae00ab124eaa3264beca2ac49e53c51790f798b52b0f548660bfd01c8d66e953513e29069e77388f23101f53e7846c0c16c27be8b58fb56b0df8a2292e43b415e18492ed03ba5e60eed22ab4d22d6df3902998a4c45fb5a660e270f3144829fa5e5da7eb16ee468da123ed79e1685c0d3b4f6c897e3fb5177642c0e501b22c70e241e82d96735088d256c746df0761a66223f6fa9a56884c06b8e22516fd808ab38eb45b5fcce93b758379bf29467b40453b2afba8a656aab9c88052016533e0624b3ff5f29bd2aa9df7b95f7ce3a73a4e740f7db3f005cf775c81fa0fd4e2a058be4a7e09a40ffe5c7f41f28d5791cada3498701da3a619dacd80af472be383ea3c68a174cb4059c5287f3aa520e02a4c0f88b2217bf125c6dfd2641b5c70fbf99f2b252e10c51f4c65100d0fc8bea2efaf297e6de3f75eacd9611816317e8b3a02ae849dad595d7b96eab4937ae76fa8f3b95431c33802c893d2fb5fde088be3c80241778198fb0a650b155f3b375b6a6062d954f995d05878e1a22180db2f61dc25be768ef7993711b2e17e78dd8414ef63baa298c9f46de56"
	verifierArtifactTestProofData           = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a089976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3"
	verifierArtifactTestProof               = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000280627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a089976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3"
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
	if artifact.VerifierInterface.Groth16Fixture.ProofBytesEncoding != "abi.encode(bytes32 transitionWitnessHash, uint256[2] a, uint256[2][2] b, uint256[2] c)" {
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
		"0xbdb8adff0902424ca22e6b3a15581cb9",
		"0xb23705ba8192448826e23449e8128626",
		"0x89976dac11073c7be3e8e6290a7fef4a",
		"0x049f44ccce9dcba57bcc0b65c7c534e8",
	}
	if !reflect.DeepEqual(artifact.VerifierInterface.Groth16Fixture.PublicInputs, expectedGroth16PublicInputs) {
		t.Fatalf(
			"groth16 public_inputs = %v, want %v",
			artifact.VerifierInterface.Groth16Fixture.PublicInputs,
			expectedGroth16PublicInputs,
		)
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0] != "0x18868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc38" {
		t.Fatalf("unexpected groth16 proof tuple a[0]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1] != "0x2c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b" {
		t.Fatalf("unexpected groth16 proof tuple b[1][1]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.C[1] != "0x2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d992" {
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
	authBatch, targetBatch, _ := verifierGateTestBatches(t)
	firstArtifact, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierArtifactBundle(first) returned error: %v", err)
	}
	firstContext := firstArtifact.VerifierInterface.Calldata.Context
	firstProofBytes, firstPublicInputs, _, err := buildBatchSpecificGroth16Proof(firstContext)
	if err != nil {
		t.Fatalf("buildBatchSpecificGroth16Proof(first) returned error: %v", err)
	}

	secondPublicInputs := firstContext.PublicInputs
	secondPublicInputs.NextStateRoot = "0x590e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5"
	secondDigest, err := buildVerifierGateDigestContract(secondPublicInputs, firstContext.AuthProofHash)
	if err != nil {
		t.Fatalf("buildVerifierGateDigestContract(second) returned error: %v", err)
	}
	secondContext := SolidityVerifierGateContext{
		BatchEncodingHash: firstContext.BatchEncodingHash,
		PublicInputs:      secondPublicInputs,
		AuthProofHash:     firstContext.AuthProofHash,
		VerifierGateHash:  secondDigest.VerifierGateHash,
	}
	secondProofBytes, secondPublicInputsHex, _, err := buildBatchSpecificGroth16Proof(secondContext)
	if err != nil {
		t.Fatalf("buildBatchSpecificGroth16Proof(second) returned error: %v", err)
	}

	if "0x"+hex.EncodeToString(firstProofBytes) != verifierArtifactTestGroth16ProofBytes {
		t.Fatalf("unexpected first proof bytes: 0x%s", hex.EncodeToString(firstProofBytes))
	}
	if "0x"+hex.EncodeToString(secondProofBytes) != verifierArtifactSecondGroth16ProofBytes {
		t.Fatalf("unexpected second proof bytes: 0x%s", hex.EncodeToString(secondProofBytes))
	}
	if reflect.DeepEqual(firstPublicInputs, secondPublicInputsHex) {
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
