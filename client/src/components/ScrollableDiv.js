import React from 'react';

const ScrollableDiv = ({ children, className, style }) => {
    const defaultStyle = {
        // maxHeight: '500px',
        overflowY: 'auto',
        ...style
    };

    return (
        <div className={className} style={defaultStyle}>
            {children}
        </div>
    );
};

export default ScrollableDiv;
