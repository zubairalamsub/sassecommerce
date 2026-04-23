using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Repositories;
using Ecommerce.InventoryService.Services;
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
    c.SwaggerDoc("v1", new() { Title = "Inventory Service API", Version = "v1" });
});

// Configure Database
var connectionString = builder.Configuration.GetConnectionString("DefaultConnection")
    ?? $"Host={builder.Configuration["DB_HOST"] ?? "localhost"};" +
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

// Register Services
builder.Services.AddScoped<IInventoryService, InventoryService>();

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
