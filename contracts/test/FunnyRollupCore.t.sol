// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FunnyRollupCore} from "../src/FunnyRollupCore.sol";
import {FunnyRollupVerifier, IFunnyRollupBatchVerifier, FunnyRollupVerifierTypes} from "../src/FunnyRollupVerifier.sol";
import {DSTest} from "./DSTest.sol";

abstract contract FunnyRollupArtifactFixtures is DSTest {
    bytes32 internal constant GO_ARTIFACT_BATCH_ENCODING_HASH =
        hex"3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb";
    bytes32 internal constant GO_ARTIFACT_AUTH_PROOF_HASH =
        hex"1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795";
    bytes32 internal constant GO_ARTIFACT_VERIFIER_GATE_HASH =
        hex"ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec5";
    bytes32 internal constant GO_ARTIFACT_SECOND_VERIFIER_GATE_HASH =
        hex"aded06975aa053c2bcc1d21f42e5b7f293723cfd9c0baaa25a8d49143d3fc9a1";
    bytes32 internal constant GO_ARTIFACT_PROOF_SCHEMA_HASH =
        hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7";
    bytes32 internal constant GO_ARTIFACT_PUBLIC_SIGNALS_HASH =
        hex"404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf";
    bytes32 internal constant GO_ARTIFACT_PROOF_DATA_SCHEMA_HASH =
        hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f7";
    bytes32 internal constant GO_ARTIFACT_PROOF_VERSION_HASH =
        hex"4c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a23";

    function goArtifactProofBytes() internal pure returns (bytes memory) {
        return hex"2ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9";
    }

    function goArtifactProofData() internal pure returns (bytes memory) {
        return hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001002ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9";
    }

    function goArtifactProof() internal pure returns (bytes memory) {
        return hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001e0627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795ad3c7037b47a17484ee261126e1d047fb971227cbd4d47f7e7cdce7a07da2ec500000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001002ae38f93d01f95c5d2dd378d9d0bff5fcdb70378c695c92589b46162d984666a1f6714793fef7239056704dde6c791e4b51ff265c01dd3c8ece3dad0fe61e01c17007c33247837bb42b9ff55729610fe64b5351dc6a7747f398ee4c4946414301145b55ded49008dc1990656386723d1011ade15765551a24ef9fdb695d6df0404b7a73e36d81c7b256c19e307b6d602f900fdcc929376dfc83d7e8f8d41503e23302c497d3c163c9229f1085fa9c3e1f19a51dadfa6f7e34411437b30e8cc1926fb84c15db12174f7056497b0fc8fc6fa86e451dab4c1690a90107394e60d060b7fdb3503a6cfe7d869f02271bebeb0f026ce0ba65b808178af834b64a68be9";
    }

    function goArtifactSecondProofBytes() internal pure returns (bytes memory) {
        return hex"14f1c8679783fc23c9cb13cf16dedb0144fff7e565f40fa101488c1faf22014f2db41a1351cdc25bfb0f574284194c0771c910a12056d2f958bd70a4e89105181a4b416b0638f16b171230fdeea324eecfb5f2915fb81eaec3d0dfb9fa37e7fa1931cbc56620a048c44b51031194d4aa2d0dfa0224517fe2d92e14744a6e127a01b9fb505ae13d4559ccff39af8944fb44911bcfc725cdb68592534b8abbdf8219b7c28f1a414fb3c4be1776612a6439caf330b1a4d8d46481a938263785a31320652243d775a66961fe63706bfda0f21772729871bd3a53074af781fb625b8a26eca88d3b9b55d5eedc9b64343f0721b574bb04242a6dede27f4259c40acec3";
    }

    function goArtifactSecondProofData() internal pure returns (bytes memory) {
        return hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795aded06975aa053c2bcc1d21f42e5b7f293723cfd9c0baaa25a8d49143d3fc9a100000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000010014f1c8679783fc23c9cb13cf16dedb0144fff7e565f40fa101488c1faf22014f2db41a1351cdc25bfb0f574284194c0771c910a12056d2f958bd70a4e89105181a4b416b0638f16b171230fdeea324eecfb5f2915fb81eaec3d0dfb9fa37e7fa1931cbc56620a048c44b51031194d4aa2d0dfa0224517fe2d92e14744a6e127a01b9fb505ae13d4559ccff39af8944fb44911bcfc725cdb68592534b8abbdf8219b7c28f1a414fb3c4be1776612a6439caf330b1a4d8d46481a938263785a31320652243d775a66961fe63706bfda0f21772729871bd3a53074af781fb625b8a26eca88d3b9b55d5eedc9b64343f0721b574bb04242a6dede27f4259c40acec3";
    }

    function goArtifactSecondProof() internal pure returns (bytes memory) {
        return hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795aded06975aa053c2bcc1d21f42e5b7f293723cfd9c0baaa25a8d49143d3fc9a100000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001e0627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795aded06975aa053c2bcc1d21f42e5b7f293723cfd9c0baaa25a8d49143d3fc9a100000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000010014f1c8679783fc23c9cb13cf16dedb0144fff7e565f40fa101488c1faf22014f2db41a1351cdc25bfb0f574284194c0771c910a12056d2f958bd70a4e89105181a4b416b0638f16b171230fdeea324eecfb5f2915fb81eaec3d0dfb9fa37e7fa1931cbc56620a048c44b51031194d4aa2d0dfa0224517fe2d92e14744a6e127a01b9fb505ae13d4559ccff39af8944fb44911bcfc725cdb68592534b8abbdf8219b7c28f1a414fb3c4be1776612a6439caf330b1a4d8d46481a938263785a31320652243d775a66961fe63706bfda0f21772729871bd3a53074af781fb625b8a26eca88d3b9b55d5eedc9b64343f0721b574bb04242a6dede27f4259c40acec3";
    }
}

