export enum AlienFormat {
    TEXT = "text",
    IMAGE = "image"
}

type DecodeRequestText = {
    type: AlienFormat.TEXT;
    text: string;
}

type DecodeRequestImage = {
    type: AlienFormat.IMAGE;
    image: string;
}

export type DecodeRequest = DecodeRequestText | DecodeRequestImage

export type DecodeResponse = {
    phonetics: string;
    alien: string;
}

export type EncodeRequest = {
    type: AlienFormat;
    text: string;
}

const BASE_URL = "/api/v1"

export type EncodeResponse = {
    text: string;
    image: string;
}

export async function decode(req: DecodeRequest) {
    return (await fetch(`${BASE_URL}/decode`, {
        method: "POST",
        body: JSON.stringify(req)
    })).json() as Promise<DecodeResponse>
}

export async function encode(req: EncodeRequest) {
    return (await fetch(req.type === AlienFormat.IMAGE ? `${BASE_URL}/encode/image` : `${BASE_URL}/encode/text`, {
        method: "POST",
        body: JSON.stringify(req)
    })).json() as Promise<EncodeResponse>
}