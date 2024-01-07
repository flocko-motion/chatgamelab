import React from 'react';
import { Card, CardBody, CardTitle, CardText } from 'reactstrap';
import ScrollableDiv from './ScrollableDiv';

const DebugLogsComponent = ({ logs }) => {
    return (
        <div className="flex-grow-1 overflow-auto p-2 bg-dark">
            {logs.map((log, index) => (
                <Card key={index} className="mb-2">
                    <CardBody>
                        <CardTitle tag="h5">Request {index + 1}</CardTitle>
                        <CardText>
                            <strong>Request:</strong> {log.request}
                        </CardText>
                        <CardText>
                            <strong>Response:</strong> {log.response}
                        </CardText>
                    </CardBody>
                </Card>
            ))}
        </div>
    );
};

export default DebugLogsComponent;
