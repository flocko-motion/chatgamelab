import React, { useState, useEffect } from 'react';
import {Button, Form, FormGroup, Label, Input, Col, Row, FormText} from 'reactstrap';
import {SharePlayUrl} from "../utils/urls";
import {faMinus, faPlus} from "@fortawesome/free-solid-svg-icons";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";

const GameEditForm = ({ initialGame, onSave, onCancel }) => {
    const [formData, setFormData] = useState({
        ...initialGame,
        statusFields: initialGame.statusFields || []
    });

    console.log("game edit form: ", initialGame);

    useEffect(() => {
        setFormData({
            ...initialGame,
            statusFields: initialGame.statusFields || []
        });
    }, [initialGame]);

    const handleChange = (event, index) => {
        const { name, value } = event.target;

        if (index !== undefined) {
            // Handle changes in statusFields
            const updatedStatusFields = formData.statusFields.map((field, idx) =>
                idx === index ? { ...field, [name]: value } : field
            );
            setFormData({ ...formData, statusFields: updatedStatusFields });
        } else {
            // Handle changes in other fields
            setFormData({ ...formData, [name]: value });
        }
    };

    const handleToggle = (event) => {
        const { name, checked } = event.target;
        setFormData({ ...formData, [name]: checked });
    };

    const handleAddStatusField = () => {
        setFormData({
            ...formData,
            statusFields: [...formData.statusFields, { name: '', value: '' }]
        });
    };

    const handleRemoveStatusField = (index) => {
        const updatedStatusFields = formData.statusFields.filter((_, idx) => idx !== index);
        setFormData({ ...formData, statusFields: updatedStatusFields });
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
                <Label for="scenario" sm={2}>Game Scenario</Label>
                <Col sm={10}>
                    <Input
                        type="textarea"
                        name="scenario"
                        id="scenario"
                        value={formData.scenario || ''}
                        onChange={handleChange}
                    />
                    <FormText color="muted">
                        What is the game about? How does it work? What role does the player have? What's the world like?
                    </FormText>
                </Col>
            </FormGroup>

            <FormGroup row>
                <Label for="sessionStartSyscall" sm={2}>Game Opening</Label>
                <Col sm={10}>
                    <Input type="textarea" name="sessionStartSyscall" id="sessionStartSyscall" value={formData.sessionStartSyscall || ''} onChange={handleChange} />
                    <FormText color="muted">
                        How should the game start? What's the first scene? How is the player welcomed?
                    </FormText>
                </Col>
            </FormGroup>


            <FormGroup row>
                <Label for="imageStyle" sm={2}>Image Style</Label>
                <Col sm={10}>
                    <Input type="textarea" name="imageStyle" id="imageStyle" value={formData.imageStyle || ''} onChange={handleChange} />
                    <FormText color="muted">
                        What art style should be used for generating the images?
                    </FormText>
                </Col>
            </FormGroup>
            {/* Status Fields Section */}
            <FormGroup row>
                <Label for="statusFields" sm={2}>GPT Status Fields</Label>
                <Col sm={10}>
                    {formData.statusFields.map((field, index) => (
                        <Row key={index} className="mb-3" >
                            <Col >
                                <Input
                                    type="text"
                                    name="name"
                                    value={field.name || ''}
                                    placeholder="Name"
                                    onChange={(e) => handleChange(e, index)}
                                />
                            </Col>
                            <Col >
                                <Input
                                    type="text"
                                    name="value"
                                    value={field.value || ''}
                                    placeholder="Value"
                                    onChange={(e) => handleChange(e, index)}
                                />
                            </Col>
                            <Col>
                                <Button color="danger" onClick={() => handleRemoveStatusField(index)}><FontAwesomeIcon icon={faMinus}/></Button>
                            </Col>
                        </Row>
                    ))}


                    <Button  color="primary" onClick={handleAddStatusField}><FontAwesomeIcon icon={faPlus}/></Button>
                </Col>
            </FormGroup>



            <FormGroup row>
                <Label for="sharePlayActive" sm={2}>Public</Label>
                <Col sm={10}>
                    <Input type="checkbox" name="sharePlayActive" id="sharePlayActive" checked={formData.sharePlayActive || false} onChange={handleToggle} />
                </Col>
            </FormGroup>

            {formData.sharePlayActive && (
                <FormGroup row>
                    <Label for="sharePlayUrl" sm={2}>Public URL</Label>
                    <Col sm={10}>
                        <Input type="text" name="sharePlayUrl" id="sharePlayUrl" readOnly value={SharePlayUrl(formData.sharePlayHash) || ''} />
                        <FormText color="muted">
                            Playing the game via this URL will not require a login. Your public-playing-key will be used, which generates costs for you for each game played.
                        </FormText>
                    </Col>
                </FormGroup>
            )}

            <Button color="primary" type="submit">Save</Button>
            <Button color="secondary" onClick={onCancel} className="ml-2">Cancel</Button>
        </Form>
    );
};

export default GameEditForm;
