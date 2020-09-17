package iam

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			"Forbidden":

			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case
			// iam.ErrCodeLimitExceededException for service response error code
			// "LimitExceeded".
			//
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limitations. The error message describes the limit
			// exceeded.
			iam.ErrCodeLimitExceededException,

			// iam.ErrCodeReportGenerationLimitExceededException for service response error code
			// "ReportGenerationLimitExceeded".
			//
			// The request failed because the maximum number of concurrent requests for
			// this account are already running.
			iam.ErrCodeReportGenerationLimitExceededException:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case

			// iam.ErrCodeCredentialReportExpiredException for service response error code
			// "ReportExpired".
			//
			// The request was rejected because the most recent credential report has expired.
			// To generate a new credential report, use GenerateCredentialReport. For more
			// information about credential report expiration, see Getting Credential Reports
			// (https://docs.aws.amazon.com/IAM/latest/UserGuide/credential-reports.html)
			// in the IAM User Guide.
			iam.ErrCodeCredentialReportExpiredException,

			// iam.ErrCodeCredentialReportNotPresentException for service response error code
			// "ReportNotPresent".
			//
			// The request was rejected because the credential report does not exist. To
			// generate a credential report, use GenerateCredentialReport.
			iam.ErrCodeCredentialReportNotPresentException,

			// iam.ErrCodeCredentialReportNotReadyException for service response error code
			// "ReportInProgress".
			//
			// The request was rejected because the credential report is still being generated.
			iam.ErrCodeCredentialReportNotReadyException,

			// iam.ErrCodeDeleteConflictException for service response error code
			// "DeleteConflict".
			//
			// The request was rejected because it attempted to delete a resource that has
			// attached subordinate entities. The error message describes these entities.
			iam.ErrCodeDeleteConflictException,

			// iam.ErrCodeDuplicateCertificateException for service response error code
			// "DuplicateCertificate".
			//
			// The request was rejected because the same certificate is associated with
			// an IAM user in the account.
			iam.ErrCodeDuplicateCertificateException,

			// iam.ErrCodeDuplicateSSHPublicKeyException for service response error code
			// "DuplicateSSHPublicKey".
			//
			// The request was rejected because the SSH public key is already associated
			// with the specified IAM user.
			iam.ErrCodeDuplicateSSHPublicKeyException,

			// iam.ErrCodeEntityAlreadyExistsException for service response error code
			// "EntityAlreadyExists".
			//
			// The request was rejected because it attempted to create a resource that already
			// exists.
			iam.ErrCodeEntityAlreadyExistsException,

			// iam.ErrCodeConcurrentModificationException for service response error code
			// "ConcurrentModification".
			//
			// The request was rejected because multiple requests to change this object
			// were submitted simultaneously. Wait a few minutes and submit your request
			// again.
			iam.ErrCodeConcurrentModificationException:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// iam.ErrCodeEntityTemporarilyUnmodifiableException for service response error code
			// "EntityTemporarilyUnmodifiable".
			//
			// The request was rejected because it referenced an entity that is temporarily
			// unmodifiable, such as a user name that was deleted and then recreated. The
			// error indicates that the request is likely to succeed if you try again after
			// waiting several minutes. The error message describes the entity.
			iam.ErrCodeEntityTemporarilyUnmodifiableException,

			// iam.ErrCodeInvalidAuthenticationCodeException for service response error code
			// "InvalidAuthenticationCode".
			//
			// The request was rejected because the authentication code was not recognized.
			// The error message describes the specific error.
			iam.ErrCodeInvalidAuthenticationCodeException,

			// iam.ErrCodeInvalidCertificateException for service response error code
			// "InvalidCertificate".
			//
			// The request was rejected because the certificate is invalid.
			iam.ErrCodeInvalidCertificateException,

			// iam.ErrCodeInvalidInputException for service response error code
			// "InvalidInput".
			//
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			iam.ErrCodeInvalidInputException,

			// iam.ErrCodeInvalidPublicKeyException for service response error code
			// "InvalidPublicKey".
			//
			// The request was rejected because the public key is malformed or otherwise
			// invalid.
			iam.ErrCodeInvalidPublicKeyException,

			// iam.ErrCodeInvalidUserTypeException for service response error code
			// "InvalidUserType".
			//
			// The request was rejected because the type of user for the transaction was
			// incorrect.
			iam.ErrCodeInvalidUserTypeException,

			// iam.ErrCodeKeyPairMismatchException for service response error code
			// "KeyPairMismatch".
			//
			// The request was rejected because the public key certificate and the private
			// key do not match.
			iam.ErrCodeKeyPairMismatchException,

			// iam.ErrCodeMalformedCertificateException for service response error code
			// "MalformedCertificate".
			//
			// The request was rejected because the certificate was malformed or expired.
			// The error message describes the specific error.
			iam.ErrCodeMalformedCertificateException,

			// iam.ErrCodeMalformedPolicyDocumentException for service response error code
			// "MalformedPolicyDocument".
			//
			// The request was rejected because the policy document was malformed. The error
			// message describes the specific error.
			iam.ErrCodeMalformedPolicyDocumentException,

			// iam.ErrCodePasswordPolicyViolationException for service response error code
			// "PasswordPolicyViolation".
			//
			// The request was rejected because the provided password did not meet the requirements
			// imposed by the account password policy.
			iam.ErrCodePasswordPolicyViolationException,

			// iam.ErrCodePolicyEvaluationException for service response error code
			// "PolicyEvaluation".
			//
			// The request failed because a provided policy could not be successfully evaluated.
			// An additional detailed message indicates the source of the failure.
			iam.ErrCodePolicyEvaluationException,

			// iam.ErrCodePolicyNotAttachableException for service response error code
			// "PolicyNotAttachable".
			//
			// The request failed because AWS service role policies can only be attached
			// to the service-linked role for that service.
			iam.ErrCodePolicyNotAttachableException,

			// iam.ErrCodeServiceNotSupportedException for service response error code
			// "NotSupportedService".
			//
			// The specified service does not support service-specific credentials.
			iam.ErrCodeServiceNotSupportedException,

			// iam.ErrCodeUnmodifiableEntityException for service response error code
			// "UnmodifiableEntity".
			//
			// The request was rejected because only the service that depends on the service-linked
			// role can modify or delete the role on your behalf. The error message includes
			// the name of the service that depends on this service-linked role. You must
			// request the change through that service.
			iam.ErrCodeUnmodifiableEntityException,

			// iam.ErrCodeUnrecognizedPublicKeyEncodingException for service response error code
			// "UnrecognizedPublicKeyEncoding".
			//
			// The request was rejected because the public key encoding format is unsupported
			// or unrecognized.
			iam.ErrCodeUnrecognizedPublicKeyEncodingException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// iam.ErrCodeNoSuchEntityException for service response error code
			// "NoSuchEntity".
			//
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			iam.ErrCodeNoSuchEntityException:

			return apierror.New(apierror.ErrNotFound, msg, aerr)

		case

			// iam.ErrCodeServiceFailureException for service response error code
			// "ServiceFailure".
			//
			// The request processing has failed because of an unknown error, exception
			// or failure.
			iam.ErrCodeServiceFailureException:

			return apierror.New(apierror.ErrServiceUnavailable, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
