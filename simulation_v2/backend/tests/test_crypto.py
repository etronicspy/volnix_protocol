from volnix.crypto.hashing import hash_obj, sha256_hex
from volnix.crypto.keys import derive_keypair, public_standin_signature, verify_signature
from volnix.crypto.merkle import EMPTY_HASH, merkle_root, merkle_root_bytes


def test_sha256_stable():
    assert sha256_hex(b"volnix") == sha256_hex(b"volnix")
    assert sha256_hex(b"a") != sha256_hex(b"b")


def test_canonical_hash_key_order():
    assert hash_obj({"b": 1, "a": 2}) == hash_obj({"a": 2, "b": 1})


def test_merkle_empty():
    assert merkle_root_bytes([]) == EMPTY_HASH


def test_merkle_single_and_pair():
    a = merkle_root([b"one"])
    b = merkle_root([b"one", b"two"])
    c = merkle_root([b"one", b"two"])
    assert a != b
    assert b == c


def test_keypair_address_and_sign():
    kp = derive_keypair("seed-a")
    assert kp.address.startswith("volnix1")
    assert len(kp.address) == 47
    msg = b"hello"
    sig = kp.sign(msg)
    assert len(sig) == 64
    standin = public_standin_signature(kp.pub_hex, msg)
    assert verify_signature(kp.pub_hex, msg, standin)
    other = derive_keypair("seed-b")
    assert other.address != kp.address
