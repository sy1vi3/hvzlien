import { AlienFormat as AlienFormat } from "../utils/api";
import styles from "./RadioButtons.module.css"

export function AlienFormatSelector(props: { onSelect: (format: AlienFormat) => void, format: AlienFormat }) {
    return (
        <div class={styles.RadioButtons}>
            <label>
                <input type="radio" checked={props.format === AlienFormat.TEXT} onChange={() => props.onSelect(AlienFormat.TEXT)} />
                Alien text
            </label>
            <label>
                <input type="radio" checked={props.format === AlienFormat.IMAGE} onChange={() => props.onSelect(AlienFormat.IMAGE)} />
                Image
            </label>
        </div>
    )
}

export enum AlienDirection {
    DECODE = 'decode',
    ENCODE = 'encode'
}

export function AlienDirectionSelector(props: { onSelect: (format: AlienDirection) => void, direction: AlienDirection }) {
    return (
        <div class={styles.RadioButtons}>
            <label>
                <input type="radio" checked={props.direction === AlienDirection.DECODE} onChange={() => props.onSelect(AlienDirection.DECODE)} />
                Decode
            </label>
            <label>
                <input type="radio" checked={props.direction === AlienDirection.ENCODE} onChange={() => props.onSelect(AlienDirection.ENCODE)} />
                Encode
            </label>
        </div>
    )
}
