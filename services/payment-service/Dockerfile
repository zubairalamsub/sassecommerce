# Build stage
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy csproj and restore dependencies
COPY ["src/Ecommerce.PaymentService/Ecommerce.PaymentService.csproj", "Ecommerce.PaymentService/"]
RUN dotnet restore "Ecommerce.PaymentService/Ecommerce.PaymentService.csproj"

# Copy everything else and build
COPY src/Ecommerce.PaymentService/. Ecommerce.PaymentService/
WORKDIR "/src/Ecommerce.PaymentService"
RUN dotnet build "Ecommerce.PaymentService.csproj" -c Release -o /app/build

# Publish stage
FROM build AS publish
RUN dotnet publish "Ecommerce.PaymentService.csproj" -c Release -o /app/publish /p:UseAppHost=false

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS final
WORKDIR /app
EXPOSE 8085

# Copy published app
COPY --from=publish /app/publish .

# Set environment variables
ENV ASPNETCORE_URLS=http://+:8085
ENV ASPNETCORE_ENVIRONMENT=Production

ENTRYPOINT ["dotnet", "Ecommerce.PaymentService.dll"]
