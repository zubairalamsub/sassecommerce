using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Messaging;
using Ecommerce.PaymentService.Repositories;
using Ecommerce.PaymentService.Services;
using Microsoft.EntityFrameworkCore;
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
var connectionString = builder.Configuration.GetConnectionString("DefaultConnection")
    ?? $"Host={builder.Configuration["DB_HOST"] ?? "localhost"};" +
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

// Register Services
builder.Services.AddScoped<IPaymentService, PaymentService>();
builder.Services.AddScoped<IPaymentGateway, SimulatedPaymentGateway>();

// Configure CORS
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(builder =>
    {
        builder.AllowAnyOrigin()
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

app.UseSerilogRequestLogging();

app.UseCors();

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
        await dbContext.Database.MigrateAsync();
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
