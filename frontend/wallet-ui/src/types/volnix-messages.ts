// Volnix Protocol Custom Message Types
// Using protobuf.js Writer for proper CosmJS compatibility

import { Writer, Reader } from 'protobufjs/minimal';

export interface MsgChangeRoleEncodeObject {
  readonly typeUrl: '/volnix.ident.v1.MsgChangeRole';
  readonly value: {
    readonly address: string;
    readonly newRole: number;
    readonly zkpProof: string;
    readonly changeFee?: {
      readonly denom: string;
      readonly amount: string;
    };
  };
}

// Encoder for MsgChangeRole using protobuf.js Writer
export function encodeMsgChangeRole(message: any, writer?: Writer): Writer {
  if (!writer) {
    writer = Writer.create();
  }
  
  // Field 1: address (string)
  if (message.address !== undefined && message.address !== '') {
    writer.uint32(10).string(message.address);
  }
  
  // Field 2: new_role (int32)
  if (message.newRole !== undefined || message.new_role !== undefined) {
    const role = message.newRole !== undefined ? message.newRole : message.new_role;
    writer.uint32(16).int32(role);
  }
  
  // Field 3: zkp_proof (string)
  if (message.zkpProof !== undefined && message.zkpProof !== '') {
    writer.uint32(26).string(message.zkpProof);
  } else if (message.zkp_proof !== undefined && message.zkp_proof !== '') {
    writer.uint32(26).string(message.zkp_proof);
  }
  
  // Field 4: change_fee (message) - optional
  if (message.changeFee) {
    writer.uint32(34).fork();
    if (message.changeFee.denom) {
      writer.uint32(10).string(message.changeFee.denom);
    }
    if (message.changeFee.amount) {
      writer.uint32(18).string(message.changeFee.amount);
    }
    writer.ldelim();
  }
  
  return writer;
}

// Decoder for MsgChangeRole
export function decodeMsgChangeRole(input: Uint8Array | Reader, length?: number): any {
  const reader = input instanceof Uint8Array ? Reader.create(input) : input;
  const end = length === undefined ? reader.len : reader.pos + length;
  const message: any = {
    address: '',
    newRole: 0,
    zkpProof: '',
  };
  
  while (reader.pos < end) {
    const tag = reader.uint32();
    switch (tag >>> 3) {
      case 1:
        message.address = reader.string();
        break;
      case 2:
        message.newRole = reader.int32();
        break;
      case 3:
        message.zkpProof = reader.string();
        break;
      case 4:
        // change_fee (nested message)
        const changeFeeLength = reader.uint32();
        const changeFeeEnd = reader.pos + changeFeeLength;
        message.changeFee = {};
        while (reader.pos < changeFeeEnd) {
          const changeFeeTag = reader.uint32();
          switch (changeFeeTag >>> 3) {
            case 1:
              message.changeFee.denom = reader.string();
              break;
            case 2:
              message.changeFee.amount = reader.string();
              break;
            default:
              reader.skipType(changeFeeTag & 7);
              break;
          }
        }
        break;
      default:
        reader.skipType(tag & 7);
        break;
    }
  }
  
  return message;
}

// Create method for MsgChangeRole
export function createMsgChangeRole(properties?: any): any {
  return {
    address: properties?.address || '',
    newRole: properties?.newRole !== undefined ? properties.newRole : (properties?.new_role !== undefined ? properties.new_role : 0),
    zkpProof: properties?.zkpProof || properties?.zkp_proof || '',
    changeFee: properties?.changeFee || properties?.change_fee || undefined,
  };
}

// GeneratedType for CosmJS Registry
// Must include encode, decode, and create methods for PbjsGeneratedType
export const MsgChangeRoleType = {
  encode: encodeMsgChangeRole,
  decode: decodeMsgChangeRole,
  create: createMsgChangeRole,
};

// ============================================================================
// MsgActivateLZN (/volnix.lizenz.v1.MsgActivateLZN)
// Fields: validator (1), amount (2), identity_hash (3)
// ============================================================================

export function encodeMsgActivateLZN(message: {
  validator?: string;
  amount?: string;
  identityHash?: string;
  identity_hash?: string;
}, writer?: Writer): Writer {
  if (!writer) writer = Writer.create();
  if (message.validator) writer.uint32(10).string(message.validator);
  if (message.amount) writer.uint32(18).string(message.amount);
  const ih = message.identityHash || message.identity_hash;
  if (ih) writer.uint32(26).string(ih);
  return writer;
}

