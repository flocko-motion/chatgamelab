import React from 'react';

const Content = ({ children, className, style }) => {
    const defaultStyle = {
        overflowY: 'auto',  // Enable vertical scrolling
        height: "100%",     // Fill the parent div
        ...style            // Allow additional styles to be applied
    };

    return (
        <div className={"p-4 " + className} style={defaultStyle}>
            {children}
        </div>
    );
};

export default Content;
