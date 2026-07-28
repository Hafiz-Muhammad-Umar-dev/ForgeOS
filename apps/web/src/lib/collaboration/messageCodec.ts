export interface WsMessage {
  readonly type: string;
  readonly intent_id?: string;
  readonly data?: string;
  readonly error?: string;
  readonly session_id?: string;
}

export function encodeMessage(type: string, data?: string, intentId?: string): string {
  const msg: Record<string, string | undefined> = { type };
  if (intentId !== undefined) {
    msg.intent_id = intentId;
  }
  if (data !== undefined) {
    msg.data = data;
  }
  return JSON.stringify(msg);
}

export function decodeMessage(raw: string): WsMessage {
  return JSON.parse(raw) as WsMessage;
}

export function encodeBase64(bytes: Uint8Array): string {
  let binary = "";
  const len = bytes.byteLength;
  for (let i = 0; i < len; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

export function decodeBase64(base64: string): Uint8Array {
  const binary = atob(base64);
  const len = binary.length;
  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
