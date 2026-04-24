using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace Ecommerce.PaymentService.Services;

public class SslCommerzConfig
{
    public string StoreId { get; set; } = string.Empty;
    public string StorePassword { get; set; } = string.Empty;
    public bool IsSandbox { get; set; } = true;
    public string SuccessUrl { get; set; } = string.Empty;
    public string FailUrl { get; set; } = string.Empty;
    public string CancelUrl { get; set; } = string.Empty;
    public string IpnUrl { get; set; } = string.Empty;

    public string BaseUrl => IsSandbox
        ? "https://sandbox.sslcommerz.com"
        : "https://securepay.sslcommerz.com";
}

/// <summary>
/// SSLCommerz payment gateway integration for Bangladesh.
/// Supports bKash, Nagad, Rocket, bank cards, internet banking.
/// Uses redirect-based flow: initiate session → redirect customer → IPN callback.
/// </summary>
public class SslCommerzPaymentGateway : IPaymentGateway
{
    private readonly HttpClient _httpClient;
    private readonly SslCommerzConfig _config;
    private readonly ILogger<SslCommerzPaymentGateway> _logger;

    public SslCommerzPaymentGateway(
        HttpClient httpClient,
        SslCommerzConfig config,
        ILogger<SslCommerzPaymentGateway> logger)
    {
        _httpClient = httpClient;
        _config = config;
        _logger = logger;
    }

    public string Name => "SSLCommerz";

    public async Task<GatewayResponse> ChargeAsync(GatewayChargeRequest request, CancellationToken cancellationToken = default)
    {
        var tranId = $"TXN_{DateTime.UtcNow:yyyyMMddHHmmss}_{Guid.NewGuid():N}"[..32];

        _logger.LogInformation("Initiating SSLCommerz session: TranId={TranId}, Amount={Amount} {Currency}, Order={OrderId}",
            tranId, request.Amount, request.Currency, request.OrderId);

        var formData = new Dictionary<string, string>
        {
            ["store_id"] = _config.StoreId,
            ["store_passwd"] = _config.StorePassword,
            ["total_amount"] = request.Amount.ToString("F2"),
            ["currency"] = request.Currency ?? "BDT",
            ["tran_id"] = tranId,
            ["success_url"] = _config.SuccessUrl,
            ["fail_url"] = _config.FailUrl,
            ["cancel_url"] = _config.CancelUrl,
            ["ipn_url"] = _config.IpnUrl,
            ["cus_name"] = request.CustomerId ?? "Customer",
            ["cus_email"] = "customer@example.com",
            ["cus_phone"] = "01700000000",
            ["cus_add1"] = "Dhaka",
            ["cus_city"] = "Dhaka",
            ["cus_country"] = "Bangladesh",
            ["shipping_method"] = "NO",
            ["product_name"] = request.Description ?? $"Order #{request.OrderId}",
            ["product_category"] = "ecommerce",
            ["product_profile"] = "general",
            ["value_a"] = request.Metadata.GetValueOrDefault("tenant_id", ""),
            ["value_b"] = request.Metadata.GetValueOrDefault("payment_id", ""),
            ["value_c"] = request.OrderId ?? "",
            ["value_d"] = request.CustomerId ?? "",
        };

        try
        {
            var content = new FormUrlEncodedContent(formData);
            var response = await _httpClient.PostAsync(
                $"{_config.BaseUrl}/gwprocess/v4/api.php",
                content,
                cancellationToken);

            var responseBody = await response.Content.ReadAsStringAsync(cancellationToken);

            _logger.LogDebug("SSLCommerz init response: {Response}", responseBody);

            var jsonDoc = JsonDocument.Parse(responseBody);
            var root = jsonDoc.RootElement;

            var status = root.GetProperty("status").GetString();

            if (status == "SUCCESS")
            {
                var gatewayPageUrl = root.GetProperty("GatewayPageURL").GetString();
                var sessionKey = root.GetProperty("sessionkey").GetString();

                _logger.LogInformation("SSLCommerz session created: TranId={TranId}, SessionKey={SessionKey}",
                    tranId, sessionKey);

                return new GatewayResponse
                {
                    Success = true,
                    TransactionId = tranId,
                    RedirectUrl = gatewayPageUrl,
                    RawResponse = responseBody
                };
            }
            else
            {
                var failReason = root.TryGetProperty("failedreason", out var fr) ? fr.GetString() : "Session creation failed";

                _logger.LogWarning("SSLCommerz session failed: TranId={TranId}, Reason={Reason}", tranId, failReason);

                return new GatewayResponse
                {
                    Success = false,
                    TransactionId = tranId,
                    ErrorCode = "session_failed",
                    ErrorMessage = failReason,
                    RawResponse = responseBody
                };
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "SSLCommerz API call failed: TranId={TranId}", tranId);

            return new GatewayResponse
            {
                Success = false,
                TransactionId = tranId,
                ErrorCode = "api_error",
                ErrorMessage = ex.Message
            };
        }
    }

