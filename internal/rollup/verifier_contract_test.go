package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	sharedauth "funnyoption/internal/shared/auth"
)

const (
	verifierArtifactTestBatchEncodingHash   = "0x3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb"
	verifierArtifactTestAuthProofHash       = "0x1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795"
	verifierArtifactTestVerifierGateHash    = "0xefefbcd563252d443f9828f2a30139f9ae0e25ad24a23bbfa43286618af69ba5"
	verifierArtifactSecondVerifierGateHash  = "0x7ddc06edb632783fa18398652eadb1ce9d24679ac28b356af36aad73cff4bdd9"
	verifierArtifactTestProofSchemaHash     = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7"
	verifierArtifactTestPublicSignalsHash   = "0x404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf"
	verifierArtifactTestProofDataHash       = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f7"
	verifierArtifactTestProofVersionHash    = "0x61d926910cd37f0427ac13baa3386a8b649e9c83c83a845ed4355ec4df7b8d2e"
	verifierArtifactTestGroth16ProofBytes   = "0xf87b07b17aed58b7ed4f12adc3a9b558f6f6405efaa177e8be6cc4f5ee464b82da6118bacf3be9f7e55ebcd74e742b58befea375db7650fee74e42a7ad27e17d412d96f28c9bd212ba27d0da92a15f27337e6ea50cbfaf4c4223af098c47ccf3b45deacc486e6a3ccaa3dbd3ead0e5cde3c3ce6aeea8c9088ea4d53036fdd94e9431ddbfb4276462d98f2fcf32458a0bfe21839c42a8f82bba3847de86d323d73a89b167b1513f5eaac606f903af4688218714b356282bac1cce90e1186c82f953fec9e4f5725b75bf8c0501257419c4fb7aba8c3024792652e9bb81b494a211eaf1c7d70ef5145001f9116853718f2bb6c919df8e7ac47dd74b0ed9be4d55632be8de8b7a68ebe83143f17ba5adc06112af77398b9106f21ea3a0fb8dd30165222a757b0cf1a9a2f2f563cfcdc04171d756ee12a4574a5f7bd9a8fd07d3d2210def9ead58e1b4c674c065c659c36c4f7d44738dec650d2d836c9df754ffe268094fcf497c12d650e90ac8199f72d97b49456ee60605651993279f61025f08ff14364ad5244e50987cadcca61e9540c272c776760580e94cb51aa32d307645070702dafcfc6a25cd4b80bc23b69c3d20218e7aab5b3731356d7bae79f33fc53710b039a909ca9f217f729d990ea23ba67aa95e942e96253f55d67e36c9fa2c722f6fe8d802b80cf963795bfe7c5c6e8d50280be96a0f9afb88deedf441f7e24c02af9381436360048f80b6c38ad8b40cd66f8397075ed2d3c1c505ca0c36afe01f51b361faf4d4a26cc5cb3c47b59a0399ea012e36dd7659073db66e4d2f2f06015aae143471880055c136e7d792b643d9126073c367e9573af3eb947d17bf7622f7b08ce4d1fac36e24465aef2e89e9ca8de9c7a3e4f03d36b2d4616a488790026b7632df3f5ee591f37afedb8c0578907669500ae8ea3d519bef746d8f6c76"
	verifierArtifactSecondGroth16ProofBytes = "0x9419e9657fc9acc263c7cc9f779e5e24365c6de05587bd9bcc125676274e2322da6118bacf3be9f7e55ebcd74e742b58befea375db7650fee74e42a7ad27e17d412d96f28c9bd212ba27d0da92a15f27337e6ea50cbfaf4c4223af098c47ccf3b45deacc486e6a3ccaa3dbd3ead0e5cde3c3ce6aeea8c9088ea4d53036fdd94e9431ddbfb4276462d98f2fcf32458a0bfe21839c42a8f82bba3847de86d323d73a89b167b1513f5eaac606f903af4688218714b356282bac1cce90e1186c82f953fec9e4f5725b75bf8c0501257419c4fb7aba8c3024792652e9bb81b494a211ff85a89bea5a71c56064cef7dea74bc38c141009ecc684347edd4acedfffedcf2be8de8b7a68ebe83143f17ba5adc06112af77398b9106f21ea3a0fb8dd301651057a3107db087c8d1450e370a7bae1808f1d9015b9c7648d0ff2318a1fa1576293fe8c44720bf756fb9a233cb0f650a48fa954c89b12d88629e8fa501f3965718dca7321376e53acf685e9df07496ebedf5a31d149e32821a9882895ed31d052c48922cd78845eb7ca1ba69039bb44fc0003488e587e2e9bbb4f6efdccc1d9d004c85a0b228eaff287ed24518842d2cdb7315f4289e0c0d94ef45349470f83c2ecd0e7c59b9b2298be8b29570f7d89c51f6e7aa0572c967a9006acbb31048531fc0e9f94ba86a1137841fef1cef17a88a048242123a3d510ceeff131d97876709ce9cc9954447062c16e5718e5d044297140e1f0ebde637d8c17242c5bbb605204273f49d855f1f5ef4736939d5565a35c8d5a3c16b9b4fdd0ef2a3e0e0059924cef37342ba0a293eb781c5992a86371381ec0a49609e53a347c5ae4ef6f50021e23a330523e7ecbb56df7a0c0582e9a804b68f10742fa595d4b1cdcc9662621f4417a6bed3027eb2d9d1470537c193bfebe8090e9b2868b9353130f6fb5671"
	verifierArtifactTestProofData           = "0x627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f761d926910cd37f0427ac13baa3386a8b649e9c83c83a845ed4355ec4df7b8d2e3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795efefbcd563252d443f9828f2a30139f9ae0e25ad24a23bbfa43286618af69ba500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000002a0f87b07b17aed58b7ed4f12adc3a9b558f6f6405efaa177e8be6cc4f5ee464b82da6118bacf3be9f7e55ebcd74e742b58befea375db7650fee74e42a7ad27e17d412d96f28c9bd212ba27d0da92a15f27337e6ea50cbfaf4c4223af098c47ccf3b45deacc486e6a3ccaa3dbd3ead0e5cde3c3ce6aeea8c9088ea4d53036fdd94e9431ddbfb4276462d98f2fcf32458a0bfe21839c42a8f82bba3847de86d323d73a89b167b1513f5eaac606f903af4688218714b356282bac1cce90e1186c82f953fec9e4f5725b75bf8c0501257419c4fb7aba8c3024792652e9bb81b494a211eaf1c7d70ef5145001f9116853718f2bb6c919df8e7ac47dd74b0ed9be4d55632be8de8b7a68ebe83143f17ba5adc06112af77398b9106f21ea3a0fb8dd30165222a757b0cf1a9a2f2f563cfcdc04171d756ee12a4574a5f7bd9a8fd07d3d2210def9ead58e1b4c674c065c659c36c4f7d44738dec650d2d836c9df754ffe268094fcf497c12d650e90ac8199f72d97b49456ee60605651993279f61025f08ff14364ad5244e50987cadcca61e9540c272c776760580e94cb51aa32d307645070702dafcfc6a25cd4b80bc23b69c3d20218e7aab5b3731356d7bae79f33fc53710b039a909ca9f217f729d990ea23ba67aa95e942e96253f55d67e36c9fa2c722f6fe8d802b80cf963795bfe7c5c6e8d50280be96a0f9afb88deedf441f7e24c02af9381436360048f80b6c38ad8b40cd66f8397075ed2d3c1c505ca0c36afe01f51b361faf4d4a26cc5cb3c47b59a0399ea012e36dd7659073db66e4d2f2f06015aae143471880055c136e7d792b643d9126073c367e9573af3eb947d17bf7622f7b08ce4d1fac36e24465aef2e89e9ca8de9c7a3e4f03d36b2d4616a488790026b7632df3f5ee591f37afedb8c0578907669500ae8ea3d519bef746d8f6c76"
	verifierArtifactTestProof               = "0x45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795efefbcd563252d443f9828f2a30139f9ae0e25ad24a23bbfa43286618af69ba500000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000380627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f761d926910cd37f0427ac13baa3386a8b649e9c83c83a845ed4355ec4df7b8d2e3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795efefbcd563252d443f9828f2a30139f9ae0e25ad24a23bbfa43286618af69ba500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000002a0f87b07b17aed58b7ed4f12adc3a9b558f6f6405efaa177e8be6cc4f5ee464b82da6118bacf3be9f7e55ebcd74e742b58befea375db7650fee74e42a7ad27e17d412d96f28c9bd212ba27d0da92a15f27337e6ea50cbfaf4c4223af098c47ccf3b45deacc486e6a3ccaa3dbd3ead0e5cde3c3ce6aeea8c9088ea4d53036fdd94e9431ddbfb4276462d98f2fcf32458a0bfe21839c42a8f82bba3847de86d323d73a89b167b1513f5eaac606f903af4688218714b356282bac1cce90e1186c82f953fec9e4f5725b75bf8c0501257419c4fb7aba8c3024792652e9bb81b494a211eaf1c7d70ef5145001f9116853718f2bb6c919df8e7ac47dd74b0ed9be4d55632be8de8b7a68ebe83143f17ba5adc06112af77398b9106f21ea3a0fb8dd30165222a757b0cf1a9a2f2f563cfcdc04171d756ee12a4574a5f7bd9a8fd07d3d2210def9ead58e1b4c674c065c659c36c4f7d44738dec650d2d836c9df754ffe268094fcf497c12d650e90ac8199f72d97b49456ee60605651993279f61025f08ff14364ad5244e50987cadcca61e9540c272c776760580e94cb51aa32d307645070702dafcfc6a25cd4b80bc23b69c3d20218e7aab5b3731356d7bae79f33fc53710b039a909ca9f217f729d990ea23ba67aa95e942e96253f55d67e36c9fa2c722f6fe8d802b80cf963795bfe7c5c6e8d50280be96a0f9afb88deedf441f7e24c02af9381436360048f80b6c38ad8b40cd66f8397075ed2d3c1c505ca0c36afe01f51b361faf4d4a26cc5cb3c47b59a0399ea012e36dd7659073db66e4d2f2f06015aae143471880055c136e7d792b643d9126073c367e9573af3eb947d17bf7622f7b08ce4d1fac36e24465aef2e89e9ca8de9c7a3e4f03d36b2d4616a488790026b7632df3f5ee591f37afedb8c0578907669500ae8ea3d519bef746d8f6c76"
)

