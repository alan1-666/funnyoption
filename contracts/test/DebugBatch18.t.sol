// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/FunnyRollupCore.sol";
import "../src/FunnyRollupVerifier.sol";

contract DebugBatch18Test is Test {
    FunnyRollupCore core = FunnyRollupCore(0xd004c6FD8A231A116aD78D1A8843c3c8c2A8a053);
    FunnyRollupVerifier verifier;

    function setUp() public {
        vm.createSelectFork("https://data-seed-prebsc-1-s1.bnbchain.org:8545");
        verifier = FunnyRollupVerifier(address(core.verifier()));
    }

    function test_debugBatch18AcceptCalldata() public {
        FunnyRollupCore.VerifierPublicInputs memory publicInputs = FunnyRollupCore.VerifierPublicInputs({
            batchId: 18,
            firstSequenceNo: 113,
            lastSequenceNo: 114,
            entryCount: 2,
            batchDataHash: 0x4a3737d55508107d2d266501d671002be51bfa277158cb634e408ee6bd1d03e1,
            prevStateRoot: 0x444ca6ff4f1e52736965f9e6fe78d97c5d95d7f92fda9119833f0790a89fa299,
            balancesRoot: 0x4a7c4c770b61f5d37888eddb20af9503b6e23cbae7fe504354226bff4b281dae,
            ordersRoot: 0xc845cf12fe838cfef7beacc043271ba34609758f63d9333839b2eb745ee99ef8,
            positionsFundingRoot: 0x04a1db17fdb9bb3abfdeded23c181ca9c05bf38a3564df717b61edc5ee5375f2,
            withdrawalsRoot: 0x4d8d05be4ce388de39e86fd781a0625c0f833a07b3a815a5c99e5917c8302c96,
            nextStateRoot: 0x3c6357723a9e0283928a119c633c7001edbba24f8c0606fac1a555c1c5e29668,
            conservationHash: 0x99fa1219e95995f8c396427d8ccd8a1efd8ca98a4d76843333ab6f4f8df6828d
        });

        FunnyRollupCore.AuthJoinStatus[] memory authStatuses = new FunnyRollupCore.AuthJoinStatus[](0);

        bytes32 authProofHash = core.hashAuthStatuses(authStatuses);
        emit log_named_bytes32("authProofHash", authProofHash);

        bytes32 verifierGateHash = core.hashVerifierGateBatch(publicInputs, authProofHash);
        emit log_named_bytes32("verifierGateHash", verifierGateHash);

        FunnyRollupVerifierTypes.VerifierContext memory context = core.buildVerifierContext(publicInputs, authProofHash);
        emit log_named_bytes32("context.batchEncodingHash", context.batchEncodingHash);
        emit log_named_bytes32("context.authProofHash", context.authProofHash);
        emit log_named_bytes32("context.verifierGateHash", context.verifierGateHash);

        // Extract the verifierProof from the accept calldata
        bytes memory verifierProof = hex"45d038e607974e1baa9a64b2ee0ae0d345a16b4ad5ccf32c35924a22abac02b7404acb1bd1ca34653909e83ee6c144d1e74d98467ed0b421dc6792f57aa00cbf3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb569e75fc77c1a856f6daaf9e69d8a9566ca34aa47f9133711ce065a571af0cfd799a8cc5d69b522a0c6430bdf852fe788773c223a8cd6fe02d1ce64d19c134c900000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000380627084a901f511fbfdc855a140c870c7de43da7b9b900117c3f719b0121e09f761d926910cd37f0427ac13baa3386a8b649e9c83c83a845ed4355ec4df7b8d2e3b6489209bd528a9779ecc9db44d4d05dceb8faba670a6922ff939d841f202cb569e75fc77c1a856f6daaf9e69d8a9566ca34aa47f9133711ce065a571af0cfd799a8cc5d69b522a0c6430bdf852fe788773c223a8cd6fe02d1ce64d19c134c900000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000002a0b5c6b7d3ec8065d8bbd2b7daf6b9ead85b2d74a4e402c7394a13291bb8a29eb2fb0644a457307cdc17b0e6aeffea9d21f6369e1f6e611b949e9a0a86dc478da37cd7f49aad8f1f873741d91fd8f7d07212afc08554dc0c2597395238a1a4f2e0b683cb93c93b418050e7eba866cb2d60a3090a0581463a1769b190c0057a533ddce6c85ef43c4470b30aa6fd6c53ba39425c7bceba97c033b936b91da751fabb29f0f220130110cf6850eebcbe7890f32eff0a366a189609791804d9de6b579353fec9e4f5725b75bf8c0501257419c4fb7aba8c3024792652e9bb81b494a211c33df906a068b832cf532fb3116c161f5931cc14082218d7a163a9d7b76d75ddbdb458f31086747ebcf282b66bae6ce7a7114f39a139d695929f47abf302604500080c308a03d7e6d1c9882a3f56d8f18cb1767cf4d577ba98a46abdc3c338f10912d963ee7bd7f3c5f8368ff0069d3a3708eed3966a4e102e38a0de1884acf92322399d313067d4f8dd6ed6a41bec9c4051e6a12b893e26b8828aac4f91d6042f3baa7d3dd917ed1d209bd34ecb492bac577c464685fd2e278bcc1eebc307531709439bbfb384a7eb123e3a373c8ee2488aab8bf89f9e372b375b28ac65e41823ce7ff3d193356742184f46e32a05bb343923e50b1c808692013fa2c1cb78920b16e12cd8b979fff1775fb604632d61a187ec46d122595897ac71534d006eee2b6685bd4febd09de8f5707fa9f85d84ff8ce6ea770ec0bfbd1083115ffcf2f12b76719df8ea7de9473132e86a4619cb02e1c3eb83bc48c05fc3d5d557544c0c23e7d79bbe22ece3d25d83d849afcfb252c4f76b49174c16a855d90355da3e5a1bfc26bbc41c9ad3a13f1845d8d10fa8d1dc04f3998dae25c9f5c941b6abc0ed1cb2780c3690e7fe157b7a28f3d4948ca48d635adde72a13120d765164856387";

        bool result = verifier.verifyBatch(context, verifierProof);
        emit log_named_string("verifyBatch result", result ? "TRUE" : "FALSE");

        // Now test individual steps
        emit log_named_bytes32("SHADOW_BATCH_V1_HASH (expected)", context.batchEncodingHash);

        bytes32 recomputedGateHash = verifier.hashVerifierGate(context);
        emit log_named_bytes32("recomputedGateHash", recomputedGateHash);
        emit log_named_string("gateHash match", recomputedGateHash == context.verifierGateHash ? "YES" : "NO");

        // Check proofBytes length
        // The verifierProof is nested: decode outer proof schema first
        emit log_named_uint("verifierProof length", verifierProof.length);

        // Call hashTransitionWitnessWithMaterial using the hashes from proofBytes
        // proofBytes inner (inside proofData) starts with proofDataSchemaHash etc, 
        // The actual groth16 proof data is inside. Let me decode it from the verifierProof.
        // For now, just check if the overall verifyBatch returns true
        
        assertEq(result, true, "verifyBatch should return true for batch 18");
    }
}
