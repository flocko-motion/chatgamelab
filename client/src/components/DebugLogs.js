import React from 'react';
import { Card, CardBody, CardTitle, CardText } from 'reactstrap';
import ScrollableDiv from './ScrollableDiv';

const DebugLogsComponent = ({ logs }) => {
    return (
        <ScrollableDiv>
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
        </ScrollableDiv>
    );
};

export default DebugLogsComponent;