    public async Task<GatewayResponse> RefundAsync(GatewayRefundRequest request, CancellationToken cancellationToken = default)
    {
        var refundId = $"REF_{DateTime.UtcNow:yyyyMMddHHmmss}_{Guid.NewGuid():N}"[..32];

        _logger.LogInformation("Initiating SSLCommerz refund: BankTranId={TransactionId}, Amount={Amount}",
            request.TransactionId, request.Amount);

        var formData = new Dictionary<string, string>
        {
            ["store_id"] = _config.StoreId,
            ["store_passwd"] = _config.StorePassword,
            ["bank_tran_id"] = request.TransactionId,
            ["refund_amount"] = request.Amount.ToString("F2"),
            ["refund_remarks"] = request.Reason ?? "Customer refund request",
            ["refe_id"] = refundId
        };

        try
        {
            var content = new FormUrlEncodedContent(formData);
            var response = await _httpClient.PostAsync(
                $"{_config.BaseUrl}/validator/api/merchantTransIDvalidationAPI.php",
                content,
                cancellationToken);

            var responseBody = await response.Content.ReadAsStringAsync(cancellationToken);
            var jsonDoc = JsonDocument.Parse(responseBody);
            var root = jsonDoc.RootElement;

            var apiConnect = root.TryGetProperty("APIConnect", out var ac) ? ac.GetString() : "";

            if (apiConnect == "DONE")
            {
                _logger.LogInformation("SSLCommerz refund initiated: RefundId={RefundId}", refundId);

                return new GatewayResponse
                {
                    Success = true,
                    TransactionId = refundId,
                    RawResponse = responseBody
                };
            }
            else
            {
                var errorMsg = root.TryGetProperty("errorReason", out var er) ? er.GetString() : "Refund failed";

                return new GatewayResponse
                {
                    Success = false,
                    TransactionId = refundId,
                    ErrorCode = "refund_failed",
                    ErrorMessage = errorMsg,
                    RawResponse = responseBody
                };
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "SSLCommerz refund API call failed");

            return new GatewayResponse
            {
                Success = false,
                TransactionId = refundId,
                ErrorCode = "api_error",
                ErrorMessage = ex.Message
            };
        }
    }

    public Task<GatewayResponse> VoidAsync(string transactionId, CancellationToken cancellationToken = default)
    {
        // SSLCommerz doesn't support void — use refund for full amount
        _logger.LogWarning("Void not supported by SSLCommerz, use refund instead. TransactionId={TransactionId}", transactionId);

        return Task.FromResult(new GatewayResponse
        {
            Success = false,
            TransactionId = transactionId,
            ErrorCode = "not_supported",
            ErrorMessage = "SSLCommerz does not support void. Use refund instead."
        });
    }

    public Task<string> TokenizeCardAsync(string cardNumber, int expiryMonth, int expiryYear, string cvv, CancellationToken cancellationToken = default)
    {
        // SSLCommerz handles card input on their hosted page — no direct tokenization
        _logger.LogInformation("SSLCommerz uses hosted payment page — card tokenization handled by gateway");
        var token = $"sslcz_{Guid.NewGuid():N}";
        return Task.FromResult(token);
    }

