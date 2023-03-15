import React, {CSSProperties} from "react";

interface Props {
    id: string;
    options: string[];
    onChange: (val: string) => void;
    style?: CSSProperties
    disabled?: boolean;
}

const Dropdown = ({id, options, onChange, style, disabled}: Props) => {
    const onSelectChange = (evt: React.FormEvent<HTMLSelectElement>) =>
        onChange(evt.currentTarget.value);

    return (
        <select style={style} onChange={onSelectChange} id={id} disabled={disabled}>
            {disabled
                ? <option key='na' value='na'>N/A</option>
                : options.map((opt: string) =>
                    <option key={opt} value={opt}>{opt}</option>)
            }
        </select>
    );
};

export default Dropdown;
