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

interface IFunnyVaultClaimProcessor {
    function processClaim(bytes32 claimId, address wallet, uint256 amount, address recipient) external;
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
        bytes32 conservationHash;
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
        bytes32 conservationHash;
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

    struct EscapeCollateralRoot {
        bytes32 merkleRoot;
        uint64 leafCount;
        uint256 totalAmount;
        uint64 anchoredAt;
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
    uint64 public latestEscapeCollateralBatchId;
    bytes32 public latestEscapeCollateralRoot;

    mapping(uint64 => BatchMetadata) public batchMetadata;
    mapping(uint64 => bool) public batchDataPublished;
    mapping(uint64 => AcceptedBatch) public acceptedBatches;
    mapping(uint64 => bytes32) public acceptedWithdrawalRoots;
    mapping(uint64 => ForcedWithdrawalRequest) public forcedWithdrawalRequests;
    mapping(uint64 => EscapeCollateralRoot) public escapeCollateralRoots;

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
    event EscapeCollateralRootRecorded(
        uint64 indexed batchId, bytes32 indexed merkleRoot, uint64 leafCount, uint256 totalAmount
    );
    event BatchDataPublished(uint64 indexed batchId, uint256 dataLength);
    event WithdrawalClaimed(
        uint64 indexed batchId,
        uint256 leafIndex,
        bytes32 indexed withdrawalId,
        address indexed wallet,
        uint256 amount,
        address recipient
    );
    event EscapeCollateralClaimed(
        uint64 indexed batchId, bytes32 indexed claimId, address indexed wallet, uint256 amount, address recipient
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
    error InvalidEscapeCollateralRoot();
    error EscapeCollateralBatchNotAccepted();
    error EscapeCollateralBatchOutOfOrder();
    error EscapeCollateralRootNotAnchored();
    error InvalidEscapeCollateralProof();
    error EscapeCollateralLeafIndexOutOfRange();
    error DataAlreadyPublished();
    error DataHashMismatch();
    error DataNotPublished();
    error BatchNotAccepted();
    error NoWithdrawalRoot();
    error InvalidWithdrawalProof();
    error WithdrawalLeafIndexOutOfRange();

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

    function recordEscapeCollateralRoot(uint64 batchId, bytes32 merkleRoot, uint64 leafCount, uint256 totalAmount)
        external
    {
        if (msg.sender != operator) revert OnlyOperator();
        if (frozen) revert RollupIsFrozen();
        if (batchId == 0 || batchId > latestAcceptedBatchId) revert EscapeCollateralBatchNotAccepted();
        if (acceptedBatches[batchId].nextStateRoot == bytes32(0)) revert EscapeCollateralBatchNotAccepted();
        if (latestEscapeCollateralBatchId != 0 && batchId < latestEscapeCollateralBatchId) {
            revert EscapeCollateralBatchOutOfOrder();
        }
        if (merkleRoot == bytes32(0)) revert InvalidEscapeCollateralRoot();

        escapeCollateralRoots[batchId] = EscapeCollateralRoot({
            merkleRoot: merkleRoot,
            leafCount: leafCount,
            totalAmount: totalAmount,
            anchoredAt: uint64(block.timestamp)
        });
        latestEscapeCollateralBatchId = batchId;
        latestEscapeCollateralRoot = merkleRoot;

        emit EscapeCollateralRootRecorded(batchId, merkleRoot, leafCount, totalAmount);
    }

    function claimEscapeCollateral(
        uint64 batchId,
        uint64 leafIndex,
        uint256 amount,
        address recipient,
        bytes32[] calldata proof
    ) external {
        if (!frozen) revert RollupIsFrozen();
        if (vault == address(0)) revert VaultNotConfigured();
        if (recipient == address(0)) revert InvalidRecipient();
        if (batchId == 0 || batchId != latestEscapeCollateralBatchId) revert EscapeCollateralRootNotAnchored();

        EscapeCollateralRoot memory anchored = escapeCollateralRoots[batchId];
        if (anchored.merkleRoot == bytes32(0)) revert EscapeCollateralRootNotAnchored();
        if (leafIndex >= anchored.leafCount) revert EscapeCollateralLeafIndexOutOfRange();
        if (amount == 0) revert InvalidAmount();

        bytes32 leaf = _hashEscapeCollateralLeaf(batchId, leafIndex, msg.sender, amount);
        if (!_verifyEscapeCollateralProof(anchored.merkleRoot, leaf, leafIndex, proof)) {
            revert InvalidEscapeCollateralProof();
        }

        bytes32 claimId = keccak256(abi.encodePacked("funny-rollup-escape-claim-v1", batchId, leafIndex, msg.sender, amount));
        IFunnyVaultClaimProcessor(vault).processClaim(claimId, msg.sender, amount, recipient);

        emit EscapeCollateralClaimed(batchId, claimId, msg.sender, amount, recipient);
    }

    function publishBatchData(uint64 batchId, bytes calldata batchData) external {
        if (msg.sender != operator) revert OnlyOperator();
        if (frozen) revert RollupIsFrozen();
        if (batchMetadata[batchId].batchDataHash == bytes32(0)) revert BatchMetadataNotRecorded();
        if (batchDataPublished[batchId]) revert DataAlreadyPublished();

        batchDataPublished[batchId] = true;
        emit BatchDataPublished(batchId, batchData.length);
    }

    function claimAcceptedWithdrawal(
        uint64 batchId,
        uint64 leafIndex,
        bytes32 withdrawalId,
        uint256 amount,
        address recipient,
        bytes32[] calldata proof
    ) external {
        if (vault == address(0)) revert VaultNotConfigured();
        if (recipient == address(0)) revert InvalidRecipient();
        if (amount == 0) revert InvalidAmount();
        if (acceptedBatches[batchId].nextStateRoot == bytes32(0)) revert BatchNotAccepted();

        bytes32 root = acceptedWithdrawalRoots[batchId];
        if (root == bytes32(0)) revert NoWithdrawalRoot();

        bytes32 leaf = _hashWithdrawalLeaf(batchId, leafIndex, withdrawalId, msg.sender, amount, recipient);
        if (!_verifyMerkleProof(root, leaf, leafIndex, proof)) {
            revert InvalidWithdrawalProof();
        }

        bytes32 claimId = keccak256(
            abi.encodePacked("funny-rollup-withdrawal-claim-v1", batchId, leafIndex, withdrawalId, msg.sender)
        );
        IFunnyVaultClaimProcessor(vault).processClaim(claimId, msg.sender, amount, recipient);

        emit WithdrawalClaimed(batchId, leafIndex, withdrawalId, msg.sender, amount, recipient);
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
        if (!batchDataPublished[publicInputs.batchId]) revert DataNotPublished();

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
            conservationHash: publicInputs.conservationHash,
            authProofHash: authProofHash,
            verifierGateHash: verifierContext.verifierGateHash
        });
        latestAcceptedBatchId = publicInputs.batchId;
        latestAcceptedStateRoot = publicInputs.nextStateRoot;
        if (publicInputs.withdrawalsRoot != bytes32(0)) {
            acceptedWithdrawalRoots[publicInputs.batchId] = publicInputs.withdrawalsRoot;
        }

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
                publicInputs.conservationHash,
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
                nextStateRoot: publicInputs.nextStateRoot,
                conservationHash: publicInputs.conservationHash
            }),
            authProofHash: authProofHash,
            verifierGateHash: hashVerifierGateBatch(publicInputs, authProofHash)
        });
    }

    function hashEscapeCollateralLeaf(uint64 batchId, uint64 leafIndex, address wallet, uint256 amount)
        external
        pure
        returns (bytes32)
    {
        return _hashEscapeCollateralLeaf(batchId, leafIndex, wallet, amount);
    }

    function verifyEscapeCollateralProof(uint64 batchId, uint64 leafIndex, address wallet, uint256 amount, bytes32[] calldata proof)
        external
        view
        returns (bool)
    {
        EscapeCollateralRoot memory anchored = escapeCollateralRoots[batchId];
        if (anchored.merkleRoot == bytes32(0)) {
            return false;
        }
        return _verifyEscapeCollateralProof(
            anchored.merkleRoot,
            _hashEscapeCollateralLeaf(batchId, leafIndex, wallet, amount),
            leafIndex,
            proof
        );
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

    function _hashEscapeCollateralLeaf(uint64 batchId, uint64 leafIndex, address wallet, uint256 amount)
        internal
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked("funny-rollup-escape-collateral-v1", batchId, leafIndex, wallet, amount));
    }

    function _verifyEscapeCollateralProof(bytes32 root, bytes32 leaf, uint64 leafIndex, bytes32[] calldata proof)
        internal
        pure
        returns (bool)
    {
        return _verifyMerkleProof(root, leaf, leafIndex, proof);
    }

    function _hashWithdrawalLeaf(
        uint64 batchId,
        uint64 leafIndex,
        bytes32 withdrawalId,
        address wallet,
        uint256 amount,
        address recipient
    ) internal pure returns (bytes32) {
        return keccak256(
            abi.encodePacked("funny-rollup-withdrawal-leaf-v1", batchId, leafIndex, withdrawalId, wallet, amount, recipient)
        );
    }

    function _verifyMerkleProof(bytes32 root, bytes32 leaf, uint64 leafIndex, bytes32[] calldata proof)
        internal
        pure
        returns (bool)
    {
        bytes32 computed = leaf;
        uint64 index = leafIndex;
        for (uint256 i = 0; i < proof.length; i++) {
            bytes32 sibling = proof[i];
            if (index % 2 == 0) {
                computed = keccak256(abi.encodePacked(computed, sibling));
            } else {
                computed = keccak256(abi.encodePacked(sibling, computed));
            }
            index = index / 2;
        }
        return computed == root;
    }

    function hashWithdrawalLeaf(
        uint64 batchId,
        uint64 leafIndex,
        bytes32 withdrawalId,
        address wallet,
        uint256 amount,
        address recipient
    ) external pure returns (bytes32) {
        return _hashWithdrawalLeaf(batchId, leafIndex, withdrawalId, wallet, amount, recipient);
    }

    function verifyWithdrawalProof(
        uint64 batchId,
        uint64 leafIndex,
        bytes32 withdrawalId,
        address wallet,
        uint256 amount,
        address recipient,
        bytes32[] calldata proof
    ) external view returns (bool) {
        bytes32 root = acceptedWithdrawalRoots[batchId];
        if (root == bytes32(0)) return false;
        return _verifyMerkleProof(
            root,
            _hashWithdrawalLeaf(batchId, leafIndex, withdrawalId, wallet, amount, recipient),
            leafIndex,
            proof
        );
    }
}
