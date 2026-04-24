using Ecommerce.PaymentService.Services;
using Microsoft.AspNetCore.Mvc;

namespace Ecommerce.PaymentService.Controllers;

/// <summary>
/// Handles SSLCommerz IPN (Instant Payment Notification) and redirect callbacks.
/// These endpoints are called by SSLCommerz servers, not by our frontend directly.
/// No [Authorize] — SSLCommerz callbacks are verified via IPN hash signature.
/// </summary>
[ApiController]
[Route("api/v1/payments/sslcommerz")]
public class SslCommerzCallbackController : ControllerBase
{
    private readonly IPaymentService _paymentService;
    private readonly SslCommerzPaymentGateway? _gateway;
    private readonly ILogger<SslCommerzCallbackController> _logger;

    public SslCommerzCallbackController(
        IPaymentService paymentService,
        IPaymentGateway gateway,
        ILogger<SslCommerzCallbackController> logger)
    {
        _paymentService = paymentService;
        _gateway = gateway as SslCommerzPaymentGateway;
        _logger = logger;
    }

    /// <summary>
    /// IPN (Instant Payment Notification) — called by SSLCommerz server-to-server.
    /// This is the most reliable callback. Always validate the transaction here.
    /// </summary>
    [HttpPost("ipn")]
    public async Task<IActionResult> HandleIpn(CancellationToken cancellationToken)
    {
        if (_gateway == null)
        {
            _logger.LogWarning("SSLCommerz IPN received but gateway is not SSLCommerz");
            return Ok();
        }

        var form = await Request.ReadFormAsync(cancellationToken);
        var ipnData = form.ToDictionary(x => x.Key, x => x.Value.ToString());

        _logger.LogInformation("SSLCommerz IPN received: TranId={TranId}, Status={Status}",
            ipnData.GetValueOrDefault("tran_id", ""),
            ipnData.GetValueOrDefault("status", ""));

        // Verify IPN hash signature
        if (!_gateway.VerifyIpnHash(ipnData))
        {
            _logger.LogWarning("SSLCommerz IPN hash verification failed");
            return BadRequest("Invalid signature");
        }

        var tranId = ipnData.GetValueOrDefault("tran_id", "");
        var status = ipnData.GetValueOrDefault("status", "");
        var valId = ipnData.GetValueOrDefault("val_id", "");

        if (string.IsNullOrEmpty(tranId))
        {
            return BadRequest("Missing tran_id");
        }

        if (status == "VALID" || status == "VALIDATED")
        {
            // Validate with SSLCommerz API for extra security
            var validation = await _gateway.ValidateTransactionAsync(valId, cancellationToken);

            if (validation.IsValid)
            {
                var rawResponse = System.Text.Json.JsonSerializer.Serialize(ipnData);
                await _paymentService.CompleteGatewayPaymentAsync(
                    tranId, validation.BankTranId, validation.Amount, rawResponse, cancellationToken);

                _logger.LogInformation("SSLCommerz IPN processed successfully: TranId={TranId}", tranId);
            }
            else
            {
                _logger.LogWarning("SSLCommerz validation failed for IPN: TranId={TranId}, ValidationStatus={Status}",
                    tranId, validation.Status);

                var rawResponse = System.Text.Json.JsonSerializer.Serialize(ipnData);
                await _paymentService.FailGatewayPaymentAsync(tranId, "Validation failed: " + validation.Status, rawResponse, cancellationToken);
            }
        }
        else if (status == "FAILED")
        {
            var rawResponse = System.Text.Json.JsonSerializer.Serialize(ipnData);
            var reason = ipnData.GetValueOrDefault("error", "Payment failed at gateway");
            await _paymentService.FailGatewayPaymentAsync(tranId, reason, rawResponse, cancellationToken);
        }
        else if (status == "CANCELLED")
        {
            var rawResponse = System.Text.Json.JsonSerializer.Serialize(ipnData);
            await _paymentService.FailGatewayPaymentAsync(tranId, "Payment cancelled by customer", rawResponse, cancellationToken);
        }

        return Ok();
    }

    /// <summary>
    /// Success redirect — customer is redirected here after successful payment on SSLCommerz.
    /// Redirects to the frontend success page. The actual payment confirmation happens via IPN.
    /// </summary>
    [HttpPost("success")]
    public IActionResult HandleSuccess()
    {
        var tranId = Request.Form["tran_id"].ToString();
        _logger.LogInformation("SSLCommerz success redirect: TranId={TranId}", tranId);

        // Redirect to frontend success page
        var frontendUrl = Environment.GetEnvironmentVariable("FRONTEND_URL") ?? "http://localhost:3000";
        return Redirect($"{frontendUrl}/checkout/success?tran_id={tranId}");
    }

    /// <summary>
    /// Fail redirect — customer is redirected here after payment failure on SSLCommerz.
    /// </summary>
    [HttpPost("fail")]
    public IActionResult HandleFail()
    {
        var tranId = Request.Form["tran_id"].ToString();
        _logger.LogInformation("SSLCommerz fail redirect: TranId={TranId}", tranId);

        var frontendUrl = Environment.GetEnvironmentVariable("FRONTEND_URL") ?? "http://localhost:3000";
        return Redirect($"{frontendUrl}/checkout/failed?tran_id={tranId}");
    }

    /// <summary>
    /// Cancel redirect — customer cancelled payment on SSLCommerz page.
    /// </summary>
    [HttpPost("cancel")]
    public IActionResult HandleCancel()
    {
        var tranId = Request.Form["tran_id"].ToString();
        _logger.LogInformation("SSLCommerz cancel redirect: TranId={TranId}", tranId);

        var frontendUrl = Environment.GetEnvironmentVariable("FRONTEND_URL") ?? "http://localhost:3000";
        return Redirect($"{frontendUrl}/checkout/cancelled?tran_id={tranId}");
    }
}
