using Microsoft.Extensions.Configuration;

namespace Ecommerce.Shared.Configuration;

public static class ConfigurationExtensions
{
    public static string GetRequiredValue(this IConfiguration configuration, string key)
    {
        var value = configuration[key];
        if (string.IsNullOrEmpty(value))
        {
            throw new InvalidOperationException($"Configuration key '{key}' is required but not found");
        }
        return value;
    }

    public static T GetRequiredSection<T>(this IConfiguration configuration, string sectionName)
        where T : class, new()
    {
        var section = configuration.GetSection(sectionName);
        if (!section.Exists())
        {
            throw new InvalidOperationException($"Configuration section '{sectionName}' is required but not found");
        }

        var config = new T();
        section.Bind(config);
        return config;
    }

    public static bool IsProduction(this IConfiguration configuration)
    {
        var environment = configuration.GetValue<string>("ASPNETCORE_ENVIRONMENT") ?? "Development";
        return environment.Equals("Production", StringComparison.OrdinalIgnoreCase);
    }

    public static bool IsDevelopment(this IConfiguration configuration)
    {
        var environment = configuration.GetValue<string>("ASPNETCORE_ENVIRONMENT") ?? "Development";
        return environment.Equals("Development", StringComparison.OrdinalIgnoreCase);
    }

    public static bool IsTest(this IConfiguration configuration)
    {
        var environment = configuration.GetValue<string>("ASPNETCORE_ENVIRONMENT") ?? "Development";
        return environment.Equals("Test", StringComparison.OrdinalIgnoreCase) ||
               environment.Equals("Testing", StringComparison.OrdinalIgnoreCase);
    }

    public static string GetEnvironment(this IConfiguration configuration)
    {
        return configuration.GetValue<string>("ASPNETCORE_ENVIRONMENT") ?? "Development";
    }

    public static string[] GetStringArray(this IConfiguration configuration, string key, char separator = ',')
    {
        var value = configuration[key];
        if (string.IsNullOrEmpty(value))
        {
            return Array.Empty<string>();
        }

        return value.Split(separator, StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
    }
}
