using System.Net;
using System.Net.Mail;
using Microsoft.Extensions.Options;

namespace Identity.Svc.Services;

public sealed class EmailOptions
{
    public string From { get; set; } = "";
    public SmtpOptions Smtp { get; set; } = new();
}

public sealed class SmtpOptions
{
    public string Host { get; set; } = "";
    public int Port { get; set; } = 587;
    public string UserName { get; set; } = "";
    public string Password { get; set; } = "";
    public bool EnableSsl { get; set; } = true;
}

public interface IAccountEmailSender
{
    Task SendAsync(string email, string subject, string htmlMessage, CancellationToken cancellationToken);
}

public sealed class SmtpAccountEmailSender(IOptions<EmailOptions> options, ILogger<SmtpAccountEmailSender> logger) : IAccountEmailSender
{
    public async Task SendAsync(string email, string subject, string htmlMessage, CancellationToken cancellationToken)
    {
        var settings = options.Value;
        if (string.IsNullOrWhiteSpace(settings.Smtp.Host))
        {
            logger.LogWarning("Email delivery is not configured. Subject: {Subject}; recipient: {Recipient}", subject, email);
            return;
        }

        using var client = new SmtpClient(settings.Smtp.Host, settings.Smtp.Port)
        {
            EnableSsl = settings.Smtp.EnableSsl,
            Credentials = string.IsNullOrWhiteSpace(settings.Smtp.UserName)
                ? CredentialCache.DefaultNetworkCredentials
                : new NetworkCredential(settings.Smtp.UserName, settings.Smtp.Password)
        };
        using var message = new MailMessage(settings.From, email, subject, htmlMessage) { IsBodyHtml = true };
        await client.SendMailAsync(message, cancellationToken);
    }
}
