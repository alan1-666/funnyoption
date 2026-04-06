// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {FunnyRollupGroth16Backend} from "./FunnyRollupGroth16Backend.sol";

library FunnyRollupVerifierTypes {
    struct VerifierPublicInputs {
        uint64 batchId;
        uint64 firstSequenceNo;
        uint64 lastSequenceNo;
        uint64 entryCount;
        bytes32 batchDataHash;
        bytes32 prevStateRoot;
        bytes32 balancesRoot;
        bytes32 ordersRoot;
        bytes32 positionsFundingRoot;
        bytes32 withdrawalsRoot;
        bytes32 nextStateRoot;
        bytes32 conservationHash;
    }

    struct VerifierContext {
        bytes32 batchEncodingHash;
        VerifierPublicInputs publicInputs;
        bytes32 authProofHash;
        bytes32 verifierGateHash;
    }

    struct ProofPublicSignals {
        bytes32 batchEncodingHash;
        bytes32 authProofHash;
        bytes32 verifierGateHash;
    }

    struct ProofData {
        bytes32 proofDataSchemaHash;
        bytes32 proofTypeHash;
        bytes32 batchEncodingHash;
        bytes32 authProofHash;
        bytes32 verifierGateHash;
        bytes proofBytes;
    }
}

interface IFunnyRollupBatchVerifier {
    function verifyBatch(FunnyRollupVerifierTypes.VerifierContext calldata context, bytes calldata verifierProof)
        external
        returns (bool);
}

