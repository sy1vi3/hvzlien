import { createSignal, Match, Switch } from "solid-js"
import { AlienFormat, DecodeRequest, EncodeRequest } from "../utils/api"
import { AlienDirection, AlienFormatSelector } from "./RadioButtons"

export type InputProps = {
    direction: AlienDirection.DECODE
    setPayload: (req: DecodeRequest) => void
    payload?: DecodeRequest
} | {
    direction: AlienDirection.ENCODE
    setPayload: (req: EncodeRequest) => void
    payload?: EncodeRequest
}

function toBase64(file: File) {
    return new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.readAsDataURL(file);
        reader.onload = () => {
            let encoded = reader.result!.toString().replace(/^data:(.*,)?/, '');
            if ((encoded.length % 4) > 0) {
                encoded += '='.repeat(4 - (encoded.length % 4));
            }
            resolve(encoded);
        };
        reader.onerror = error => reject(error);
    });
}


export function Input(props: InputProps) {
    return (
        <Switch>
            <Match when={props.direction === AlienDirection.DECODE}>
                <InputDecode setDecodeRequest={props.setPayload as (req: DecodeRequest) => void} />
            </Match>
            <Match when={props.direction === AlienDirection.ENCODE}>
                <InputEncode setEncodeRequest={props.setPayload as (req: EncodeRequest) => void} />
            </Match>
        </Switch>
    )
}

function InputDecode(props: { setDecodeRequest: (request: DecodeRequest) => void }) {
    const [format, setFormat] = createSignal<AlienFormat>(AlienFormat.TEXT)
    const [text, setText] = createSignal("");
    const [image, setImage] = createSignal<string | undefined>(undefined);

    return <div class="flex flex-col gap-2">
        <div class="flex flex-row gap-2">
            <span>Alien input format:</span>
            <AlienFormatSelector format={format()} onSelect={setFormat} />
        </div>
        <div class="p-3 border-2 border-dashed border-gray-400 bg-amber-50">
            <Switch>
                <Match when={format() === AlienFormat.TEXT}>
                    <div class="flex flex-col gap-2">
                        <textarea class={"w-full h-24 box-border resize-y"} placeholder={"Alien text..."} onchange={(e) => {
                            setText(e.target.value)
                        }}></textarea>
                        <div class="flex flex-col">
                            <span>
                                Alien language:
                            </span>
                            <span>
                                ☀ ☁ ☂ ☃ ☄ ★ ☆ ☇ ☈ ☉ ☊ ☋ ☌ ☍ ☎ ☏ ☐ ☑ ☒ ☓ ☔ ☕ ☖ ☗ ☘ ☙ ☚ ☛ ☜ ☝ ☞ ☟ ☠ ☡ ☢ ☣ ☤ ☥ ☦ ☧ ☨
                            </span>
                            <span class="text-gray-600 pt-2">
                                Copy and paste the alien letters above to build your string
                            </span>
                        </div>
                    </div>
                </Match>
                <Match when={format() === AlienFormat.IMAGE}>
                    <div class="flex flex-col gap-1">
                        <input type="file" accept="image/png, image/jpeg" onChange={async (e) => {
                            if (!e.target.files || e.target.files.length !== 1) {
                                return
                            }

                            setImage(await toBase64(e.target.files![0]))
                        }} />
                        <span class="text-sm/7 text-gray-600">
                            Accepts jpeg & png images
                        </span>
                    </div>
                </Match>
            </Switch>
        </div>

        <button onclick={() => {
            if (format() === AlienFormat.TEXT && text().length === 0) {
                return alert("Input text cannot be empty")
            }
            if (format() === AlienFormat.IMAGE && (image() ?? "").length === 0) {
                return alert("No input image")
            }
            props.setDecodeRequest({
                type: format(),
                text: format() === AlienFormat.TEXT ? text() : "",
                image: format() === AlienFormat.IMAGE ? image()! : "",
            })
        }}>Decode</button>
    </div>
}

function InputEncode(props: { setEncodeRequest: (request: EncodeRequest) => void }) {
    const [outputFormat, setOutputFormat] = createSignal<AlienFormat>(AlienFormat.TEXT)
    const [text, setText] = createSignal("");

    return <div class="flex flex-col gap-2">
        <div class="flex flex-col gap-2 p-3 border-2 border-dashed border-gray-400 bg-amber-50">
            <textarea class={"w-full h-24 box-border resize-y"} placeholder={"Non-alien text..."} onchange={(e) => {
                setText(e.target.value)
            }}></textarea>

        </div>
        <div class={"flex flex-row gap-2"}>
            <span>Output format:</span>
            <AlienFormatSelector format={outputFormat()} onSelect={setOutputFormat} />
        </div>
        <button class="w-fill" onclick={() => {
            props.setEncodeRequest({
                type: outputFormat(),
                text: text()
            })
        }}>Encode</button>
    </div>
}