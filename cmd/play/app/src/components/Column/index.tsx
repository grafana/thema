import React from "react";

interface ColumnProps {
    title: string;
    color: string;
}

const Column = (props: React.PropsWithChildren<ColumnProps>) =>
    <div style={{height: '80vh', width: '33vw', paddingTop: '20px'}}>
        <h4 style={{textAlign: 'left', paddingLeft: '1vw'}}>{`${props.title}:`}</h4>
        <div style={{height: '80vh', margin: '1vh 1vw', border: `1px dashed ${props.color}`}}>
            {props.children}
        </div>
    </div>;

export default Column;
