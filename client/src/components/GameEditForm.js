import React, { useState, useEffect } from 'react';
import { Button, Form, FormGroup, Label, Input, Col, FormText } from 'reactstrap';
import {SharePlayUrl} from "../utils/urls";

const GameEditForm = ({ initialGame, onSave, onCancel }) => {
    const [formData, setFormData] = useState(initialGame);

    useEffect(() => {
        setFormData(initialGame);
    }, [initialGame]);

    const handleChange = (event) => {
        const { name, value } = event.target;
        setFormData({ ...formData, [name]: value });
    };

    const handleToggle = (event) => {
        const { name, checked } = event.target;
        setFormData({ ...formData, [name]: checked });
    };

    const handleSave = (event) => {
        event.preventDefault();
        onSave(formData);
    };

    return (
        <Form onSubmit={handleSave}>
            <FormGroup row>
                <Label for="owner" sm={2}>Owner</Label>
                <Col sm={10}>
                    <Input type="text" name="owner" id="owner" readOnly value={formData.userName || ''} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="title" sm={2}>Title</Label>
                <Col sm={10}>
                    <Input type="text" name="title" id="title" value={formData.title || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="description" sm={2}>Description</Label>
                <Col sm={10}>
                    <Input type="textarea" name="description" id="description" value={formData.description || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="scenario" sm={2}>Scenario</Label>
                <Col sm={10}>
                    <Input type="textarea" name="scenario" id="scenario" value={formData.scenario || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="imageStyle" sm={2}>Image Style</Label>
                <Col sm={10}>
                    <Input type="textarea" name="imageStyle" id="imageStyle" value={formData.imageStyle || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="sessionStartSyscall" sm={2}>Session Start Syscall</Label>
                <Col sm={10}>
                    <Input type="textarea" name="sessionStartSyscall" id="sessionStartSyscall" value={formData.sessionStartSyscall || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="postActionSyscall" sm={2}>Post Action Syscall</Label>
                <Col sm={10}>
                    <Input type="textarea" name="postActionSyscall" id="postActionSyscall" value={formData.postActionSyscall || ''} onChange={handleChange} />
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="sharePlayActive" sm={2}>Share Play</Label>
                <Col sm={10}>
                    <Input type="checkbox" name="sharePlayActive" id="sharePlayActive" checked={formData.sharePlayActive || false} onChange={handleToggle} />
                </Col>
            </FormGroup>

            {formData.sharePlayActive && (
                <FormGroup row>
                    <Label for="sharePlayUrl" sm={2}>Share Play URL</Label>
                    <Col sm={10}>
                        <Input type="text" name="sharePlayUrl" id="sharePlayUrl" readOnly value={SharePlayUrl(formData.sharePlayHash) || ''} />
                    </Col>
                </FormGroup>
            )}

            <Button color="primary" type="submit">Save</Button>
            <Button color="secondary" onClick={onCancel} className="ml-2">Cancel</Button>
        </Form>
    );
};

export default GameEditForm;
