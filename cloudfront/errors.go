package cloudfront

import (
	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			// cloudfront.ErrCodeAccessDenied for service response error code
			// "AccessDenied".
			//
			// Access denied.
			cloudfront.ErrCodeAccessDenied:
			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case
			// cloudfront.ErrCodeBatchTooLarge for service response error code
			// "BatchTooLarge".
			cloudfront.ErrCodeBatchTooLarge,

			// cloudfront.ErrCodeFieldLevelEncryptionProfileSizeExceeded for service response error code
			// "FieldLevelEncryptionProfileSizeExceeded".
			//
			// The maximum size of a profile for field-level encryption was exceeded.
			cloudfront.ErrCodeFieldLevelEncryptionProfileSizeExceeded,

			// cloudfront.ErrCodeTooManyCacheBehaviors for service response error code
			// "TooManyCacheBehaviors".
			//
			// You cannot create more cache behaviors for the distribution.
			cloudfront.ErrCodeTooManyCacheBehaviors,

			// cloudfront.ErrCodeTooManyCertificates for service response error code
			// "TooManyCertificates".
			//
			// You cannot create anymore custom SSL/TLS certificates.
			cloudfront.ErrCodeTooManyCertificates,

			// cloudfront.ErrCodeTooManyCloudFrontOriginAccessIdentities for service response error code
			// "TooManyCloudFrontOriginAccessIdentities".
			//
			// Processing your request would cause you to exceed the maximum number of origin
			// access identities allowed.
			cloudfront.ErrCodeTooManyCloudFrontOriginAccessIdentities,

			// cloudfront.ErrCodeTooManyCookieNamesInWhiteList for service response error code
			// "TooManyCookieNamesInWhiteList".
			//
			// Your request contains more cookie names in the whitelist than are allowed
			// per cache behavior.
			cloudfront.ErrCodeTooManyCookieNamesInWhiteList,

			// cloudfront.ErrCodeTooManyDistributionCNAMEs for service response error code
			// "TooManyDistributionCNAMEs".
			//
			// Your request contains more CNAMEs than are allowed per distribution.
			cloudfront.ErrCodeTooManyDistributionCNAMEs,

			// cloudfront.ErrCodeTooManyDistributions for service response error code
			// "TooManyDistributions".
			//
			// Processing your request would cause you to exceed the maximum number of distributions
			// allowed.
			cloudfront.ErrCodeTooManyDistributions,

			// cloudfront.ErrCodeTooManyDistributionsAssociatedToFieldLevelEncryptionConfig for service response error code
			// "TooManyDistributionsAssociatedToFieldLevelEncryptionConfig".
			//
			// The maximum number of distributions have been associated with the specified
			// configuration for field-level encryption.
			cloudfront.ErrCodeTooManyDistributionsAssociatedToFieldLevelEncryptionConfig,

			// cloudfront.ErrCodeTooManyDistributionsWithLambdaAssociations for service response error code
			// "TooManyDistributionsWithLambdaAssociations".
			//
			// Processing your request would cause the maximum number of distributions with
			// Lambda function associations per owner to be exceeded.
			cloudfront.ErrCodeTooManyDistributionsWithLambdaAssociations,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionConfigs for service response error code
			// "TooManyFieldLevelEncryptionConfigs".
			//
			// The maximum number of configurations for field-level encryption have been
			// created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionConfigs,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionContentTypeProfiles for service response error code
			// "TooManyFieldLevelEncryptionContentTypeProfiles".
			//
			// The maximum number of content type profiles for field-level encryption have
			// been created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionContentTypeProfiles,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionEncryptionEntities for service response error code
			// "TooManyFieldLevelEncryptionEncryptionEntities".
			//
			// The maximum number of encryption entities for field-level encryption have
			// been created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionEncryptionEntities,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionFieldPatterns for service response error code
			// "TooManyFieldLevelEncryptionFieldPatterns".
			//
			// The maximum number of field patterns for field-level encryption have been
			// created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionFieldPatterns,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionProfiles for service response error code
			// "TooManyFieldLevelEncryptionProfiles".
			//
			// The maximum number of profiles for field-level encryption have been created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionProfiles,

			// cloudfront.ErrCodeTooManyFieldLevelEncryptionQueryArgProfiles for service response error code
			// "TooManyFieldLevelEncryptionQueryArgProfiles".
			//
			// The maximum number of query arg profiles for field-level encryption have
			// been created.
			cloudfront.ErrCodeTooManyFieldLevelEncryptionQueryArgProfiles,

			// cloudfront.ErrCodeTooManyHeadersInForwardedValues for service response error code
			// "TooManyHeadersInForwardedValues".
			cloudfront.ErrCodeTooManyHeadersInForwardedValues,

			// cloudfront.ErrCodeTooManyInvalidationsInProgress for service response error code
			// "TooManyInvalidationsInProgress".
			//
			// You have exceeded the maximum number of allowable InProgress invalidation
			// batch requests, or invalidation objects.
			cloudfront.ErrCodeTooManyInvalidationsInProgress,

			// cloudfront.ErrCodeTooManyLambdaFunctionAssociations for service response error code
			// "TooManyLambdaFunctionAssociations".
			//
			// Your request contains more Lambda function associations than are allowed
			// per distribution.
			cloudfront.ErrCodeTooManyLambdaFunctionAssociations,

			// cloudfront.ErrCodeTooManyOriginCustomHeaders for service response error code
			// "TooManyOriginCustomHeaders".
			cloudfront.ErrCodeTooManyOriginCustomHeaders,

			// cloudfront.ErrCodeTooManyOriginGroupsPerDistribution for service response error code
			// "TooManyOriginGroupsPerDistribution".
			//
			// Processing your request would cause you to exceed the maximum number of origin
			// groups allowed.
			cloudfront.ErrCodeTooManyOriginGroupsPerDistribution,

			// cloudfront.ErrCodeTooManyOrigins for service response error code
			// "TooManyOrigins".
			//
			// You cannot create more origins for the distribution.
			cloudfront.ErrCodeTooManyOrigins,

			// cloudfront.ErrCodeTooManyPublicKeys for service response error code
			// "TooManyPublicKeys".
			//
			// The maximum number of public keys for field-level encryption have been created.
			// To create a new public key, delete one of the existing keys.
			cloudfront.ErrCodeTooManyPublicKeys,

			// cloudfront.ErrCodeTooManyQueryStringParameters for service response error code
			// "TooManyQueryStringParameters".
			cloudfront.ErrCodeTooManyQueryStringParameters,

			// cloudfront.ErrCodeTooManyStreamingDistributionCNAMEs for service response error code
			// "TooManyStreamingDistributionCNAMEs".
			cloudfront.ErrCodeTooManyStreamingDistributionCNAMEs,

			// cloudfront.ErrCodeTooManyStreamingDistributions for service response error code
			// "TooManyStreamingDistributions".
			//
			// Processing your request would cause you to exceed the maximum number of streaming
			// distributions allowed.
			cloudfront.ErrCodeTooManyStreamingDistributions,

			// cloudfront.ErrCodeTooManyTrustedSigners for service response error code
			// "TooManyTrustedSigners".
			//
			// Your request contains more trusted signers than are allowed per distribution.
			cloudfront.ErrCodeTooManyTrustedSigners:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case
			// cloudfront.ErrCodeCNAMEAlreadyExists for service response error code
			// "CNAMEAlreadyExists".
			cloudfront.ErrCodeCNAMEAlreadyExists,
			// cloudfront.ErrCodeDistributionAlreadyExists for service response error code
			// "DistributionAlreadyExists".
			//
			// The caller reference you attempted to create the distribution with is associated
			// with another distribution.
			cloudfront.ErrCodeDistributionAlreadyExists,

			// cloudfront.ErrCodeFieldLevelEncryptionConfigAlreadyExists for service response error code
			// "FieldLevelEncryptionConfigAlreadyExists".
			//
			// The specified configuration for field-level encryption already exists.
			cloudfront.ErrCodeFieldLevelEncryptionConfigAlreadyExists,

			// cloudfront.ErrCodeFieldLevelEncryptionConfigInUse for service response error code
			// "FieldLevelEncryptionConfigInUse".
			//
			// The specified configuration for field-level encryption is in use.
			cloudfront.ErrCodeFieldLevelEncryptionConfigInUse,

			// cloudfront.ErrCodeFieldLevelEncryptionProfileAlreadyExists for service response error code
			// "FieldLevelEncryptionProfileAlreadyExists".
			//
			// The specified profile for field-level encryption already exists.
			cloudfront.ErrCodeFieldLevelEncryptionProfileAlreadyExists,

			// cloudfront.ErrCodeFieldLevelEncryptionProfileInUse for service response error code
			// "FieldLevelEncryptionProfileInUse".
			//
			// The specified profile for field-level encryption is in use.
			cloudfront.ErrCodeFieldLevelEncryptionProfileInUse,

			// cloudfront.ErrCodeOriginAccessIdentityAlreadyExists for service response error code
			// "CloudFrontOriginAccessIdentityAlreadyExists".
			//
			// If the CallerReference is a value you already sent in a previous request
			// to create an identity but the content of the CloudFrontOriginAccessIdentityConfig
			// is different from the original request, CloudFront returns a CloudFrontOriginAccessIdentityAlreadyExists
			// error.
			cloudfront.ErrCodeOriginAccessIdentityAlreadyExists,

			// cloudfront.ErrCodeOriginAccessIdentityInUse for service response error code
			// "CloudFrontOriginAccessIdentityInUse".
			cloudfront.ErrCodeOriginAccessIdentityInUse,

			// cloudfront.ErrCodePublicKeyAlreadyExists for service response error code
			// "PublicKeyAlreadyExists".
			//
			// The specified public key already exists.
			cloudfront.ErrCodePublicKeyAlreadyExists,

			// cloudfront.ErrCodePublicKeyInUse for service response error code
			// "PublicKeyInUse".
			//
			// The specified public key is in use.
			cloudfront.ErrCodePublicKeyInUse,

			// cloudfront.ErrCodeStreamingDistributionAlreadyExists for service response error code
			// "StreamingDistributionAlreadyExists".
			cloudfront.ErrCodeStreamingDistributionAlreadyExists:

			// return a conflict
			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// cloudfront.ErrCodeCannotChangeImmutablePublicKeyFields for service response error code
			// "CannotChangeImmutablePublicKeyFields".
			//
			// You can't change the value of a public key.
			cloudfront.ErrCodeCannotChangeImmutablePublicKeyFields,

			// cloudfront.ErrCodeDistributionNotDisabled for service response error code
			// "DistributionNotDisabled".
			cloudfront.ErrCodeDistributionNotDisabled,

			// cloudfront.ErrCodeIllegalFieldLevelEncryptionConfigAssociationWithCacheBehavior for service response error code
			// "IllegalFieldLevelEncryptionConfigAssociationWithCacheBehavior".
			//
			// The specified configuration for field-level encryption can't be associated
			// with the specified cache behavior.
			cloudfront.ErrCodeIllegalFieldLevelEncryptionConfigAssociationWithCacheBehavior,

			// cloudfront.ErrCodeIllegalUpdate for service response error code
			// "IllegalUpdate".
			//
			// Origin and CallerReference cannot be updated.
			cloudfront.ErrCodeIllegalUpdate,

			// cloudfront.ErrCodeInconsistentQuantities for service response error code
			// "InconsistentQuantities".
			//
			// The value of Quantity and the size of Items don't match.
			cloudfront.ErrCodeInconsistentQuantities,

			// cloudfront.ErrCodeInvalidArgument for service response error code
			// "InvalidArgument".
			//
			// The argument is invalid.
			cloudfront.ErrCodeInvalidArgument,

			// cloudfront.ErrCodeInvalidDefaultRootObject for service response error code
			// "InvalidDefaultRootObject".
			//
			// The default root object file name is too big or contains an invalid character.
			cloudfront.ErrCodeInvalidDefaultRootObject,

			// cloudfront.ErrCodeInvalidErrorCode for service response error code
			// "InvalidErrorCode".
			cloudfront.ErrCodeInvalidErrorCode,

			// cloudfront.ErrCodeInvalidForwardCookies for service response error code
			// "InvalidForwardCookies".
			//
			// Your request contains forward cookies option which doesn't match with the
			// expectation for the whitelisted list of cookie names. Either list of cookie
			// names has been specified when not allowed or list of cookie names is missing
			// when expected.
			cloudfront.ErrCodeInvalidForwardCookies,

			// cloudfront.ErrCodeInvalidGeoRestrictionParameter for service response error code
			// "InvalidGeoRestrictionParameter".
			cloudfront.ErrCodeInvalidGeoRestrictionParameter,

			// cloudfront.ErrCodeInvalidHeadersForS3Origin for service response error code
			// "InvalidHeadersForS3Origin".
			cloudfront.ErrCodeInvalidHeadersForS3Origin,

			// cloudfront.ErrCodeInvalidIfMatchVersion for service response error code
			// "InvalidIfMatchVersion".
			//
			// The If-Match version is missing or not valid for the distribution.
			cloudfront.ErrCodeInvalidIfMatchVersion,

			// cloudfront.ErrCodeInvalidLambdaFunctionAssociation for service response error code
			// "InvalidLambdaFunctionAssociation".
			//
			// The specified Lambda function association is invalid.
			cloudfront.ErrCodeInvalidLambdaFunctionAssociation,

			// cloudfront.ErrCodeInvalidLocationCode for service response error code
			// "InvalidLocationCode".
			cloudfront.ErrCodeInvalidLocationCode,

			// cloudfront.ErrCodeInvalidMinimumProtocolVersion for service response error code
			// "InvalidMinimumProtocolVersion".
			cloudfront.ErrCodeInvalidMinimumProtocolVersion,

			// cloudfront.ErrCodeInvalidOrigin for service response error code
			// "InvalidOrigin".
			//
			// The Amazon S3 origin server specified does not refer to a valid Amazon S3
			// bucket.
			cloudfront.ErrCodeInvalidOrigin,

			// cloudfront.ErrCodeInvalidOriginAccessIdentity for service response error code
			// "InvalidOriginAccessIdentity".
			//
			// The origin access identity is not valid or doesn't exist.
			cloudfront.ErrCodeInvalidOriginAccessIdentity,

			// cloudfront.ErrCodeInvalidOriginKeepaliveTimeout for service response error code
			// "InvalidOriginKeepaliveTimeout".
			cloudfront.ErrCodeInvalidOriginKeepaliveTimeout,

			// cloudfront.ErrCodeInvalidOriginReadTimeout for service response error code
			// "InvalidOriginReadTimeout".
			cloudfront.ErrCodeInvalidOriginReadTimeout,

			// cloudfront.ErrCodeInvalidProtocolSettings for service response error code
			// "InvalidProtocolSettings".
			//
			// You cannot specify SSLv3 as the minimum protocol version if you only want
			// to support only clients that support Server Name Indication (SNI).
			cloudfront.ErrCodeInvalidProtocolSettings,

			// cloudfront.ErrCodeInvalidQueryStringParameters for service response error code
			// "InvalidQueryStringParameters".
			cloudfront.ErrCodeInvalidQueryStringParameters,

			// cloudfront.ErrCodeInvalidRelativePath for service response error code
			// "InvalidRelativePath".
			//
			// The relative path is too big, is not URL-encoded, or does not begin with
			// a slash (/).
			cloudfront.ErrCodeInvalidRelativePath,

			// cloudfront.ErrCodeInvalidRequiredProtocol for service response error code
			// "InvalidRequiredProtocol".
			//
			// This operation requires the HTTPS protocol. Ensure that you specify the HTTPS
			// protocol in your request, or omit the RequiredProtocols element from your
			// distribution configuration.
			cloudfront.ErrCodeInvalidRequiredProtocol,

			// cloudfront.ErrCodeInvalidResponseCode for service response error code
			// "InvalidResponseCode".
			cloudfront.ErrCodeInvalidResponseCode,

			// cloudfront.ErrCodeInvalidTTLOrder for service response error code
			// "InvalidTTLOrder".
			cloudfront.ErrCodeInvalidTTLOrder,

			// cloudfront.ErrCodeInvalidTagging for service response error code
			// "InvalidTagging".
			cloudfront.ErrCodeInvalidTagging,

			// cloudfront.ErrCodeInvalidViewerCertificate for service response error code
			// "InvalidViewerCertificate".
			cloudfront.ErrCodeInvalidViewerCertificate,

			// cloudfront.ErrCodeInvalidWebACLId for service response error code
			// "InvalidWebACLId".
			cloudfront.ErrCodeInvalidWebACLId,

			// cloudfront.ErrCodeMissingBody for service response error code
			// "MissingBody".
			//
			// This operation requires a body. Ensure that the body is present and the Content-Type
			// header is set.
			cloudfront.ErrCodeMissingBody,

			// cloudfront.ErrCodePreconditionFailed for service response error code
			// "PreconditionFailed".
			//
			// The precondition given in one or more of the request-header fields evaluated
			// to false.
			cloudfront.ErrCodePreconditionFailed,

			// cloudfront.ErrCodeQueryArgProfileEmpty for service response error code
			// "QueryArgProfileEmpty".
			//
			// No profile specified for the field-level encryption query argument.
			cloudfront.ErrCodeQueryArgProfileEmpty,

			// cloudfront.ErrCodeStreamingDistributionNotDisabled for service response error code
			// "StreamingDistributionNotDisabled".
			cloudfront.ErrCodeStreamingDistributionNotDisabled:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// cloudfront.ErrCodeNoSuchCloudFrontOriginAccessIdentity for service response error code
			// "NoSuchCloudFrontOriginAccessIdentity".
			//
			// The specified origin access identity does not exist.
			cloudfront.ErrCodeNoSuchCloudFrontOriginAccessIdentity,

			// cloudfront.ErrCodeNoSuchDistribution for service response error code
			// "NoSuchDistribution".
			//
			// The specified distribution does not exist.
			cloudfront.ErrCodeNoSuchDistribution,

			// cloudfront.ErrCodeNoSuchFieldLevelEncryptionConfig for service response error code
			// "NoSuchFieldLevelEncryptionConfig".
			//
			// The specified configuration for field-level encryption doesn't exist.
			cloudfront.ErrCodeNoSuchFieldLevelEncryptionConfig,

			// cloudfront.ErrCodeNoSuchFieldLevelEncryptionProfile for service response error code
			// "NoSuchFieldLevelEncryptionProfile".
			//
			// The specified profile for field-level encryption doesn't exist.
			cloudfront.ErrCodeNoSuchFieldLevelEncryptionProfile,

			// cloudfront.ErrCodeNoSuchInvalidation for service response error code
			// "NoSuchInvalidation".
			//
			// The specified invalidation does not exist.
			cloudfront.ErrCodeNoSuchInvalidation,

			// cloudfront.ErrCodeNoSuchOrigin for service response error code
			// "NoSuchOrigin".
			//
			// No origin exists with the specified Origin Id.
			cloudfront.ErrCodeNoSuchOrigin,

			// cloudfront.ErrCodeNoSuchPublicKey for service response error code
			// "NoSuchPublicKey".
			//
			// The specified public key doesn't exist.
			cloudfront.ErrCodeNoSuchPublicKey,

			// cloudfront.ErrCodeNoSuchResource for service response error code
			// "NoSuchResource".
			cloudfront.ErrCodeNoSuchResource,

			// cloudfront.ErrCodeNoSuchStreamingDistribution for service response error code
			// "NoSuchStreamingDistribution".
			//
			// The specified streaming distribution does not exist.
			cloudfront.ErrCodeNoSuchStreamingDistribution,

			// cloudfront.ErrCodeTrustedSignerDoesNotExist for service response error code
			// "TrustedSignerDoesNotExist".
			//
			// One or more of your trusted signers don't exist.
			cloudfront.ErrCodeTrustedSignerDoesNotExist:
			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrBadRequest, msg, err)
}
