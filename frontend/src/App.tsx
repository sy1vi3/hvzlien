import { createResource, createSignal, Match, Switch } from 'solid-js'
import { AlienDirection, AlienDirectionSelector } from './components/RadioButtons'
import { Input } from './components/Input'
import { Output } from './components/Output'
import { Meowlien } from './components/Meowlien'
import { decode, DecodeRequest, encode, EncodeRequest } from './utils/api'

function App() {
    const [direction, setDirection] = createSignal<AlienDirection>(AlienDirection.DECODE)
    const [payload, setPayload] = createSignal<EncodeRequest | DecodeRequest | undefined>()

    const [data, { mutate: setData }] = createResource(payload, async (req: EncodeRequest | DecodeRequest) => {
        if (direction() === AlienDirection.DECODE) {
            return await decode(req as DecodeRequest)
        }
        if (direction() === AlienDirection.ENCODE) {
            return await encode(req as EncodeRequest)
        }
    })

    return (
        <div class={"w-full sm:w-xl"}>
            <h1>HVZlien Tool <Meowlien /></h1>
            <div class={"flex flex-row gap-2"}>
                <span>Direction:</span>
                <AlienDirectionSelector direction={direction()} onSelect={(d) => {
                    setData(undefined);
                    setDirection(d);
                }} />
            </div>

            <h2>Input ({direction() === AlienDirection.DECODE ? "Alien" : "Text"})</h2>
            <Input direction={direction()} setPayload={setPayload} />

            <h2>Output ({direction() === AlienDirection.ENCODE ? "Alien" : "Text"})</h2>
            <Switch fallback={<pre>Error :(</pre>}>
                <Match when={data.loading}>
                    Loading...
                </Match>
                <Match when={data.error}>
                    Error executing request:{" "}
                    {data.error.toString()}
                </Match>
                <Match when={data.error === undefined && data()}>
                    <Output payload={data()!} direction={direction()} />
                </Match>
                <Match when={!data()}>
                    Press the "{direction() === AlienDirection.DECODE ? "Decode" : "Encode"}" button to see the output here
                </Match>
            </Switch>
        </div>
    )
}


export default App