    /// <summary>
    /// Validates a transaction with SSLCommerz using the val_id from IPN callback.
    /// Call this to confirm that a payment notification is authentic.
    /// </summary>
    public async Task<SslCommerzValidationResult> ValidateTransactionAsync(string valId, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Validating SSLCommerz transaction: ValId={ValId}", valId);

        try
        {
            var url = $"{_config.BaseUrl}/validator/api/validationserverAPI.php" +
                      $"?val_id={valId}&store_id={_config.StoreId}&store_passwd={_config.StorePassword}&format=json";

            var response = await _httpClient.GetAsync(url, cancellationToken);
            var responseBody = await response.Content.ReadAsStringAsync(cancellationToken);

            _logger.LogDebug("SSLCommerz validation response: {Response}", responseBody);

            var jsonDoc = JsonDocument.Parse(responseBody);
            var root = jsonDoc.RootElement;

            var status = root.TryGetProperty("status", out var s) ? s.GetString() : "";
            var tranId = root.TryGetProperty("tran_id", out var t) ? t.GetString() : "";
            var amount = root.TryGetProperty("amount", out var a) ? a.GetString() : "0";
            var currency = root.TryGetProperty("currency", out var c) ? c.GetString() : "BDT";
            var bankTranId = root.TryGetProperty("bank_tran_id", out var b) ? b.GetString() : "";
            var cardType = root.TryGetProperty("card_type", out var ct) ? ct.GetString() : "";
            var cardBrand = root.TryGetProperty("card_brand", out var cb) ? cb.GetString() : "";

            return new SslCommerzValidationResult
            {
                IsValid = status == "VALID" || status == "VALIDATED",
                Status = status ?? "",
                TranId = tranId ?? "",
                BankTranId = bankTranId ?? "",
                Amount = decimal.TryParse(amount, out var amt) ? amt : 0,
                Currency = currency ?? "BDT",
                CardType = cardType ?? "",
                CardBrand = cardBrand ?? "",
                RawResponse = responseBody
            };
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "SSLCommerz validation failed: ValId={ValId}", valId);

            return new SslCommerzValidationResult
            {
                IsValid = false,
                Status = "ERROR",
                RawResponse = ex.Message
            };
        }
    }

    /// <summary>
    /// Verifies the IPN hash to ensure the callback is genuinely from SSLCommerz.
    /// </summary>
    public bool VerifyIpnHash(Dictionary<string, string> ipnData)
    {
        if (!ipnData.TryGetValue("verify_sign", out var receivedSign) || string.IsNullOrEmpty(receivedSign))
            return false;

        if (!ipnData.TryGetValue("verify_key", out var verifyKey) || string.IsNullOrEmpty(verifyKey))
            return false;

        var keyFields = verifyKey.Split(',');
        var dataToHash = new StringBuilder();

        foreach (var field in keyFields.OrderBy(f => f))
        {
            if (ipnData.TryGetValue(field, out var value))
            {
                if (dataToHash.Length > 0) dataToHash.Append('&');
                dataToHash.Append($"{field}={value}");
            }
        }

        dataToHash.Append($"&store_passwd={ComputeMd5(_config.StorePassword)}");

        var computedSign = ComputeMd5(dataToHash.ToString());
        return string.Equals(computedSign, receivedSign, StringComparison.OrdinalIgnoreCase);
    }

    private static string ComputeMd5(string input)
    {
        var bytes = MD5.HashData(Encoding.UTF8.GetBytes(input));
        return Convert.ToHexString(bytes).ToLowerInvariant();
    }
}

public class SslCommerzValidationResult
{
    public bool IsValid { get; set; }
    public string Status { get; set; } = string.Empty;
    public string TranId { get; set; } = string.Empty;
    public string BankTranId { get; set; } = string.Empty;
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";
    public string CardType { get; set; } = string.Empty;
    public string CardBrand { get; set; } = string.Empty;
    public string RawResponse { get; set; } = string.Empty;
}