contract FunnyRollupVerifier is IFunnyRollupBatchVerifier {
    bytes32 public constant SHADOW_BATCH_V1_HASH = keccak256("shadow-batch-v1");
    bytes32 public constant PROOF_SCHEMA_V1_HASH = keccak256("funny-rollup-proof-envelope-v1");
    bytes32 public constant PUBLIC_SIGNALS_V1_HASH = keccak256("funny-rollup-public-signals-v1");
    bytes32 public constant PROOF_DATA_SCHEMA_V1_HASH = keccak256("funny-rollup-proof-data-v1");
    bytes32 public constant PLACEHOLDER_PROOF_V1_HASH = keccak256("funny-rollup-proof-placeholder-v1");
    // The first planned cryptographic lane keeps proofData-v1 and uses
    // proofBytes = abi.encode(bytes32 transitionWitnessHash, uint256[2] a, uint256[2][2] b, uint256[2] c),
    // while proofTypeHash fixes the full verifier-facing contract: proving
    // system/curve, bytes32 signal lifting, exact circuit/vk, and byte codec.
    bytes32 public constant GROTH16_BN254_2X128_SHADOW_STATE_ROOT_GATE_V1_HASH =
        keccak256("funny-rollup-proof-groth16-bn254-2x128-shadow-state-root-gate-v1");
    uint256 internal constant GROTH16_PROOF_BYTES_LENGTH = 0x1A0;

    FunnyRollupGroth16Backend public immutable groth16Backend;

    constructor() {
        groth16Backend = new FunnyRollupGroth16Backend();
    }

    function verifyBatch(FunnyRollupVerifierTypes.VerifierContext calldata context, bytes calldata verifierProof)
        external
        view
        returns (bool)
    {
        if (context.batchEncodingHash != SHADOW_BATCH_V1_HASH) {
            return false;
        }

        bytes32 recomputedVerifierGateHash = _hashVerifierGate(context);
        if (context.verifierGateHash != recomputedVerifierGateHash) {
            return false;
        }

        (
            bytes32 proofSchemaHash,
            bytes32 publicSignalsSchemaHash,
            FunnyRollupVerifierTypes.ProofPublicSignals memory publicSignals,
            FunnyRollupVerifierTypes.ProofData memory proofData,
            bool ok
        ) = _decodeProof(verifierProof);
        if (!ok) {
            return false;
        }
        if (proofSchemaHash != PROOF_SCHEMA_V1_HASH) {
            return false;
        }
        if (publicSignalsSchemaHash != PUBLIC_SIGNALS_V1_HASH) {
            return false;
        }
        if (publicSignals.batchEncodingHash != context.batchEncodingHash) {
            return false;
        }
        if (publicSignals.authProofHash != context.authProofHash) {
            return false;
        }
        if (publicSignals.verifierGateHash != context.verifierGateHash) {
            return false;
        }
        if (proofData.proofDataSchemaHash != PROOF_DATA_SCHEMA_V1_HASH) {
            return false;
        }
        if (proofData.batchEncodingHash != publicSignals.batchEncodingHash) {
            return false;
        }
        if (proofData.authProofHash != publicSignals.authProofHash) {
            return false;
        }
        if (proofData.verifierGateHash != publicSignals.verifierGateHash) {
            return false;
        }
        if (publicSignals.verifierGateHash != recomputedVerifierGateHash) {
            return false;
        }
        if (proofData.verifierGateHash != recomputedVerifierGateHash) {
            return false;
        }

        if (proofData.proofTypeHash != GROTH16_BN254_2X128_SHADOW_STATE_ROOT_GATE_V1_HASH) {
            return false;
        }

        bytes32 transitionWitnessHash = _hashTransitionWitness(context);
        (
            bytes32 proofTransitionWitnessHash,
            uint256[2] memory a,
            uint256[2][2] memory b,
            uint256[2] memory c,
            uint256[2] memory commitments,
            uint256[2] memory commitmentPok,
            bool decodedTupleOk
        ) = _decodeGroth16ProofTuple(proofData.proofBytes);
        if (!decodedTupleOk) {
            return false;
        }
        if (proofTransitionWitnessHash != transitionWitnessHash) {
            return false;
        }

        uint256[8] memory publicInputs = deriveGroth16PublicInputs(
            publicSignals.batchEncodingHash,
            publicSignals.authProofHash,
            publicSignals.verifierGateHash,
            transitionWitnessHash
        );

        return groth16Backend.verifyTupleProofWithCommitments(a, b, c, commitments, commitmentPok, publicInputs);
    }

    function hashVerifierGate(FunnyRollupVerifierTypes.VerifierContext calldata context)
        external
        pure
        returns (bytes32)
    {
        return _hashVerifierGate(context);
    }

    function hashTransitionWitness(FunnyRollupVerifierTypes.VerifierContext calldata context)
        external
        pure
        returns (bytes32)
    {
        return _hashTransitionWitness(context);
    }

    function deriveGroth16PublicInputs(
        bytes32 batchEncodingHash,
        bytes32 authProofHash,
        bytes32 verifierGateHash,
        bytes32 transitionWitnessHash
    )
        public
        pure
        returns (uint256[8] memory inputs)
    {
        (inputs[0], inputs[1]) = _splitBytes32(batchEncodingHash);
        (inputs[2], inputs[3]) = _splitBytes32(authProofHash);
        (inputs[4], inputs[5]) = _splitBytes32(verifierGateHash);
        (inputs[6], inputs[7]) = _splitBytes32(transitionWitnessHash);
    }

    function _hashVerifierGate(FunnyRollupVerifierTypes.VerifierContext calldata context)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(
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
                context.publicInputs.conservationHash,
                context.authProofHash
            )
        );
    }

    function _hashTransitionWitness(FunnyRollupVerifierTypes.VerifierContext calldata context)
        internal
        pure
        returns (bytes32)
    {
        return sha256(
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
                context.publicInputs.conservationHash,
                context.authProofHash
            )
        );
    }

    function _decodeProof(bytes calldata verifierProof)
        internal
        pure
        returns (
            bytes32 proofSchemaHash,
            bytes32 publicSignalsSchemaHash,
            FunnyRollupVerifierTypes.ProofPublicSignals memory publicSignals,
            FunnyRollupVerifierTypes.ProofData memory proofData,
            bool ok
        )
    {
        if (verifierProof.length < 0xe0) {
            return (bytes32(0), bytes32(0), _emptyPublicSignals(), _emptyProofData(), false);
        }

        bytes32 proofBatchEncodingHash;
        bytes32 proofAuthProofHash;
        bytes32 proofVerifierGateHash;
        uint256 proofDataOffset;
        uint256 proofDataLength;
        uint256 totalProofLength;

        assembly {
            proofSchemaHash := calldataload(verifierProof.offset)
            publicSignalsSchemaHash := calldataload(add(verifierProof.offset, 0x20))
            proofBatchEncodingHash := calldataload(add(verifierProof.offset, 0x40))
            proofAuthProofHash := calldataload(add(verifierProof.offset, 0x60))
            proofVerifierGateHash := calldataload(add(verifierProof.offset, 0x80))
            proofDataOffset := calldataload(add(verifierProof.offset, 0xa0))
            proofDataLength := calldataload(add(verifierProof.offset, 0xc0))
        }
        totalProofLength = 0xe0 + _paddedLength(proofDataLength);

        if (proofDataOffset != 0xc0 || verifierProof.length != totalProofLength) {
            return (bytes32(0), bytes32(0), _emptyPublicSignals(), _emptyProofData(), false);
        }

        publicSignals = FunnyRollupVerifierTypes.ProofPublicSignals({
            batchEncodingHash: proofBatchEncodingHash,
            authProofHash: proofAuthProofHash,
            verifierGateHash: proofVerifierGateHash
        });
        (proofData, ok) = _decodeProofData(verifierProof[0xe0:0xe0 + proofDataLength]);
        if (!ok) {
            return (bytes32(0), bytes32(0), _emptyPublicSignals(), _emptyProofData(), false);
        }

        return (proofSchemaHash, publicSignalsSchemaHash, publicSignals, proofData, true);
    }

    function _decodeProofData(bytes calldata proofDataBytes)
        internal
        pure
        returns (FunnyRollupVerifierTypes.ProofData memory proofData, bool ok)
    {
        if (proofDataBytes.length < 0xe0) {
            return (_emptyProofData(), false);
        }

        uint256 proofBytesOffset;
        uint256 proofBytesLength;
        bytes32 proofDataSchemaHash;
        bytes32 proofTypeHash;
        bytes32 proofDataBatchEncodingHash;
        bytes32 proofDataAuthProofHash;
        bytes32 proofDataVerifierGateHash;

        assembly {
            proofDataSchemaHash := calldataload(proofDataBytes.offset)
            proofTypeHash := calldataload(add(proofDataBytes.offset, 0x20))
            proofDataBatchEncodingHash := calldataload(add(proofDataBytes.offset, 0x40))
            proofDataAuthProofHash := calldataload(add(proofDataBytes.offset, 0x60))
            proofDataVerifierGateHash := calldataload(add(proofDataBytes.offset, 0x80))
            proofBytesOffset := calldataload(add(proofDataBytes.offset, 0xa0))
            proofBytesLength := calldataload(add(proofDataBytes.offset, 0xc0))
        }

        if (proofBytesOffset != 0xc0 || proofDataBytes.length != 0xe0 + _paddedLength(proofBytesLength)) {
            return (_emptyProofData(), false);
        }

        proofData = FunnyRollupVerifierTypes.ProofData({
            proofDataSchemaHash: proofDataSchemaHash,
            proofTypeHash: proofTypeHash,
            batchEncodingHash: proofDataBatchEncodingHash,
            authProofHash: proofDataAuthProofHash,
            verifierGateHash: proofDataVerifierGateHash,
            proofBytes: _copyCalldataBytes(proofDataBytes[0xe0:0xe0 + proofBytesLength])
        });

        return (proofData, true);
    }

    function _decodeGroth16ProofTuple(bytes memory proofBytes)
        internal
        pure
        returns (
            bytes32 transitionWitnessHash,
            uint256[2] memory a,
            uint256[2][2] memory b,
            uint256[2] memory c,
            uint256[2] memory commitments,
            uint256[2] memory commitmentPok,
            bool ok
        )
    {
        if (proofBytes.length != GROTH16_PROOF_BYTES_LENGTH) {
            return (bytes32(0), a, b, c, commitments, commitmentPok, false);
        }
        (transitionWitnessHash, a, b, c, commitments, commitmentPok) = abi.decode(
            proofBytes, (bytes32, uint256[2], uint256[2][2], uint256[2], uint256[2], uint256[2])
        );
        return (transitionWitnessHash, a, b, c, commitments, commitmentPok, true);
    }

    function _splitBytes32(bytes32 value) internal pure returns (uint256 hi, uint256 lo) {
        uint256 widened = uint256(value);
        hi = widened >> 128;
        lo = uint128(widened);
    }

    function _paddedLength(uint256 length) internal pure returns (uint256) {
        return (length + 0x1f) & ~uint256(0x1f);
    }

    function _copyCalldataBytes(bytes calldata data) internal pure returns (bytes memory out) {
        out = new bytes(data.length);
        if (data.length == 0) {
            return out;
        }

        assembly {
            calldatacopy(add(out, 0x20), data.offset, data.length)
        }
    }

    function _emptyPublicSignals()
        internal
        pure
        returns (FunnyRollupVerifierTypes.ProofPublicSignals memory publicSignals)
    {
        publicSignals = FunnyRollupVerifierTypes.ProofPublicSignals({
            batchEncodingHash: bytes32(0), authProofHash: bytes32(0), verifierGateHash: bytes32(0)
        });
    }

    function _emptyProofData() internal pure returns (FunnyRollupVerifierTypes.ProofData memory proofData) {
        proofData = FunnyRollupVerifierTypes.ProofData({
            proofDataSchemaHash: bytes32(0),
            proofTypeHash: bytes32(0),
            batchEncodingHash: bytes32(0),
            authProofHash: bytes32(0),
            verifierGateHash: bytes32(0),
            proofBytes: new bytes(0)
        });
    }
}
