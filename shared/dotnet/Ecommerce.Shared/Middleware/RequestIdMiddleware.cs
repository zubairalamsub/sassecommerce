using Microsoft.AspNetCore.Http;

namespace Ecommerce.Shared.Middleware;

public class RequestIdMiddleware
{
    private readonly RequestDelegate _next;
    private const string RequestIdHeader = "X-Request-ID";

    public RequestIdMiddleware(RequestDelegate next)
    {
        _next = next;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        // Check if request ID already exists in header
        var requestId = context.Request.Headers[RequestIdHeader].FirstOrDefault();

        // Generate new request ID if not present
        if (string.IsNullOrEmpty(requestId))
        {
            requestId = Guid.NewGuid().ToString();
        }

        // Set request ID in context and response header
        context.Items["RequestId"] = requestId;
        context.Response.Headers.Add(RequestIdHeader, requestId);

        await _next(context);
    }
}

public static class RequestIdMiddlewareExtensions
{
    public static IApplicationBuilder UseRequestId(this IApplicationBuilder builder)
    {
        return builder.UseMiddleware<RequestIdMiddleware>();
    }

    public static string? GetRequestId(this HttpContext context)
    {
        return context.Items["RequestId"] as string;
    }
}
