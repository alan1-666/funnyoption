// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IFunnyRollupBatchVerifier, FunnyRollupVerifierTypes} from "./FunnyRollupVerifier.sol";

interface IFunnyVaultClaimReader {
    function processedClaims(bytes32 claimId) external view returns (bool);
    function processedClaimRecords(bytes32 claimId)
        external
        view
        returns (address wallet, uint256 amount, address recipient);
}

contract FunnyRollupCore {
    bytes32 public constant SHADOW_BATCH_V1_HASH = keccak256("shadow-batch-v1");

    enum AuthJoinStatus {
        UNSPECIFIED,
        JOINED,
        MISSING_TRADING_KEY_AUTHORIZED,
        NON_VERIFIER_ELIGIBLE
    }

    enum ForcedWithdrawalStatus {
        NONE,
        REQUESTED,
        SATISFIED,
        FROZEN
    }

    struct BatchMetadata {
        bytes32 batchDataHash;
        bytes32 prevStateRoot;
        bytes32 nextStateRoot;
    }

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
    }

    struct L1BatchMetadata {
        uint64 batchId;
        bytes32 batchDataHash;
        bytes32 prevStateRoot;
        bytes32 nextStateRoot;
    }

    struct AcceptedBatch {
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
        bytes32 authProofHash;
        bytes32 verifierGateHash;
    }

    struct ForcedWithdrawalRequest {
        address wallet;
        address recipient;
        uint256 amount;
        uint64 requestedAt;
        uint64 deadlineAt;
        bytes32 satisfiedClaimId;
        uint64 satisfiedAt;
        uint64 frozenAt;
        ForcedWithdrawalStatus status;
    }

    address public immutable operator;
    IFunnyRollupBatchVerifier public verifier;
    address public vault;
    uint64 public forcedWithdrawalGracePeriod;
    bool public frozen;
    uint64 public frozenAt;
    uint64 public freezeRequestId;
    uint64 public forcedWithdrawalRequestCount;
    uint64 public latestBatchId;
    bytes32 public latestStateRoot;
    uint64 public latestAcceptedBatchId;
    bytes32 public latestAcceptedStateRoot;

    mapping(uint64 => BatchMetadata) public batchMetadata;
    mapping(uint64 => AcceptedBatch) public acceptedBatches;
    mapping(uint64 => ForcedWithdrawalRequest) public forcedWithdrawalRequests;

    event BatchMetadataRecorded(
        uint64 indexed batchId, bytes32 indexed batchDataHash, bytes32 indexed prevStateRoot, bytes32 nextStateRoot
    );
    event VerifierUpdated(address indexed verifier);
    event VaultUpdated(address indexed vault);
    event ForcedWithdrawalGracePeriodUpdated(uint64 gracePeriod);
    event ForcedWithdrawalRequested(
        uint64 indexed requestId, address indexed wallet, address indexed recipient, uint256 amount, uint64 deadlineAt
    );
    event ForcedWithdrawalSatisfied(uint64 indexed requestId, bytes32 indexed claimId);
    event FrozenForForcedWithdrawal(uint64 indexed requestId, uint64 frozenAt);
    event VerifiedBatchAccepted(
        uint64 indexed batchId,
        bytes32 indexed verifierGateHash,
        bytes32 indexed nextStateRoot,
        bytes32 prevStateRoot,
        bytes32 authProofHash
    );

    error InvalidOperator();
    error OnlyOperator();
    error InvalidBatchId();
    error InvalidStateRoot();
    error InvalidAmount();
    error InvalidRecipient();
    error PrevStateRootMismatch();
    error InvalidVerifier();
    error InvalidVault();
    error VerifierNotConfigured();
    error BatchMetadataNotRecorded();
    error MetadataMismatch();
    error AuthProofNotFullyJoined(uint256 index, AuthJoinStatus status);
    error InvalidVerifierVerdict();
    error RollupIsFrozen();
    error VaultNotConfigured();
    error InvalidForcedWithdrawalRequest();
    error ForcedWithdrawalRequestNotPending();
    error ForcedWithdrawalClaimNotProcessed();
    error ForcedWithdrawalClaimMismatch();
    error ForcedWithdrawalDeadlineNotReached();
    error AlreadyFrozen();

    constructor(address operator_, bytes32 genesisStateRoot_) {
        if (operator_ == address(0)) revert InvalidOperator();
        if (genesisStateRoot_ == bytes32(0)) revert InvalidStateRoot();

        operator = operator_;
        latestStateRoot = genesisStateRoot_;
        latestAcceptedStateRoot = genesisStateRoot_;
    }

    function setVerifier(address verifier_) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (verifier_ == address(0)) revert InvalidVerifier();

        verifier = IFunnyRollupBatchVerifier(verifier_);
        emit VerifierUpdated(verifier_);
    }

    function setVault(address vault_) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (vault_ == address(0)) revert InvalidVault();

        vault = vault_;
        emit VaultUpdated(vault_);
    }

    function setForcedWithdrawalGracePeriod(uint64 gracePeriod_) external {
        if (msg.sender != operator) revert OnlyOperator();

        forcedWithdrawalGracePeriod = gracePeriod_;
        emit ForcedWithdrawalGracePeriodUpdated(gracePeriod_);
    }

    function requestForcedWithdrawal(address recipient, uint256 amount) external returns (uint64 requestId) {
        if (frozen) revert RollupIsFrozen();
        if (recipient == address(0)) revert InvalidRecipient();
        if (amount == 0) revert InvalidAmount();

        requestId = forcedWithdrawalRequestCount + 1;
        forcedWithdrawalRequestCount = requestId;

        uint64 requestedAt = uint64(block.timestamp);
        uint64 deadlineAt = requestedAt + forcedWithdrawalGracePeriod;
        forcedWithdrawalRequests[requestId] = ForcedWithdrawalRequest({
            wallet: msg.sender,
            recipient: recipient,
            amount: amount,
            requestedAt: requestedAt,
            deadlineAt: deadlineAt,
            satisfiedClaimId: bytes32(0),
            satisfiedAt: 0,
            frozenAt: 0,
            status: ForcedWithdrawalStatus.REQUESTED
        });

        emit ForcedWithdrawalRequested(requestId, msg.sender, recipient, amount, deadlineAt);
    }

    function satisfyForcedWithdrawal(uint64 requestId, bytes32 claimId) external {
        if (vault == address(0)) revert VaultNotConfigured();

        ForcedWithdrawalRequest storage request = forcedWithdrawalRequests[requestId];
        if (request.status == ForcedWithdrawalStatus.NONE) revert InvalidForcedWithdrawalRequest();
        if (request.status != ForcedWithdrawalStatus.REQUESTED) revert ForcedWithdrawalRequestNotPending();

        IFunnyVaultClaimReader vaultReader = IFunnyVaultClaimReader(vault);
        if (!vaultReader.processedClaims(claimId)) revert ForcedWithdrawalClaimNotProcessed();

        (address wallet, uint256 amount, address recipient) = vaultReader.processedClaimRecords(claimId);
        if (wallet != request.wallet || amount != request.amount || recipient != request.recipient) {
            revert ForcedWithdrawalClaimMismatch();
        }

        request.satisfiedClaimId = claimId;
        request.satisfiedAt = uint64(block.timestamp);
        request.status = ForcedWithdrawalStatus.SATISFIED;

        emit ForcedWithdrawalSatisfied(requestId, claimId);
    }

    function freezeForMissedForcedWithdrawal(uint64 requestId) external {
        if (frozen) revert AlreadyFrozen();

        ForcedWithdrawalRequest storage request = forcedWithdrawalRequests[requestId];
        if (request.status == ForcedWithdrawalStatus.NONE) revert InvalidForcedWithdrawalRequest();
        if (request.status != ForcedWithdrawalStatus.REQUESTED) revert ForcedWithdrawalRequestNotPending();
        if (uint64(block.timestamp) < request.deadlineAt) revert ForcedWithdrawalDeadlineNotReached();

        frozen = true;
        frozenAt = uint64(block.timestamp);
        freezeRequestId = requestId;
        request.frozenAt = frozenAt;
        request.status = ForcedWithdrawalStatus.FROZEN;

        emit FrozenForForcedWithdrawal(requestId, frozenAt);
    }

    function escapeHatchEnabled() external view returns (bool) {
        return frozen;
    }

    function recordBatchMetadata(uint64 batchId, bytes32 batchDataHash, bytes32 prevStateRoot, bytes32 nextStateRoot)
        external
    {
        if (msg.sender != operator) revert OnlyOperator();
        if (frozen) revert RollupIsFrozen();
        if (batchId == 0 || batchId != latestBatchId + 1) revert InvalidBatchId();
        if (nextStateRoot == bytes32(0)) revert InvalidStateRoot();
        if (prevStateRoot != latestStateRoot) revert PrevStateRootMismatch();

        batchMetadata[batchId] =
            BatchMetadata({batchDataHash: batchDataHash, prevStateRoot: prevStateRoot, nextStateRoot: nextStateRoot});
        latestBatchId = batchId;
        latestStateRoot = nextStateRoot;

        emit BatchMetadataRecorded(batchId, batchDataHash, prevStateRoot, nextStateRoot);
    }

    function acceptVerifiedBatch(
        VerifierPublicInputs calldata publicInputs,
        L1BatchMetadata calldata metadataSubset,
        AuthJoinStatus[] calldata authStatuses,
        bytes calldata verifierProof
    ) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (frozen) revert RollupIsFrozen();
        if (address(verifier) == address(0)) revert VerifierNotConfigured();
        if (publicInputs.batchId == 0 || publicInputs.batchId != latestAcceptedBatchId + 1) revert InvalidBatchId();
        if (publicInputs.nextStateRoot == bytes32(0)) revert InvalidStateRoot();
        if (publicInputs.prevStateRoot != latestAcceptedStateRoot) revert PrevStateRootMismatch();

        _assertMetadataSubsetMatches(publicInputs, metadataSubset);

        BatchMetadata memory recordedMetadata = batchMetadata[publicInputs.batchId];
        if (publicInputs.batchId > latestBatchId || recordedMetadata.nextStateRoot == bytes32(0)) {
            revert BatchMetadataNotRecorded();
        }
        _assertRecordedMetadataMatches(recordedMetadata, publicInputs);

        for (uint256 i = 0; i < authStatuses.length; ++i) {
            if (authStatuses[i] != AuthJoinStatus.JOINED) {
                revert AuthProofNotFullyJoined(i, authStatuses[i]);
            }
        }

        bytes32 authProofHash = hashAuthStatuses(authStatuses);
        FunnyRollupVerifierTypes.VerifierContext memory verifierContext =
            buildVerifierContext(publicInputs, authProofHash);
        if (!verifier.verifyBatch(verifierContext, verifierProof)) revert InvalidVerifierVerdict();

        acceptedBatches[publicInputs.batchId] = AcceptedBatch({
            firstSequenceNo: publicInputs.firstSequenceNo,
            lastSequenceNo: publicInputs.lastSequenceNo,
            entryCount: publicInputs.entryCount,
            batchDataHash: publicInputs.batchDataHash,
            prevStateRoot: publicInputs.prevStateRoot,
            balancesRoot: publicInputs.balancesRoot,
            ordersRoot: publicInputs.ordersRoot,
            positionsFundingRoot: publicInputs.positionsFundingRoot,
            withdrawalsRoot: publicInputs.withdrawalsRoot,
            nextStateRoot: publicInputs.nextStateRoot,
            authProofHash: authProofHash,
            verifierGateHash: verifierContext.verifierGateHash
        });
        latestAcceptedBatchId = publicInputs.batchId;
        latestAcceptedStateRoot = publicInputs.nextStateRoot;

        emit VerifiedBatchAccepted(
            publicInputs.batchId,
            verifierContext.verifierGateHash,
            publicInputs.nextStateRoot,
            publicInputs.prevStateRoot,
            authProofHash
        );
    }

    function hashAuthStatuses(AuthJoinStatus[] calldata authStatuses) public pure returns (bytes32) {
        return keccak256(abi.encode(authStatuses));
    }

    function hashVerifierGateBatch(VerifierPublicInputs calldata publicInputs, bytes32 authProofHash)
        public
        pure
        returns (bytes32)
    {
        return keccak256(
            abi.encode(
                SHADOW_BATCH_V1_HASH,
                publicInputs.batchId,
                publicInputs.firstSequenceNo,
                publicInputs.lastSequenceNo,
                publicInputs.entryCount,
                publicInputs.batchDataHash,
                publicInputs.prevStateRoot,
                publicInputs.balancesRoot,
                publicInputs.ordersRoot,
                publicInputs.positionsFundingRoot,
                publicInputs.withdrawalsRoot,
                publicInputs.nextStateRoot,
                authProofHash
            )
        );
    }

    function buildVerifierContext(VerifierPublicInputs calldata publicInputs, bytes32 authProofHash)
        public
        pure
        returns (FunnyRollupVerifierTypes.VerifierContext memory)
    {
        return FunnyRollupVerifierTypes.VerifierContext({
            batchEncodingHash: SHADOW_BATCH_V1_HASH,
            publicInputs: FunnyRollupVerifierTypes.VerifierPublicInputs({
                batchId: publicInputs.batchId,
                firstSequenceNo: publicInputs.firstSequenceNo,
                lastSequenceNo: publicInputs.lastSequenceNo,
                entryCount: publicInputs.entryCount,
                batchDataHash: publicInputs.batchDataHash,
                prevStateRoot: publicInputs.prevStateRoot,
                balancesRoot: publicInputs.balancesRoot,
                ordersRoot: publicInputs.ordersRoot,
                positionsFundingRoot: publicInputs.positionsFundingRoot,
                withdrawalsRoot: publicInputs.withdrawalsRoot,
                nextStateRoot: publicInputs.nextStateRoot
            }),
            authProofHash: authProofHash,
            verifierGateHash: hashVerifierGateBatch(publicInputs, authProofHash)
        });
    }

    function _assertMetadataSubsetMatches(
        VerifierPublicInputs calldata publicInputs,
        L1BatchMetadata calldata metadataSubset
    ) internal pure {
        if (
            metadataSubset.batchId != publicInputs.batchId || metadataSubset.batchDataHash != publicInputs.batchDataHash
                || metadataSubset.prevStateRoot != publicInputs.prevStateRoot
                || metadataSubset.nextStateRoot != publicInputs.nextStateRoot
        ) revert MetadataMismatch();
    }

    function _assertRecordedMetadataMatches(
        BatchMetadata memory recordedMetadata,
        VerifierPublicInputs calldata publicInputs
    ) internal pure {
        if (
            recordedMetadata.batchDataHash != publicInputs.batchDataHash
                || recordedMetadata.prevStateRoot != publicInputs.prevStateRoot
                || recordedMetadata.nextStateRoot != publicInputs.nextStateRoot
        ) revert MetadataMismatch();
    }
}
