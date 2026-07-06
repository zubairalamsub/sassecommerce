using System.Text;
using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Messaging;
using Ecommerce.InventoryService.Repositories;
using Ecommerce.InventoryService.Services;
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
    c.SwaggerDoc("v1", new() { Title = "Inventory Service API", Version = "v1" });
});

// Configure Database
var configuredConn = builder.Configuration.GetConnectionString("DefaultConnection");
var connectionString = !string.IsNullOrEmpty(configuredConn)
    ? configuredConn
    : $"Host={builder.Configuration["DB_HOST"] ?? "localhost"};" +
      $"Port={builder.Configuration["DB_PORT"] ?? "5432"};" +
      $"Database={builder.Configuration["DB_NAME"] ?? "inventory_db"};" +
      $"Username={builder.Configuration["DB_USER"] ?? "postgres"};" +
      $"Password={builder.Configuration["DB_PASSWORD"] ?? "postgres"}";

builder.Services.AddDbContext<InventoryDbContext>(options =>
    options.UseNpgsql(connectionString));

// Configure AutoMapper
builder.Services.AddAutoMapper(typeof(Program));

// Register Repositories
builder.Services.AddScoped<IWarehouseRepository, WarehouseRepository>();
builder.Services.AddScoped<IInventoryRepository, InventoryRepository>();
builder.Services.AddScoped<IStockMovementRepository, StockMovementRepository>();
builder.Services.AddScoped<IStockReservationRepository, StockReservationRepository>();

// Register Kafka Event Publisher
builder.Services.AddSingleton<IEventPublisher, KafkaEventPublisher>();

// Register Kafka Event Consumer (background service)
builder.Services.AddHostedService<OrderEventConsumer>();

// Register Services
builder.Services.AddScoped<IInventoryService, InventoryService>();

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

// Security headers land first so they apply to every response — including
// errors raised by middleware further down the pipeline. Values mirror the
// shared Go SecurityHeaders middleware defaults for a JSON-only API.
app.Use(async (context, next) =>
{
    var headers = context.Response.Headers;
    headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains";
    headers["X-Content-Type-Options"] = "nosniff";
    headers["X-Frame-Options"] = "DENY";
    headers["Referrer-Policy"] = "strict-origin-when-cross-origin";
    headers["Permissions-Policy"] = "camera=(), microphone=(), geolocation=(), interest-cohort=()";
    headers["X-XSS-Protection"] = "0";
    headers["Content-Security-Policy"] = "default-src 'none'; frame-ancestors 'none'";
    await next();
});

// Configure the HTTP request pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseSerilogRequestLogging();

app.UseCors();

app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();

// Health check endpoints
app.MapGet("/health", () => Results.Ok(new
{
    status = "healthy",
    service = "inventory-service",
    timestamp = DateTime.UtcNow
}));

app.MapGet("/ready", async (InventoryDbContext dbContext) =>
{
    try
    {
        await dbContext.Database.CanConnectAsync();
        return Results.Ok(new
        {
            status = "ready",
            service = "inventory-service",
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
    var dbContext = scope.ServiceProvider.GetRequiredService<InventoryDbContext>();
    try
    {
        await dbContext.Database.EnsureCreatedAsync();
        Log.Information("Database schema ensured successfully");
    }
    catch (Exception ex)
    {
        Log.Error(ex, "An error occurred while migrating the database");
    }
}

try
{
    Log.Information("Starting Inventory Service");
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
