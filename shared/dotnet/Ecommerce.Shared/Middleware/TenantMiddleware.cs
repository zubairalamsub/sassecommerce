using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Logging;

namespace Ecommerce.Shared.Middleware;

public class TenantMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<TenantMiddleware> _logger;
    private readonly TenantMiddlewareOptions _options;

    private const string TenantIdHeader = "X-Tenant-ID";
    private const string TenantSlugHeader = "X-Tenant-Slug";

    public TenantMiddleware(
        RequestDelegate next,
        ILogger<TenantMiddleware> logger,
        TenantMiddlewareOptions options)
    {
        _next = next;
        _logger = logger;
        _options = options;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        string? tenantId = null;
        string? tenantSlug = null;

        // 1. Try to get from header
        if (_options.AllowHeader)
        {
            tenantId = context.Request.Headers[TenantIdHeader].FirstOrDefault();
            tenantSlug = context.Request.Headers[TenantSlugHeader].FirstOrDefault();
        }

        // 2. Try to get from subdomain
        if (string.IsNullOrEmpty(tenantId) && _options.AllowSubdomain)
        {
            var host = context.Request.Host.Host;
            var parts = host.Split('.');
            if (parts.Length > 2)
            {
                // Extract subdomain (e.g., "tenant1.api.example.com" -> "tenant1")
                tenantSlug = parts[0];
            }
        }

        // 3. Try to get from route parameter
        if (string.IsNullOrEmpty(tenantId) && _options.AllowPath)
        {
            if (context.Request.RouteValues.TryGetValue(_options.PathParam, out var routeTenantId))
            {
                tenantId = routeTenantId?.ToString();
            }
        }

        // Check if tenant ID is required
        if (_options.Required && string.IsNullOrEmpty(tenantId) && string.IsNullOrEmpty(tenantSlug))
        {
            _logger.LogWarning("Tenant identification required but not provided");
            context.Response.StatusCode = StatusCodes.Status400BadRequest;
            context.Response.ContentType = "application/json";
            await context.Response.WriteAsJsonAsync(new
            {
                error = "Tenant identification required",
                details = "Provide tenant ID via X-Tenant-ID header, subdomain, or path parameter"
            });
            return;
        }

        // Set tenant information in context
        if (!string.IsNullOrEmpty(tenantId))
        {
            context.Items["TenantId"] = tenantId;
        }
        if (!string.IsNullOrEmpty(tenantSlug))
        {
            context.Items["TenantSlug"] = tenantSlug;
        }

        await _next(context);
    }
}

public class TenantMiddlewareOptions
{
    public bool Required { get; set; } = true;
    public bool AllowHeader { get; set; } = true;
    public bool AllowSubdomain { get; set; } = true;
    public bool AllowPath { get; set; } = true;
    public string PathParam { get; set; } = "tenantId";
}

public static class TenantMiddlewareExtensions
{
    public static IApplicationBuilder UseTenant(
        this IApplicationBuilder builder,
        Action<TenantMiddlewareOptions>? configure = null)
    {
        var options = new TenantMiddlewareOptions();
        configure?.Invoke(options);

        return builder.UseMiddleware<TenantMiddleware>(options);
    }

    public static string? GetTenantId(this HttpContext context)
    {
        return context.Items["TenantId"] as string;
    }

    public static string? GetTenantSlug(this HttpContext context)
    {
        return context.Items["TenantSlug"] as string;
    }
}
