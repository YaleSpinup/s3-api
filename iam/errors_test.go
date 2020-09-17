package iam

import (
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		iam.ErrCodeLimitExceededException:                 apierror.ErrLimitExceeded,
		iam.ErrCodeReportGenerationLimitExceededException: apierror.ErrLimitExceeded,

		iam.ErrCodeCredentialReportExpiredException:    apierror.ErrConflict,
		iam.ErrCodeCredentialReportNotPresentException: apierror.ErrConflict,
		iam.ErrCodeCredentialReportNotReadyException:   apierror.ErrConflict,
		iam.ErrCodeDeleteConflictException:             apierror.ErrConflict,
		iam.ErrCodeDuplicateCertificateException:       apierror.ErrConflict,
		iam.ErrCodeDuplicateSSHPublicKeyException:      apierror.ErrConflict,
		iam.ErrCodeEntityAlreadyExistsException:        apierror.ErrConflict,
		iam.ErrCodeConcurrentModificationException:     apierror.ErrConflict,

		iam.ErrCodeEntityTemporarilyUnmodifiableException: apierror.ErrBadRequest,
		iam.ErrCodeInvalidAuthenticationCodeException:     apierror.ErrBadRequest,
		iam.ErrCodeInvalidCertificateException:            apierror.ErrBadRequest,
		iam.ErrCodeInvalidInputException:                  apierror.ErrBadRequest,
		iam.ErrCodeInvalidPublicKeyException:              apierror.ErrBadRequest,
		iam.ErrCodeInvalidUserTypeException:               apierror.ErrBadRequest,
		iam.ErrCodeKeyPairMismatchException:               apierror.ErrBadRequest,
		iam.ErrCodeMalformedCertificateException:          apierror.ErrBadRequest,
		iam.ErrCodeMalformedPolicyDocumentException:       apierror.ErrBadRequest,
		iam.ErrCodePasswordPolicyViolationException:       apierror.ErrBadRequest,
		iam.ErrCodePolicyEvaluationException:              apierror.ErrBadRequest,
		iam.ErrCodePolicyNotAttachableException:           apierror.ErrBadRequest,
		iam.ErrCodeServiceNotSupportedException:           apierror.ErrBadRequest,
		iam.ErrCodeUnmodifiableEntityException:            apierror.ErrBadRequest,
		iam.ErrCodeUnrecognizedPublicKeyEncodingException: apierror.ErrBadRequest,

		iam.ErrCodeNoSuchEntityException:   apierror.ErrNotFound,
		iam.ErrCodeServiceFailureException: apierror.ErrInternalError,
	}

	for awsErr, apiErr := range apiErrorTestCases {
		err := ErrCode("test error", awserr.New(awsErr, awsErr, nil))
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected cloudwatch error %s to be an apierror.Error %s, got %s", awsErr, apiErr, err)
		}
	}

	err := ErrCode("test error", errors.New("Unknown"))
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		t.Logf("got apierror '%s'", aerr)
	} else {
		t.Errorf("expected unknown error to be an apierror.ErrInternalError, got %s", err)
	}
}
