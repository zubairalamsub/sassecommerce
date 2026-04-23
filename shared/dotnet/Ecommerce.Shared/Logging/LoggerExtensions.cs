using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using Serilog;
using Serilog.Events;
using Serilog.Formatting.Compact;

namespace Ecommerce.Shared.Logging;

public static class LoggerExtensions
{
    public static IServiceCollection AddCustomLogging(
        this IServiceCollection services,
        IConfiguration configuration)
    {
        var logLevel = configuration.GetValue<string>("Logging:LogLevel:Default") ?? "Information";
        var logFormat = configuration.GetValue<string>("Logging:Format") ?? "json";
        var logOutput = configuration.GetValue<string>("Logging:Output") ?? "console";
        var serviceName = configuration.GetValue<string>("ServiceName") ?? "app";

        var logEventLevel = Enum.Parse<LogEventLevel>(logLevel, true);

        var loggerConfiguration = new LoggerConfiguration()
            .MinimumLevel.Is(logEventLevel)
            .Enrich.FromLogContext()
            .Enrich.WithProperty("Service", serviceName)
            .Enrich.WithProperty("Environment", Environment.GetEnvironmentVariable("ASPNETCORE_ENVIRONMENT") ?? "Development");

        if (logFormat.Equals("json", StringComparison.OrdinalIgnoreCase))
        {
            if (logOutput.Equals("console", StringComparison.OrdinalIgnoreCase))
            {
                loggerConfiguration.WriteTo.Console(new CompactJsonFormatter());
            }
            else
            {
                loggerConfiguration.WriteTo.File(
                    new CompactJsonFormatter(),
                    logOutput,
                    rollingInterval: RollingInterval.Day);
            }
        }
        else
        {
            if (logOutput.Equals("console", StringComparison.OrdinalIgnoreCase))
            {
                loggerConfiguration.WriteTo.Console(
                    outputTemplate: "[{Timestamp:HH:mm:ss} {Level:u3}] {Message:lj} {Properties:j}{NewLine}{Exception}");
            }
            else
            {
                loggerConfiguration.WriteTo.File(
                    logOutput,
                    rollingInterval: RollingInterval.Day,
                    outputTemplate: "[{Timestamp:HH:mm:ss} {Level:u3}] {Message:lj} {Properties:j}{NewLine}{Exception}");
            }
        }

        Log.Logger = loggerConfiguration.CreateLogger();

        services.AddLogging(loggingBuilder =>
        {
            loggingBuilder.ClearProviders();
            loggingBuilder.AddSerilog(dispose: true);
        });

        return services;
    }
}
