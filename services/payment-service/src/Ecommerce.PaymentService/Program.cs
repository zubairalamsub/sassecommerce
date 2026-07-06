using System.Text;
using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Messaging;
using Ecommerce.PaymentService.Middleware;
using Ecommerce.PaymentService.Repositories;
using Ecommerce.PaymentService.Services;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.EntityFrameworkCore;
using Microsoft.IdentityModel.Tokens;
using Serilog;

var builder = WebApplication.CreateBuilder(args);

// Configure Serilog
Log.Logger = new LoggerConfiguration()
    .ReadFrom.Configuration(builder.Configuration)
    .Enrich.FromLogContext()
    .WriteTo.Console()
    .CreateLogger();

builder.Host.UseSerilog();

// Add services to the container
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(c =>
{
    c.SwaggerDoc("v1", new() { Title = "Payment Service API", Version = "v1" });
});

// Configure Database
var configuredConn = builder.Configuration.GetConnectionString("DefaultConnection");
var connectionString = !string.IsNullOrEmpty(configuredConn)
    ? configuredConn
    : $"Host={builder.Configuration["DB_HOST"] ?? "localhost"};" +
      $"Port={builder.Configuration["DB_PORT"] ?? "5432"};" +
      $"Database={builder.Configuration["DB_NAME"] ?? "payment_db"};" +
      $"Username={builder.Configuration["DB_USER"] ?? "postgres"};" +
      $"Password={builder.Configuration["DB_PASSWORD"] ?? "postgres"}";

builder.Services.AddDbContext<PaymentDbContext>(options =>
    options.UseNpgsql(connectionString));

// Configure AutoMapper
builder.Services.AddAutoMapper(typeof(Program));

// Register Repositories
builder.Services.AddScoped<IPaymentRepository, PaymentRepository>();
builder.Services.AddScoped<IPaymentMethodRepository, PaymentMethodRepository>();
builder.Services.AddScoped<IPaymentTransactionRepository, PaymentTransactionRepository>();
builder.Services.AddScoped<IRefundRepository, RefundRepository>();

// Register Kafka Event Publisher
builder.Services.AddSingleton<IEventPublisher, KafkaEventPublisher>();

// Register Kafka Event Consumer (background service)
builder.Services.AddHostedService<OrderEventConsumer>();

// Register Services
builder.Services.AddScoped<IPaymentService, PaymentService>();

// Register Payment Gateway — SSLCommerz for production, Simulated for development
var sslCommerzStoreId = builder.Configuration["SSLCOMMERZ_STORE_ID"];
if (!string.IsNullOrEmpty(sslCommerzStoreId))
{
    var paymentServiceBaseUrl = builder.Configuration["PAYMENT_SERVICE_URL"] ?? "http://localhost:8085";
    var sslCommerzConfig = new SslCommerzConfig
    {
        StoreId = sslCommerzStoreId,
        StorePassword = builder.Configuration["SSLCOMMERZ_STORE_PASSWORD"] ?? "",
        IsSandbox = builder.Configuration["SSLCOMMERZ_SANDBOX"]?.ToLower() != "false",
        SuccessUrl = $"{paymentServiceBaseUrl}/api/v1/payments/sslcommerz/success",
        FailUrl = $"{paymentServiceBaseUrl}/api/v1/payments/sslcommerz/fail",
        CancelUrl = $"{paymentServiceBaseUrl}/api/v1/payments/sslcommerz/cancel",
        IpnUrl = $"{paymentServiceBaseUrl}/api/v1/payments/sslcommerz/ipn",
    };

    builder.Services.AddSingleton(sslCommerzConfig);
    builder.Services.AddHttpClient<SslCommerzPaymentGateway>();
    builder.Services.AddScoped<IPaymentGateway, SslCommerzPaymentGateway>();

    Log.Information("Payment gateway: SSLCommerz ({Mode})", sslCommerzConfig.IsSandbox ? "Sandbox" : "Production");
}
else
{
    builder.Services.AddScoped<IPaymentGateway, SimulatedPaymentGateway>();
    Log.Information("Payment gateway: Simulated (set SSLCOMMERZ_STORE_ID to enable SSLCommerz)");
}

// Configure JWT Authentication
var jwtSecret = builder.Configuration["JWT_SECRET"]
    ?? throw new InvalidOperationException("JWT_SECRET is required but not set - refusing to start without a token signing secret");
if (Encoding.UTF8.GetByteCount(jwtSecret) < 32)
{
    throw new InvalidOperationException("JWT_SECRET must be at least 32 bytes");
}
builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuerSigningKey = true,
            IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwtSecret)),
            ValidateIssuer = false,
            ValidateAudience = false,
            ValidateLifetime = true,
            ClockSkew = TimeSpan.Zero
        };
    });

// Configure CORS. Origins come from CORS_ALLOWED_ORIGINS (comma-separated);
// outside production an empty value falls back to the localhost dev origins.
// Never AllowAnyOrigin: combined with credentials it permits cross-origin
// reads of authenticated responses.
var corsOrigins = (builder.Configuration["CORS_ALLOWED_ORIGINS"] ?? string.Empty)
    .Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
if (corsOrigins.Length == 0)
{
    if (builder.Environment.IsProduction())
    {
        throw new InvalidOperationException("CORS_ALLOWED_ORIGINS is required in production");
    }
    corsOrigins = new[] { "http://localhost:3000", "http://localhost:3001" };
}
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        policy.WithOrigins(corsOrigins)
              .AllowAnyMethod()
              .AllowAnyHeader();
    });
});

var app = builder.Build();

// Configure the HTTP request pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseRequestResponseLogging();

app.UseCors();

app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();

// Health check endpoints
app.MapGet("/health", () => Results.Ok(new
{
    status = "healthy",
    service = "payment-service",
    timestamp = DateTime.UtcNow
}));

app.MapGet("/ready", async (PaymentDbContext dbContext) =>
{
    try
    {
        await dbContext.Database.CanConnectAsync();
        return Results.Ok(new
        {
            status = "ready",
            service = "payment-service",
            database = "connected"
        });
    }
    catch
    {
        return Results.StatusCode(503);
    }
});

// Run database migrations
using (var scope = app.Services.CreateScope())
{
    var dbContext = scope.ServiceProvider.GetRequiredService<PaymentDbContext>();
    try
    {
        await dbContext.Database.EnsureCreatedAsync();
        Log.Information("Database migration completed successfully");
    }
    catch (Exception ex)
    {
        Log.Error(ex, "An error occurred while migrating the database");
    }
}

try
{
    Log.Information("Starting Payment Service");
    app.Run();
}
catch (Exception ex)
{
    Log.Fatal(ex, "Application terminated unexpectedly");
}
finally
{
    Log.CloseAndFlush();
}