contract FunnyRollupCoreTest is FunnyRollupArtifactFixtures {
    function testRecordBatchMetadata() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);

        bytes32 batchDataHash = keccak256("batch_data_1");
        bytes32 nextStateRoot = keccak256("next_state_root_1");
        core.recordBatchMetadata(1, batchDataHash, genesisStateRoot, nextStateRoot);

        (bytes32 storedDataHash, bytes32 storedPrevStateRoot, bytes32 storedNextStateRoot) = core.batchMetadata(1);
        assertEq(uint256(core.latestBatchId()), 1, "latestBatchId mismatch");
        assertEqBytes32(core.latestStateRoot(), nextStateRoot, "latestStateRoot mismatch");
        assertEqBytes32(storedDataHash, batchDataHash, "batchDataHash mismatch");
        assertEqBytes32(storedPrevStateRoot, genesisStateRoot, "prevStateRoot mismatch");
        assertEqBytes32(storedNextStateRoot, nextStateRoot, "nextStateRoot mismatch");
    }

    function testRejectsNonOperatorRecorder() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        BatchRecorderCaller caller = new BatchRecorderCaller();

        (bool ok,) = caller.callRecordBatchMetadata(
            core, 1, keccak256("batch_data_1"), genesisStateRoot, keccak256("next_state_root_1")
        );
        assertTrue(!ok, "expected only-operator revert");
    }

    function testRejectsPrevStateRootMismatch() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);

        try core.recordBatchMetadata(
            1, keccak256("batch_data_1"), keccak256("wrong_prev_state_root"), keccak256("next_state_root_1")
        ) {
            revert("expected prev state root mismatch");
        } catch {}
    }

    function testAcceptVerifiedBatchAdvancesAcceptedStateRootWhenAllAuthRowsAreJoined() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier bootstrapVerifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(bootstrapVerifier));

        FunnyRollupCore.VerifierPublicInputs memory bootstrapPublicInputs =
            buildPublicInputs(genesisStateRoot, goArtifactPublicInputs().prevStateRoot);
        FunnyRollupCore.L1BatchMetadata memory bootstrapMetadataSubset = buildMetadataSubset(bootstrapPublicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory bootstrapAuthStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        bootstrapAuthStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;
        bytes32 bootstrapVerifierGateHash =
            core.hashVerifierGateBatch(bootstrapPublicInputs, core.hashAuthStatuses(bootstrapAuthStatuses));
        bootstrapVerifier.setVerdict(bootstrapVerifierGateHash, true);
        core.recordBatchMetadata(
            1,
            bootstrapPublicInputs.batchDataHash,
            bootstrapPublicInputs.prevStateRoot,
            bootstrapPublicInputs.nextStateRoot
        );
        core.acceptVerifiedBatch(bootstrapPublicInputs, bootstrapMetadataSubset, bootstrapAuthStatuses, hex"1234");

        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs = goArtifactPublicInputs();
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        core.recordBatchMetadata(2, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);

        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            core.buildVerifierContext(publicInputs, authProofHash);
        bytes32 verifierGateHash = verifierContext.verifierGateHash;

        core.acceptVerifiedBatch(
            publicInputs, metadataSubset, authStatuses, buildVerifierProof(verifier, verifierContext)
        );

        assertEq(uint256(core.latestAcceptedBatchId()), 2, "latestAcceptedBatchId mismatch");
        assertEqBytes32(core.latestAcceptedStateRoot(), publicInputs.nextStateRoot, "latestAcceptedStateRoot mismatch");
        assertEqBytes32(verifierContext.authProofHash, authProofHash, "authProofHash mismatch");
        assertEqBytes32(verifierContext.verifierGateHash, verifierGateHash, "verifierGateHash mismatch");
    }

    function testRejectsNonJoinedAuthStatusBeforeVerifierCall() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.MISSING_TRADING_KEY_AUTHORIZED;
        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234") {
            revert("expected auth proof rejection");
        } catch {}

        assertEq(verifier.verifyCalls(), 0, "verifier should not be called for non-JOINED auth rows");
        assertEq(uint256(core.latestAcceptedBatchId()), 0, "latestAcceptedBatchId should stay unset");
        assertEqBytes32(core.latestAcceptedStateRoot(), genesisStateRoot, "accepted state root should stay at genesis");
    }

    function testRejectsFailedVerifierVerdict() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            core.buildVerifierContext(publicInputs, authProofHash);
        bytes memory verifierProof = buildVerifierProofWithSignals(
            verifier, verifierContext.batchEncodingHash, verifierContext.authProofHash, bytes32(uint256(1))
        );

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, verifierProof) {
            revert("expected verifier verdict rejection");
        } catch {}

        assertEq(uint256(core.latestAcceptedBatchId()), 0, "latestAcceptedBatchId should stay unset");
        assertEqBytes32(core.latestAcceptedStateRoot(), genesisStateRoot, "accepted state root should stay at genesis");
    }

    function testAcceptVerifiedBatchPassesFullVerifierContext() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            core.buildVerifierContext(publicInputs, authProofHash);
        bytes32 verifierGateHash = verifierContext.verifierGateHash;
        verifier.setVerdict(verifierGateHash, true);

        core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234");

        assertEqBytes32(verifier.lastBatchEncodingHash(), core.SHADOW_BATCH_V1_HASH(), "batchEncodingHash mismatch");
        assertEqBytes32(verifier.lastAuthProofHash(), authProofHash, "authProofHash mismatch");
        assertEqBytes32(verifier.lastVerifierGateHash(), verifierGateHash, "verifierGateHash mismatch");
        assertEq(uint256(verifier.lastBatchId()), publicInputs.batchId, "batchId mismatch");
    }

    function testVerifierGateHashMatchesGoArtifactParityFixture() public {
        FunnyRollupCore core = new FunnyRollupCore(address(this), bytes32(uint256(1)));
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();

        FunnyRollupCore.VerifierPublicInputs memory publicInputs = FunnyRollupCore.VerifierPublicInputs({
            batchId: 2,
            firstSequenceNo: 7,
            lastSequenceNo: 7,
            entryCount: 1,
            batchDataHash: hex"8a6a2bdede255ffcf3de22c1c07efd93df958681f7582e293ce0bbe19a47ffe0",
            prevStateRoot: hex"749de8c4520e934e38bb7cd42bb62208a408b53f7373fa2b333cc56c62102e46",
            balancesRoot: hex"a18255dc375d022d3d805eb70bb97aa1d44562a0dc9a6d5b0a3b4a103b6ad319",
            ordersRoot: hex"1854c9b450264fa6410c58d2f66c3b7f32425fc528d88fac9f5624d2839f93ce",
            positionsFundingRoot: hex"069f457419a48abd7327a9a22cc0a53c18101bf4799eb86074dbed63db7f6ac3",
            withdrawalsRoot: hex"4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96",
            nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5"
        });
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, authProofHash);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            core.buildVerifierContext(publicInputs, authProofHash);
        bytes32 verifierProofHash = verifier.GROTH16_BN254_2X128_SHADOW_STATE_ROOT_GATE_V1_HASH();
        bytes memory verifierProof = buildVerifierProof(verifier, verifierContext);
        uint256[6] memory groth16PublicInputs = verifier.deriveGroth16PublicInputs(
            verifierContext.batchEncodingHash, verifierContext.authProofHash, verifierContext.verifierGateHash
        );

        assertEqBytes32(
            core.SHADOW_BATCH_V1_HASH(), GO_ARTIFACT_BATCH_ENCODING_HASH, "batchEncodingHash fixture mismatch"
        );
        assertEqBytes32(authProofHash, GO_ARTIFACT_AUTH_PROOF_HASH, "authProofHash fixture mismatch");
        assertEqBytes32(verifierGateHash, GO_ARTIFACT_VERIFIER_GATE_HASH, "verifierGateHash fixture mismatch");
        assertEqBytes32(
            verifier.PROOF_SCHEMA_V1_HASH(), GO_ARTIFACT_PROOF_SCHEMA_HASH, "proofSchemaHash fixture mismatch"
        );
        assertEqBytes32(
            verifier.PUBLIC_SIGNALS_V1_HASH(), GO_ARTIFACT_PUBLIC_SIGNALS_HASH, "publicSignalsHash fixture mismatch"
        );
        assertEqBytes32(
            verifier.PROOF_DATA_SCHEMA_V1_HASH(),
            GO_ARTIFACT_PROOF_DATA_SCHEMA_HASH,
            "proofDataSchemaHash fixture mismatch"
        );
        assertEqBytes32(verifierProofHash, GO_ARTIFACT_PROOF_VERSION_HASH, "proofVersionHash fixture mismatch");
        assertEqBytes32(
            verifierContext.batchEncodingHash, GO_ARTIFACT_BATCH_ENCODING_HASH, "context batchEncodingHash mismatch"
        );
        assertEqBytes32(
            verifierContext.verifierGateHash, GO_ARTIFACT_VERIFIER_GATE_HASH, "context verifierGateHash mismatch"
        );
        assertEqBytes32(
            verifier.hashVerifierGate(verifierContext),
            GO_ARTIFACT_VERIFIER_GATE_HASH,
            "verifier contract hash mismatch"
        );
        assertEq(groth16PublicInputs[0], 0x3b6489209bd528a9779ecc9db44d4d05, "batchEncodingHashHi mismatch");
        assertEq(groth16PublicInputs[1], 0xdceb8faba670a6922ff939d841f202cb, "batchEncodingHashLo mismatch");
        assertEq(groth16PublicInputs[2], 0x1e7c5c1c118b439a090ebf5654651794, "authProofHashHi mismatch");
        assertEq(groth16PublicInputs[3], 0x76e94bae5ba6a5ae0f146ec3866c8795, "authProofHashLo mismatch");
        assertEq(groth16PublicInputs[4], 0xad3c7037b47a17484ee261126e1d047f, "verifierGateHashHi mismatch");
        assertEq(groth16PublicInputs[5], 0xb971227cbd4d47f7e7cdce7a07da2ec5, "verifierGateHashLo mismatch");
        assertEqBytes32(
            keccak256(
                buildProofData(
                    verifier,
                    verifierContext.batchEncodingHash,
                    verifierContext.authProofHash,
                    verifierContext.verifierGateHash
                )
            ),
            keccak256(goArtifactProofData()),
            "proofData fixture mismatch"
        );
        assertEqBytes32(keccak256(verifierProof), keccak256(goArtifactProof()), "verifierProof fixture mismatch");
    }

    function testRejectsAcceptanceWhenMetadataWasNotRecorded() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234") {
            revert("expected missing recorded metadata rejection");
        } catch {}

        assertEq(verifier.verifyCalls(), 0, "verifier should not be called when metadata was not recorded");
        assertEq(uint256(core.latestAcceptedBatchId()), 0, "latestAcceptedBatchId should stay unset");
        assertEqBytes32(core.latestAcceptedStateRoot(), genesisStateRoot, "accepted state root should stay at genesis");
    }

    function testRejectsAcceptanceWhenRecordedMetadataMismatchesCalldata() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        core.recordBatchMetadata(
            1, keccak256("different_batch_data"), publicInputs.prevStateRoot, publicInputs.nextStateRoot
        );
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234") {
            revert("expected recorded metadata mismatch rejection");
        } catch {}

        assertEq(verifier.verifyCalls(), 0, "verifier should not be called when recorded metadata mismatches");
        assertEq(uint256(core.latestAcceptedBatchId()), 0, "latestAcceptedBatchId should stay unset");
        assertEqBytes32(core.latestAcceptedStateRoot(), genesisStateRoot, "accepted state root should stay at genesis");
    }

    function buildPublicInputs(bytes32 prevStateRoot, bytes32 nextStateRoot)
        internal
        pure
        returns (FunnyRollupCore.VerifierPublicInputs memory)
    {
        return FunnyRollupCore.VerifierPublicInputs({
            batchId: 1,
            firstSequenceNo: 11,
            lastSequenceNo: 17,
            entryCount: 4,
            batchDataHash: keccak256("batch_data_1"),
            prevStateRoot: prevStateRoot,
            balancesRoot: keccak256("balances_root_1"),
            ordersRoot: keccak256("orders_root_1"),
            positionsFundingRoot: keccak256("positions_funding_root_1"),
            withdrawalsRoot: keccak256("withdrawals_root_1"),
            nextStateRoot: nextStateRoot
        });
    }

    function buildMetadataSubset(FunnyRollupCore.VerifierPublicInputs memory publicInputs)
        internal
        pure
        returns (FunnyRollupCore.L1BatchMetadata memory)
    {
        return FunnyRollupCore.L1BatchMetadata({
            batchId: publicInputs.batchId,
            batchDataHash: publicInputs.batchDataHash,
            prevStateRoot: publicInputs.prevStateRoot,
            nextStateRoot: publicInputs.nextStateRoot
        });
    }

    function goArtifactPublicInputs() internal pure returns (FunnyRollupCore.VerifierPublicInputs memory) {
        return FunnyRollupCore.VerifierPublicInputs({
            batchId: 2,
            firstSequenceNo: 7,
            lastSequenceNo: 7,
            entryCount: 1,
            batchDataHash: hex"8a6a2bdede255ffcf3de22c1c07efd93df958681f7582e293ce0bbe19a47ffe0",
            prevStateRoot: hex"749de8c4520e934e38bb7cd42bb62208a408b53f7373fa2b333cc56c62102e46",
            balancesRoot: hex"a18255dc375d022d3d805eb70bb97aa1d44562a0dc9a6d5b0a3b4a103b6ad319",
            ordersRoot: hex"1854c9b450264fa6410c58d2f66c3b7f32425fc528d88fac9f5624d2839f93ce",
            positionsFundingRoot: hex"069f457419a48abd7327a9a22cc0a53c18101bf4799eb86074dbed63db7f6ac3",
            withdrawalsRoot: hex"4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96",
            nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5"
        });
    }

    function buildVerifierProof(FunnyRollupVerifier verifier, FunnyRollupVerifierTypes.VerifierContext memory context)
        internal
        view
        returns (bytes memory)
    {
        return buildVerifierProofWithSignals(
            verifier, context.batchEncodingHash, context.authProofHash, context.verifierGateHash
        );
    }

    function buildVerifierProofWithSignals(
        FunnyRollupVerifier verifier,
        bytes32 batchEncodingHash,
        bytes32 authProofHash,
        bytes32 verifierGateHash
    ) internal view returns (bytes memory) {
        return abi.encode(
            verifier.PROOF_SCHEMA_V1_HASH(),
            verifier.PUBLIC_SIGNALS_V1_HASH(),
            FunnyRollupVerifierTypes.ProofPublicSignals({
                batchEncodingHash: batchEncodingHash, authProofHash: authProofHash, verifierGateHash: verifierGateHash
            }),
            buildProofData(verifier, batchEncodingHash, authProofHash, verifierGateHash)
        );
    }

    function buildProofData(
        FunnyRollupVerifier verifier,
        bytes32 batchEncodingHash,
        bytes32 authProofHash,
        bytes32 verifierGateHash
    ) internal view returns (bytes memory) {
        return abi.encode(
            verifier.PROOF_DATA_SCHEMA_V1_HASH(),
            GO_ARTIFACT_PROOF_VERSION_HASH,
            batchEncodingHash,
            authProofHash,
            verifierGateHash,
            buildGroth16ProofBytes()
        );
    }

    function buildGroth16ProofBytes() internal pure returns (bytes memory) {
        return goArtifactProofBytes();
    }
}

