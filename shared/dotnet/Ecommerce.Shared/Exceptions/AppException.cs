using System.Net;

namespace Ecommerce.Shared.Exceptions;

public class AppException : Exception
{
    public string ErrorCode { get; }
    public HttpStatusCode StatusCode { get; }
    public object? Details { get; }

    public AppException(
        string errorCode,
        string message,
        HttpStatusCode statusCode,
        object? details = null)
        : base(message)
    {
        ErrorCode = errorCode;
        StatusCode = statusCode;
        Details = details;
    }
}

public static class CommonExceptions
{
    public static AppException BadRequest(string message)
        => new("BAD_REQUEST", message, HttpStatusCode.BadRequest);

    public static AppException Unauthorized(string? message = null)
        => new("UNAUTHORIZED", message ?? "Unauthorized access", HttpStatusCode.Unauthorized);

    public static AppException Forbidden(string? message = null)
        => new("FORBIDDEN", message ?? "Access forbidden", HttpStatusCode.Forbidden);

    public static AppException NotFound(string resource)
        => new("NOT_FOUND", $"{resource} not found", HttpStatusCode.NotFound);

    public static AppException Conflict(string message)
        => new("CONFLICT", message, HttpStatusCode.Conflict);

    public static AppException ValidationError(string message, object? details = null)
        => new("VALIDATION_ERROR", message, HttpStatusCode.UnprocessableEntity, details);

    public static AppException InternalError(string? message = null)
        => new("INTERNAL_ERROR", message ?? "Internal server error", HttpStatusCode.InternalServerError);

    public static AppException ServiceUnavailable(string service)
        => new("SERVICE_UNAVAILABLE", $"{service} service unavailable", HttpStatusCode.ServiceUnavailable);

    public static AppException TooManyRequests(string? message = null)
        => new("TOO_MANY_REQUESTS", message ?? "Too many requests", HttpStatusCode.TooManyRequests);
}
