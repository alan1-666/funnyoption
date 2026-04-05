// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract FunnyRollupGroth16Backend {
    error PublicInputNotInField();
    error ProofInvalid();

    uint256 internal constant PRECOMPILE_ADD = 0x06;
    uint256 internal constant PRECOMPILE_MUL = 0x07;
    uint256 internal constant PRECOMPILE_VERIFY = 0x08;

    uint256 internal constant R = 0x30644e72e131a029b85045b68181585d2833e84879b9709143e1f593f0000001;

    uint256 internal constant ALPHA_X = 17988391925738443002830003776861124374006548207319419832138193707836945444359;
    uint256 internal constant ALPHA_Y = 7131249788563547066353224374659822413223679694688631240767924692406317298233;

    uint256 internal constant BETA_NEG_X_0 =
        11442848367466845599508486204253376090750601578118380036561004086135082320279;
    uint256 internal constant BETA_NEG_X_1 =
        20889485738355333312399472141098602324147025143628065258622393741934505524300;
    uint256 internal constant BETA_NEG_Y_0 =
        5956453587868412494531569411815953096471364522448033199149139687498375716311;
    uint256 internal constant BETA_NEG_Y_1 =
        16500568703055685089521882022779515809215116274653552964515605463350691946397;

    uint256 internal constant GAMMA_NEG_X_0 =
        9867459934665246188855528029846613763625684717552302433780183137001311086306;
    uint256 internal constant GAMMA_NEG_X_1 =
        5780537943860201120533934858327160481418949406700980510536096428065561143092;
    uint256 internal constant GAMMA_NEG_Y_0 =
        18774615119467786151310267563853931107510052713548904052617394909543912354053;
    uint256 internal constant GAMMA_NEG_Y_1 =
        21739617427274696639058529364291428787734278144796010996613215033503229824693;

    uint256 internal constant DELTA_NEG_X_0 =
        94467196617270450807650204387252788984643461532113064753519382870381672937;
    uint256 internal constant DELTA_NEG_X_1 =
        17624041307715061680264977883973581923092248063649038756286478042694178068910;
    uint256 internal constant DELTA_NEG_Y_0 =
        6659635663146941451847559879443093295789507925026868897479935713335811969685;
    uint256 internal constant DELTA_NEG_Y_1 =
        829538873065516901802410888796879778654334869917164224536793152696957805814;

    uint256 internal constant CONSTANT_X = 4794900876306622586522806142958822625302505837960310358647291104778838112607;
    uint256 internal constant CONSTANT_Y = 1358488799037023507767741648024952675968417240892347375658462871327036963269;
    uint256 internal constant PUB_0_X = 5757654639045681729655895956079223702075234351160865956391030725158488087284;
    uint256 internal constant PUB_0_Y = 11809105763705843095775622109129660403229700473042003345068990510017478749036;
    uint256 internal constant PUB_1_X = 20853477249256986485100124206486868345616307435140591331442627316510333893;
    uint256 internal constant PUB_1_Y = 8125779745068455565659579799530754813351222388830606440773510025802480170582;
    uint256 internal constant PUB_2_X = 6577122341543780558742968569606238687089106152463507217907111105669715430250;
    uint256 internal constant PUB_2_Y = 3431372038450328258332894702531107431721167190519624528243201896382744166210;
    uint256 internal constant PUB_3_X = 9819907088086391843379335994379547248876696186136128344561612171397412346702;
    uint256 internal constant PUB_3_Y = 6632779480416247767500303860988568581215326118176314608905944663985965716682;
    uint256 internal constant PUB_4_X = 19569225151746741802766748238897121652484886580902248876701692503848036564205;
    uint256 internal constant PUB_4_Y = 2587651760388467229614125649182484502147095333329616787354030414446836993296;
    uint256 internal constant PUB_5_X = 2042272694734063908341304651035757911336986045209361846188798348506415062624;
    uint256 internal constant PUB_5_Y = 532396587595852843545829655889182075640483826544697618013250249933309347528;

    function verifyTupleProof(
        uint256[2] calldata a,
        uint256[2][2] calldata b,
        uint256[2] calldata c,
        uint256[6] calldata input
    ) external view returns (bool) {
        uint256[8] memory proof;
        proof[0] = a[0];
        proof[1] = a[1];
        proof[2] = b[0][0];
        proof[3] = b[0][1];
        proof[4] = b[1][0];
        proof[5] = b[1][1];
        proof[6] = c[0];
        proof[7] = c[1];

        try this.verifyProof(proof, input) {
            return true;
        } catch {
            return false;
        }
    }

    function verifyProof(uint256[8] calldata proof, uint256[6] calldata input) public view {
        (uint256 x, uint256 y) = publicInputMSM(input);

        bool success;
        assembly ("memory-safe") {
            let f := mload(0x40)

            calldatacopy(f, proof, 0x100)

            mstore(add(f, 0x100), DELTA_NEG_X_1)
            mstore(add(f, 0x120), DELTA_NEG_X_0)
            mstore(add(f, 0x140), DELTA_NEG_Y_1)
            mstore(add(f, 0x160), DELTA_NEG_Y_0)
            mstore(add(f, 0x180), ALPHA_X)
            mstore(add(f, 0x1a0), ALPHA_Y)
            mstore(add(f, 0x1c0), BETA_NEG_X_1)
            mstore(add(f, 0x1e0), BETA_NEG_X_0)
            mstore(add(f, 0x200), BETA_NEG_Y_1)
            mstore(add(f, 0x220), BETA_NEG_Y_0)
            mstore(add(f, 0x240), x)
            mstore(add(f, 0x260), y)
            mstore(add(f, 0x280), GAMMA_NEG_X_1)
            mstore(add(f, 0x2a0), GAMMA_NEG_X_0)
            mstore(add(f, 0x2c0), GAMMA_NEG_Y_1)
            mstore(add(f, 0x2e0), GAMMA_NEG_Y_0)

            let output := add(f, 0x300)
            success := staticcall(gas(), PRECOMPILE_VERIFY, f, 0x300, output, 0x20)
            success := and(success, eq(mload(output), 1))
        }
        if (!success) {
            revert ProofInvalid();
        }
    }

    function publicInputMSM(uint256[6] calldata input) internal view returns (uint256 x, uint256 y) {
        bool success = true;
        assembly ("memory-safe") {
            let f := mload(0x40)
            let g := add(f, 0x40)
            let s

            mstore(f, CONSTANT_X)
            mstore(add(f, 0x20), CONSTANT_Y)

            mstore(g, PUB_0_X)
            mstore(add(g, 0x20), PUB_0_Y)
            s := calldataload(input)
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            mstore(g, PUB_1_X)
            mstore(add(g, 0x20), PUB_1_Y)
            s := calldataload(add(input, 32))
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            mstore(g, PUB_2_X)
            mstore(add(g, 0x20), PUB_2_Y)
            s := calldataload(add(input, 64))
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            mstore(g, PUB_3_X)
            mstore(add(g, 0x20), PUB_3_Y)
            s := calldataload(add(input, 96))
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            mstore(g, PUB_4_X)
            mstore(add(g, 0x20), PUB_4_Y)
            s := calldataload(add(input, 128))
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            mstore(g, PUB_5_X)
            mstore(add(g, 0x20), PUB_5_Y)
            s := calldataload(add(input, 160))
            mstore(add(g, 0x40), s)
            success := and(success, lt(s, R))
            success := and(success, staticcall(gas(), PRECOMPILE_MUL, g, 0x60, g, 0x40))
            success := and(success, staticcall(gas(), PRECOMPILE_ADD, f, 0x80, f, 0x40))

            x := mload(f)
            y := mload(add(f, 0x20))
        }
        if (!success) {
            revert PublicInputNotInField();
        }
    }
}
