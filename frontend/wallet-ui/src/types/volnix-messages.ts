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
