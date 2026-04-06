// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FunnyRollupCore} from "../src/FunnyRollupCore.sol";
import {FunnyRollupVerifier, IFunnyRollupBatchVerifier, FunnyRollupVerifierTypes} from "../src/FunnyRollupVerifier.sol";
import {FunnyVault} from "../src/FunnyVault.sol";
import {MockUSDT} from "../src/MockUSDT.sol";
import {DSTest} from "./DSTest.sol";

abstract contract FunnyRollupArtifactFixtures is DSTest {
    bytes32 internal constant GO_ARTIFACT_BATCH_ENCODING_HASH =
        hex"3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb";
    bytes32 internal constant GO_ARTIFACT_AUTH_PROOF_HASH =
        hex"1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795";
    bytes32 internal constant GO_ARTIFACT_VERIFIER_GATE_HASH =
        hex"bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e8128626";
    bytes32 internal constant GO_ARTIFACT_SECOND_VERIFIER_GATE_HASH =
        hex"795a355fc2c2e98cbac5561fa98476a65d079471a264f5999e37158d9440e026";
    bytes32 internal constant GO_ARTIFACT_PROOF_SCHEMA_HASH =
        hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7";
    bytes32 internal constant GO_ARTIFACT_PUBLIC_SIGNALS_HASH =
        hex"404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf";
    bytes32 internal constant GO_ARTIFACT_PROOF_DATA_SCHEMA_HASH =
        hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f7";
    bytes32 internal constant GO_ARTIFACT_PROOF_VERSION_HASH =
        hex"4c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a23";

    function goArtifactProofBytes() internal pure returns (bytes memory) {
        return hex"89976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3";
    }

    function goArtifactProofData() internal pure returns (bytes memory) {
        return hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a089976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3";
    }

    function goArtifactProof() internal pure returns (bytes memory) {
        return hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000280627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795bdb8adff0902424ca22e6b3a15581cb9b23705ba8192448826e23449e812862600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a089976dac11073c7be3e8e6290a7fef4a049f44ccce9dcba57bcc0b65c7c534e818868700fa735f25df27d493c3ee7489775c5a5bfb70c5fe1bb910a15ef3fc381cb8cf62a1fa97a484e66c014834a0117e99e5d356a849a855bdb1cd4c2e62cc21b2c6db60136c8744e9f0d5694c71dcd9a513ba0f8f68ad38d0be7ce07e74441eb2fe77dc8b67d30fa18aa0838fe8e8bda225bcd11d3bd836d379d0dd9e09f22299a7d4101f03a810e63e95419b3d02faca6735f835650aca893256e0d3e7e62c669bdb792ee7f45d1610d66e8ef4602e9c49c8b63bcf9d65e2bc10d4ea012b11887cdf149acb3a29f485e6e35d603bc2c6787daf06c7e424b3ac6b6f413c0c2c4609f176329a48b5c2478c941b2ba2b648fdb94f3b016d039c6a054df3d99222c23e22120cabd75aee443c80a3b0929ed0c6943cb489617cd8bd9e7b9b712c0e46ca3d303bf15b33d91d80dcb51d95a017a1a0726a8932a97e5ddf5b5286a40c4706d5920b4717c94ee7ef987052492e0f34480cf12bb626cbc4b79aed1c1e276ef44cb21ffc49b99fb63f14f2e83ffd1855199c5bb96fb6b49678403b15b3";
    }

    function goArtifactSecondProofBytes() internal pure returns (bytes memory) {
        return hex"22781bd0ce3ae00ab124eaa3264beca2ac49e53c51790f798b52b0f548660bfd01c8d66e953513e29069e77388f23101f53e7846c0c16c27be8b58fb56b0df8a2292e43b415e18492ed03ba5e60eed22ab4d22d6df3902998a4c45fb5a660e270f3144829fa5e5da7eb16ee468da123ed79e1685c0d3b4f6c897e3fb5177642c0e501b22c70e241e82d96735088d256c746df0761a66223f6fa9a56884c06b8e22516fd808ab38eb45b5fcce93b758379bf29467b40453b2afba8a656aab9c88052016533e0624b3ff5f29bd2aa9df7b95f7ce3a73a4e740f7db3f005cf775c81fa0fd4e2a058be4a7e09a40ffe5c7f41f28d5791cada3498701da3a619dacd80af472be383ea3c68a174cb4059c5287f3aa520e02a4c0f88b2217bf125c6dfd2641b5c70fbf99f2b252e10c51f4c65100d0fc8bea2efaf297e6de3f75eacd9611816317e8b3a02ae849dad595d7b96eab4937ae76fa8f3b95431c33802c893d2fb5fde088be3c80241778198fb0a650b155f3b375b6a6062d954f995d05878e1a22180db2f61dc25be768ef7993711b2e17e78dd8414ef63baa298c9f46de56";
    }

    function goArtifactSecondProofData() internal pure returns (bytes memory) {
        return hex"627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795795a355fc2c2e98cbac5561fa98476a65d079471a264f5999e37158d9440e02600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a022781bd0ce3ae00ab124eaa3264beca2ac49e53c51790f798b52b0f548660bfd01c8d66e953513e29069e77388f23101f53e7846c0c16c27be8b58fb56b0df8a2292e43b415e18492ed03ba5e60eed22ab4d22d6df3902998a4c45fb5a660e270f3144829fa5e5da7eb16ee468da123ed79e1685c0d3b4f6c897e3fb5177642c0e501b22c70e241e82d96735088d256c746df0761a66223f6fa9a56884c06b8e22516fd808ab38eb45b5fcce93b758379bf29467b40453b2afba8a656aab9c88052016533e0624b3ff5f29bd2aa9df7b95f7ce3a73a4e740f7db3f005cf775c81fa0fd4e2a058be4a7e09a40ffe5c7f41f28d5791cada3498701da3a619dacd80af472be383ea3c68a174cb4059c5287f3aa520e02a4c0f88b2217bf125c6dfd2641b5c70fbf99f2b252e10c51f4c65100d0fc8bea2efaf297e6de3f75eacd9611816317e8b3a02ae849dad595d7b96eab4937ae76fa8f3b95431c33802c893d2fb5fde088be3c80241778198fb0a650b155f3b375b6a6062d954f995d05878e1a22180db2f61dc25be768ef7993711b2e17e78dd8414ef63baa298c9f46de56";
    }

    function goArtifactSecondProof() internal pure returns (bytes memory) {
        return hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795795a355fc2c2e98cbac5561fa98476a65d079471a264f5999e37158d9440e02600000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000280627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f74c620ff3da228edbea86e6c62707674cb14e8cfc20fee57eb70e8adfb03c1a233b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb1e7c5c1c118b439a090ebf565465179476e94bae5ba6a5ae0f146ec3866c8795795a355fc2c2e98cbac5561fa98476a65d079471a264f5999e37158d9440e02600000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000001a022781bd0ce3ae00ab124eaa3264beca2ac49e53c51790f798b52b0f548660bfd01c8d66e953513e29069e77388f23101f53e7846c0c16c27be8b58fb56b0df8a2292e43b415e18492ed03ba5e60eed22ab4d22d6df3902998a4c45fb5a660e270f3144829fa5e5da7eb16ee468da123ed79e1685c0d3b4f6c897e3fb5177642c0e501b22c70e241e82d96735088d256c746df0761a66223f6fa9a56884c06b8e22516fd808ab38eb45b5fcce93b758379bf29467b40453b2afba8a656aab9c88052016533e0624b3ff5f29bd2aa9df7b95f7ce3a73a4e740f7db3f005cf775c81fa0fd4e2a058be4a7e09a40ffe5c7f41f28d5791cada3498701da3a619dacd80af472be383ea3c68a174cb4059c5287f3aa520e02a4c0f88b2217bf125c6dfd2641b5c70fbf99f2b252e10c51f4c65100d0fc8bea2efaf297e6de3f75eacd9611816317e8b3a02ae849dad595d7b96eab4937ae76fa8f3b95431c33802c893d2fb5fde088be3c80241778198fb0a650b155f3b375b6a6062d954f995d05878e1a22180db2f61dc25be768ef7993711b2e17e78dd8414ef63baa298c9f46de56";
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
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        core.setVerifier(address(verifier));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        core.publishBatchData(1, bytes("batch_data_1"));
        core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234");

        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        assertEq(uint256(core.latestAcceptedBatchId()), 1, "latestAcceptedBatchId mismatch");
        assertEqBytes32(core.latestAcceptedStateRoot(), publicInputs.nextStateRoot, "latestAcceptedStateRoot mismatch");
    }

    function testRejectsAcceptWithoutDataPublished() public {
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

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234") {
            revert("expected data-not-published revert");
        } catch {}
    }

    function testPublishBatchDataHappyPath() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);

        bytes memory batchData = bytes("batch_data_1");
        bytes32 batchDataHash = keccak256(batchData);
        core.recordBatchMetadata(1, batchDataHash, genesisStateRoot, keccak256("next_state_root_1"));
        core.publishBatchData(1, batchData);

        assertTrue(core.batchDataPublished(1), "batchDataPublished should be true");
    }

    function testPublishBatchDataRejectsWrongData() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);

        core.recordBatchMetadata(1, keccak256("batch_data_1"), genesisStateRoot, keccak256("next_state_root_1"));

        try core.publishBatchData(1, bytes("wrong_batch_data")) {
            revert("expected data-hash-mismatch revert");
        } catch {}
    }

    function testPublishBatchDataRejectsDoublePublish() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);

        bytes memory batchData = bytes("batch_data_1");
        core.recordBatchMetadata(1, keccak256(batchData), genesisStateRoot, keccak256("next_state_root_1"));
        core.publishBatchData(1, batchData);

        try core.publishBatchData(1, batchData) {
            revert("expected already-published revert");
        } catch {}
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
        core.publishBatchData(1, bytes("batch_data_1"));

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
        core.publishBatchData(1, bytes("batch_data_1"));
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
        core.publishBatchData(1, bytes("batch_data_1"));
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
            nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5",
            conservationHash: hex"6ad792b876594c7e9876d4166919c3431ad4ff4d76504fbad8e786ff5e43ab23"
        });
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;

        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, authProofHash);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            core.buildVerifierContext(publicInputs, authProofHash);
        bytes32 verifierProofHash = verifier.GROTH16_BN254_2X128_SHADOW_STATE_ROOT_GATE_V1_HASH();
        bytes memory verifierProof = buildVerifierProof(verifier, verifierContext);
        bytes32 transitionWitnessHash = verifier.hashTransitionWitness(verifierContext);
        uint256[8] memory groth16PublicInputs = verifier.deriveGroth16PublicInputs(
            verifierContext.batchEncodingHash, verifierContext.authProofHash, verifierContext.verifierGateHash, transitionWitnessHash
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
        assertEq(groth16PublicInputs[4], 0xbdb8adff0902424ca22e6b3a15581cb9, "verifierGateHashHi mismatch");
        assertEq(groth16PublicInputs[5], 0xb23705ba8192448826e23449e8128626, "verifierGateHashLo mismatch");
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

    function testRequestForcedWithdrawalStoresRequest() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        ForcedWithdrawalRequester requester = new ForcedWithdrawalRequester();

        core.setForcedWithdrawalGracePeriod(3600);
        uint64 requestId = requester.request(core, address(0xBEEF), 1250);

        (
            address wallet,
            address recipient,
            uint256 amount,
            uint64 requestedAt,
            uint64 deadlineAt,
            bytes32 satisfiedClaimId,
            uint64 satisfiedAt,
            uint64 frozenRequestAt,
            FunnyRollupCore.ForcedWithdrawalStatus status
        ) = core.forcedWithdrawalRequests(requestId);

        assertEq(uint256(requestId), 1, "requestId mismatch");
        assertEq(uint256(core.forcedWithdrawalRequestCount()), 1, "forcedWithdrawalRequestCount mismatch");
        assertEq(uint256(uint160(wallet)), uint256(uint160(address(requester))), "wallet mismatch");
        assertEq(uint256(uint160(recipient)), uint256(uint160(address(0xBEEF))), "recipient mismatch");
        assertEq(amount, 1250, "amount mismatch");
        assertTrue(requestedAt > 0, "requestedAt should be set");
        assertEq(uint256(deadlineAt), uint256(requestedAt) + 3600, "deadlineAt mismatch");
        assertEqBytes32(satisfiedClaimId, bytes32(0), "satisfiedClaimId should be zero");
        assertEq(uint256(satisfiedAt), 0, "satisfiedAt should be zero");
        assertEq(uint256(frozenRequestAt), 0, "frozenAt should be zero");
        assertEq(
            uint256(uint8(status)), uint256(uint8(FunnyRollupCore.ForcedWithdrawalStatus.REQUESTED)), "status mismatch"
        );
    }

    function testSatisfyForcedWithdrawalMarksRequestSatisfied() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        MockUSDT token = new MockUSDT();
        FunnyVault vault = new FunnyVault(address(token), address(this));
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        ForcedWithdrawalRequester requester = new ForcedWithdrawalRequester();

        core.setVault(address(vault));
        core.setForcedWithdrawalGracePeriod(3600);

        uint64 requestId = requester.request(core, address(0xCAFE), 900);
        bytes32 claimId = keccak256("forced_claim_1");
        token.mint(address(vault), 900);
        vault.processClaim(claimId, address(requester), 900, address(0xCAFE));

        core.satisfyForcedWithdrawal(requestId, claimId);

        (
            address wallet,
            address recipient,
            uint256 amount,
            uint64 requestedAt,
            uint64 deadlineAt,
            bytes32 satisfiedClaimId,
            uint64 satisfiedAt,
            uint64 frozenRequestAt,
            FunnyRollupCore.ForcedWithdrawalStatus status
        ) = core.forcedWithdrawalRequests(requestId);

        assertEq(uint256(uint160(wallet)), uint256(uint160(address(requester))), "wallet mismatch");
        assertEq(uint256(uint160(recipient)), uint256(uint160(address(0xCAFE))), "recipient mismatch");
        assertEq(amount, 900, "amount mismatch");
        assertTrue(requestedAt > 0, "requestedAt should stay set");
        assertTrue(deadlineAt >= requestedAt, "deadlineAt should stay set");
        assertEqBytes32(satisfiedClaimId, claimId, "satisfiedClaimId mismatch");
        assertTrue(satisfiedAt > 0, "satisfiedAt should be set");
        assertEq(uint256(frozenRequestAt), 0, "frozenAt should stay zero");
        assertEq(
            uint256(uint8(status)), uint256(uint8(FunnyRollupCore.ForcedWithdrawalStatus.SATISFIED)), "status mismatch"
        );
    }

    function testFreezeBlocksBatchAdvancementAfterMissedDeadline() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        ForcedWithdrawalRequester requester = new ForcedWithdrawalRequester();

        core.setVerifier(address(verifier));
        core.recordBatchMetadata(1, keccak256("batch_data_1"), genesisStateRoot, keccak256("next_state_root_1"));
        core.setForcedWithdrawalGracePeriod(0);
        requester.request(core, address(0xF00D), 777);
        core.freezeForMissedForcedWithdrawal(1);

        assertTrue(core.escapeHatchEnabled(), "escape hatch should be enabled when frozen");
        assertEq(uint256(core.freezeRequestId()), 1, "freezeRequestId mismatch");
        assertTrue(core.frozenAt() > 0, "frozenAt should be set");

        try core.recordBatchMetadata(
            2, keccak256("batch_data_2"), keccak256("next_state_root_1"), keccak256("next_state_root_2")
        ) {
            revert("expected recordBatchMetadata to be frozen");
        } catch {}

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        try core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234") {
            revert("expected acceptVerifiedBatch to be frozen");
        } catch {}
    }

    function testFrozenUserCanClaimEscapeCollateralFromAnchoredRoot() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        MockUSDT token = new MockUSDT();
        FunnyVault vault = new FunnyVault(address(token), address(this));
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        ForcedWithdrawalRequester requester = new ForcedWithdrawalRequester();

        core.setVerifier(address(verifier));
        core.setVault(address(vault));
        vault.setRollupCore(address(core));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        core.publishBatchData(1, bytes("batch_data_1"));
        core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234");

        uint256 escapeAmount = 9_000_000;
        bytes32 leaf = core.hashEscapeCollateralLeaf(1, 0, address(requester), escapeAmount);
        bytes32[] memory proof = new bytes32[](0);
        core.recordEscapeCollateralRoot(1, leaf, 1, escapeAmount);

        token.mint(address(vault), escapeAmount);
        core.setForcedWithdrawalGracePeriod(0);
        requester.request(core, address(requester), escapeAmount);
        core.freezeForMissedForcedWithdrawal(1);

        requester.claim(core, 1, 0, escapeAmount, address(0xCAFE), proof);

        bytes32 claimId = keccak256(
            abi.encodePacked("funny-rollup-escape-claim-v1", uint64(1), uint64(0), address(requester), escapeAmount)
        );
        assertTrue(core.escapeHatchEnabled(), "escape hatch should stay enabled");
        assertEq(uint256(core.latestEscapeCollateralBatchId()), 1, "latestEscapeCollateralBatchId mismatch");
        assertEqBytes32(core.latestEscapeCollateralRoot(), leaf, "latestEscapeCollateralRoot mismatch");
        assertTrue(vault.processedClaims(claimId), "vault should mark escape claim processed");
        assertEq(token.balanceOf(address(0xCAFE)), escapeAmount, "recipient token balance mismatch");
    }

    function testClaimAcceptedWithdrawalHappyPath() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        MockUSDT token = new MockUSDT();
        FunnyVault vault = new FunnyVault(address(token), address(this));
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        WithdrawalClaimer claimer = new WithdrawalClaimer();

        core.setVerifier(address(verifier));
        core.setVault(address(vault));
        vault.setRollupCore(address(core));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        uint256 withdrawalAmount = 5_000_000;
        bytes32 withdrawalId = keccak256("wd_1");
        bytes32 leaf = core.hashWithdrawalLeaf(1, 0, withdrawalId, address(claimer), withdrawalAmount, address(0xBEEF));
        publicInputs.withdrawalsRoot = leaf;
        metadataSubset = buildMetadataSubset(publicInputs);
        verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        core.publishBatchData(1, bytes("batch_data_1"));
        core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234");

        token.mint(address(vault), withdrawalAmount);
        bytes32[] memory proof = new bytes32[](0);
        claimer.claimWithdrawal(core, 1, 0, withdrawalId, withdrawalAmount, address(0xBEEF), proof);

        bytes32 claimId = keccak256(
            abi.encodePacked("funny-rollup-withdrawal-claim-v1", uint64(1), uint64(0), withdrawalId, address(claimer))
        );
        assertTrue(vault.processedClaims(claimId), "vault should mark withdrawal claim processed");
        assertEq(token.balanceOf(address(0xBEEF)), withdrawalAmount, "recipient token balance mismatch");
    }

    function testClaimAcceptedWithdrawalRejectsInvalidProof() public {
        bytes32 genesisStateRoot = keccak256("genesis_state_root");
        MockUSDT token = new MockUSDT();
        FunnyVault vault = new FunnyVault(address(token), address(this));
        FunnyRollupCore core = new FunnyRollupCore(address(this), genesisStateRoot);
        MockFunnyRollupVerifier verifier = new MockFunnyRollupVerifier();
        WithdrawalClaimer claimer = new WithdrawalClaimer();

        core.setVerifier(address(verifier));
        core.setVault(address(vault));
        vault.setRollupCore(address(core));

        FunnyRollupCore.VerifierPublicInputs memory publicInputs =
            buildPublicInputs(genesisStateRoot, keccak256("next_state_root_1"));
        FunnyRollupCore.L1BatchMetadata memory metadataSubset = buildMetadataSubset(publicInputs);
        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](1);
        authStatuses[0] = FunnyRollupCore.AuthJoinStatus.JOINED;
        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, core.hashAuthStatuses(authStatuses));
        verifier.setVerdict(verifierGateHash, true);

        core.recordBatchMetadata(1, publicInputs.batchDataHash, publicInputs.prevStateRoot, publicInputs.nextStateRoot);
        core.publishBatchData(1, bytes("batch_data_1"));
        core.acceptVerifiedBatch(publicInputs, metadataSubset, authStatuses, hex"1234");

        token.mint(address(vault), 1000);
        bytes32[] memory proof = new bytes32[](0);

        try claimer.claimWithdrawal(core, 1, 0, keccak256("wd_1"), 1000, address(0xBEEF), proof) {
            revert("expected invalid-proof revert");
        } catch {}
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
            nextStateRoot: nextStateRoot,
            conservationHash: keccak256("conservation_hash_1")
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
            nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5",
            conservationHash: hex"6ad792b876594c7e9876d4166919c3431ad4ff4d76504fbad8e786ff5e43ab23"
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
        bytes32 transitionWitnessHash = verifier.hashTransitionWitness(context);
        uint256[8] memory publicInputs = verifier.deriveGroth16PublicInputs(
            context.batchEncodingHash, context.authProofHash, context.verifierGateHash, transitionWitnessHash
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
        assertEq(publicInputs[4], 0x795a355fc2c2e98cbac5561fa98476a6, "second verifierGateHashHi mismatch");
        assertEq(publicInputs[5], 0x5d079471a264f5999e37158d9440e026, "second verifierGateHashLo mismatch");
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
        bytes32 transitionWitnessHash = verifier.hashTransitionWitness(context);
        uint256[8] memory publicInputs = verifier.deriveGroth16PublicInputs(
            context.batchEncodingHash, context.authProofHash, context.verifierGateHash, transitionWitnessHash
        );

        assertEq(publicInputs[0], 0x3b6489209bd528a9779ecc9db44d4d05, "batchEncodingHashHi mismatch");
        assertEq(publicInputs[1], 0xdceb8faba670a6922ff939d841f202cb, "batchEncodingHashLo mismatch");
        assertEq(publicInputs[2], 0x1e7c5c1c118b439a090ebf5654651794, "authProofHashHi mismatch");
        assertEq(publicInputs[3], 0x76e94bae5ba6a5ae0f146ec3866c8795, "authProofHashLo mismatch");
        assertEq(publicInputs[4], 0xbdb8adff0902424ca22e6b3a15581cb9, "verifierGateHashHi mismatch");
        assertEq(publicInputs[5], 0xb23705ba8192448826e23449e8128626, "verifierGateHashLo mismatch");
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
                nextStateRoot: hex"490e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5",
                conservationHash: hex"6ad792b876594c7e9876d4166919c3431ad4ff4d76504fbad8e786ff5e43ab23"
            }),
            authProofHash: GO_ARTIFACT_AUTH_PROOF_HASH,
            verifierGateHash: GO_ARTIFACT_VERIFIER_GATE_HASH
        });
    }

    function buildSecondGoArtifactContext() internal pure returns (FunnyRollupVerifierTypes.VerifierContext memory) {
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
                nextStateRoot: hex"590e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5",
                conservationHash: hex"6ad792b876594c7e9876d4166919c3431ad4ff4d76504fbad8e786ff5e43ab23"
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

contract ForcedWithdrawalRequester {
    function request(FunnyRollupCore core, address recipient, uint256 amount) external returns (uint64) {
        return core.requestForcedWithdrawal(recipient, amount);
    }

    function claim(
        FunnyRollupCore core,
        uint64 batchId,
        uint64 leafIndex,
        uint256 amount,
        address recipient,
        bytes32[] calldata proof
    ) external {
        core.claimEscapeCollateral(batchId, leafIndex, amount, recipient, proof);
    }
}

contract WithdrawalClaimer {
    function claimWithdrawal(
        FunnyRollupCore core,
        uint64 batchId,
        uint64 leafIndex,
        bytes32 withdrawalId,
        uint256 amount,
        address recipient,
        bytes32[] calldata proof
    ) external {
        core.claimAcceptedWithdrawal(batchId, leafIndex, withdrawalId, amount, recipient, proof);
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
        {
            bytes memory part1 = abi.encode(
                context.batchEncodingHash,
                context.publicInputs.batchId,
                context.publicInputs.firstSequenceNo,
                context.publicInputs.lastSequenceNo,
                context.publicInputs.entryCount,
                context.publicInputs.batchDataHash,
                context.publicInputs.prevStateRoot
            );
            bytes memory part2 = abi.encode(
                context.publicInputs.balancesRoot,
                context.publicInputs.ordersRoot,
                context.publicInputs.positionsFundingRoot,
                context.publicInputs.withdrawalsRoot,
                context.publicInputs.nextStateRoot,
                context.publicInputs.conservationHash,
                context.authProofHash
            );
            if (context.verifierGateHash != keccak256(abi.encodePacked(part1, part2))) {
                return false;
            }
        }
        if (expectedVerifierGateHash != bytes32(0) && context.verifierGateHash != expectedVerifierGateHash) {
            return false;
        }
        return verdict;
    }
}
