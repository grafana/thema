import React from "react";

interface ColumnProps {
    title: string;
    color: string;
}

const styles = {
    title: {
        textAlign: 'left' as const,
        paddingLeft: '1vw',
        fontWeight: 700,
    },
    contents: {
        height: '80vh',
        margin: '1vh 1vw',
        padding: '10px',
        border: 'rgba(204, 204, 220, 0.07) solid 1px',
        borderRadius: '2px',
    }
}
const Column = (props: React.PropsWithChildren<ColumnProps>) =>
    <div style={{height: '80vh', width: '33vw', paddingTop: '20px'}}>
        <h4 style={styles.title}>{`${props.title}:`}</h4>
        <div className='column-contents' style={styles.contents}>
            {props.children}
        </div>
    </div>;

export default Column;