export function decodeMsgActivateLZN(input: Uint8Array | Reader, length?: number): any {
  const reader = input instanceof Uint8Array ? Reader.create(input) : input;
  const end = length === undefined ? reader.len : reader.pos + length;
  const message: any = { validator: '', amount: '', identityHash: '' };
  while (reader.pos < end) {
    const tag = reader.uint32();
    switch (tag >>> 3) {
      case 1: message.validator = reader.string(); break;
      case 2: message.amount = reader.string(); break;
      case 3: message.identityHash = reader.string(); break;
      default: reader.skipType(tag & 7); break;
    }
  }
  return message;
}

export const MsgActivateLZNType = {
  encode: encodeMsgActivateLZN,
  decode: decodeMsgActivateLZN,
  create: (p?: any) => ({
    validator: p?.validator || '',
    amount: p?.amount || '',
    identityHash: p?.identityHash || p?.identity_hash || '',
  }),
};

// ============================================================================
// MsgDeactivateLZN (/volnix.lizenz.v1.MsgDeactivateLZN)
// Fields: validator (1), amount (2), reason (3)
// ============================================================================

export function encodeMsgDeactivateLZN(message: {
  validator?: string;
  amount?: string;
  reason?: string;
}, writer?: Writer): Writer {
  if (!writer) writer = Writer.create();
  if (message.validator) writer.uint32(10).string(message.validator);
  if (message.amount) writer.uint32(18).string(message.amount);
  if (message.reason) writer.uint32(26).string(message.reason);
  return writer;
}

export function decodeMsgDeactivateLZN(input: Uint8Array | Reader, length?: number): any {
  const reader = input instanceof Uint8Array ? Reader.create(input) : input;
  const end = length === undefined ? reader.len : reader.pos + length;
  const message: any = { validator: '', amount: '', reason: '' };
  while (reader.pos < end) {
    const tag = reader.uint32();
    switch (tag >>> 3) {
      case 1: message.validator = reader.string(); break;
      case 2: message.amount = reader.string(); break;
      case 3: message.reason = reader.string(); break;
      default: reader.skipType(tag & 7); break;
    }
  }
  return message;
}

export const MsgDeactivateLZNType = {
  encode: encodeMsgDeactivateLZN,
  decode: decodeMsgDeactivateLZN,
  create: (p?: any) => ({
    validator: p?.validator || '',
    amount: p?.amount || '',
    reason: p?.reason || '',
  }),
};

// ============================================================================
// MsgPlaceOrder (/volnix.anteil.v1.MsgPlaceOrder)
// Fields: owner (1), order_type (2), order_side (3), ant_amount (4), price (5), identity_hash (6)
// OrderType: ORDER_TYPE_LIMIT=1, ORDER_TYPE_MARKET=2
// OrderSide: ORDER_SIDE_BUY=1, ORDER_SIDE_SELL=2
// ============================================================================

export function encodeMsgPlaceOrder(message: {
  owner?: string;
  orderType?: number;
  orderSide?: number;
  antAmount?: string;
  price?: string;
  identityHash?: string;
}, writer?: Writer): Writer {
  if (!writer) writer = Writer.create();
  if (message.owner) writer.uint32(10).string(message.owner);
  if (message.orderType !== undefined) writer.uint32(16).int32(message.orderType);
  if (message.orderSide !== undefined) writer.uint32(24).int32(message.orderSide);
  if (message.antAmount) writer.uint32(34).string(message.antAmount);
  if (message.price) writer.uint32(42).string(message.price);
  const ih = message.identityHash;
  if (ih) writer.uint32(50).string(ih);
  return writer;
}

export function decodeMsgPlaceOrder(input: Uint8Array | Reader, length?: number): any {
  const reader = input instanceof Uint8Array ? Reader.create(input) : input;
  const end = length === undefined ? reader.len : reader.pos + length;
  const message: any = { owner: '', orderType: 1, orderSide: 1, antAmount: '', price: '', identityHash: '' };
  while (reader.pos < end) {
    const tag = reader.uint32();
    switch (tag >>> 3) {
      case 1: message.owner = reader.string(); break;
      case 2: message.orderType = reader.int32(); break;
      case 3: message.orderSide = reader.int32(); break;
      case 4: message.antAmount = reader.string(); break;
      case 5: message.price = reader.string(); break;
      case 6: message.identityHash = reader.string(); break;
      default: reader.skipType(tag & 7); break;
    }
  }
  return message;
}

export const MsgPlaceOrderType = {
  encode: encodeMsgPlaceOrder,
  decode: decodeMsgPlaceOrder,
  create: (p?: any) => ({
    owner: p?.owner || '',
    orderType: p?.orderType ?? 1,
    orderSide: p?.orderSide ?? 1,
    antAmount: p?.antAmount || '',
    price: p?.price || '',
    identityHash: p?.identityHash || '',
  }),
};