contract FunnyRollupVerifierTest is FunnyRollupArtifactFixtures {
    function testAcceptsGoArtifactParityFixture() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildGoArtifactContext();
        bytes memory verifierProof = buildGoArtifactProof(verifier, context);

        assertTrue(verifier.verifyBatch(context, verifierProof), "expected Go artifact fixture to verify");
    }

    function testAcceptsSecondBatchSpecificGoArtifact() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildSecondGoArtifactContext();
        bytes memory verifierProof = buildSecondGoArtifactProof(verifier, context);
        uint256[6] memory publicInputs = verifier.deriveGroth16PublicInputs(
            context.batchEncodingHash, context.authProofHash, context.verifierGateHash
        );

        assertEqBytes32(
            verifier.hashVerifierGate(context),
            GO_ARTIFACT_SECOND_VERIFIER_GATE_HASH,
            "second verifierGateHash mismatch"
        );
        assertEqBytes32(keccak256(verifierProof), keccak256(goArtifactSecondProof()), "second proof mismatch");
        assertEqBytes32(
            keccak256(
                buildProofData(
                    verifier,
                    context.batchEncodingHash,
                    context.authProofHash,
                    context.verifierGateHash,
                    goArtifactSecondProofBytes()
                )
            ),
            keccak256(goArtifactSecondProofData()),
            "second proofData mismatch"
        );
        assertEq(publicInputs[4], 0xaded06975aa053c2bcc1d21f42e5b7f2, "second verifierGateHashHi mismatch");
        assertEq(publicInputs[5], 0x93723cfd9c0baaa25a8d49143d3fc9a1, "second verifierGateHashLo mismatch");
        assertTrue(
            keccak256(goArtifactProofBytes()) != keccak256(goArtifactSecondProofBytes()),
            "expected batch-specific proof bytes divergence"
        );
        assertTrue(verifier.verifyBatch(context, verifierProof), "expected second Go artifact to verify");
    }

    function testRejectsContextVerifierGateHashMismatch() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildGoArtifactContext();
        bytes memory verifierProof = buildGoArtifactProof(verifier, context);

        context.verifierGateHash = bytes32(uint256(1));

        assertTrue(!verifier.verifyBatch(context, verifierProof), "expected mismatched verifierGateHash rejection");
    }

    function testRejectsProofPublicSignalAuthProofHashMismatch() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildGoArtifactContext();
        bytes memory verifierProof = abi.encode(
            verifier.PROOF_SCHEMA_V1_HASH(),
            verifier.PUBLIC_SIGNALS_V1_HASH(),
            FunnyRollupVerifierTypes.ProofPublicSignals({
                batchEncodingHash: context.batchEncodingHash,
                authProofHash: bytes32(uint256(1)),
                verifierGateHash: context.verifierGateHash
            }),
            buildProofData(
                verifier,
                context.batchEncodingHash,
                bytes32(uint256(1)),
                context.verifierGateHash,
                buildGroth16ProofBytes()
            )
        );

        assertTrue(!verifier.verifyBatch(context, verifierProof), "expected authProofHash public-signal rejection");
    }

    function testRejectsMalformedInnerGroth16ProofBytes() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildGoArtifactContext();
        bytes memory verifierProof = abi.encode(
            verifier.PROOF_SCHEMA_V1_HASH(),
            verifier.PUBLIC_SIGNALS_V1_HASH(),
            FunnyRollupVerifierTypes.ProofPublicSignals({
                batchEncodingHash: context.batchEncodingHash,
                authProofHash: context.authProofHash,
                verifierGateHash: context.verifierGateHash
            }),
            buildProofData(
                verifier, context.batchEncodingHash, context.authProofHash, context.verifierGateHash, hex"1234"
            )
        );

        assertTrue(!verifier.verifyBatch(context, verifierProof), "expected malformed inner proof bytes rejection");
    }

    function testRejectsMalformedProofBytes() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();

        assertTrue(!verifier.verifyBatch(buildGoArtifactContext(), hex"1234"), "expected malformed proof rejection");
    }

    function testDerivesGoArtifactGroth16PublicInputs() public {
        FunnyRollupVerifier verifier = new FunnyRollupVerifier();
        FunnyRollupVerifierTypes.VerifierContext memory context = buildGoArtifactContext();
        uint256[6] memory publicInputs = verifier.deriveGroth16PublicInputs(
            context.batchEncodingHash, context.authProofHash, context.verifierGateHash
        );

        assertEq(publicInputs[0], 0x3b6489209bd528a9779ecc9db44d4d05, "batchEncodingHashHi mismatch");
        assertEq(publicInputs[1], 0xdceb8faba670a6922ff939d841f202cb, "batchEncodingHashLo mismatch");
        assertEq(publicInputs[2], 0x1e7c5c1c118b439a090ebf5654651794, "authProofHashHi mismatch");
        assertEq(publicInputs[3], 0x76e94bae5ba6a5ae0f146ec3866c8795, "authProofHashLo mismatch");
        assertEq(publicInputs[4], 0xad3c7037b47a17484ee261126e1d047f, "verifierGateHashHi mismatch");
        assertEq(publicInputs[5], 0xb971227cbd4d47f7e7cdce7a07da2ec5, "verifierGateHashLo mismatch");
    }

    function buildGoArtifactContext() internal pure returns (FunnyRollupVerifierTypes.VerifierContext memory) {
        return FunnyRollupVerifierTypes.VerifierContext({
            batchEncodingHash: GO_ARTIFACT_BATCH_ENCODING_HASH,
            publicInputs: FunnyRollupVerifierTypes.VerifierPublicInputs({
                batchId: 2,
                firstSequenceNo: 7,
                lastSequenceNo: 7,
                entryCount: 1,
                batchDataHash: hex"8a6a2bdede255ffcf3de22c1c07efd93df958681f7582e293ce0bbe19a47ffe0",
                prevStateRoot: hex"749de8c4520e934e38bb7cd42bb62208a408b53f7373fa2b333cc56c62102e46",
                balancesRoot: hex"a18255dc375d022d3d805eb70bb97aa1d44562a0dc9a6d5b0a3b4a103b6ad319",
                ordersRoot: hex"1854c9b450264fa6410c58d2f66c3b7f32425fc528d88fac9f5624d2839f93ce",
                positionsFundingRoot: hex"069f457419a48abd7327a9a22cc0a53c18101bf4799eb86074dbed63db7f6ac3",
                withdrawalsRoot: hex"4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96",
                nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5"
            }),
            authProofHash: GO_ARTIFACT_AUTH_PROOF_HASH,
            verifierGateHash: GO_ARTIFACT_VERIFIER_GATE_HASH
        });
    }

    function buildSecondGoArtifactContext() internal pure returns (FunnyRollupVerifierTypes.VerifierContext memory) {
        return FunnyRollupVerifierTypes.VerifierContext({
            batchEncodingHash: GO_ARTIFACT_BATCH_ENCODING_HASH,
            publicInputs: FunnyRollupVerifierTypes.VerifierPublicInputs({
                batchId: 1,
                firstSequenceNo: 11,
                lastSequenceNo: 17,
                entryCount: 4,
                batchDataHash: keccak256("batch_data_1"),
                prevStateRoot: keccak256("genesis_state_root"),
                balancesRoot: keccak256("balances_root_1"),
                ordersRoot: keccak256("orders_root_1"),
                positionsFundingRoot: keccak256("positions_funding_root_1"),
                withdrawalsRoot: keccak256("withdrawals_root_1"),
                nextStateRoot: keccak256("next_state_root_1")
            }),
            authProofHash: GO_ARTIFACT_AUTH_PROOF_HASH,
            verifierGateHash: GO_ARTIFACT_SECOND_VERIFIER_GATE_HASH
        });
    }

    function buildGoArtifactProof(FunnyRollupVerifier verifier, FunnyRollupVerifierTypes.VerifierContext memory context)
        internal
        view
        returns (bytes memory)
    {
        assertEqBytes32(verifier.PROOF_SCHEMA_V1_HASH(), GO_ARTIFACT_PROOF_SCHEMA_HASH, "proof schema mismatch");
        assertEqBytes32(
            verifier.PUBLIC_SIGNALS_V1_HASH(), GO_ARTIFACT_PUBLIC_SIGNALS_HASH, "public signals schema mismatch"
        );
        assertEqBytes32(
            verifier.PROOF_DATA_SCHEMA_V1_HASH(), GO_ARTIFACT_PROOF_DATA_SCHEMA_HASH, "proof data schema mismatch"
        );

        return abi.encode(
            verifier.PROOF_SCHEMA_V1_HASH(),
            verifier.PUBLIC_SIGNALS_V1_HASH(),
            FunnyRollupVerifierTypes.ProofPublicSignals({
                batchEncodingHash: context.batchEncodingHash,
                authProofHash: context.authProofHash,
                verifierGateHash: context.verifierGateHash
            }),
            buildProofData(
                verifier,
                context.batchEncodingHash,
                context.authProofHash,
                context.verifierGateHash,
                goArtifactProofBytes()
            )
        );
    }

    function buildSecondGoArtifactProof(
        FunnyRollupVerifier verifier,
        FunnyRollupVerifierTypes.VerifierContext memory context
    ) internal view returns (bytes memory) {
        return abi.encode(
            verifier.PROOF_SCHEMA_V1_HASH(),
            verifier.PUBLIC_SIGNALS_V1_HASH(),
            FunnyRollupVerifierTypes.ProofPublicSignals({
                batchEncodingHash: context.batchEncodingHash,
                authProofHash: context.authProofHash,
                verifierGateHash: context.verifierGateHash
            }),
            buildProofData(
                verifier,
                context.batchEncodingHash,
                context.authProofHash,
                context.verifierGateHash,
                goArtifactSecondProofBytes()
            )
        );
    }

    function buildProofData(
        FunnyRollupVerifier verifier,
        bytes32 batchEncodingHash,
        bytes32 authProofHash,
        bytes32 verifierGateHash,
        bytes memory proofBytes
    ) internal view returns (bytes memory) {
        return abi.encode(
            verifier.PROOF_DATA_SCHEMA_V1_HASH(),
            GO_ARTIFACT_PROOF_VERSION_HASH,
            batchEncodingHash,
            authProofHash,
            verifierGateHash,
            proofBytes
        );
    }

    function buildGroth16ProofBytes() internal pure returns (bytes memory) {
        return goArtifactProofBytes();
    }
}

