from fractions import Fraction

from volnix.app.modules.mint import distribute_rewards
from volnix.app.modules.povb import (
    corridor_bounds,
    entry_burn,
    process_endblocker,
    suggest_declare_bs,
)
from volnix.app.state import SCALE, Account, AppState, DeclareRecord
from volnix.types.role import Role
from volnix.types.validator import Validator, ValidatorSet, voting_power


def _state_with_validators(n: int, l_i: int, ant: int) -> tuple[AppState, ValidatorSet]:
    st = AppState()
    st.height = 1
    st.params.lambda_num, st.params.lambda_den = 1, 3
    st.params.alpha_num, st.params.alpha_den = 1, 50
    st.params.max_active_validators = 150
    vals = []
    for i in range(n):
        addr = f"volnix1{'%040d' % i}"
        acc = Account(
            address=addr,
            pub_hex="ab" * 32,
            role=Role.VALIDATOR,
            lzn_activated=l_i,
            ant=ant,
        )
        st.accounts[addr] = acc
        vals.append(
            Validator(
                address=addr,
                pub_hex=acc.pub_hex,
                l_i=l_i,
                s_i=l_i,
                w_i=1.0,
                power=voting_power(l_i, l_i),
            )
        )
    return st, ValidatorSet(validators=vals)


def test_entry_burn_alpha():
    """DeclareEntryAlpha α=1/50 → f_i = ⌊L_i / 50⌋ (§5.4 / §6.3(5))."""
    assert entry_burn(SCALE, Fraction(1, 50)) == 20_000
    assert entry_burn(SCALE, Fraction(1, 50)) == SCALE // 50
    assert entry_burn(50, Fraction(1, 50)) == 1
    assert entry_burn(1, Fraction(1, 50)) == 0
    assert entry_burn(49, Fraction(1, 50)) == 0


def test_suggest_declare_bs_genesis_6_3_5():
    pair = suggest_declare_bs(SCALE, 2 * SCALE, Fraction(1, 50))
    assert pair == (500_000, 400_000)
    f = entry_burn(SCALE, Fraction(1, 50))
    assert f + pair[0] + pair[1] == 920_000


def test_corridor_bounds_integer():
    # λ=1/3, L_decl=1_000_000 → B_min=⌈L/3⌉=333334, B_max=⌊2L/3⌋=666666
    b_min, b_max = corridor_bounds(SCALE, Fraction(1, 3))
    assert b_min == 333_334
    assert b_max == 666_666


def test_voting_power_floor():
    assert voting_power(400_000, SCALE) == 400_000
    assert voting_power(1, SCALE) == 1
    assert voting_power(SCALE, SCALE) == 1_000_000


def test_constraint_f_b_s_exceeds_l():
    st, vset = _state_with_validators(1, SCALE, SCALE)
    addr = list(st.accounts)[0]
    st.declares[addr] = DeclareRecord(validator=addr, b_i=SCALE, s_i=SCALE)
    nxt, _, trace = process_endblocker(st, vset)
    rec = next(d for d in trace["declares"] if d["validator"] == addr)
    assert rec["valid"] is False
    assert rec["reason"] == "f+b+s_exceeds_L"


def test_s_i_zero_rejected():
    st, vset = _state_with_validators(1, SCALE, SCALE)
    addr = list(st.accounts)[0]
    st.declares[addr] = DeclareRecord(validator=addr, b_i=400_000, s_i=0)
    _, _, trace = process_endblocker(st, vset)
    rec = next(d for d in trace["declares"] if d["validator"] == addr)
    assert rec["valid"] is False
    assert rec["reason"] == "s_i_zero"


def test_l_decl_not_l_total():
    """Only one of two validators declares → L_decl = L_i, not 2·L_i."""
    st, vset = _state_with_validators(2, SCALE, 2 * SCALE)
    addrs = sorted(st.accounts)
    # only first declares; corridor must use L_decl=SCALE
    st.declares[addrs[0]] = DeclareRecord(validator=addrs[0], b_i=500_000, s_i=400_000)
    _, _, trace = process_endblocker(st, vset)
    assert trace["l_decl"] == SCALE
    assert trace["l_total"] == 2 * SCALE
    assert trace["fill"] <= trace["l_decl"]
    assert trace["set_updated"] is True
    # Fill = f + b + s = 20000 + 500000 + 400000 = 920000
    assert trace["fill"] == 920_000


