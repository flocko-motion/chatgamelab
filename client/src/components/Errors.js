import React from 'react';
import { useRecoilState } from "recoil";
import { errorsState } from "../api/atoms";
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faExclamationTriangle } from '@fortawesome/free-solid-svg-icons';

const Errors = () => {
    const [errors, setErrors] = useRecoilState(errorsState);

    const handleClose = (index) => {
        const updatedErrors = errors.filter((_, i) => i !== index);
        setErrors(updatedErrors);
    }

    return <div className="errors"> {
        errors.map((error, index) => (
        <div key={index} className="alert alert-danger" role="alert">
            <FontAwesomeIcon icon={faExclamationTriangle} /> {error}
            <button type="button" className="close" aria-label="Close" onClick={() => handleClose(index)}>
                <span aria-hidden="true">&times;</span>
            </button>
        </div>
        ))
    }
    </div>
};

export default Errors;
