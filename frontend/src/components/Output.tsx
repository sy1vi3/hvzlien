import { createSignal, Match, onCleanup, onMount, Switch } from "solid-js"
import { DecodeResponse, EncodeResponse } from "../utils/api"
import { AlienDirection } from "./RadioButtons"

export type OutputProps = {
    direction: AlienDirection
    payload: DecodeResponse | EncodeResponse
}

export function Output(props: OutputProps) {
    return (
        <Switch>
            <Match when={props.direction === AlienDirection.DECODE}>
                <OutputDecode payload={props.payload as DecodeResponse} />
            </Match>

            <Match when={props.direction === AlienDirection.ENCODE}>
                <OutputEncode payload={props.payload as EncodeResponse} />
            </Match>
        </Switch>
    )
}

function Base64Image(props: { image: string }) {
    const [blobUrl, setBlobUrl] = createSignal<string | null>(null);

    onMount(() => {
        const binaryString = atob(props.image);
        const byteArray = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            byteArray[i] = binaryString.charCodeAt(i);
        }
        const blob = new Blob([byteArray], { type: 'image/png' });
        const url = URL.createObjectURL(blob);
        setBlobUrl(url);

        onCleanup(() => {
            URL.revokeObjectURL(url);
        });
    });

    return <>
        {blobUrl() && <img class="object-contain w-full" src={blobUrl()!} alt="Blob Image" />}
    </>
};

function OutputDecode(props: { payload: DecodeResponse }) {
    return <div class="p-3 border-2 border-dashed border-gray-400 bg-green-100 flex flex-col gap-2">
        <div>
            <span>Parsed alien:</span>
            <textarea readOnly={true} class={"w-full h-16 box-border resize-y"}>
                {props.payload.alien}
            </textarea>
        </div>
        <div>
            <span>Translated phonetics:</span>
            <textarea readOnly={true} class={"w-full h-16 box-border resize-y"}>
                {props.payload.phonetics}
            </textarea>
        </div>
    </div>
}

function OutputEncode(props: { payload: EncodeResponse }) {
    return <div class="p-3 border-2 border-dashed border-gray-400 bg-green-100">
        <Switch fallback={<>No output :(</>}>
            <Match when={props.payload.image.length !== 0}>
                <Base64Image image={props.payload.image} />
            </Match>
            <Match when={props.payload.text.length !== 0}>
                <textarea readOnly={true} class={"w-full h-16 box-border resize-y"}>
                    {props.payload.text}
                </textarea>
            </Match>
        </Switch>
    </div>
}