contract BatchRecorderCaller {
    function callRecordBatchMetadata(
        FunnyRollupCore core,
        uint64 batchId,
        bytes32 batchDataHash,
        bytes32 prevStateRoot,
        bytes32 nextStateRoot
    ) external returns (bool, bytes memory) {
        return address(core)
            .call(abi.encodeCall(core.recordBatchMetadata, (batchId, batchDataHash, prevStateRoot, nextStateRoot)));
    }
}

contract MockFunnyRollupVerifier is IFunnyRollupBatchVerifier {
    bytes32 public expectedVerifierGateHash;
    bool public verdict;
    uint256 public verifyCalls;
    bytes32 public lastBatchEncodingHash;
    bytes32 public lastAuthProofHash;
    bytes32 public lastVerifierGateHash;
    uint64 public lastBatchId;

    function setVerdict(bytes32 expectedVerifierGateHash_, bool verdict_) external {
        expectedVerifierGateHash = expectedVerifierGateHash_;
        verdict = verdict_;
    }

    function verifyBatch(FunnyRollupVerifierTypes.VerifierContext calldata context, bytes calldata)
        external
        returns (bool)
    {
        verifyCalls += 1;
        lastBatchEncodingHash = context.batchEncodingHash;
        lastAuthProofHash = context.authProofHash;
        lastVerifierGateHash = context.verifierGateHash;
        lastBatchId = context.publicInputs.batchId;
        if (context.batchEncodingHash != keccak256("shadow-batch-v1")) {
            return false;
        }
        if (
            context.verifierGateHash
                != keccak256(
                    abi.encode(
                        context.batchEncodingHash,
                        context.publicInputs.batchId,
                        context.publicInputs.firstSequenceNo,
                        context.publicInputs.lastSequenceNo,
                        context.publicInputs.entryCount,
                        context.publicInputs.batchDataHash,
                        context.publicInputs.prevStateRoot,
                        context.publicInputs.balancesRoot,
                        context.publicInputs.ordersRoot,
                        context.publicInputs.positionsFundingRoot,
                        context.publicInputs.withdrawalsRoot,
                        context.publicInputs.nextStateRoot,
                        context.authProofHash
                    )
                )
        ) {
            return false;
        }
        if (expectedVerifierGateHash != bytes32(0) && context.verifierGateHash != expectedVerifierGateHash) {
            return false;
        }
        return verdict;
    }
}