def test_genesis_arithmetic_6_3_5():
    """§6.3(5): L=1e6, b=500k, s=400k → Fill=920k, VotingPower=400k."""
    st, vset = _state_with_validators(1, SCALE, 2 * SCALE)
    addr = list(st.accounts)[0]
    st.declares[addr] = DeclareRecord(validator=addr, b_i=500_000, s_i=400_000)
    nxt, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is True
    assert trace["fill"] == 920_000
    assert nxt.validators[0].power == 400_000


def test_corridor_and_fill():
    st, vset = _state_with_validators(2, SCALE, 2 * SCALE)
    addrs = sorted(st.accounts)
    for a in addrs:
        st.declares[a] = DeclareRecord(validator=a, b_i=SCALE // 2, s_i=SCALE // 4)
    _, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is True
    assert trace["l_decl"] == 2 * SCALE
    assert trace["fill"] <= trace["l_decl"]
    sum_b = sum(d["b_i"] for d in trace["declares"] if d["passed"])
    assert trace["b_min"] <= sum_b <= trace["b_max"]


def test_step6_set_not_updated_when_under_floor():
    st, vset = _state_with_validators(1, SCALE, SCALE)
    addr = list(st.accounts)[0]
    # b_i small → under B_min; s_i > 0 required
    st.declares[addr] = DeclareRecord(validator=addr, b_i=0, s_i=SCALE // 2)
    nxt, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is False
    assert nxt.validators[0].address == addr
    f = entry_burn(SCALE, Fraction(1, 50))
    assert st.accounts[addr].ant == SCALE - f


def test_exclude_from_tail_equal_w():
    """Equal w_i: exclude larger address (tail), keep smaller (head)."""
    st, vset = _state_with_validators(3, SCALE, 3 * SCALE)
    addrs = sorted(st.accounts)  # 0, 1, 2
    st.params.max_active_validators = 2
    for a in addrs:
        # equal w_i = 0.25; b high enough that after top-K floor holds
        st.declares[a] = DeclareRecord(validator=a, b_i=SCALE // 2, s_i=SCALE // 4)
    _, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is True
    passed = set(trace["passed"])
    assert addrs[0] in passed
    assert addrs[1] in passed
    assert addrs[2] not in passed
    excluded = [d for d in trace["declares"] if d["excluded"] == "topk"]
    assert any(d["validator"] == addrs[2] for d in excluded)


def test_subsidy_by_b_i_not_l():
    st, vset = _state_with_validators(2, SCALE, 2 * SCALE)
    addrs = sorted(st.accounts)
    # same L, different b
    st.declares[addrs[0]] = DeclareRecord(validator=addrs[0], b_i=600_000, s_i=200_000)
    st.declares[addrs[1]] = DeclareRecord(validator=addrs[1], b_i=200_000, s_i=200_000)
    # L_decl=2e6, B_min≈666667, sum_b=800000 OK; B_max≈1333333
    st.params.lambda_num, st.params.lambda_den = 1, 3
    _, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is True
    st.params.base_block_reward = 1_000_000
    before = {a: st.accounts[a].wrt for a in addrs}
    distribute_rewards(st, fee_pool=0)
    # shares 600k/800k and 200k/800k of 1e6 → 750000 and 250000
    assert st.accounts[addrs[0]].wrt - before[addrs[0]] == 750_000
    assert st.accounts[addrs[1]].wrt - before[addrs[1]] == 250_000


def test_b_zero_no_reward_share():
    st, vset = _state_with_validators(2, SCALE, 2 * SCALE)
    addrs = sorted(st.accounts)
    st.params.lambda_num, st.params.lambda_den = 1, 100  # tiny floor
    st.declares[addrs[0]] = DeclareRecord(validator=addrs[0], b_i=SCALE // 2, s_i=SCALE // 4)
    st.declares[addrs[1]] = DeclareRecord(validator=addrs[1], b_i=0, s_i=SCALE // 2)
    _, _, trace = process_endblocker(st, vset)
    before = {a: st.accounts[a].wrt for a in addrs}
    distribute_rewards(st, fee_pool=1_000_000)
    assert st.accounts[addrs[1]].wrt == before[addrs[1]]


def test_fee_burn_when_b_zero_set_not_updated():
    st, vset = _state_with_validators(1, SCALE, SCALE)
    addr = list(st.accounts)[0]
    st.declares[addr] = DeclareRecord(validator=addr, b_i=0, s_i=SCALE // 2)
    _, _, trace = process_endblocker(st, vset)
    assert trace["set_updated"] is False
    st.wrt_supply = 5_000_000
    events = distribute_rewards(st, fee_pool=1000)
    assert st.wrt_supply == 5_000_000 - 1000
    assert any(e.type == "consensus.fee_burned" for e in events)
