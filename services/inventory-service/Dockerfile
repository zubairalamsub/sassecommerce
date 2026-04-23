# Build stage
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy csproj and restore dependencies
COPY ["src/Ecommerce.InventoryService/Ecommerce.InventoryService.csproj", "Ecommerce.InventoryService/"]
RUN dotnet restore "Ecommerce.InventoryService/Ecommerce.InventoryService.csproj"

# Copy everything else and build
COPY src/Ecommerce.InventoryService/. Ecommerce.InventoryService/
WORKDIR "/src/Ecommerce.InventoryService"
RUN dotnet build "Ecommerce.InventoryService.csproj" -c Release -o /app/build

# Publish stage
FROM build AS publish
RUN dotnet publish "Ecommerce.InventoryService.csproj" -c Release -o /app/publish /p:UseAppHost=false

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS final
WORKDIR /app
EXPOSE 8084

# Copy published app
COPY --from=publish /app/publish .

# Set environment variables
ENV ASPNETCORE_URLS=http://+:8084
ENV ASPNETCORE_ENVIRONMENT=Production

ENTRYPOINT ["dotnet", "Ecommerce.InventoryService.dll"]
