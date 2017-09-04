package journal

// File rotation frequency
const (
	ROT_NONE     = 0
	ROT_DAILY    = 1
	ROT_WEEKLY   = 2
	ROT_MONTHLY  = 3
	ROT_ANNUALLY = 4
)

// Output selection
const (
	OUT_FILE            = 0
	OUT_STDOUT          = 1
	OUT_FILE_AND_STDOUT = 2
)

// Log columns
const (
	COL_DATE_YYMMDD             = 0
	COL_DATE_YYMMDD_HHMMSS      = 1
	COL_DATE_YYMMDD_HHMMSS_NANO = 2
	COL_TIMESTAMP               = 3
	COL_SERVICE                 = 4
	COL_INSTANCE                = 5
	COL_CALLER                  = 6
	COL_MSG_TYPE_SHORT          = 7
	COL_MSG_TYPE_INT            = 8
	COL_MSG_TYPE_STR            = 9
	COL_MSG                     = 10
	COL_FILE                    = 11
	COL_LINE                    = 12
)

// colname returns a column's textual representation
func colname(col int64) string {

	switch col {
	case COL_DATE_YYMMDD:
		return "Date"
	case COL_DATE_YYMMDD_HHMMSS:
		return "Date"
	case COL_DATE_YYMMDD_HHMMSS_NANO:
		return "Date"
	case COL_TIMESTAMP:
		return "Date"
	case COL_SERVICE:
		return "Service"
	case COL_INSTANCE:
		return "Instance"
	case COL_CALLER:
		return "Caller"
	case COL_MSG_TYPE_SHORT:
		return "Type"
	case COL_MSG_TYPE_INT:
		return "Type_INT"
	case COL_MSG_TYPE_STR:
		return "Type_STR"
	case COL_MSG:
		return "Message"
	case COL_FILE:
		return "File"
	case COL_LINE:
		return "Line"
	default:
		return "Unknown"
	}

}

// Code contains a single message type with an indicator of whether this
// message should be treated as an error.
type Code struct {
	Error bool
	Type  string
}

// defaultCodes contains default message codes used by the logger
var defaultCodes = map[int]Code{
	0:   Code{false, "Notification"},
	1:   Code{true, "GeneralError"},
	2:   Code{true, "ConfigurationError"},
	3:   Code{true, "FailedAction"},
	4:   Code{true, "UserError"},
	10:  Code{true, "CatastrophicFailure"},
	100: Code{false, "HTTP-StatusContinue"},
	101: Code{false, "HTTP-StatusSwitchingProtocols"},
	102: Code{false, "HTTP-StatusProcessing"},
	200: Code{false, "HTTP-StatusOK"},
	201: Code{false, "HTTP-StatusCreated"},
	202: Code{false, "HTTP-StatusAccepted"},
	203: Code{false, "HTTP-StatusNonAuthoritativeInfo"},
	204: Code{false, "HTTP-StatusNoContent"},
	205: Code{false, "HTTP-StatusResetContent"},
	206: Code{false, "HTTP-StatusPartialContent"},
	207: Code{false, "HTTP-StatusMultiStatus"},
	208: Code{false, "HTTP-StatusAlreadyReported"},
	226: Code{false, "HTTP-StatusIMUsed"},
	300: Code{false, "HTTP-StatusMultipleChoices"},
	301: Code{false, "HTTP-StatusMovedPermanently"},
	302: Code{false, "HTTP-StatusFound"},
	303: Code{false, "HTTP-StatusSeeOther"},
	304: Code{false, "HTTP-StatusNotModified"},
	305: Code{false, "HTTP-StatusUseProxy"},
	307: Code{false, "HTTP-StatusTemporaryRedirect"},
	308: Code{false, "HTTP-StatusPermanentRedirect"},
	400: Code{true, "HTTP-StatusBadRequest"},
	401: Code{true, "HTTP-StatusUnauthorized"},
	402: Code{true, "HTTP-StatusPaymentRequired"},
	403: Code{true, "HTTP-StatusForbidden"},
	404: Code{true, "HTTP-StatusNotFound"},
	405: Code{true, "HTTP-StatusMethodNotAllowed"},
	406: Code{true, "HTTP-StatusNotAcceptable"},
	407: Code{true, "HTTP-StatusProxyAuthRequired"},
	408: Code{true, "HTTP-StatusRequestTimeout"},
	409: Code{true, "HTTP-StatusConflict"},
	410: Code{true, "HTTP-StatusGone"},
	411: Code{true, "HTTP-StatusLengthRequired"},
	412: Code{true, "HTTP-StatusPreconditionFailed"},
	413: Code{true, "HTTP-StatusRequestEntityTooLarge"},
	414: Code{true, "HTTP-StatusRequestURITooLong"},
	415: Code{true, "HTTP-StatusUnsupportedMediaType"},
	416: Code{true, "HTTP-StatusRequestedRangeNotSatisfiable"},
	417: Code{true, "HTTP-StatusExpectationFailed"},
	418: Code{true, "HTTP-StatusTeapot"},
	422: Code{true, "HTTP-StatusUnprocessableEntity"},
	423: Code{true, "HTTP-StatusLocked"},
	424: Code{true, "HTTP-StatusFailedDependency"},
	426: Code{true, "HTTP-StatusUpgradeRequired"},
	428: Code{true, "HTTP-StatusPreconditionRequired"},
	429: Code{true, "HTTP-StatusTooManyRequests"},
	431: Code{true, "HTTP-StatusRequestHeaderFieldsTooLarge"},
	451: Code{true, "HTTP-StatusUnavailableForLegalReasons"},
	500: Code{true, "HTTP-StatusInternalServerError"},
	501: Code{true, "HTTP-StatusNotImplemented"},
	502: Code{true, "HTTP-StatusBadGateway"},
	503: Code{true, "HTTP-StatusServiceUnavailable"},
	504: Code{true, "HTTP-StatusGatewayTimeout"},
	505: Code{true, "HTTP-StatusHTTPVersionNotSupported"},
	506: Code{true, "HTTP-StatusVariantAlsoNegotiates"},
	507: Code{true, "HTTP-StatusInsufficientStorage"},
	508: Code{true, "HTTP-StatusLoopDetected"},
	510: Code{true, "HTTP-StatusNotExtended"},
	511: Code{true, "HTTP-StatusNetworkAuthenticationRequired"},
	999: Code{true, "Exception/Unintended"},
}

// defaultCols contains default log columns
var defaultCols = []int64{COL_DATE_YYMMDD_HHMMSS_NANO, COL_SERVICE, COL_INSTANCE, COL_MSG_TYPE_SHORT,
	COL_MSG_TYPE_INT, COL_MSG_TYPE_STR, COL_MSG, COL_FILE, COL_LINE}
