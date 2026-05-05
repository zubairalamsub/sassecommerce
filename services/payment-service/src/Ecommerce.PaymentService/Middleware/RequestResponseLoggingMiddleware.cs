using System.Diagnostics;
using System.Text;

namespace Ecommerce.PaymentService.Middleware;

public class RequestResponseLoggingMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<RequestResponseLoggingMiddleware> _logger;
    private const int MaxBodySize = 10 * 1024; // 10 KB

    private static readonly HashSet<string> SensitiveHeaders = new(StringComparer.OrdinalIgnoreCase)
    {
        "Authorization", "Cookie", "X-Api-Key"
    };

    private static readonly HashSet<string> SkipPaths = new(StringComparer.OrdinalIgnoreCase)
    {
        "/health", "/ready"
    };

    public RequestResponseLoggingMiddleware(RequestDelegate next, ILogger<RequestResponseLoggingMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        if (SkipPaths.Contains(context.Request.Path.Value ?? ""))
        {
            await _next(context);
            return;
        }

        var stopwatch = Stopwatch.StartNew();
        var requestId = context.TraceIdentifier;

        // Read request body
        context.Request.EnableBuffering();
        var requestBody = await ReadBodyAsync(context.Request.Body);
        context.Request.Body.Position = 0;

        // Capture request headers
        var requestHeaders = GetSafeHeaders(context.Request.Headers);

        // Capture response body
        var originalResponseBody = context.Response.Body;
        using var responseBodyStream = new MemoryStream();
        context.Response.Body = responseBodyStream;

        try
        {
            await _next(context);
        }
        finally
        {
            stopwatch.Stop();

            responseBodyStream.Seek(0, SeekOrigin.Begin);
            var responseBody = await ReadBodyAsync(responseBodyStream);
            responseBodyStream.Seek(0, SeekOrigin.Begin);
            await responseBodyStream.CopyToAsync(originalResponseBody);
            context.Response.Body = originalResponseBody;

            var statusCode = context.Response.StatusCode;
            var logLevel = statusCode >= 500 ? LogLevel.Error
                : statusCode >= 400 ? LogLevel.Warning
                : LogLevel.Information;

            _logger.Log(logLevel,
                "HTTP {Method} {Path}{Query} responded {StatusCode} in {Duration}ms | " +
                "RequestId: {RequestId} | ClientIP: {ClientIP} | " +
                "ReqHeaders: {RequestHeaders} | ReqBody: {RequestBody} | " +
                "RespBody: {ResponseBody}",
                context.Request.Method,
                context.Request.Path.Value,
                context.Request.QueryString.Value,
                statusCode,
                stopwatch.ElapsedMilliseconds,
                requestId,
                context.Connection.RemoteIpAddress?.ToString(),
                requestHeaders,
                Truncate(requestBody),
                Truncate(responseBody));
        }
    }

    private static async Task<string> ReadBodyAsync(Stream stream)
    {
        using var reader = new StreamReader(stream, Encoding.UTF8, leaveOpen: true);
        var buffer = new char[MaxBodySize];
        var charsRead = await reader.ReadAsync(buffer, 0, MaxBodySize);
        return new string(buffer, 0, charsRead);
    }

    private static Dictionary<string, string> GetSafeHeaders(IHeaderDictionary headers)
    {
        var result = new Dictionary<string, string>();
        foreach (var header in headers)
        {
            result[header.Key] = SensitiveHeaders.Contains(header.Key)
                ? "[REDACTED]"
                : header.Value.ToString();
        }
        return result;
    }

    private static string Truncate(string value)
    {
        if (string.IsNullOrEmpty(value)) return "";
        return value.Length > MaxBodySize
            ? value[..MaxBodySize] + "...[truncated]"
            : value;
    }
}

public static class RequestResponseLoggingExtensions
{
    public static IApplicationBuilder UseRequestResponseLogging(this IApplicationBuilder app)
    {
        return app.UseMiddleware<RequestResponseLoggingMiddleware>();
    }
}
