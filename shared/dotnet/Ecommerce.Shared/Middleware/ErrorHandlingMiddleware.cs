using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Logging;
using Ecommerce.Shared.Exceptions;
using System.Net;
using System.Text.Json;

namespace Ecommerce.Shared.Middleware;

public class ErrorHandlingMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<ErrorHandlingMiddleware> _logger;

    public ErrorHandlingMiddleware(RequestDelegate next, ILogger<ErrorHandlingMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await _next(context);
        }
        catch (Exception ex)
        {
            await HandleExceptionAsync(context, ex);
        }
    }

    private async Task HandleExceptionAsync(HttpContext context, Exception exception)
    {
        var requestId = context.GetRequestId();
        var tenantId = context.GetTenantId();

        _logger.LogError(exception,
            "Error occurred. RequestId: {RequestId}, TenantId: {TenantId}, Method: {Method}, Path: {Path}",
            requestId, tenantId, context.Request.Method, context.Request.Path);

        var statusCode = HttpStatusCode.InternalServerError;
        var errorCode = "INTERNAL_ERROR";
        var message = "An internal server error occurred";
        object? details = null;

        // Handle specific exception types
        if (exception is AppException appException)
        {
            statusCode = appException.StatusCode;
            errorCode = appException.ErrorCode;
            message = appException.Message;
            details = appException.Details;
        }
        else if (exception is UnauthorizedAccessException)
        {
            statusCode = HttpStatusCode.Unauthorized;
            errorCode = "UNAUTHORIZED";
            message = "Unauthorized access";
        }
        else if (exception is ArgumentException or ArgumentNullException)
        {
            statusCode = HttpStatusCode.BadRequest;
            errorCode = "BAD_REQUEST";
            message = exception.Message;
        }

        context.Response.StatusCode = (int)statusCode;
        context.Response.ContentType = "application/json";

        var response = new
        {
            success = false,
            error = message,
            code = errorCode,
            details,
            request_id = requestId
        };

        var options = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase
        };

        await context.Response.WriteAsJsonAsync(response, options);
    }
}

public static class ErrorHandlingMiddlewareExtensions
{
    public static IApplicationBuilder UseErrorHandling(this IApplicationBuilder builder)
    {
        return builder.UseMiddleware<ErrorHandlingMiddleware>();
    }
}
