using Identity.Svc.Configurations;
using Microsoft.Extensions.Options;

namespace Identity.Svc.Services;

public sealed class OpenFgaProvisioner(
    IOptions<OpenFgaConfig> config,
    HttpClient client, IConfiguration configuration, ILogger<OpenFgaProvisioner> logger)
{
    public async Task AssignOrganizationAdminAsync(Guid organizationId, string userId, CancellationToken cancellationToken)
    {
        var baseUrl = config.Value.ServiceUrl;
        var token = config.Value.ProvisioningToken;
        if (string.IsNullOrWhiteSpace(baseUrl) || string.IsNullOrWhiteSpace(token))
        {
            logger.LogWarning("OpenFGA provisioning is not configured; no admin role was assigned for organization {OrganizationId}", organizationId);
            return;
        }

        using var request = new HttpRequestMessage(HttpMethod.Post,
            $"{baseUrl.TrimEnd('/')}/v1/internal/organizations/{organizationId}/bootstrap-admin/{userId}");
        request.Headers.Add("X-Internal-Token", token);
        using var response = await client.SendAsync(request, cancellationToken);
        response.EnsureSuccessStatusCode();
    }

    public async Task<bool> CanManageOrganizationAsync(Guid organizationId, string userId, CancellationToken cancellationToken)
    {
        if (config.Value.SkipChecksInDebug) return true;
        var baseUrl = configuration["Authz:ServiceUrl"];
        if (string.IsNullOrWhiteSpace(baseUrl)) return false;
        using var response = await client.PostAsJsonAsync($"{baseUrl.TrimEnd('/')}/v1/check", new
        {
            subjectId = userId,
            organizationId = organizationId.ToString(),
            permission = "list_users"
        }, cancellationToken);
        if (!response.IsSuccessStatusCode) return false;
        var decision = await response.Content.ReadFromJsonAsync<AuthzDecision>(cancellationToken: cancellationToken);
        return decision?.Allowed == true;
    }

    private sealed record AuthzDecision(bool Allowed);
}
