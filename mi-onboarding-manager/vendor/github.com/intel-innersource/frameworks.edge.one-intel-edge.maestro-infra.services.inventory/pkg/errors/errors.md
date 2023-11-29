Errors
---------------------------

The errors functions implemented in this package offers helpers to easily wrap
and manage errors in EdgeI components. Errors are always augmented with context
information to allow debugging using a "single pane of glass": if the client
gets an error you can easily print the stacktrace with the information of the
error faced and where it happened in the server which means you don't have to look
in parallel the server logs and hope to catch the exact moment you got the error

Note this is not meant to replace the tracing, but it offers at no cost and no
time an easy way to debug.

Note also that the errors returned by this package are actually grpc status
which implements the Error() interface. grpc status can transport arbitrary
messages using the `details` field.

Key component is the `errors.proto` which describes the proto message `ErrorInfo`,
this is used to carry additional information over the errors. This message is
structured and parseable automatically by a machine. The latter can use the fields
defined to act upon failures.

## API Documentation

See the Go doc of the package for detailed function descriptions. Here is the
general workflow:

```go
// assuming the import is called errors

// Wraps an existing error errrr, coming from a 3rd party library
// Add the support for additionally libraries!!!
err := errors.Wrap(errrr)

// Creates a new error, using a fmt string
err := errors.Errorf("I am an %s", "error")

// Creates a new error, using a fmt string and code
err := errors.Errorfc(codes.PermissionDenied, "I am an %s", "error")

// Creates a new error, using a fmt string and reason
err := errors.Errorfr(errors.Reason_UNKNOWN_CLIENT, "I am an %s", "error")

// Check if it is an Unknown client
resp, err := gcli.Invapi.CreateResource(ctx, &req)
if errors.IsUnKnownClient(err) {
    // do something
}

// Check if it is a Not Found error
resp, err := gcli.Invapi.GetResource(ctx, &req)
if errors.IsNotFound(err) {
// do something
}

// Debug errors
log.Debug().Msgf("Detail %s", errors.ErrorToStringWithDetails(err))

// Print minimum available info
log.Info().Msgf("Msg %s", errors.ErrorToString(err))

// Convert to HTTP status
err := errors.Wrap(context.Canceled)
httpStatus = errors.ErrorToHttpStatus(err)

// Returns directly to the caller
func (s *server) RegisterSomethingByID(ctx context.Context, in *Message) (*Response, error) {
	if something_went_wrong {
        // No need to further transcoding. NOTE that
        // the string invalid request won't be leaked
        // and by default will transparent to the caller
		return nil, errors.Errorf("invalid request")
	}
}

```