func TestBuildVerifierGateBatchContractJoinsCanonicalAuthorizationRefAcrossBatches(t *testing.T) {
	authBatch, targetBatch, authRef := verifierGateTestBatches(t)

	contract, err := BuildVerifierGateBatchContract([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierGateBatchContract returned error: %v", err)
	}
	if contract.PublicInputs.BatchDataHash != canonicalBatchDataHash(targetBatch) {
		t.Fatalf("batch_data_hash = %s, want %s", contract.PublicInputs.BatchDataHash, canonicalBatchDataHash(targetBatch))
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
	if contract.SolidityExport.Calldata.PublicInputs.BatchDataHash != "0x"+canonicalBatchDataHash(targetBatch) {
		t.Fatalf("solidity public_inputs.batch_data_hash = %s, want %s", contract.SolidityExport.Calldata.PublicInputs.BatchDataHash, "0x"+canonicalBatchDataHash(targetBatch))
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
	if artifact.VerifierInterface.Groth16Fixture.ProofBytesEncoding != "abi.encode(bytes32 transitionWitnessHash, bytes32 entrySetHash, bytes32 acceptedBalancesHash, bytes32 acceptedPositionsHash, bytes32 acceptedPayoutsHash, bytes32 acceptedWithdrawalRootHash, bytes32 acceptedWithdrawalLeavesHash, bytes32 escapeCollateralRootHash, bytes32 escapeCollateralLeavesHash, uint256[2] a, uint256[2][2] b, uint256[2] c, uint256[2] commitments, uint256[2] commitmentPok)" {
		t.Fatalf("unexpected groth16 proof bytes encoding: %s", artifact.VerifierInterface.Groth16Fixture.ProofBytesEncoding)
	}
	if !artifact.VerifierInterface.Groth16Fixture.ExpectedVerdict {
		t.Fatalf("expected groth16 fixture verdict to stay true")
	}
	expectedGroth16PublicInputs, err := buildGroth16PublicInputsHex(
		artifact.VerifierInterface.Calldata.Context.BatchEncodingHash,
		artifact.VerifierInterface.Calldata.Context.AuthProofHash,
		artifact.VerifierInterface.Calldata.Context.VerifierGateHash,
		func() string {
			witnessMaterial, witnessErr := BuildStateTransitionWitnessMaterial([]StoredBatch{authBatch}, targetBatch)
			if witnessErr != nil {
				t.Fatalf("BuildStateTransitionWitnessMaterial returned error: %v", witnessErr)
			}
			transitionWitnessHash, transitionErr := buildTransitionWitnessHash(artifact.VerifierInterface.Calldata.Context, witnessMaterial)
			if transitionErr != nil {
				t.Fatalf("buildTransitionWitnessHash returned error: %v", transitionErr)
			}
			return transitionWitnessHash
		}(),
	)
	if err != nil {
		t.Fatalf("buildGroth16PublicInputsHex returned error: %v", err)
	}
	if !reflect.DeepEqual(artifact.VerifierInterface.Groth16Fixture.PublicInputs, expectedGroth16PublicInputs) {
		t.Fatalf(
			"groth16 public_inputs = %v, want %v",
			artifact.VerifierInterface.Groth16Fixture.PublicInputs,
			expectedGroth16PublicInputs,
		)
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0] != "0x222a757b0cf1a9a2f2f563cfcdc04171d756ee12a4574a5f7bd9a8fd07d3d221" {
		t.Fatalf("unexpected groth16 proof tuple a[0]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.A[0])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1] != "0x10b039a909ca9f217f729d990ea23ba67aa95e942e96253f55d67e36c9fa2c72" {
		t.Fatalf("unexpected groth16 proof tuple b[1][1]: %s", artifact.VerifierInterface.Groth16Fixture.ProofTuple.B[1][1])
	}
	if artifact.VerifierInterface.Groth16Fixture.ProofTuple.C[1] != "0x02af9381436360048f80b6c38ad8b40cd66f8397075ed2d3c1c505ca0c36afe0" {
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
	firstMaterial, err := BuildStateTransitionWitnessMaterial([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildStateTransitionWitnessMaterial returned error: %v", err)
	}
	firstArtifact, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, targetBatch)
	if err != nil {
		t.Fatalf("BuildVerifierArtifactBundle(first) returned error: %v", err)
	}
	firstContext := firstArtifact.VerifierInterface.Calldata.Context
	firstProofBytes, firstPublicInputs, _, err := buildBatchSpecificGroth16Proof(firstContext, firstMaterial)
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
	secondBatch := targetBatch
	secondBatch.StateRoot = strings.TrimPrefix(secondPublicInputs.NextStateRoot, "0x")
	secondMaterial, err := BuildStateTransitionWitnessMaterial([]StoredBatch{authBatch}, secondBatch)
	if err != nil {
		t.Fatalf("BuildStateTransitionWitnessMaterial(second) returned error: %v", err)
	}
	secondArtifact, err := BuildVerifierArtifactBundle([]StoredBatch{authBatch}, secondBatch)
	if err != nil {
		t.Fatalf("BuildVerifierArtifactBundle(second) returned error: %v", err)
	}
	secondProofBytes, secondPublicInputsHex, _, err := buildBatchSpecificGroth16Proof(secondContext, secondMaterial)
	if err != nil {
		t.Fatalf("buildBatchSpecificGroth16Proof(second) returned error: %v", err)
	}

	if "0x"+hex.EncodeToString(firstProofBytes) != firstArtifact.VerifierInterface.Calldata.ProofDataFields.ProofBytes {
		t.Fatalf("unexpected first proof bytes: 0x%s", hex.EncodeToString(firstProofBytes))
	}
	if "0x"+hex.EncodeToString(secondProofBytes) != secondArtifact.VerifierInterface.Calldata.ProofDataFields.ProofBytes {
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
